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
	"github.com/pkg/errors"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
)

// GetAlterContainerArgs returns arguments for the alter container for containerd
func GetAlterContainerArgs() ([]string, []string) {
	runArgs := []string{
		// privileged is required for "ctr image pull" permissions
		"--privileged",
		// the snapshot storage must be a volume.
		// see the info in Commit()
		"-v=/var/lib/containerd",
		// enable the actual entry point in the kind base image
		"--entrypoint=/usr/local/bin/entrypoint",
	}
	runCommands := []string{
		// pass the init binary to the entrypoint
		"/sbin/init",
	}
	return runArgs, runCommands
}

// StartRuntime starts the runtime
func StartRuntime(bc *bits.BuildContext) error {
	log.Info("starting containerd")
	go func() {
		bc.RunInContainer("containerd")
		log.Info("containerd stopped")
	}()
	return nil
}

// StopRuntime stops the runtime
func StopRuntime(bc *bits.BuildContext) error {
	return bc.RunInContainer("pkill", "-f", "containerd")
}

// PullImages pulls a set of images using ctr
func PullImages(bc *bits.BuildContext, images []string, targetPath string) error {
	// Supposedly this should be enough for containerd to snapshot the images, but it does not work.
	// TODO: commit pre-pulled images for containerd.
	for _, image := range images {
		if err := bc.RunInContainer("bash", "-c", "ctr image pull "+image+" > /dev/null"); err != nil {
			return errors.Wrapf(err, "could not pull image: %s", image)
		}
	}
	return nil
}

// PreLoadInitImages preload images required by kubeadm-init into the containerd runtime installed that exists inside a kind(er) node
func PreLoadInitImages(bc *bits.BuildContext) error {
	// NB. this code is an extract from "sigs.k8s.io/kind/pkg/build/node"

	return bc.RunInContainer(
		// NB. the ctr call bellow used to include "--no-unpack", but his flag is not longer available
		// TODO: importing the images, deleting the tars, committing the changes to the image and then creating
		// a container from the image results in no images in the container, so this preload does not work.
		"bash", "-c",
		`containerd & find /kind/images -name *.tar -print0 | xargs -r -0 -n 1 -P $(nproc) ctr --namespace=k8s.io images import && kill %1 && rm -rf /kind/images/*`,
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
