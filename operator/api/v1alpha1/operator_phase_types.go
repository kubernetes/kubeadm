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

// OperationPhase is a string representation of a Operation Phase.
//
// This type is a high-level indicator of the status of the Operation as it is provisioned,
// from the API user’s perspective.
//
// The value should not be interpreted by any software components as a reliable indication
// of the actual state of the Operation, and controllers should not use the Operation Phase field
// value when making decisions about what action to take.
//
// Controllers should always look at the actual state of the Operation’s fields to make those decisions.
type OperationPhase string

const (
	// OperationPhasePending is the first state a Operation is assigned after being created.
	OperationPhasePending = OperationPhase("Pending")

	// OperationPhaseRunning is the Operation state when it has
	// started its actuation.
	OperationPhaseRunning = OperationPhase("Running")

	// OperationPhasePaused is the Operation state when it is paused.
	OperationPhasePaused = OperationPhase("Paused")

	// OperationPhaseSucceeded is the Operation state when all the
	// desired RuntimeTasks are succeeded.
	OperationPhaseSucceeded = OperationPhase("Succeeded")

	// OperationPhaseFailed is the Operation state when the system
	// might require user intervention.
	OperationPhaseFailed = OperationPhase("Failed")

	// OperationPhaseDeleted is the Operation state when the object
	// is deleted and ready to be garbage collected by the API Server.
	OperationPhaseDeleted = OperationPhase("Deleted")

	//OperationPhaseUnknown is returned if the Operation state cannot be determined.
	OperationPhaseUnknown = OperationPhase("")
)

// SetTypedPhase sets the Phase field to the string representation of OperationPhase.
func (s *OperationStatus) SetTypedPhase(p OperationPhase) {
	s.Phase = string(p)
}

// GetTypedPhase attempts to parse the Phase field and return
// the typed OperationPhase representation.
func (s *OperationStatus) GetTypedPhase() OperationPhase {
	switch phase := OperationPhase(s.Phase); phase {
	case
		OperationPhasePending,
		OperationPhaseRunning,
		OperationPhasePaused,
		OperationPhaseSucceeded,
		OperationPhaseFailed,
		OperationPhaseDeleted:
		return phase
	default:
		return OperationPhaseUnknown
	}
}
