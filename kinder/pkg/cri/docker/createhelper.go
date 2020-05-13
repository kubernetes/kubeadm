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
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cri/util"
	"k8s.io/kubeadm/kinder/pkg/exec"
	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
)

// CreateNode creates a container that internally hosts the docker cri runtime
func CreateNode(cluster, name, image, role string, volumes []string) error {
	args, err := util.CommonArgs(cluster, name, role)
	if err != nil {
		return err
	}

	args, err = util.RunArgsForNode(role, volumes, args)
	if err != nil {
		return err
	}

	// Add run args for docker in docker
	args = runArgsForDocker(args)

	// Specify the image to run
	args = append(args, image)

	// dd container args for docker in docker
	args = containerArgsForDocker(args)

	// creates the container
	if err := exec.NewHostCmd("docker", args...).Run(); err != nil {
		return err
	}

	// Deletes the machine-id embedded in the node image and regenerate a new one.
	// This is necessary because both kubelet and other components like weave net
	// use machine-id internally to distinguish nodes.
	if err := fixMachineID(name); err != nil {
		return err
	}

	// we need to change a few mounts once we have the container
	// we'd do this ahead of time if we could, but --privileged implies things
	// that don't seem to be configurable, and we need that flag
	if err := fixMounts(name); err != nil {
		return err
	}

	// signal the node container entrypoint to continue booting into systemd
	if err := signalStart(name); err != nil {
		return err
	}

	// wait for docker to be ready
	if !waitForDocker(name, time.Now().Add(time.Second*30)) {
		return errors.Errorf("timed out waiting for docker to be ready on node %s", name)
	}

	// load the docker image artifacts into the docker daemon
	loadImages(name)

	return nil
}

func fixMachineID(name string) error {
	if err := exec.NewNodeCmd(name, "rm", "-f", "/etc/machine-id").Silent().Run(); err != nil {
		return errors.Wrap(err, "machine-id-setup error")
	}
	if err := exec.NewNodeCmd(name, "systemd-machine-id-setup").Silent().Run(); err != nil {
		return errors.Wrap(err, "machine-id-setup error")
	}
	return nil
}

func runArgsForDocker(args []string) []string {
	args = append(args,
		"--entrypoint=/usr/local/bin/entrypoint",
	)

	return args
}

func containerArgsForDocker(args []string) []string {
	args = append(args,
		"/sbin/init",
	)

	return args
}

// fixMounts will correct mounts in the node container to meet the right
// sharing and permissions for systemd and Docker / Kubernetes
func fixMounts(name string) error {
	// Check if userns-remap is enabled
	if kinddocker.UsernsRemap() {
		// The binary /bin/mount should be owned by root:root in order to execute
		// the following mount commands
		if err := exec.NewNodeCmd(name, "chown", "root:root", "/bin/mount").Silent().Run(); err != nil {
			return err
		}
		// The binary /bin/mount should have the setuid bit
		if err := exec.NewNodeCmd(name, "chmod", "-s", "/bin/mount").Silent().Run(); err != nil {
			return err
		}
	}

	// systemd-in-a-container should have read only /sys
	// https://systemd.io/CONTAINER_INTERFACE/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := exec.NewNodeCmd(name, "mount", "-o", "remount,ro", "/sys").Silent().Run(); err != nil {
		return err
	}
	// kubernetes needs shared mount propagation
	if err := exec.NewNodeCmd(name, "mount", "--make-shared", "/").Silent().Run(); err != nil {
		return err
	}
	if err := exec.NewNodeCmd(name, "mount", "--make-shared", "/run").Silent().Run(); err != nil {
		return err
	}
	if err := exec.NewNodeCmd(name, "mount", "--make-shared", "/var/lib/docker").Silent().Run(); err != nil {
		return err
	}
	return nil
}

// signalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func signalStart(name string) error {
	return kinddocker.Kill("SIGUSR1", name)
}

// waitForDocker waits for Docker to be ready on the node
// it returns true on success, and false on a timeout
func waitForDocker(name string, until time.Time) bool {
	return tryUntil(until, func() bool {
		out, err := exec.NewNodeCmd(name, "systemctl", "is-active", "docker").Silent().RunAndCapture()
		if err != nil {
			return false
		}
		return len(out) == 1 && out[0] == "active"
	})
}

// helper that calls `try()`` in a loop until the deadline `until`
// has passed or `try()`returns true, returns whether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}

// loadImages loads image tarballs stored on the node into docker on the node
func loadImages(name string) {
	// load images cached on the node into docker
	if err := exec.NewNodeCmd(name,
		"/bin/bash", "-c",
		// use xargs to load images in parallel
		`find /kind/images -name *.tar -print0 | xargs -0 -n 1 -P $(nproc) docker load -i`,
	).Silent().Run(); err != nil {
		log.Warningf("Failed to preload docker images from /kind/images: %v", err)
		return
	}
}
