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

package docker

import (
	"os"
	"os/exec"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
)

// PreLoadInitImages preload images required by kubeadm-init into the docker runtime that exists inside a kind(er) node
func PreLoadInitImages(bc *bits.BuildContext) error {
	// in docker images are pre-loaded at create time, so this action is a no-op at alter time
	return nil
}

// Commit a kind(er) node image that uses the docker runtime internally
func Commit(containerID, targetImage string) error {
	// Save the image changes to a new image
	cmd := exec.Command("docker", "commit", containerID, targetImage)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
