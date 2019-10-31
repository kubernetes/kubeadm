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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	operatorerrors "k8s.io/kubeadm/operator/errors"
)

// RuntimeTaskSpec defines the desired state of RuntimeTask
type RuntimeTaskSpec struct {
	// NodeName is a request to schedule this RuntimeTask onto a specific node.
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// RecoveryMode sets the strategy to use when a command is failed.
	// +optional
	RecoveryMode string `json:"recoveryMode,omitempty"`

	// Commands provide the list of commands to be performed when executing a RuntimeTask on a node
	Commands []CommandDescriptor `json:"commands"`
}

// RuntimeTaskStatus defines the observed state of RuntimeTask
type RuntimeTaskStatus struct {

	// StartTime represents time when the RuntimeTask execution was started by the controller.
	// It is represented in RFC3339 form and is in UTC.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CurrentCommand
	// +optional
	CurrentCommand int32 `json:"currentCommand,omitempty"`

	// CommandProgress
	// Please note that this field is only for allowing a better visal representation of status
	// +optional
	CommandProgress string `json:"commandProgress,omitempty"`

	// Paused indicates that the RuntimeTask is paused.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// CompletionTime represents time when the RuntimeTask was completed.
	// It is represented in RFC3339 form and is in UTC.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Phase represents the current phase of RuntimeTask actuation.
	// E.g. Pending, Running, Completed, Failed etc.
	// +optional
	Phase string `json:"phase,omitempty"`

	// ErrorReason will be set in the event that there is a problem in executing
	// the RuntimeTasks and will contain a succinct value suitable
	// for machine interpretation.
	// +optional
	ErrorReason *operatorerrors.RuntimeTaskStatusError `json:"errorReason,omitempty"`

	// ErrorMessage will be set in the event that there is a problem in executing
	// the RuntimeTasks and will contain a more verbose string suitable
	// for logging and human consumption.
	// +optional
	ErrorMessage *string `json:"errorMessage,omitempty"`
}

// SetStartTime is a utility method for setting the StartTime field to Now.
func (s *RuntimeTaskStatus) SetStartTime() {
	now := metav1.Now()
	s.StartTime = &now
}

// NextCurrentCommand is a utility method for setting the CurrentCommand and CommandProgress.
func (s *RuntimeTaskStatus) NextCurrentCommand(commands []CommandDescriptor) {
	s.CurrentCommand = s.CurrentCommand + 1
	s.CommandProgress = fmt.Sprintf("%d/%d", s.CurrentCommand, len(commands))
}

// SetCompletionTime is a utility method for setting the CompletionTime field to Now.
func (s *RuntimeTaskStatus) SetCompletionTime() {
	now := metav1.Now()
	s.CompletionTime = &now

	// ensure all the status attributes are clean after complete
	s.Paused = false
	s.ResetError()
}

// SetError is a utility method for setting the ErrorReason and ErrorMessage fields
// given a RuntimeTaskError object.
func (s *RuntimeTaskStatus) SetError(err *operatorerrors.RuntimeTaskError) {
	reason := err.Reason
	s.ErrorReason = &reason
	s.ErrorMessage = pointer.StringPtr(err.Message)
}

// ResetError is a utility method for cleaning the ErrorReason and ErrorMessage fields
func (s *RuntimeTaskStatus) ResetError() {
	s.ErrorReason = nil
	s.ErrorMessage = nil
}

// +kubebuilder:resource:path=runtimetasks,scope=Cluster,categories=kubeadm-operator
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".status.startTime"
// +kubebuilder:printcolumn:name="Command",type="string",JSONPath=".status.commandProgress"
// +kubebuilder:printcolumn:name="CompletionTime",type="date",JSONPath=".status.completionTime"

// RuntimeTask is the Schema for the runtimetasks API
type RuntimeTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuntimeTaskSpec   `json:"spec,omitempty"`
	Status RuntimeTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuntimeTaskList contains a list of RuntimeTask
type RuntimeTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RuntimeTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RuntimeTask{}, &RuntimeTaskList{})
}
