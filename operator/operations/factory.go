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
	"github.com/pkg/errors"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

// DaemonSetNodeSelectorLabels labels for limiting the nodes where the operation agent will be deployed
func DaemonSetNodeSelectorLabels(operation *operatorv1.Operation) (map[string]string, error) {
	if operation.Spec.RenewCertificates != nil {
		return setupRenewCertificates(), nil
	}

	if operation.Spec.Upgrade != nil {
		return setupUpgrade(), nil
	}

	if operation.Spec.CustomOperation != nil {
		return setupCustom(), nil
	}

	return nil, errors.New("Invalid Operation.Spec.OperatorDescriptor. There are no operation implementation matching this spec")
}

// TaskGroupList return the list of TaskGroup to be performed by an operation
func TaskGroupList(operation *operatorv1.Operation) (*operatorv1.RuntimeTaskGroupList, error) {
	if operation.Spec.RenewCertificates != nil {
		return planRenewCertificates(operation, operation.Spec.RenewCertificates), nil
	}

	if operation.Spec.Upgrade != nil {
		return planUpgrade(operation, operation.Spec.Upgrade), nil
	}

	if operation.Spec.CustomOperation != nil {
		return planCustom(operation, operation.Spec.CustomOperation), nil
	}

	return nil, errors.New("Invalid Operation.Spec.OperatorDescriptor. There are no operation implementation matching this spec")
}
