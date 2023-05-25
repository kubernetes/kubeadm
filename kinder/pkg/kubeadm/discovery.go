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
	"k8s.io/kubeadm/kinder/pkg/constants"
)

// GetRemoveTokenPatch returns the kubeadm config patch that will instruct kubeadm
// to not uses token discovery.
func GetRemoveTokenPatch(kubeadmConfigVersion string) (PatchJSON6902, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing removeTokenPatch for kubeadm config %s", kubeadmConfigVersion)

	var patch string
	kind := "JoinConfiguration"
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = removeTokenPatchv1beta2
	case "v1beta3":
		patch = removeTokenPatchv1beta3
	default:
		return PatchJSON6902{}, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return PatchJSON6902{
		Group:   "kubeadm.k8s.io",
		Version: kubeadmConfigVersion,
		Kind:    kind,
		Patch:   patch,
	}, nil
}

const removeTokenPatchv1beta2 = `
- op: remove
  path: "/discovery/bootstrapToken"`

const removeTokenPatchv1beta3 = `
- op: remove
  path: "/discovery/bootstrapToken"`

// GetFileDiscoveryPatch returns the kubeadm config patch that will instruct kubeadm
// to use FileDiscovery.
func GetFileDiscoveryPatch(kubeadmConfigVersion string) (string, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing fileDiscoveryPatch for kubeadm config %s", kubeadmConfigVersion)

	var patch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = fileDiscoveryPatchv1beta2
	case "v1beta3":
		patch = fileDiscoveryPatchv1beta3
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(patch, constants.DiscoveryFile), nil
}

const fileDiscoveryPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
discovery:
  file:
    kubeConfigPath: %s`

const fileDiscoveryPatchv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  file:
    kubeConfigPath: %s`

// GetTLSBootstrapPatch returns the kubeadm config patch that will instruct kubeadm
// to use a TLSBootstrap token.
// NB. for sake of semplicity, we are using the same Token already used for Token discovery
func GetTLSBootstrapPatch(kubeadmConfigVersion string) (string, error) {
	// select the patches for the kubeadm config version
	log.Debugf("Preparing tlsBootstrapPatch for kubeadm config %s", kubeadmConfigVersion)

	var patch string
	switch kubeadmConfigVersion {
	case "v1beta2":
		patch = tlsBootstrapPatchv1beta2
	case "v1beta3":
		patch = tlsBootstrapPatchv1beta3
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	return fmt.Sprintf(patch, constants.Token), nil
}

const tlsBootstrapPatchv1beta2 = `apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
discovery:
  tlsBootstrapToken: %s`

const tlsBootstrapPatchv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  tlsBootstrapToken: %s`
