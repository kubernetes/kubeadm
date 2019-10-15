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

// RuntimeTaskGroupCreatePolicy is a string representation of a RuntimeTaskGroup create policy.
type RuntimeTaskGroupCreatePolicy string

const (
	// RuntimeTaskGroupCreateSerialStrategy forces the RuntimeTaskGroup controller to create RuntimeTasks in sequential order.
	// New RuntimeTask are created only after the current RuntimeTime is completed successfully.
	RuntimeTaskGroupCreateSerialStrategy = RuntimeTaskGroupCreatePolicy("Serial")

	// RuntimeTaskGroupCreateUnknownStrategy is returned if the RuntimeTaskGroupCreatePolicy cannot be determined.
	RuntimeTaskGroupCreateUnknownStrategy = RuntimeTaskGroupCreatePolicy("")
)

// GetTypedTaskGroupCreateStrategy attempts to parse the CreateStrategy field and return
// the typed RuntimeTaskGroupCreatePolicy representation.
func (s *RuntimeTaskGroupSpec) GetTypedTaskGroupCreateStrategy() RuntimeTaskGroupCreatePolicy {
	switch mode := RuntimeTaskGroupCreatePolicy(s.CreateStrategy); mode {
	case
		RuntimeTaskGroupCreateSerialStrategy:
		return mode
	default:
		return RuntimeTaskGroupCreateUnknownStrategy
	}
}
