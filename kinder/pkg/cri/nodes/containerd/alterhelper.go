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
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes/common"
)

// GetAlterContainerArgs returns arguments for the alter container for containerd
func GetAlterContainerArgs() ([]string, []string) {
	// NB. using /usr/local/bin/entrypoint or /sbin/init both throw errors
	// for base image "kindest/base:v20191105-ee880e9b".
	// Use "sleep infinity" instead, but still make sure containerd can run.
	runArgs := []string{
		// privileged is required for "ctr image pull" permissions
		"--privileged",
		// override the entrypoint
		"--entrypoint=/bin/sleep",
	}
	runCommands := []string{
		// pass this to the entrypoint
		"infinity",
	}
	return runArgs, runCommands
}

// StartRuntime starts the runtime
func StartRuntime(bc *bits.BuildContext) error {
	log.Info("starting containerd")
	go func() {
		bc.RunInContainer("bash", "-c", "nohup containerd > /dev/null 2>&1 &")
	}()

	duration := 10 * time.Second
	result := common.TryUntil(time.Now().Add(duration), func() bool {
		return bc.RunInContainer("bash", "-c", "crictl ps &> /dev/null") == nil
	})
	if !result {
		return errors.Errorf("containerd did not start in %v", duration)
	}
	log.Info("containerd started")
	return nil
}

// StopRuntime stops the runtime
func StopRuntime(bc *bits.BuildContext) error {
	return bc.RunInContainer("pkill", "-f", "containerd")
}

// ImportImage import a TAR file into the CR and delete it
func ImportImage(bc *bits.BuildContext, tar string) error {
	if err := bc.RunInContainer("ctr", "--namespace=k8s.io", "image", "import", tar, "--no-unpack"); err != nil {
		return errors.Wrapf(err, "could not import image file %q", tar)
	}
	if err := bc.RunInContainer("rm", tar); err != nil {
		return errors.Wrapf(err, "could not delete the file %q", tar)
	}
	return nil
}

// PreLoadInitImages preload images required by kubeadm-init into the containerd runtime that exists inside a kind(er) node
func PreLoadInitImages(bc *bits.BuildContext, srcFolder string) error {
	// NB. this code is an extract from "sigs.k8s.io/kind/pkg/build/node"
	return bc.RunInContainer(
		"bash", "-c",
		`find `+srcFolder+` -name *.tar -print0 | xargs -0 -n 1 -P $(nproc) ctr --namespace=k8s.io images import --all-platforms --no-unpack && rm -rf `+srcFolder+`/*.tar`,
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
