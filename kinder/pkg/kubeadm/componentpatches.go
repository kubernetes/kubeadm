/*
Copyright 2021 The Kubernetes Authors.

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

// GetPatchesDirectoryPatches returns the kubeadm config patches that will instruct kubeadm
// to use patches directory.
func GetPatchesDirectoryPatches(kubeadmConfigVersion string) ([]string, error) {
	log.Debugf("Preparing patches directory for kubeadm config %s", kubeadmConfigVersion)
	var patchInit, patchJoin, patchUpgradeApply, patchUpgradeNode string
	switch kubeadmConfigVersion {
	case "v1beta3":
		patchInit = patchesDirectoryPatchInitv1beta3
		patchJoin = patchesDirectoryPatchJoinv1beta3
		return []string{
			fmt.Sprintf(patchInit, constants.PatchesDir),
			fmt.Sprintf(patchJoin, constants.PatchesDir),
		}, nil
	case "v1beta4":
		patchInit = patchesDirectoryPatchInitv1beta4
		patchJoin = patchesDirectoryPatchJoinv1beta4
		patchUpgradeApply = patchesDirectoryPatchUpgradeApplyv1beta4
		patchUpgradeNode = patchesDirectoryPatchUpgradeNodev1beta4
		return []string{
			fmt.Sprintf(patchInit, constants.PatchesDir),
			fmt.Sprintf(patchJoin, constants.PatchesDir),
			fmt.Sprintf(patchUpgradeApply, constants.PatchesDir),
			fmt.Sprintf(patchUpgradeNode, constants.PatchesDir),
		}, nil
	default:
		return []string{}, errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}
}

const patchesDirectoryPatchInitv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
patches:
  directory: %s`

const patchesDirectoryPatchJoinv1beta3 = `apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
patches:
  directory: %s`

const patchesDirectoryPatchInitv1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
patches:
  directory: %s`

const patchesDirectoryPatchJoinv1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: JoinConfiguration
patches:
  directory: %s`

const patchesDirectoryPatchUpgradeApplyv1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: UpgradeConfiguration
apply:
  patches:
    directory: %s`

const patchesDirectoryPatchUpgradeNodev1beta4 = `apiVersion: kubeadm.k8s.io/v1beta4
kind: UpgradeConfiguration
node:
  patches:
    directory: %s`
