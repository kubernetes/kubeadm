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

// OperationStatusError defines error conditions for OperationStatus
type OperationStatusError string

const (
	// OperationReplicaError represent error in creating RuntimeTaskGroups
	OperationReplicaError OperationStatusError = "FailedReplics"

	// OperationReconciliationError represent an unexpected condition in controlled RuntimeTaskGroups
	OperationReconciliationError OperationStatusError = "InvalidState"
)

// OperationError provides a more descriptive kind of error that represents an error condition that
// should be set in the Operation.Status. The "Reason" field is meant for short,
// enum-style constants meant to be interpreted by machines. The "Message"
// field is meant to be read by humans.
type OperationError struct {
	Reason  OperationStatusError
	Message string
}

// The Error message
func (e *OperationError) Error() string {
	return e.Message
}

// NewOperationReplicaError returns a new OperationReplicaError
func NewOperationReplicaError(msg string, args ...interface{}) *OperationError {
	return &OperationError{
		Reason:  OperationReplicaError,
		Message: fmt.Sprintf(msg, args...),
	}
}

// NewOperationReconciliationError returns a new OperationReconciliationError
func NewOperationReconciliationError(msg string, args ...interface{}) *OperationError {
	return &OperationError{
		Reason:  OperationReconciliationError,
		Message: fmt.Sprintf(msg, args...),
	}
}
