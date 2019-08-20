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

package containerd

import (
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	kindCRI "sigs.k8s.io/kind/pkg/container/cri"
)

// CreateControlPlaneNode creates a kind(er) control-plane node that uses containerd runtime internally
func CreateControlPlaneNode(name, image, clusterLabel, listenAddress string, port int32, mounts []kindCRI.Mount, portMappings []kindCRI.PortMapping) error {
	_, err := kindnodes.CreateControlPlaneNode(name, image, clusterLabel, listenAddress, port, mounts, portMappings)
	return err
}

// CreateWorkerNode creates a kind(er) worker node node that uses containerd runtime internally
func CreateWorkerNode(name, image, clusterLabel string, mounts []kindCRI.Mount, portMappings []kindCRI.PortMapping) error {
	_, err := kindnodes.CreateWorkerNode(name, image, clusterLabel, mounts, portMappings)
	return err
}
