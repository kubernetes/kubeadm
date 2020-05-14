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
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
)

// GetAlterContainerArgs returns arguments for alter container for Docker
func GetAlterContainerArgs() ([]string, []string) {
	runArgs := []string{
		// privileged is required for "dockerd" iptables permissions
		"--privileged",
	}
	return runArgs, []string{}
}

// StartRuntime starts the runtime
func StartRuntime(bc *bits.BuildContext) error {
	log.Info("starting dockerd")
	go func() {
		bc.RunInContainer("dockerd")
		log.Info("dockerd stopped")
	}()

	duration := 10 * time.Second
	result := tryUntil(time.Now().Add(duration), func() bool {
		return bc.RunInContainer("bash", "-c", "docker info &> /dev/null") == nil
	})
	if !result {
		return errors.Errorf("dockerd did not start in %v", duration)
	}
	return nil
}

// StopRuntime stops the runtime
func StopRuntime(bc *bits.BuildContext) error {
	return bc.RunInContainer("pkill", "-f", "dockerd")
}

// PullImages pulls a set of images using docker
func PullImages(bc *bits.BuildContext, images []string, targetPath string) error {
	// pull the images
	for _, image := range images {
		if err := bc.RunInContainer("bash", "-c", "docker pull "+image+" > /dev/null"); err != nil {
			return errors.Wrapf(err, "could not pull image: %s", image)
		}
		// extract the image name; assumes the format is "repository/image:tag"
		r := regexp.MustCompile("[/:]")
		s := r.Split(image, -1)
		if len(s) < 3 {
			return errors.Errorf("unsupported image URL: %s", image)
		}
		path := filepath.Join(targetPath, s[len(s)-2])
		if err := bc.RunInContainer("docker", "save", "-o="+path+".tar", image); err != nil {
			return errors.Wrapf(err, "could not save image %q to path %q", image, targetPath)
		}
	}
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
