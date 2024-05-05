/*
Copyright 2020 The Kubernetes Authors.

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

package host

import (
	"time"

	"k8s.io/kubeadm/kinder/pkg/exec"
)

// PullImage will pull an image if it is not present locally
// retrying up to retries times
// it returns true if it attempted to pull, and any errors from pulling
func PullImage(image string, retries int) (bool, error) {
	// once we have configurable log levels
	// if this did not return an error, then the image exists locally
	if err := exec.NewHostCmd("docker", "inspect", "--type=image", image).Run(); err == nil {
		return false, nil
	}

	// otherwise try to pull it
	var err error
	if err = exec.NewHostCmd("docker", "pull", image).Run(); err != nil {
		for i := 0; i < retries; i++ {
			time.Sleep(time.Second * time.Duration(i+1))
			if err = exec.NewHostCmd("docker", "pull", image).Run(); err == nil {
				break
			}
		}
	}
	return true, err
}
