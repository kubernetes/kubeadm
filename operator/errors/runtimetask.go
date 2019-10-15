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

// RuntimeTaskStatusError defines error conditions for RuntimeTaskStatus
type RuntimeTaskStatusError string

const (
	// RuntimeTaskExecutionError represent an error during task execution
	RuntimeTaskExecutionError RuntimeTaskStatusError = "ExecutionError"

	// RuntimeTaskIndexOutOfRangeError represent an error while accessing RuntimeTask commands
	RuntimeTaskIndexOutOfRangeError RuntimeTaskStatusError = "IndexOutOfRange"
)

// RuntimeTaskError provides a more descriptive kind of error that represents an error condition that
// should be set in the RuntimeTask.Status. The "Reason" field is meant for short,
// enum-style constants meant to be interpreted by machines. The "Message"
// field is meant to be read by humans.
type RuntimeTaskError struct {
	Reason  RuntimeTaskStatusError
	Message string
}

// The Error message
func (e *RuntimeTaskError) Error() string {
	return e.Message
}

// NewRuntimeTaskExecutionError returns a new RuntimeTaskExecutionError
func NewRuntimeTaskExecutionError(msg string, args ...interface{}) *RuntimeTaskError {
	return &RuntimeTaskError{
		Reason:  RuntimeTaskExecutionError,
		Message: fmt.Sprintf(msg, args...),
	}
}

// NewRuntimeTaskIndexOutOfRangeError returns a new RuntimeTaskIndexOutOfRangeError
func NewRuntimeTaskIndexOutOfRangeError(msg string, args ...interface{}) *RuntimeTaskError {
	return &RuntimeTaskError{
		Reason:  RuntimeTaskIndexOutOfRangeError,
		Message: fmt.Sprintf(msg, args...),
	}
}
