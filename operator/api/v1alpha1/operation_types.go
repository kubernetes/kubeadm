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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	operatorerrors "k8s.io/kubeadm/operator/errors"
)

// OperationSpec defines the spec of an Operation to be performed by the kubeadm-operator.
// Please note that once the operation will be completed, the operator will stop to reconcile on any instance of this object.
type OperationSpec struct {
	// Paused indicates that the operation is paused.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// ExecutionMode indicates how the controller should handle RuntimeTask/Command execution for this operation.
	// If missing, auto mode will be used.
	// +optional
	ExecutionMode string `json:"executionMode,omitempty"`

	// OperatorDescriptor provide description of the operator content
	OperatorDescriptor `json:",inline"`
}

// OperationStatus defines the observed state of Operation
type OperationStatus struct {

	// StartTime represents time when the Operation execution was started by the controller.
	// It is represented in RFC3339 form and is in UTC.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Paused indicates that the Operation is paused.
	// This fields is set when the OperationSpec.Paused value is processed by the controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// CompletionTime represents time when the Operation was completed.
	// It is represented in RFC3339 form and is in UTC.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Groups returns the number of RuntimeTaskGroups belonging to this operation.
	// +optional
	Groups int32 `json:"groups,omitempty"`

	// RunningGroups
	// +optional
	RunningGroups int32 `json:"runningGroups,omitempty"`

	// SucceededGroups return the number of RuntimeTaskGroups where all the RuntimeTask succeeded
	// +optional
	SucceededGroups int32 `json:"succeededGroups,omitempty"`

	// FailedGroups return the number of RuntimeTaskGroups with at least one RuntimeTask failure
	// +optional
	FailedGroups int32 `json:"failedGroups,omitempty"`

	// InvalidGroups return the number of RuntimeTaskGroups with at least one RuntimeTask inconsistencies like e.g. orphans
	// +optional
	InvalidGroups int32 `json:"invalidGroups,omitempty"`

	// Phase represents the current phase of Operation actuation.
	// E.g. pending, running, succeeded, failed etc.
	// +optional
	Phase string `json:"phase,omitempty"`

	// ErrorReason will be set in the event that there is a problem in executing
	// one of the Operation's RuntimeTasks and will contain a succinct value suitable
	// for machine interpretation.
	// +optional
	ErrorReason *operatorerrors.OperationStatusError `json:"errorReason,omitempty"`

	// ErrorMessage will be set in the event that there is a problem in executing
	// one of the Operation's RuntimeTasks and will contain a more verbose string suitable
	// for logging and human consumption.
	// +optional
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// SetStartTime is a utility method for setting the StartTime field to Now.
func (s *OperationStatus) SetStartTime() {
	now := metav1.Now()
	s.StartTime = &now
}

// SetCompletionTime is a utility method for setting the CompletionTime field to Now.
func (s *OperationStatus) SetCompletionTime() {
	now := metav1.Now()
	s.CompletionTime = &now

	// ensure all the status attributes are clean after complete
	s.Paused = false
	s.ErrorReason = nil
	s.ErrorMessage = nil
}

// SetError is a utility method for setting the ErrorReason and ErrorMessage fields
// given an OperationError object.
func (s *OperationStatus) SetError(err *operatorerrors.OperationError) {
	reason := err.Reason
	s.ErrorReason = &reason
	s.ErrorMessage = pointer.StringPtr(err.Message)
}

// ResetError is a utility method for resetting the ErrorReason and ErrorMessage fields
func (s *OperationStatus) ResetError() {
	s.ErrorReason = nil
	s.ErrorMessage = nil
}

// +kubebuilder:resource:path=operations,scope=Cluster,categories=kubeadm-operator
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Groups",type="integer",JSONPath=".status.groups"
// +kubebuilder:printcolumn:name="Succeeded",type="integer",JSONPath=".status.succeededGroups"
// +kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=".status.failedGroups"

// Operation is the Schema for the operations API
type Operation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperationSpec   `json:"spec,omitempty"`
	Status OperationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OperationList contains a list of Operation
type OperationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Operation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Operation{}, &OperationList{})
}
