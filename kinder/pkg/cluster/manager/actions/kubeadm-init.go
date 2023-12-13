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
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/manager/actions/assets"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

// KubeadmInit executes the kubeadm init workflow including also post init task
// like installing the CNI network plugin
func KubeadmInit(c *status.Cluster, usePhases bool, copyCertsMode CopyCertsMode, kubeadmConfigVersion, patchesDir, ignorePreflightErrors, featureGates string, wait time.Duration, vLevel int) (err error) {
	cp1 := c.BootstrapControlPlane()

	if err := copyPatchesToNode(cp1, patchesDir); err != nil {
		return err
	}

	// checks pre-loaded images available on the node (this will report missing images, if any)
	kubeVersion, err := cp1.KubeVersion()
	if err != nil {
		return err
	}

	if err := checkImagesForVersion(cp1, kubeVersion); err != nil {
		return err
	}

	// prepares the kubeadm config on this node
	if err := KubeadmInitConfig(c, kubeadmConfigVersion, copyCertsMode, featureGates, cp1); err != nil {
		return err
	}

	// prepares the loadbalancer config
	if err := LoadBalancer(c, cp1); err != nil {
		return err
	}

	// execs the kubeadm init workflow
	if usePhases {
		err = kubeadmInitWithPhases(cp1, copyCertsMode, ignorePreflightErrors, vLevel)
	} else {
		err = kubeadmInit(cp1, copyCertsMode, ignorePreflightErrors, vLevel)
	}
	if err != nil {
		return err
	}

	// completes post init task by installing the CNI network plugin
	if err := postInit(c, wait); err != nil {
		return err
	}

	return nil
}

func kubeadmInit(cp1 *status.Node, copyCertsMode CopyCertsMode, ignorePreflightErrors string, vLevel int) error {
	initArgs := []string{
		"init",
		fmt.Sprintf("--ignore-preflight-errors=%s", ignorePreflightErrors),
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
	}
	if copyCertsMode == CopyCertsModeAuto {
		initArgs = append(initArgs,
			"--upload-certs",
			// NB. certificate key is passed via the config file)
		)
	}

	if err := cp1.Command(
		"kubeadm", initArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	return nil
}

func kubeadmInitWithPhases(cp1 *status.Node, copyCertsMode CopyCertsMode, ignorePreflightErrors string, vLevel int) error {
	if err := cp1.Command(
		"kubeadm", "init", "phase", "preflight", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
		fmt.Sprintf("--ignore-preflight-errors=%s", ignorePreflightErrors),
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "kubelet-start", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "certs", "all", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "kubeconfig", "all", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	controlplaneArgs := []string{
		"init", "phase", "control-plane", "all", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	}
	if err := cp1.Command(
		"kubeadm", controlplaneArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	etcdArgs := []string{
		"init", "phase", "etcd", "local", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	}
	if err := cp1.Command(
		"kubeadm", etcdArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	cp1.Infof("waiting for the api server to start")
	if err := cp1.Command(
		"/bin/bash", "-c", //use shell to get $(...) resolved into the container
		fmt.Sprintf("while [[ \"$(curl -k https://localhost:%d/healthz -s -o /dev/null -w ''%%{http_code}'')\" != \"200\" ]]; do sleep 1; done", constants.APIServerPort),
	).Silent().Run(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "upload-config", "all", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	if copyCertsMode == CopyCertsModeAuto {
		uploadCertsArgs := []string{
			"init", "phase", "upload-certs", "--upload-certs",
			fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
			fmt.Sprintf("--v=%d", vLevel),
		}
		if err := cp1.Command(
			"kubeadm", uploadCertsArgs...,
		).RunWithEcho(); err != nil {
			return err
		}
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "mark-control-plane", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "bootstrap-token", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubeadm", "init", "phase", "addon", "all", fmt.Sprintf("--config=%s", constants.KubeadmConfigPath), fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	return nil
}

func postInit(c *status.Cluster, wait time.Duration) error {
	cp1 := c.BootstrapControlPlane()

	if err := copyKubeConfigToHost(c); err != nil {
		return err
	}

	// Apply a CNI plugin using a hardcoded manifest
	cmd := cp1.Command("kubectl", "apply", "--kubeconfig=/etc/kubernetes/admin.conf", "-f", "-")
	cp1.Infof("applying kindnet version 0.5.4")
	cmd.Stdin(strings.NewReader(assets.KindnetManifest054))
	if err := cmd.RunWithEcho(); err != nil {
		return err
	}

	if len(c.Workers()) == 0 {
		taintArgs := []string{
			"--kubeconfig=/etc/kubernetes/admin.conf", "taint", "nodes", "--all",
			"node-role.kubernetes.io/control-plane-",
		}
		if err := cp1.Command(
			"kubectl", taintArgs...,
		).RunWithEcho(); err != nil {
			return err
		}
	}

	//TODO: add the default storage class
	//if err := addDefaultStorageClass(node); err != nil {
	//	return errors.Wrap(err, "failed to add default storage class")
	//}

	if err := waitNewControlPlaneNodeReady(c, cp1, wait); err != nil {
		return err
	}

	fmt.Printf(
		"Cluster creation complete. You can now use the cluster with:\n\n"+

			"export KUBECONFIG=\"$(kinder get kubeconfig-path --name=%q)\"\n"+
			"kubectl cluster-info\n",
		c.Name(),
	)

	return nil
}

// copyKubeConfigToHost copies the admin.conf file to the host in order to make the cluster
// usable with kubectl.
// the kubeconfig file created by kubeadm internally to the node must be modified in order to use
// the random host port reserved for the API server and exposed by the node
func copyKubeConfigToHost(c *status.Cluster) error {
	c.BootstrapControlPlane().Infof("copying the admin.conf file to the host")

	hostPort, err := getAPIServerPort(c)
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	if err := writeKubeConfig(c, hostPort); err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	return nil
}

// getAPIServerPort returns the port on the host on which the APIServer is exposed
func getAPIServerPort(c *status.Cluster) (int32, error) {
	// select the external loadbalancer first
	if c.ExternalLoadBalancer() != nil {
		return c.ExternalLoadBalancer().Ports(constants.ControlPlanePort)
	}

	// fallback to the bootstrap control plane
	return c.BootstrapControlPlane().Ports(constants.APIServerPort)
}

// matches kubeconfig server entry like:
//
//	server: https://172.17.0.2:6443
//
// which we rewrite to:
//
//	server: https://$ADDRESS:$PORT
var serverAddressRE = regexp.MustCompile(`^(\s+server:) https://.*:\d+$`)

// writeKubeConfig writes a fixed KUBECONFIG to dest
// this should only be called on a control plane node
// While copying to the host machine the control plane address
// is replaced with local host and the control plane port with
// a randomly generated port reserved during node creation.
func writeKubeConfig(c *status.Cluster, hostPort int32) error {
	lines, err := c.BootstrapControlPlane().Command("cat", "/etc/kubernetes/admin.conf").Silent().RunAndCapture()
	if err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	// fix the config file, swapping out the server for the forwarded localhost:port
	var buff bytes.Buffer
	for _, line := range lines {
		match := serverAddressRE.FindStringSubmatch(line)
		if len(match) > 1 {
			addr := net.JoinHostPort("localhost", fmt.Sprintf("%d", hostPort))
			line = fmt.Sprintf("%s https://%s", match[1], addr)
		}
		buff.WriteString(line)
		buff.WriteString("\n")
	}

	// create the directory to contain the KUBECONFIG file.
	// 0755 is taken from client-go's config handling logic: https://github.com/kubernetes/client-go/blob/5d107d4ebc00ee0ea606ad7e39fd6ce4b0d9bf9e/tools/clientcmd/loader.go#L412
	dest := c.KubeConfigPath()
	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return errors.Wrap(err, "failed to create kubeconfig output directory")
	}

	return os.WriteFile(dest, buff.Bytes(), 0600)
}

func copyPatchesToNode(n *status.Node, dir string) error {
	// always create the target patch directory on the node since it's always
	// defined in the kubeadm config.
	if err := n.Command("mkdir", "-p", constants.PatchesDir).Silent().Run(); err != nil {
		return errors.Wrapf(err, "failed to create %s folder", constants.PatchesDir)
	}

	// if the source directory is not defined, skip copying patches
	if len(dir) == 0 {
		return nil
	}

	// copy the patches from the host
	n.Infof("Importing patches from %s", dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Printf("%s\n", file.Name())

		hostPath := filepath.Join(dir, file.Name())
		nodePath := filepath.Join(constants.PatchesDir, file.Name())

		if err := n.CopyTo(hostPath, nodePath); err != nil {
			return errors.Wrapf(err, "failed to copy from host path %q to node path %q for node %q",
				hostPath, nodePath, n.Name())
		}
	}

	return nil
}
