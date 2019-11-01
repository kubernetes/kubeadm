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

// CreateHelper provides CRI specific methods for node create
type CreateHelper struct {
	cri status.ContainerRuntime
}

// NewCreateHelper returns a new CreateHelper
func NewCreateHelper(cri status.ContainerRuntime) (*CreateHelper, error) {
	return &CreateHelper{
		cri: cri,
	}, nil
}

// CreateControlPlaneNode creates a kind(er) contol-plane node that uses the selected container runtime internally
func (h *CreateHelper) CreateControlPlaneNode(name, image, clusterLabel string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.CreateControlPlaneNode(name, image, clusterLabel)
	case status.DockerRuntime:
		return docker.CreateControlPlaneNode(name, image, clusterLabel)
	}

	return errors.Errorf("unknown cri: %s", h.cri)
}

// CreateWorkerNode creates a kind(er) worker node node that uses the selected container runtime internally
func (h *CreateHelper) CreateWorkerNode(name, image, clusterLabel string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.CreateWorkerNode(name, image, clusterLabel)
	case status.DockerRuntime:
		return docker.CreateWorkerNode(name, image, clusterLabel)
	}

	return errors.Errorf("unknown cri: %s", h.cri)
}
