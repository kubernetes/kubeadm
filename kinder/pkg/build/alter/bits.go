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

package alter

import (
	"path/filepath"

	"sigs.k8s.io/kind/pkg/exec"
)

// bits defines files/artifacts to be installed into the image being altered
type bits interface {
	// Get retrives a set of bits and get them into the temporary folder on the host machine
	Get(*bitsContext) error
	// Install should install (deploy) the bits on the image being altered
	Install(*bitsContext) error
}

// bitsContext provide context for populating a installing bits for the image alter process
type bitsContext struct {
	hostBasePath string
	containerID  string
}

// HostBasePath returns the path of the temporary folder on the host machine used for the image alter process
func (c *bitsContext) HostBasePath() string {
	return c.hostBasePath
}

// HostBitsPath returns the path of the temporary folder on the host machine used for staging bits before install
func (c *bitsContext) HostBitsPath() string {
	return filepath.Join(c.HostBasePath(), "bits")
}

// ContainerBasePath returns the path of the temporary folder on the container used for the image alter process
func (c *bitsContext) ContainerBasePath() string {
	return "/alter"
}

// ContainerBitsPath returns the path of the temporary folder on the container used for staging bits before install
func (c *bitsContext) ContainerBitsPath() string {
	return filepath.Join(c.ContainerBasePath(), "bits")
}

// BindToContainer binds the current bitsContext to the containerused for altering the image
func (c *bitsContext) BindToContainer(containerID string) {
	c.containerID = containerID
}

// RunInContainer a command on the container used for altering the image
func (c *bitsContext) RunInContainer(command string, args ...string) error {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", c.containerID, command},
			args...,
		)...,
	)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

// CombinedOutputLinesInContainer for a command executed on the container used for altering the image
func (c *bitsContext) CombinedOutputLinesInContainer(command string, args ...string) ([]string, error) {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", c.containerID, command},
			args...,
		)...,
	)
	return exec.CombinedOutputLines(cmd)
}
