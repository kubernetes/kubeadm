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
)

// GetKubeDNSPatch returns the kubeadm config patch that will instruct kubeadm
// to use kube-dns instead of CoreDNS.
func GetKubeDNSPatch(kubeadmConfigVersion string) (string, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing KubeDNSPatch for kubeadm config %s", kubeadmConfigVersion)
	switch kubeadmConfigVersion {
	case "v1beta2":
		return kubeDNSPatchv1beta2, nil
	case "v1beta3":
		return "", nil
	}

	return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
}

const kubeDNSPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`
