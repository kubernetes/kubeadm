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

import "sigs.k8s.io/kind/pkg/exec"

// InstallContext should be implemented by users of Bits
// to allow installing the bits in a Docker image
type installContext struct {
	basePath    string
	containerID string
}

// Returns the base path Paths() were populated relative to
func (ic *installContext) BasePath() string {
	return ic.basePath
}

func (ic *installContext) Run(command string, args ...string) error {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (ic *installContext) CombinedOutputLines(command string, args ...string) ([]string, error) {
	cmd := exec.Command(
		"docker",
		append(
			[]string{"exec", ic.containerID, command},
			args...,
		)...,
	)
	return exec.CombinedOutputLines(cmd)
}

// bits provides the locations of Kubernetes Binaries / Images
// needed on the cluster nodes
type bits interface {
	// Paths returns a map of path on host machine to desired path in the alter folder
	// Note: if Images are populated to images/, the cluster provisioning
	// will load these prior to calling kubeadm
	Paths() map[string]string
	// Install should install (deploy) the bits on the node, assuming paths
	// have been populated
	Install(*installContext) error
}
