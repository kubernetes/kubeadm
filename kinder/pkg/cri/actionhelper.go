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

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri/containerd"
	"k8s.io/kubeadm/kinder/pkg/cri/docker"
)

// ActionHelper helper provides CRI specific methods used by kind(er) actions
type ActionHelper struct {
	cri status.ContainerRuntime
}

// NewActionHelper returns a new AlterHelper
func NewActionHelper(cri status.ContainerRuntime) (*ActionHelper, error) {
	return &ActionHelper{
		cri: cri,
	}, nil
}

// PreLoadUpgradeImages preload images required by kubeadm-update into the selected container runtime that exists inside a kind(er) node
func (h *ActionHelper) PreLoadUpgradeImages(n *status.Node, srcFolder string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.PreLoadUpgradeImages(n, srcFolder)
	case status.DockerRuntime:
		return docker.PreLoadUpgradeImages(n, srcFolder)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// GetImages prints the images available in the node
func (h *ActionHelper) GetImages(n *status.Node) ([]string, error) {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.GetImages(n)
	case status.DockerRuntime:
		return docker.GetImages(n)
	}
	return nil, errors.Errorf("unknown cri: %s", h.cri)
}
