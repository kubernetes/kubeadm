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
	"fmt"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func setupCustom() map[string]string {
	return map[string]string{}
}

func planCustom(operation *operatorv1.Operation, spec *operatorv1.CustomOperationSpec) *operatorv1.RuntimeTaskGroupList {
	var items []operatorv1.RuntimeTaskGroup

	for i, t := range spec.Workflow {
		items = append(items, fixupCustomTaskGroup(operation, t, fmt.Sprintf("%02d", i+1)))
	}

	return &operatorv1.RuntimeTaskGroupList{
		Items: items,
	}
}
