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

/*
Package bits provide utilities for managing bits (files/artifacts) to be installed into the image at build time
*/
package bits

import (
	"path/filepath"

	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// Installer interface defines the behaviour of a type in charge of installing a specific set of bits (files/artifacts)
type Installer interface {
	// Prepare a set of bits into the temporary folder on the host machine
	Prepare(*BuildContext) (map[string]string, error)
	// Install should install (deploy) the bits on the image being altered
	Install(*BuildContext) error
}

// BuildContext provide context for installing bits during a build process
type BuildContext struct {
	hostBasePath string
	containerID  string
}

// NewBuildContext returns a new BuildContext
func NewBuildContext(tmpFolder string) *BuildContext {
	return &BuildContext{
		hostBasePath: tmpFolder,
	}
}

// HostBasePath returns the path of the temporary folder on the host machine used for the image build process
func (c *BuildContext) HostBasePath() string {
	return c.hostBasePath
}

// HostBitsPath returns the path of subfolder under HostBasePath where bits are stored
func (c *BuildContext) HostBitsPath() string {
	return filepath.Join(c.HostBasePath(), "bits")
}

// ContainerBasePath returns the path where the HostBasePath is mounted inside the container used for the build process
func (c *BuildContext) ContainerBasePath() string {
	return "/alter"
}

// ContainerBitsPath returns the path where the HostBitsPath is mounted inside the container used for the build process
func (c *BuildContext) ContainerBitsPath() string {
	return filepath.Join(c.ContainerBasePath(), "bits")
}

// BindToContainer binds the current BuildContext to the container used for the build process
func (c *BuildContext) BindToContainer(containerID string) {
	c.containerID = containerID
}

// RunInContainer executes a command on the container used for altering the image
func (c *BuildContext) RunInContainer(command string, args ...string) error {
	cmd := kindexec.Command(
		"docker",
		append(
			[]string{"exec", c.containerID, command},
			args...,
		)...,
	)
	kindexec.InheritOutput(cmd)
	return cmd.Run()
}

// CombinedOutputLinesInContainer executes a command on the container used for altering the image and returns CombinedOutputLines
func (c *BuildContext) CombinedOutputLinesInContainer(command string, args ...string) ([]string, error) {
	cmd := kindexec.Command(
		"docker",
		append(
			[]string{"exec", c.containerID, command},
			args...,
		)...,
	)
	return kindexec.CombinedOutputLines(cmd)
}
