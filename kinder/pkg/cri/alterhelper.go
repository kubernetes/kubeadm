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

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri/containerd"
	"k8s.io/kubeadm/kinder/pkg/cri/docker"
)

// AlterHelper provides CRI specific methods for altering a kind(er) images
type AlterHelper struct {
	cri status.ContainerRuntime
}

// NewAlterHelper returns a new AlterHelper
func NewAlterHelper(cri status.ContainerRuntime) (*AlterHelper, error) {
	return &AlterHelper{
		cri: cri,
	}, nil
}

// PreLoadInitImages preload images required by kubeadm-init into the selected container runtime that exists inside a kind(er) node
func (h *AlterHelper) PreLoadInitImages(bc *bits.BuildContext) error {
	if err := bc.RunInContainer("mkdir", "-p", "/kind/images"); err != nil {
		return err
	}

	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.PreLoadInitImages(bc)
	case status.DockerRuntime:
		return docker.PreLoadInitImages(bc)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// Commit a kind(er) node image that uses the selected container runtime internally
func (h *AlterHelper) Commit(containerID, targetImage string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.Commit(containerID, targetImage)
	case status.DockerRuntime:
		return docker.Commit(containerID, targetImage)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}
