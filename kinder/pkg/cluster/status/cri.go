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

package status

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"

	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// NB. code implemented in this package ideally should be in the CRI package, but ATM it is
// implemented here to avoid circular references. TODO: refactor

// ContainerRuntime defines CRI runtime that are supported inside a kind(er) node
type ContainerRuntime string

const (
	// DockerRuntime refers to the docker container runtime
	DockerRuntime ContainerRuntime = "docker"
	// ContainerdRuntime refers to the containerd container runtime
	ContainerdRuntime ContainerRuntime = "containerd"
)

// InspectCRIinImage inspect an image and detects the installed container runtime
func InspectCRIinImage(image string) (ContainerRuntime, error) {
	// define docker default args
	id := "kind-detect-" + uuid.New().String()
	args := []string{
		"-d", // make the client exit while the container continues to run
		"--entrypoint=sleep",
		"--name=" + id,
	}

	if err := kinddocker.Run(
		image,
		kinddocker.WithRunArgs(
			args...,
		),
		kinddocker.WithContainerArgs(
			"infinity", // sleep infinitely to keep the container around
		),
	); err != nil {
		return "", errors.Wrap(err, "error creating a temporary container for CRI detection")
	}
	defer func() {
		kindexec.Command("docker", "rm", "-f", id).Run()
	}()

	return InspectCRIinContainer(id)
}

// InspectCRIinContainer inspect a running and detects the installed container runtime
// NB. this method use raw kinddocker/kindexec commands because it is used also during alter and create
// (before an actual Cluster status exist)
func InspectCRIinContainer(id string) (ContainerRuntime, error) {

	cmder := kinddocker.ContainerCmder(id)

	cmd := cmder.Command("/bin/sh", "-c",
		`cat /kinder/cri > /dev/null 2>&1 || true`)
	lines, err := kindexec.CombinedOutputLines(cmd)
	if err != nil {
		return ContainerRuntime(""), errors.Wrap(err, "error detecting CRI")
	}

	if len(lines) == 1 {
		return ContainerRuntime(lines[0]), nil
	}

	cmd = cmder.Command("/bin/sh", "-c",
		`which docker || true`)
	lines, err = kindexec.CombinedOutputLines(cmd)

	if err != nil {
		return ContainerRuntime(""), errors.Wrap(err, "error detecting CRI")
	}

	if len(lines) == 1 {
		return DockerRuntime, nil
	}

	return ContainerdRuntime, nil
}
