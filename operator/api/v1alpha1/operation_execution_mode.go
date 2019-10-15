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

// OperationExecutionMode is a string representation of a RuntimeTaskGroup create policy.
type OperationExecutionMode string

const (
	// OperationExecutionModeAuto forces the kubeadm operator to automatically execute a new RuntimeTask/Command immediately
	// after the current RuntimeTask/Command is completed successfully.
	OperationExecutionModeAuto = OperationExecutionMode("Auto")

	// OperationExecutionModeControlled forces the kubeadm operator to pause immediately before executing a new
	// RuntimeTask/Command, so the user can check the RuntimeTask specification and decide if to proceed or not.
	OperationExecutionModeControlled = OperationExecutionMode("Controlled")

	// OperationExecutionModeDryRun forces the kubeadm operator to dry-run instead of actually executing RuntimeTasks/Commands.
	OperationExecutionModeDryRun = OperationExecutionMode("DryRun")

	// OperationExecutionModeUnknown is returned if the OperationExecutionMode cannot be determined.
	OperationExecutionModeUnknown = OperationExecutionMode("")
)

// GetTypedOperationExecutionMode attempts to parse the ExecutionMode field and return
// the typed OperationExecutionMode representation.
func (s *OperationSpec) GetTypedOperationExecutionMode() OperationExecutionMode {
	switch mode := OperationExecutionMode(s.ExecutionMode); mode {
	case
		OperationExecutionModeAuto,
		OperationExecutionModeDryRun,
		OperationExecutionModeControlled:
		return mode
	default:
		return OperationExecutionModeUnknown
	}
}
