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
	case "v1beta3":
		basePatch = dockerPatchv1beta3
	case "v1beta4":
		basePatch = dockerPatchv1beta4
	default:
		return nil, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return []string{
		fmt.Sprintf(basePatch, "InitConfiguration"),
		fmt.Sprintf(basePatch, "JoinConfiguration"),
	}, nil
}

const dockerPatchv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: %s
nodeRegistration:
  criSocket: /var/run/dockershim.sock`

const dockerPatchv1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: %s
nodeRegistration:
  criSocket: /var/run/dockershim.sock`
