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

package v1alpha1

// OperatorDescriptor represents an operation to be performed.
// Only one of its members may be specified.
type OperatorDescriptor struct {
	// Upgrade provide declarative support for the kubeadm upgrade workflow.
	// +optional
	Upgrade *UpgradeOperationSpec `json:"upgrade,omitempty"`

	// RenewCertificates provide declarative support for the kubeadm upgrade workflow.
	// +optional
	RenewCertificates *RenewCertificatesOperationSpec `json:"renewCertificates,omitempty"`

	// CustomOperation enable definition of custom list of RuntimeTaskGroup.
	// +optional
	CustomOperation *CustomOperationSpec `json:"custom,omitempty"`
}

// UpgradeOperationSpec provide declarative support for the kubeadm upgrade workflow.
type UpgradeOperationSpec struct {
	// KubernetesVersion specifies the target kubernetes version
	KubernetesVersion string `json:"kubernetesVersion"`

	// INSERT ADDITIONAL SPEC FIELDS -
	// Important: Run "make" to regenerate code after modifying this file
}

// RenewCertificatesOperationSpec provide declarative support for the kubeadm upgrade workflow.
type RenewCertificatesOperationSpec struct {

	// INSERT ADDITIONAL SPEC FIELDS -
	// Important: Run "make" to regenerate code after modifying this file
}

// CustomOperationSpec enable definition of custom list of RuntimeTaskGroup.
type CustomOperationSpec struct {
	// Workflow allows to define a custom list of RuntimeTaskGroup.
	Workflow []RuntimeTaskGroup `json:"workflow"`
}
