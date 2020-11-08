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
	"k8s.io/kubeadm/kinder/pkg/cri/nodes/common"
	"k8s.io/kubeadm/kinder/pkg/exec"
)

// CreateNode creates a container that internally hosts the containerd cri runtime
func CreateNode(cluster, name, image, role string, volumes []string) error {
	args, err := common.BaseRunArgs(cluster, name, role)
	if err != nil {
		return err
	}

	args, err = common.RunArgsForNode(role, volumes, args)
	if err != nil {
		return err
	}

	// Specify the image to run
	args = append(args, image)

	// creates the container
	if err := exec.NewHostCmd("docker", args...).Run(); err != nil {
		return err
	}

	return nil
}
