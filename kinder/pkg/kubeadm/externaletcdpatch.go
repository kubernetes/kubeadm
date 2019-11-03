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

// GetExternalEtcdPatch returns the kubeadm config patch that will instruct kubeadm
// to use external etcd.
func GetExternalEtcdPatch(kubeadmVersion *K8sVersion.Version, etcdIP string) (string, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return "", err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing externalEtcdPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)

	var externalEtcdPatch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		externalEtcdPatch = externalEtcdPatchv1beta2
	case "v1beta1":
		externalEtcdPatch = externalEtcdPatchv1beta1
	case "v1alpha3":
		externalEtcdPatch = externalEtcdPatchv1alpha3
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(externalEtcdPatch, etcdIP), nil
}

const externalEtcdPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const externalEtcdPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const externalEtcdPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`
