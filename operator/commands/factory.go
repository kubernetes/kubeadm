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

package commands

import (
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

// RunCommand run the command on a node
func RunCommand(c *operatorv1.CommandDescriptor, log logr.Logger) error {

	if c.Preflight != nil {
		return runPreflight(c.Preflight, log)
	}

	if c.KubeadmRenewCertificates != nil {
		return runKubeadmRenewCertificates(c.KubeadmRenewCertificates, log)
	}

	if c.KubeadmUpgradeApply != nil {
		return runKubeadmUpgradeApply(c.KubeadmUpgradeApply, log)
	}

	if c.KubeadmUpgradeNode != nil {
		return runKubeadmUpgradeNode(c.KubeadmUpgradeNode, log)
	}

	if c.KubectlDrain != nil {
		return runKubectlDrain(c.KubectlDrain, log)
	}

	if c.KubectlUncordon != nil {
		return runKubectlUncordon(c.KubectlUncordon, log)
	}

	if c.UpgradeKubeadm != nil {
		return runUpgradeKubeadm(c.UpgradeKubeadm, log)
	}

	if c.UpgradeKubeletAndKubeactl != nil {
		return runUpgradeKubectlAndKubelet(c.UpgradeKubeletAndKubeactl, log)
	}

	if c.Pass != nil {
		return nil
	}

	if c.Fail != nil {
		time.Sleep(5 * time.Second)
		return errors.New("command fail failed")
	}

	if c.Wait != nil {
		time.Sleep(time.Duration(c.Wait.Seconds) * time.Second)
		return nil
	}

	return errors.New("invalid Task.Spec.[]CommandDescriptor. There are no command implementations matching this spec")
}
