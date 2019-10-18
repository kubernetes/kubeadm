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

// RuntimeTaskRecoveryStrategy is a string representation of a RuntimeTask error recovery policy.
type RuntimeTaskRecoveryStrategy string

const (
	// RuntimeTaskRecoverySkippingFailedCommandStrategy forces the RuntimeTask operator to skip failed RuntimeTask/Command.
	RuntimeTaskRecoverySkippingFailedCommandStrategy = RuntimeTaskRecoveryStrategy("SkipFailedCommand")

	// RuntimeTaskRecoveryRetryingFailedCommandStrategy forces the RuntimeTask operator to retry the failed RuntimeTask/Command.
	RuntimeTaskRecoveryRetryingFailedCommandStrategy = RuntimeTaskRecoveryStrategy("RetryFailedCommand")

	// RuntimeTaskRecoveryUnknownStrategy is returned if the RuntimeTaskErrorRecoveryStrategy cannot be determined.
	RuntimeTaskRecoveryUnknownStrategy = RuntimeTaskRecoveryStrategy("")
)

// GetTypedTaskRecoveryStrategy attempts to parse the mode field and return
// the typed RuntimeTaskRecoveryStrategy representation.
func (s *RuntimeTaskSpec) GetTypedTaskRecoveryStrategy() RuntimeTaskRecoveryStrategy {
	switch mode := RuntimeTaskRecoveryStrategy(s.RecoveryMode); mode {
	case
		RuntimeTaskRecoverySkippingFailedCommandStrategy,
		RuntimeTaskRecoveryRetryingFailedCommandStrategy:
		return mode
	default:
		return RuntimeTaskRecoveryUnknownStrategy
	}
}
