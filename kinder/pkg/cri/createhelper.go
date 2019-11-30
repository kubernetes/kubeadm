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
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri/containerd"
	"k8s.io/kubeadm/kinder/pkg/cri/docker"
	"k8s.io/kubeadm/kinder/pkg/cri/util"
	"k8s.io/kubeadm/kinder/pkg/exec"
	"k8s.io/kubeadm/kinder/third_party/kind/loadbalancer"
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

// CreateNode creates a container that internally hosts the selected cri runtime
func (h *CreateHelper) CreateNode(cluster, name, image, role string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.CreateNode(cluster, name, image, role)
	case status.DockerRuntime:
		return docker.CreateNode(cluster, name, image, role)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// CreateExternalEtcd creates a container hosting a single node, insecure, external etcd cluster
func (h *CreateHelper) CreateExternalEtcd(cluster, name, image string) error {
	args, err := util.CommonArgs(cluster, name, constants.ExternalEtcdNodeRoleValue)
	if err != nil {
		return err
	}

	// Add etcd run args
	args = util.RunArgsForExternalEtcd(args)

	// Specify the image to run
	args = append(args, image)

	// Add container args for starting a single node, insecure etcd
	args = util.ContainerArgsForExternalEtcd(cluster, args)

	// creates the container
	return exec.NewHostCmd("docker", args...).Run()
}

// CreateExternalLoadBalancer creates a container hosting an external load balancer
func (h *CreateHelper) CreateExternalLoadBalancer(cluster, name string) error {
	args, err := util.CommonArgs(cluster, name, constants.ExternalLoadBalancerNodeRoleValue)
	if err != nil {
		return err
	}

	// Add load balancer run args
	args, err = util.RunArgsForExternalLoadBalancer(args)
	if err != nil {
		return err
	}

	// Specify the image to run
	args = append(args, loadbalancer.Image)

	// creates the container
	return exec.NewHostCmd("docker", args...).Run()
}
