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

package errors

import (
	"fmt"
)

// RuntimeTaskGroupStatusError defines error conditions for RuntimeTaskGroupStatus
type RuntimeTaskGroupStatusError string

const (
	// RuntimeTaskGroupReplicaError represent error in creating RuntimeTask replicas for a node
	RuntimeTaskGroupReplicaError RuntimeTaskGroupStatusError = "FailedNodes"

	// RuntimeTaskGroupReconciliationError represent an unexpected condition in controlled RuntimeTasks
	RuntimeTaskGroupReconciliationError RuntimeTaskGroupStatusError = "InvalidState"
)

// RuntimeTaskGroupError provide a more descriptive kind of error that represents an error condition that
// should be set in the RuntimeTaskGroup.Status. The "Reason" field is meant for short,
// enum-style constants meant to be interpreted by machines. The "Message"
// field is meant to be read by humans.
type RuntimeTaskGroupError struct {
	Reason  RuntimeTaskGroupStatusError
	Message string
}

// The Error message
func (e *RuntimeTaskGroupError) Error() string {
	return e.Message
}

// NewRuntimeTaskGroupReplicaError returns a new RuntimeTaskGroupReplicaError
func NewRuntimeTaskGroupReplicaError(msg string, args ...interface{}) *RuntimeTaskGroupError {
	return &RuntimeTaskGroupError{
		Reason:  RuntimeTaskGroupReplicaError,
		Message: fmt.Sprintf(msg, args...),
	}
}

// NewRuntimeTaskGroupReconciliationError returns a new RuntimeTaskGroupStatusError
func NewRuntimeTaskGroupReconciliationError(msg string, args ...interface{}) *RuntimeTaskGroupError {
	return &RuntimeTaskGroupError{
		Reason:  RuntimeTaskGroupReconciliationError,
		Message: fmt.Sprintf(msg, args...),
	}
}
