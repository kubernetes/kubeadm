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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
)

// GetKubeDNSPatch returns the kubeadm config patch that will instruct kubeadm
// to use kube-dns instead of CoreDNS.
func GetKubeDNSPatch(kubeadmVersion *K8sVersion.Version) (string, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return "", err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing KubeDNSPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)
	switch kubeadmConfigVersion {
	case "v1beta2":
		return kubeDNSPatchv1beta2, nil
	case "v1beta1":
		return kubeDNSPatchv1beta1, nil
	}

	return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
}

const kubeDNSPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`

const kubeDNSPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`
