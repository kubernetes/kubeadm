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

package cluster

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"
)

// GetKubeadmConfigPatches returns the kubeadm config patches for the Kubernetes initVersion
func GetKubeadmConfigPatches(initVersion string) (string, string, string, error) {
	// gets the kubeadm config version corresponding to a Kubernetes initVersion
	initConfigVersion, err := getKubeadmConfigVersion(initVersion)
	if err != nil {
		return "", "", "", err
	}

	// select the patches for the kubeadm config version
	log.Infof("Preparing kubeadm config patches for %s version", initConfigVersion)
	switch initConfigVersion {
	case "v1beta2":
		return kubeDNSPatchv1beta2, calicoPatchv1beta2, externalEtcdPatchv1beta2, nil
	case "v1beta1":
		return kubeDNSPatchv1beta1, calicoPatchv1beta1, externalEtcdPatchv1beta1, nil
	case "v1alpha3":
		return kubeDNSPatchv1alpha3, calicoPatchv1alpha3, externalEtcdPatchv1alpha3, nil
	case "v1alpha2":
		return kubeDNSPatchv1alpha2, calicoPatchv1alpha2, externalEtcdPatchv1alpha2, nil
	}

	return "", "", "", errors.Errorf("unknown kubeadm config version: %s", initConfigVersion)
}

// getKubeadmConfigVersion returns the kubeadm config version corresponding to a Kubernetes initVersion
func getKubeadmConfigVersion(initVersion string) (string, error) {
	// parses the initVersion
	vS, err := version.ParseSemantic(initVersion)
	if err != nil {
		return "", errors.Wrapf(err, "%s is not a valid version", initVersion)
	}

	// returns the corresponding kubeadm config version
	// nb v1alpha1 (that is Kubernetes v1.10.0) is out of support

	if vS.LessThan(version.MustParseSemantic("v1.12.0-0")) {
		return "v1alpha2", nil
	}

	if vS.LessThan(version.MustParseSemantic("v1.13.0-0")) {
		return "v1alpha3", nil
	}

	if vS.LessThan(version.MustParseSemantic("v1.15.0-0")) {
		return "v1beta1", nil
	}

	return "v1beta2", nil
}

const kubeDNSPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`

const externalEtcdPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const calicoPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
networking:
  podSubnet: "192.168.0.0/16"`

const kubeDNSPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`

const externalEtcdPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const calicoPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
networking:
  podSubnet: "192.168.0.0/16"`

const kubeDNSPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
metadata:
  name: config
featureGates:
  CoreDNS: false`

const externalEtcdPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const calicoPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
metadata:
  name: config
networking:
  podSubnet: "192.168.0.0/16"`

const kubeDNSPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: MasterConfiguration
metadata:
  name: config
featureGates:
  CoreDNS: false`

const externalEtcdPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: MasterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const calicoPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: MasterConfiguration
metadata:
  name: config
networking:
  podSubnet: "192.168.0.0/16"`
