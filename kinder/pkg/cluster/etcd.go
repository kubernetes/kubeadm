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

package cluster

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/container/docker"
)

// CreateExternalEtcd creates a docker container mocking a kind external etcd node
// this is temporary and should go away as soon as kind support external etcd node
func CreateExternalEtcd(name string) (ip string, err error) {
	// define name and labels mocking a kind external etcd node

	containerName := fmt.Sprintf("%s-%s", name, constants.ExternalEtcdNodeRoleValue)

	runArgs := []string{
		"-d",                        // run the container detached
		"--hostname", containerName, // make hostname match container name
		"--name", containerName, // ... and set the container name
		// label the node with the cluster ID
		"--label", fmt.Sprintf("%s=%s", constants.ClusterLabelKey, name),
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleKey, constants.ExternalEtcdNodeRoleValue),
	}

	// define a minimal etcd (insecure, single node, not exposed to the host machine)
	containerArgs := []string{
		"etcd",
		"--name", fmt.Sprintf("%s-etcd", name),
		"--advertise-client-urls", "http://127.0.0.1:2379",
		"--listen-client-urls", "http://0.0.0.0:2379",
	}

	_, err = docker.Run(
		"k8s.gcr.io/etcd:3.2.24",
		docker.WithRunArgs(runArgs...),
		docker.WithContainerArgs(containerArgs...),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create external etcd container")
	}

	kn, err := NewKNode(*nodes.FromName(containerName))
	if err != nil {
		return "", errors.Wrap(err, "failed to create external etcd node")
	}

	return kn.IP()
}
