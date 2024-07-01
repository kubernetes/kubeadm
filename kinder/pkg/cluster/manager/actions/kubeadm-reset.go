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

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/kubeadm"
)

// KubeadmReset executes the kubeadm reset workflow
func KubeadmReset(c *status.Cluster, vLevel int) error {
	//TODO: implements kubeadm reset with phases
	for _, n := range c.K8sNodes().EligibleForActions() {
		flags := []string{"reset", fmt.Sprintf("--v=%d", vLevel)}

		// After upgrade, the 'kubeadm version' should return the version of the kubeadm used
		// to perform the upgrade. Use this version to determine if v1beta4 is enabled. If yes,
		// use ResetConfiguration with a 'force: true', else just use the '--force' flag.
		v, err := n.KubeadmVersion()
		if err != nil {
			return errors.Wrap(err, "could not obtain the kubeadm version before calling 'kubeadm reset'")
		}
		if kubeadm.GetKubeadmConfigVersion(v) == "v1beta4" {
			if err := KubeadmResetConfig(c, n); err != nil {
				return errors.Wrap(err, "could not write kubeadm config before calling 'kubeadm reset'")
			}
			flags = append(flags, "--config", constants.KubeadmConfigPath)
		} else {
			flags = append(flags, "--force")
		}

		if err := n.Command("kubeadm", flags...).RunWithEcho(); err != nil {
			return err
		}
	}
	return nil
}
