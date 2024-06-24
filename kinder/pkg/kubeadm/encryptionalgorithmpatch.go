/*
Copyright 2024 The Kubernetes Authors.

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

// GetEncryptionAlgorithmPatch returns the kubeadm config patch that will instruct kubeadm
// to use a specific encryption algorithm
func GetEncryptionAlgorithmPatch(kubeadmConfigVersion string, algorithm string) (string, error) {
	var patch string
	log.Debugf("Preparing encryptionAlgorithm patch for kubeadm config %s", kubeadmConfigVersion)

	switch kubeadmConfigVersion {
	case "v1beta3":
		return "", errors.New("ClusterConfiguration.encryptionAlgorithm is not supported in v1beta3")
	case "v1beta4":
		patch = encryptionAlgorithmPatchV1beta4
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(patch, algorithm), nil
}

const encryptionAlgorithmPatchV1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
encryptionAlgorithm: %s
`
