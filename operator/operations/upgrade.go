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

package operations

import (
	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func setupUpgrade() map[string]string {
	return map[string]string{}
}

func planUpgrade(operation *operatorv1.Operation, spec *operatorv1.UpgradeOperationSpec) *operatorv1.RuntimeTaskGroupList {
	var items []operatorv1.RuntimeTaskGroup

	t1 := createBasicTaskGroup(operation, "01", "upgrade-cp-1")
	setCP1Selector(&t1)
	t1.Spec.NodeFilter = string(operatorv1.RuntimeTaskGroupNodeFilterHead)
	t1.Spec.Template.Spec.Commands = append(t1.Spec.Template.Spec.Commands,
		operatorv1.CommandDescriptor{
			UpgradeKubeadm: &operatorv1.UpgradeKubeadmCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			KubeadmUpgradeApply: &operatorv1.KubeadmUpgradeApplyCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			UpgradeKubeletAndKubeactl: &operatorv1.UpgradeKubeletAndKubeactlCommandSpec{},
		},
	)
	items = append(items, t1)

	t2 := createBasicTaskGroup(operation, "02", "upgrade-cp-n")
	setCPNSelector(&t2)
	t2.Spec.Template.Spec.Commands = append(t2.Spec.Template.Spec.Commands,
		operatorv1.CommandDescriptor{
			UpgradeKubeadm: &operatorv1.UpgradeKubeadmCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			KubeadmUpgradeNode: &operatorv1.KubeadmUpgradeNodeCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			UpgradeKubeletAndKubeactl: &operatorv1.UpgradeKubeletAndKubeactlCommandSpec{},
		},
	)
	items = append(items, t2)

	t3 := createBasicTaskGroup(operation, "02", "upgrade-w")
	setWSelector(&t3)
	t3.Spec.Template.Spec.Commands = append(t3.Spec.Template.Spec.Commands,
		operatorv1.CommandDescriptor{
			KubectlDrain: &operatorv1.KubectlDrainCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			UpgradeKubeadm: &operatorv1.UpgradeKubeadmCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			KubeadmUpgradeNode: &operatorv1.KubeadmUpgradeNodeCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			UpgradeKubeletAndKubeactl: &operatorv1.UpgradeKubeletAndKubeactlCommandSpec{},
		},
		operatorv1.CommandDescriptor{
			KubectlUncordon: &operatorv1.KubectlUncordonCommandSpec{},
		},
	)
	items = append(items, t3)

	return &operatorv1.RuntimeTaskGroupList{
		Items: items,
	}
}
