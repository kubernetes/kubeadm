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

package cri

import (
	"github.com/pkg/errors"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/kubeadm"
)

// ConfigHelper provides CRI specific methods for creating the kubeadm config
type ConfigHelper struct {
	cri status.ContainerRuntime
}

// NewConfigHelper returns a new ConfigHelper
func NewConfigHelper(cri status.ContainerRuntime) (*ConfigHelper, error) {
	return &ConfigHelper{
		cri: cri,
	}, nil
}

// GetKubeadmConfigPatches returns kustomize patches for configuring the kubeadm config file for using the selected container runtime
func (h *ConfigHelper) GetKubeadmConfigPatches(kubeadmVersion *K8sVersion.Version, controlPlane bool) ([]string, error) {
	switch h.cri {
	case status.ContainerdRuntime:
		// since we are using kind library for generating the kubeadm-config file, and kind uses by default containerd, no
		// additional pathches are required in this case
		return []string{}, nil
	case status.DockerRuntime:
		return kubeadm.GetDockerPatch(kubeadmVersion, controlPlane)
	}
	return nil, errors.Errorf("unknown cri: %s", h.cri)
}
