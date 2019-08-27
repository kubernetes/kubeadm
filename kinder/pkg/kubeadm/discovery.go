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
	"k8s.io/kubeadm/kinder/pkg/config"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

// GetRemoveTokenPatch returns the kubeadm config patch that will instruct kubeadm
// to not uses token discovery.
func GetRemoveTokenPatch(kubeadmVersion *K8sVersion.Version) (config.PatchJSON6902, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return config.PatchJSON6902{}, err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing removeTokenPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)

	var patch string
	kind := "JoinConfiguration"
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = removeTokenPatchv1beta2
	case "v1beta1":
		patch = removeTokenPatchv1beta1
	case "v1alpha3":
		patch = removeTokenPatchv1alpha3
	case "v1alpha2":
		kind = "NodeConfiguration"
		patch = removeTokenPatchv1alpha2
	default:
		return config.PatchJSON6902{}, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return config.PatchJSON6902{
		Group:   "kubeadm.k8s.io",
		Version: kubeadmConfigVersion,
		Kind:    kind,
		Name:    "config",
		Patch:   patch,
	}, nil
}

const removeTokenPatchv1beta2 = `
- op: remove
  path: "/discovery/bootstrapToken"`

const removeTokenPatchv1beta1 = `
- op: remove
  path: "/discovery/bootstrapToken"`

const removeTokenPatchv1alpha3 = `
- op: remove
  path: "/discoveryTokenAPIServers"
- op: remove
  path: "/token"
- op: remove
  path: "/discoveryTokenUnsafeSkipCAVerification"`

const removeTokenPatchv1alpha2 = `
- op: remove
  path: "/discoveryTokenAPIServers"
- op: remove
  path: "/token"
- op: remove
  path: "/discoveryTokenUnsafeSkipCAVerification"`

// GetFileDiscoveryPatch returns the kubeadm config patch that will instruct kubeadm
// to use FileDiscovery.
func GetFileDiscoveryPatch(kubeadmVersion *K8sVersion.Version) (string, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return "", err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing fileDiscoveryPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)

	var patch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = fileDiscoveryPatchv1beta2
	case "v1beta1":
		patch = fileDiscoveryPatchv1beta1
	case "v1alpha3":
		patch = fileDiscoveryPatchv1alpha3
	case "v1alpha2":
		patch = fileDiscoveryPatchv1alpha2
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(patch, constants.DiscoveryFile), nil
}

const fileDiscoveryPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
discovery:
  file:
    kubeConfigPath: %s`

const fileDiscoveryPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
metadata:
  name: config
discovery:
  file:
    kubeConfigPath: %s`

const fileDiscoveryPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
metadata:
  name: config
discoveryFile: %s`

const fileDiscoveryPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: NodeConfiguration
metadata:
  name: config
discoveryFile: %s`

// GetTLSBootstrapPatch returns the kubeadm config patch that will instruct kubeadm
// to use a TLSBootstrap token.
// NB. for sake of semplicity, we are using the same Token already used for Token discovery
func GetTLSBootstrapPatch(kubeadmVersion *K8sVersion.Version) (string, error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return "", err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing tlsBootstrapPatch for kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)

	var patch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = tlsBootstrapPatchv1beta2
	case "v1beta1":
		patch = tlsBootstrapPatchv1beta1
	case "v1alpha3":
		patch = tlsBootstrapPatchv1alpha3
	case "v1alpha2":
		patch = tlsBootstrapPatchv1alpha2
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(patch, constants.Token), nil
}

const tlsBootstrapPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
discovery:
  tlsBootstrapToken: %s`

const tlsBootstrapPatchv1beta1 = `apiVersion: kubeadm.k8s.io/v1beta1
kind: JoinConfiguration
metadata:
  name: config
discovery:
  tlsBootstrapToken: %s`

const tlsBootstrapPatchv1alpha3 = `apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
metadata:
  name: config
tlsBootstrapToken: %s`

const tlsBootstrapPatchv1alpha2 = `apiVersion: kubeadm.k8s.io/v1alpha2
kind: NodeConfiguration
metadata:
  name: config
tlsBootstrapToken: %s`
