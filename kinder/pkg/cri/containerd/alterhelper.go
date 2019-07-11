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
	"os"
	"os/exec"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
)

// PreLoadInitImages preload images required by kubeadm-init into the containerd runtime installed that exists inside a kind(er) node
func PreLoadInitImages(bc *bits.BuildContext) error {
	// NB. this code is an extract from "sigs.k8s.io/kind/pkg/build/node"

	return bc.RunInContainer(
		"bash", "-c",
		`containerd & find /kind/images -name *.tar -print0 | xargs -r -0 -n 1 -P $(nproc) ctr --namespace=k8s.io images import --no-unpack && kill %1 && rm -rf /kind/images/*`,
	)
}

// Commit a kind(er) node image that uses the containerd runtime internally
func Commit(containerID, targetImage string) error {
	// NB. this code is an extract from "sigs.k8s.io/kind/pkg/build/node"

	// Save the image changes to a new image
	cmd := exec.Command("docker", "commit",
		/*
			The snapshot storage must be a volume to avoid overlay on overlay

			NOTE: we do this last because changing a volume with a docker image
			must occur before defining it.

			See: https://docs.docker.com/engine/reference/builder/#volume
		*/
		"--change", `VOLUME [ "/var/lib/containerd" ]`,
		// we need to put this back after changing it when running the image
		"--change", `ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]`,
		containerID, targetImage)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
