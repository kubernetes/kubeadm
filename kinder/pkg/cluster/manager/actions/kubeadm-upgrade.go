/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes"
	"k8s.io/kubeadm/kinder/pkg/kubeadm"
)

// KubeadmUpgrade executes the kubeadm upgrade workflow, including also deployment of new
// kubeadm/kubelet/kubectl binaries; for sake of simplicity, drain/uncordon when upgrading nodes
// is not executed.
//
// The implementation assumes that the kubeadm/kubelet/kubectl binaries and all the necessary images
// for the new kubernetes version are available in the /kinder/upgrade/{version} folder.
func KubeadmUpgrade(c *status.Cluster, upgradeVersion *version.Version, patchesDir string, wait time.Duration, vLevel int) (err error) {
	if upgradeVersion == nil {
		return errors.New("kubeadm-upgrade actions requires the --upgrade-version parameter to be set")
	}

	nodeList := c.K8sNodes().EligibleForActions()

	for _, n := range nodeList {
		preloadNodeUpgradeImages(n, upgradeVersion)

		if err := copyPatchesToNode(n, patchesDir); err != nil {
			return err
		}

		// Check if the upgrade version provided on the CLI is different from what is on the node image.
		// If there is a difference print a warning and fallback to what is on the node image.
		// This is useful in debug scenarios where the ci/latest version label changed during
		// debugging a particular workflow.
		versionPath := filepath.Join("/kinder", "upgrade", "version")
		out, err := n.Command("cat", versionPath).Silent().RunAndCapture()
		if err != nil {
			return errors.Wrapf(err, "could not compare %s file before upgrade", versionPath)
		}
		if len(out) != 1 {
			return errors.Errorf("expected %s to have 1 line, got %d", versionPath, len(out))
		}
		nodeVersion := version.MustParseSemantic(out[0])
		cmp, err := nodeVersion.Compare(upgradeVersion.String())
		if err != nil {
			return errors.Wrapf(err, "cannot compare %s to provided upgrade version", versionPath)
		}
		if cmp != 0 {
			log.Warnf("provided upgrade version is %s, but the node has %s, using the node version",
				upgradeVersion, nodeVersion)
			upgradeVersion = nodeVersion
		}

		if err := upgradeKubeadmBinary(n, upgradeVersion); err != nil {
			return err
		}

		// prepares the kubeadm config on this node
		if err := KubeadmUpgradeConfig(c, upgradeVersion, n); err != nil {
			return err
		}

		v, err := n.KubeadmVersion()
		if err != nil {
			errors.Wrap(err, "could not obtain the kubeadm version before calling kubeadm reset")
		}
		kubeadmConfigVersion := kubeadm.GetKubeadmConfigVersion(v)

		if n.Name() == c.BootstrapControlPlane().Name() {
			if err := kubeadmUpgradePlan(c, n, kubeadmConfigVersion, upgradeVersion, vLevel); err != nil {
				return err
			}
			if err := kubeadmUpgradeDiff(c, n, kubeadmConfigVersion, upgradeVersion, vLevel); err != nil {
				return err
			}
			err = kubeadmUpgradeApply(c, n, kubeadmConfigVersion, upgradeVersion, patchesDir, wait, vLevel)
		} else {
			err = kubeadmUpgradeNode(c, n, kubeadmConfigVersion, upgradeVersion, patchesDir, wait, vLevel)
		}
		if err != nil {
			return err
		}
	}

	for _, n := range nodeList {
		if err := upgradeKubeletKubectl(c, n, upgradeVersion, wait); err != nil {
			return err
		}
	}

	return nil
}

func preloadNodeUpgradeImages(n *status.Node, upgradeVersion *version.Version) {
	srcFolder := filepath.Join("/kinder", "upgrade", fmt.Sprintf("v%s", upgradeVersion))

	// load images cached on the node into CRI engine
	// this should be executed on all nodes before running kubeadm upgrade apply in order to
	// get everything in place when kubeadm creates pre-pull daemonsets (if not, this might be blocking in case of
	// images not available on public registry, like e.g. pre-release images)
	n.Infof("pre-loading images required for the upgrade")
	nodeCRI, err := n.CRI()
	if err != nil {
		fmt.Printf("error detecting CRI: %v", err)
		return
	}

	actionHelper, err := nodes.NewActionHelper(nodeCRI)
	if err != nil {
		fmt.Printf("error creating the action helper: %v", err)
		return
	}

	if err := actionHelper.PreLoadUpgradeImages(n, srcFolder); err != nil {
		fmt.Printf("error PreLoadUpgradeImages: %v", err)
		return
	}

	// checks pre-loaded images available on the node (this will report missing images, if any)
	if err := checkImagesForVersion(n, upgradeVersion.String()); err != nil {
		fmt.Printf("error ReportImages: %v", err)
		return
	}
}

func upgradeKubeadmBinary(n *status.Node, upgradeVersion *version.Version) error {
	n.Infof("upgrade kubeadm binary")

	srcFolder := filepath.Join("/kinder", "upgrade", fmt.Sprintf("v%s", upgradeVersion))
	src := filepath.Join(srcFolder, "kubeadm")
	dest := filepath.Join("/usr", "bin", "kubeadm")

	if err := n.Command(
		"ln", "-sf", src, dest,
	).Silent().Run(); err != nil {
		return err
	}
	return nil
}

func kubeadmUpgradeDiff(c *status.Cluster, cp1 *status.Node, configVersion string, upgradeVersion *version.Version, vLevel int) error {
	diffArgs := []string{
		"upgrade", "diff", fmt.Sprintf("--v=%d", vLevel),
	}

	if configVersion == "v1beta4" {
		diffArgs = append(diffArgs, "--config", constants.KubeadmConfigPath)
	} else {
		diffArgs = append(diffArgs, fmt.Sprintf("v%s", upgradeVersion))
	}

	if err := cp1.Command(
		"kubeadm", diffArgs...,
	).RunWithEcho(); err != nil {
		return err
	}
	return nil
}

func kubeadmUpgradePlan(c *status.Cluster, cp1 *status.Node, configVersion string, upgradeVersion *version.Version, vLevel int) error {
	planArgs := []string{
		"upgrade", "plan", fmt.Sprintf("--v=%d", vLevel),
	}

	if configVersion == "v1beta4" {
		planArgs = append(planArgs, "--config", constants.KubeadmConfigPath)
	} else {
		planArgs = append(planArgs, "--allow-experimental-upgrades", "--allow-release-candidate-upgrades",
			fmt.Sprintf("v%s", upgradeVersion))
	}

	if err := cp1.Command(
		"kubeadm", planArgs...,
	).RunWithEcho(); err != nil {
		return err
	}
	return nil
}

func kubeadmUpgradeApply(c *status.Cluster, cp1 *status.Node, configVersion string, upgradeVersion *version.Version, patchesDir string, wait time.Duration, vLevel int) error {
	applyArgs := []string{
		"upgrade", "apply", fmt.Sprintf("--v=%d", vLevel),
	}

	if configVersion == "v1beta4" {
		applyArgs = append(applyArgs, "--config", constants.KubeadmConfigPath)
	} else {
		if patchesDir != "" {
			applyArgs = append(applyArgs, fmt.Sprintf("--patches=%s", constants.PatchesDir))
		}
		applyArgs = append(applyArgs, "-f", fmt.Sprintf("v%s", upgradeVersion.String()))
	}

	if err := cp1.Command(
		"kubeadm", applyArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	if err := waitControlPlaneUpgraded(c, cp1, upgradeVersion, wait); err != nil {
		return err
	}

	return nil
}

func kubeadmUpgradeNode(c *status.Cluster, n *status.Node, configVersion string, upgradeVersion *version.Version, patchesDir string, wait time.Duration, vLevel int) error {
	// waitKubeletHasRBAC waits for the kubelet to have access to the expected config map
	// please note that this is a temporary workaround for a problem we are observing on upgrades while
	// executing node upgrades immediately after control-plane upgrade.
	if err := waitKubeletHasRBAC(c, n, upgradeVersion, wait); err != nil {
		return err
	}

	// kubeadm upgrade node
	nodeArgs := []string{
		"upgrade", "node", fmt.Sprintf("--v=%d", vLevel),
	}

	if configVersion == "v1beta4" {
		nodeArgs = append(nodeArgs, "--config", constants.KubeadmConfigPath)
	} else {
		if patchesDir != "" {
			nodeArgs = append(nodeArgs, fmt.Sprintf("--patches=%s", constants.PatchesDir))
		}
	}

	if err := n.Command(
		"kubeadm", nodeArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	if n.IsControlPlane() {
		if err := waitControlPlaneUpgraded(c, n, upgradeVersion, wait); err != nil {
			return err
		}
	}

	return nil
}

func upgradeKubeletKubectl(c *status.Cluster, n *status.Node, upgradeVersion *version.Version, wait time.Duration) error {
	n.Infof("upgrade kubelet and kubectl binaries")

	srcFolder := filepath.Join("/kinder", "upgrade", fmt.Sprintf("v%s", upgradeVersion))

	// upgrade kubectl
	src := filepath.Join(srcFolder, "kubectl")
	dest := filepath.Join("/usr", "bin", "kubectl")

	if err := n.Command(
		"ln", "-sf", src, dest,
	).Silent().Run(); err != nil {
		return err
	}

	// upgrade kubelet
	src = filepath.Join(srcFolder, "kubelet")
	dest = filepath.Join("/usr", "bin", "kubelet")

	if err := n.Command(
		"ln", "-sf", src, dest,
	).Silent().Run(); err != nil {
		return err
	}

	// restart the kubelet
	if err := n.Command(
		"systemctl", "restart", "kubelet",
	).Silent().Run(); err != nil {
		return err
	}

	//write "/kind/version"
	if err := n.Command(
		"echo", fmt.Sprintf("\"%s\"", fmt.Sprintf("v%s", upgradeVersion)), ">", "/kind/version",
	).Silent().Run(); err != nil {
		return err
	}

	if err := waitKubeletUpgraded(c, n, upgradeVersion, wait); err != nil {
		return err
	}

	return nil
}
