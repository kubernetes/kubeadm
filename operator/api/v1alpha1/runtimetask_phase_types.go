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

// RuntimeTaskPhase is a string representation of a RuntimeTask Phase.
//
// This type is a high-level indicator of the status of the RuntimeTask as it is provisioned,
// from the API user’s perspective.
//
// The value should not be interpreted by any software components as a reliable indication
// of the actual state of the RuntimeTask, and controllers should not use the RuntimeTask Phase field
// value when making decisions about what action to take.
//
// Controllers should always look at the actual state of the RuntimeTask’s fields to make those decisions.
type RuntimeTaskPhase string

const (
	// RuntimeTaskPhasePending is the first state a RuntimeTask is assigned by
	// RuntimeTask controller after being created.
	RuntimeTaskPhasePending = RuntimeTaskPhase("Pending")

	// RuntimeTaskPhaseRunning is the RuntimeTask state when it has
	// started its actuation.
	RuntimeTaskPhaseRunning = RuntimeTaskPhase("Running")

	// RuntimeTaskPhasePaused is the RuntimeTask state when it is paused.
	RuntimeTaskPhasePaused = RuntimeTaskPhase("Paused")

	// RuntimeTaskPhaseSucceeded is the RuntimeTask state when it has
	// succeeded its actuation.
	RuntimeTaskPhaseSucceeded = RuntimeTaskPhase("Succeeded")

	// RuntimeTaskPhaseFailed is the RuntimeTask state when the system
	// might require user intervention.
	RuntimeTaskPhaseFailed = RuntimeTaskPhase("Failed")

	// RuntimeTaskPhaseDeleted is the RuntimeTask state when the object
	// is deleted and ready to be garbage collected by the API Server.
	RuntimeTaskPhaseDeleted = RuntimeTaskPhase("Deleted")

	//RuntimeTaskPhaseUnknown is returned if the RuntimeTask state cannot be determined.
	RuntimeTaskPhaseUnknown = RuntimeTaskPhase("")
)

// SetTypedPhase sets the Phase field to the string representation of RuntimeTaskPhase.
func (s *RuntimeTaskStatus) SetTypedPhase(p RuntimeTaskPhase) {
	s.Phase = string(p)
}

// GetTypedPhase attempts to parse the Phase field and return
// the typed RuntimeTaskPhase representation.
func (s *RuntimeTaskStatus) GetTypedPhase() RuntimeTaskPhase {
	switch phase := RuntimeTaskPhase(s.Phase); phase {
	case
		RuntimeTaskPhasePending,
		RuntimeTaskPhasePaused,
		RuntimeTaskPhaseRunning,
		RuntimeTaskPhaseSucceeded,
		RuntimeTaskPhaseFailed,
		RuntimeTaskPhaseDeleted:
		return phase
	default:
		return RuntimeTaskPhaseUnknown
	}
}
