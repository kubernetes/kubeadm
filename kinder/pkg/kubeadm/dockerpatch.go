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

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
)

// GetDockerPatch returns the kubeadm config patch that will instruct kubeadm
// to setup user docker CRI defaults.
func GetDockerPatch(kubeadmVersion *K8sVersion.Version, ControlPlane bool) ([]string, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return nil, err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing dockerPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)

	var basePatch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		basePatch = dockerPatchv1beta2
	case "v1beta1":
		basePatch = dockerPatchv1beta1
	case "v1alpha3":
		basePatch = dockerPatchv1alpha3
	case "v1alpha2":
		// kind kubeadm config template for v1alpha2 returns only MasterConfiguration or NodeConfiguration
		// so we should create patches accordingly
		if ControlPlane {
			return []string{
				fmt.Sprintf(dockerPatchv1alpha2, "MasterConfiguration"),
			}, nil
		}
		return []string{
			fmt.Sprintf(dockerPatchv1alpha2, "NodeConfiguration"),
		}, nil
	default:
		return nil, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	// kind kubeadm config template for v1alpha3, v1beta1,v1beta2 returns both InitConfiguration and JoinConfiguration
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

const dockerPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: %s
metadata:
  name: config
nodeRegistration:
  criSocket: /var/run/dockershim.sock`

const dockerPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: %s
metadata:
  name: config
nodeRegistration:
  criSocket: /var/run/dockershim.sock`

const dockerPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: %s
metadata:
  name: config
nodeRegistration:
  criSocket: /var/run/dockershim.sock`
