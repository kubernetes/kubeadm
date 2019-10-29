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

const (
	// OperationNameLabel is a label defined for allowing lookup of objects related
	// to one Operation.
	OperationNameLabel = "operator.kubeadm.x-k8s.io/operation"

	// OperationUIDLabel is a label defined for ensuring that objects related
	// to one Operation won't get mixed by chance.
	OperationUIDLabel = "operator.kubeadm.x-k8s.io/uid"

	// TaskGroupNameLabel is a label defined for allowing lookup of objects related
	// to one TaskGroup object.
	TaskGroupNameLabel = "operator.kubeadm.x-k8s.io/taskgroup"

	// TaskGroupOrderLabel is a label defined  allowing lookup of objects related
	// to one TaskGroup object by the TaskGroup sequential order
	// e.g. list of the Task related to the first TaskGroup.
	TaskGroupOrderLabel = "operator.kubeadm.x-k8s.io/order"
)
