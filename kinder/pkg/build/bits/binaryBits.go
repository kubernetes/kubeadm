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

package bits

import (
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// binaryBits defines a bit installer that allows to override the binary files in /usr/bin into the node image
// using the binary bits existing in the src
type binaryBits struct {
	src        string
	binaryName string
}

var _ Installer = &binaryBits{}

// NewBinaryBits returns a new binary Installer
func NewBinaryBits(src, binaryName string) Installer {
	return &binaryBits{
		src:        src,
		binaryName: binaryName,
	}
}

// Get implements Installer.Get
func (b *binaryBits) Prepare(c *BuildContext) (map[string]string, error) {
	// Creates an extractor instance, that will read the binary bit from the src,
	// where source can be one of version/build-label/file or folder containing the binary,
	// and save it to the HostBitsPath
	e := extract.NewExtractor(
		b.src, c.HostBitsPath(),
		extract.OnlyKubeadm(b.binaryName == "kubeadm"),
		extract.OnlyKubelet(b.binaryName == "kubelet"),
	)

	// Extracts the binary bit
	return e.Extract()
}

// Install implements bits.Install
func (b *binaryBits) Install(c *BuildContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join(c.ContainerBitsPath(), b.binaryName)

	// The dest path is in /usr/bin, the location of the binary used by kind(er)
	dest := filepath.Join("/usr", "bin", b.binaryName)

	// copy artifacts
	if err := c.RunInContainer("cp", src, dest); err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	// make sure we own the packages
	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
