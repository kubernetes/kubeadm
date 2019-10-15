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

// RuntimeTaskGroupPhase is a string representation of a RuntimeTaskGroup Phase.
//
// This type is a high-level indicator of the status of the RuntimeTaskGroup as it is provisioned,
// from the API user’s perspective.
//
// The value should not be interpreted by any software components as a reliable indication
// of the actual state of the RuntimeTaskGroup, and controllers should not use the RuntimeTaskGroup Phase field
// value when making decisions about what action to take.
//
// Controllers should always look at the actual state of the RuntimeTaskGroup’s fields to make those decisions.
type RuntimeTaskGroupPhase string

const (
	// RuntimeTaskGroupPhasePending is the first state a RuntimeTaskGroup is assigned by
	// Operation controller after being created.
	RuntimeTaskGroupPhasePending = RuntimeTaskGroupPhase("Pending")

	// RuntimeTaskGroupPhaseRunning is the RuntimeTaskGroup state when it has
	// started its actuation.
	RuntimeTaskGroupPhaseRunning = RuntimeTaskGroupPhase("Running")

	// RuntimeTaskGroupPhasePaused is the RuntimeTaskGroup state when it is paused.
	RuntimeTaskGroupPhasePaused = RuntimeTaskGroupPhase("Paused")

	// RuntimeTaskGroupPhaseSucceeded is the RuntimeTaskGroup state when all the
	// RuntimeTasks are succeeded.
	RuntimeTaskGroupPhaseSucceeded = RuntimeTaskGroupPhase("Succeeded")

	// RuntimeTaskGroupPhaseFailed is the RuntimeTaskGroup state when the system
	// might require user intervention.
	RuntimeTaskGroupPhaseFailed = RuntimeTaskGroupPhase("Failed")

	// RuntimeTaskGroupPhaseDeleted is the RuntimeTaskGroup state when the object
	// is deleted and ready to be garbage collected by the API Server.
	RuntimeTaskGroupPhaseDeleted = RuntimeTaskGroupPhase("Deleted")

	//RuntimeTaskGroupPhaseUnknown is returned if the RuntimeTaskGroup state cannot be determined.
	RuntimeTaskGroupPhaseUnknown = RuntimeTaskGroupPhase("")
)

// SetTypedPhase sets the Phase field to the string representation of RuntimeTaskGroupPhase.
func (s *RuntimeTaskGroupStatus) SetTypedPhase(p RuntimeTaskGroupPhase) {
	s.Phase = string(p)
}

// GetTypedPhase attempts to parse the Phase field and return
// the typed RuntimeTaskGroupPhase representation.
func (s *RuntimeTaskGroupStatus) GetTypedPhase() RuntimeTaskGroupPhase {
	switch phase := RuntimeTaskGroupPhase(s.Phase); phase {
	case
		RuntimeTaskGroupPhasePending,
		RuntimeTaskGroupPhasePaused,
		RuntimeTaskGroupPhaseRunning,
		RuntimeTaskGroupPhaseSucceeded,
		RuntimeTaskGroupPhaseFailed,
		RuntimeTaskGroupPhaseDeleted:
		return phase
	default:
		return RuntimeTaskGroupPhaseUnknown
	}
}
