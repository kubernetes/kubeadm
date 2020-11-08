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

package nodes

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes/containerd"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes/docker"
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

// GetAlterContainerArgs ...
func (h *AlterHelper) GetAlterContainerArgs() ([]string, []string) {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.GetAlterContainerArgs()
	case status.DockerRuntime:
		return docker.GetAlterContainerArgs()
	}
	return []string{}, []string{}
}

// StartCRI starts the CRI engine
func (h *AlterHelper) StartCRI(bc *bits.BuildContext) error {
	// start the container runtime
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.StartRuntime(bc)
	case status.DockerRuntime:
		return docker.StartRuntime(bc)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// PreLoadInitImages preload images required by kubeadm-init into the selected container runtime that exists inside a kind(er) node
func (h *AlterHelper) PreLoadInitImages(bc *bits.BuildContext, srcFolder string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.PreLoadInitImages(bc, srcFolder)
	case status.DockerRuntime:
		return docker.PreLoadInitImages(bc, srcFolder)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// StopCRI stops the CRI engine
func (h *AlterHelper) StopCRI(bc *bits.BuildContext) error {
	// stop the container runtime
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.StopRuntime(bc)
	case status.DockerRuntime:
		return docker.StopRuntime(bc)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// ImportImage imports a set of TAR files into the CR
func (h *AlterHelper) ImportImage(bc *bits.BuildContext, tar string) error {
	switch h.cri {
	case status.ContainerdRuntime:
		return containerd.ImportImage(bc, tar)
	case status.DockerRuntime:
		return docker.ImportImage(bc, tar)
	}
	return errors.Errorf("unknown cri: %s", h.cri)
}

// GetImagesForKubeadmBinary runs a kubeadm binary located at "binaryPath" and gets the images it returns
func (h *AlterHelper) GetImagesForKubeadmBinary(bc *bits.BuildContext, binaryPath string) ([]string, error) {
	images, err := bc.CombinedOutputLinesInContainer(
		"bash",
		"-c",
		binaryPath+" config images list --kubernetes-version=$("+binaryPath+" version -o short) 2> /dev/null | grep -v 'kube-'",
	)
	if err != nil {
		return nil, err
	}
	log.Infof("Found the following extra images from the binary %q: %v", binaryPath, images)

	return images, nil
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
