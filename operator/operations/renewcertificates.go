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

func setupRenewCertificates() map[string]string {
	return map[string]string{
		"node-role.kubernetes.io/master": "",
	}
}

func planRenewCertificates(operation *operatorv1.Operation, spec *operatorv1.RenewCertificatesOperationSpec) *operatorv1.RuntimeTaskGroupList {
	var items []operatorv1.RuntimeTaskGroup

	t1 := createBasicTaskGroup(operation, "01", "renew-cp")
	setCPSelector(&t1)
	t1.Spec.Template.Spec.Commands = append(t1.Spec.Template.Spec.Commands,
		operatorv1.CommandDescriptor{
			KubeadmRenewCertificates: &operatorv1.KubeadmRenewCertsCommandSpec{},
		},
	)
	items = append(items, t1)

	return &operatorv1.RuntimeTaskGroupList{
		Items: items,
	}
}
