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

package kubeadm

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// GetDockerPatch returns the kubeadm config patch that will instruct kubeadm
// to setup user docker CRI defaults.
func GetDockerPatch(kubeadmConfigVersion string, ControlPlane bool) ([]string, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing dockerPatch for kubeadm config %s", kubeadmConfigVersion)

	var basePatch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		basePatch = dockerPatchv1beta2
	case "v1beta3":
		basePatch = dockerPatchv1beta3
	default:
		return nil, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	// kind kubeadm config template for v1alpha3, v1beta1, v1beta2, v1beta3 returns both InitConfiguration and JoinConfiguration
	// so we should create two patches
	return []string{
		fmt.Sprintf(basePatch, "InitConfiguration"),
		fmt.Sprintf(basePatch, "JoinConfiguration"),
	}, nil
}

const dockerPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: %s
metadata:
  name: config
nodeRegistration:
  criSocket: /var/run/dockershim.sock`

const dockerPatchv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: %s
metadata:
  name: config
nodeRegistration:
  criSocket: /var/run/dockershim.sock`
