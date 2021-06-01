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
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes/common"
)

// GetAlterContainerArgs returns arguments for alter container for Docker
func GetAlterContainerArgs() ([]string, []string) {
	runArgs := []string{
		// privileged is required for "dockerd" iptables permissions
		"--privileged",
	}
	return runArgs, []string{}
}

// SetupRuntime setups the runtime
func SetupRuntime(bc *bits.BuildContext) error {
	// Rewrite the Docker daemon config to include:
	// - the "cri-containerd: true", which is something that already exists in kindest/base:v20190403-1ebf15f
	// - the cgroup driver setting (systemd)
	if err := bc.RunInContainer("bash", "-c",
		"printf '{\"cri-containerd\": true, \"exec-opts\": [\"native.cgroupdriver=systemd\"]}\n' > /etc/docker/daemon.json",
	); err != nil {
		return errors.Wrap(err, "could not overwrite /etc/docker/daemon.json")
	}
	// Workaround from https://github.com/kubernetes/kubernetes/issues/43704#issuecomment-289484654
	// Using the systemd driver in our rather old base image results in errors around the kubepods.slice.
	// Write the flags --cgroups-per-qos=false --enforce-node-allocatable="" in the KUBELET_EXTRA_ARGS file.
	//
	// It's not possible to pass these via the KubeletConfiguration because the validation / defaulting is bogus.
	// Empty slice gets defaulted to "pods" and non-empty slice fails for 'cgroupsPerQOS: false' -
	// i.e. it is not possible to pass 'none' or '[]' for the 'enforceNodeAllocatable' config field.
	// https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/pkg/kubelet/apis/config/validation/validation.go#L53-L54
	if err := bc.RunInContainer("bash", "-c",
		"printf 'KUBELET_EXTRA_ARGS=\"--cgroups-per-qos=false --enforce-node-allocatable=\"\"\"' > /etc/default/kubelet",
	); err != nil {
		return errors.Wrap(err, "could not write /etc/default/kubelet")
	}
	return nil
}

// StartRuntime starts the runtime
func StartRuntime(bc *bits.BuildContext) error {
	log.Info("starting dockerd")
	go func() {
		bc.RunInContainer("dockerd")
	}()

	duration := 10 * time.Second
	result := common.TryUntil(time.Now().Add(duration), func() bool {
		return bc.RunInContainer("bash", "-c", "docker info &> /dev/null") == nil
	})
	if !result {
		return errors.Errorf("dockerd did not start in %v", duration)
	}
	log.Info("dockerd started")
	return nil
}

// StopRuntime stops the runtime
func StopRuntime(bc *bits.BuildContext) error {
	return bc.RunInContainer("pkill", "-f", "dockerd")
}

// PreLoadInitImages preload images required by kubeadm-init into the docker runtime that exists inside a kind(er) node
func PreLoadInitImages(bc *bits.BuildContext, srcFolder string) error {
	// Currently docker images preloaded at build time gets lost at commit time, so this is a NOP
	// and images tars are loaded at node create time.
	// If we manage to get this working then we will speed up node creation time for docker;
	// A possible hint to solve is to remove VOLUME [ "/var/lib/docker" ] from the base image and add "--change", `VOLUME [ "/var/lib/docker" ]`
	// on commit, however this will force kinder to start using a new base image for all the docker jobs in ci
	return nil
}

// ImportImage import a TAR file into the CR and delete it
func ImportImage(bc *bits.BuildContext, tar string) error {
	// NO-OP for Docker
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
