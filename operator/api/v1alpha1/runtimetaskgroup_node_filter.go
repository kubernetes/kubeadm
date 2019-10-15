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

// RuntimeTaskGroupNodeFilter is a string representation of a RuntimeTaskGroup node filter.
type RuntimeTaskGroupNodeFilter string

const (
	// RuntimeTaskGroupNodeFilterAll forces the RuntimeTaskGroup controller to use all the nodes
	// returned by the NodeSelector.
	RuntimeTaskGroupNodeFilterAll = RuntimeTaskGroupNodeFilter("All")

	// RuntimeTaskGroupNodeFilterHead forces the RuntimeTaskGroup controller to use only the first node
	// returned by the NodeSelector.
	RuntimeTaskGroupNodeFilterHead = RuntimeTaskGroupNodeFilter("Head")

	// RuntimeTaskGroupNodeFilterTail forces the RuntimeTaskGroup controller to use all the nodes
	// returned by the NodeSelector except the first one.
	RuntimeTaskGroupNodeFilterTail = RuntimeTaskGroupNodeFilter("Tail")

	// RuntimeTaskGroupNodeUnknownFilter is returned if the RuntimeTaskGroupNodeFilter cannot be determined.
	RuntimeTaskGroupNodeUnknownFilter = RuntimeTaskGroupNodeFilter("")
)

// GetTypedTaskGroupNodeFilter attempts to parse the NodeFilter field and return
// the typed RuntimeTaskGroupNodeFilter representation.
func (s *RuntimeTaskGroupSpec) GetTypedTaskGroupNodeFilter() RuntimeTaskGroupNodeFilter {
	switch mode := RuntimeTaskGroupNodeFilter(s.NodeFilter); mode {
	case
		RuntimeTaskGroupNodeFilterAll,
		RuntimeTaskGroupNodeFilterHead,
		RuntimeTaskGroupNodeFilterTail:
		return mode
	default:
		return RuntimeTaskGroupNodeUnknownFilter
	}
}
