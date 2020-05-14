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
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cri/util"
	"k8s.io/kubeadm/kinder/pkg/exec"
)

// CreateNode creates a container that internally hosts the containerd cri runtime
func CreateNode(cluster, name, image, role string, volumes []string) error {
	args, err := util.CommonArgs(cluster, name, role)
	if err != nil {
		return err
	}

	args, err = util.RunArgsForNode(role, volumes, args)
	if err != nil {
		return err
	}

	// Specify the image to run
	args = append(args, image)

	// creates the container
	if err := exec.NewHostCmd("docker", args...).Run(); err != nil {
		return err
	}

	// load the image artifacts into containerd
	loadImages(name)

	return nil
}

// loadImages loads image tarballs stored on the node into containerd on the node
func loadImages(name string) {
	// load images cached on the node into containerd
	if err := exec.NewNodeCmd(name,
		"/bin/bash", "-c",
		// use xargs to load images in parallel
		`find /kind/images -name *.tar -print0 | xargs -r -0 -n 1 -P $(nproc) ctr --namespace=k8s.io images import --no-unpack`,
	).Silent().Run(); err != nil {
		log.Warningf("Failed to preload containerd images from /kind/images: %v", err)
		return
	}
}
