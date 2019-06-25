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
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
)

// CreateExternalEtcd creates a docker container mocking a kind external etcd node
// this is temporary and should go away as soon as kind support external etcd node
func CreateExternalEtcd(name, nodeImage string) (ip string, err error) {
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

	// create a temporary container from the node-image to be able to fetch the etcd version from kubeadm
	_, err = docker.Run(
		nodeImage,
		docker.WithRunArgs("-d", fmt.Sprintf("--name=%s", containerName)),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create a temporary node container")
	}
	var etcdVersionBuf bytes.Buffer
	cmder := docker.ContainerCmder(containerName)
	cmd := cmder.Command("/bin/sh", "-c", "kubeadm config images list 2> /dev/null | grep etcd")
	cmd.SetStdout(&etcdVersionBuf)
	if err := cmd.Run(); err != nil {
		return "", errors.Wrap(err, "failed to get etcd version from the temporary node container")
	}
	if err := exec.Command("docker", "rm", "-f", "-v", containerName).Run(); err != nil {
		return "", errors.Wrap(err, "failed to remove the temporary node container")
	}

	// pull the image if needed
	etcdImage := strings.TrimSpace(etcdVersionBuf.String())
	if _, err := docker.PullIfNotPresent(etcdImage, 2); err != nil {
		return "", errors.Wrap(err, "failed to pre-pull the etcd image")
	}

	// create the etcd container
	_, err = docker.Run(
		etcdImage,
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
