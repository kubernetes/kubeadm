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

package alter

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// upgradeBits implements Bits implements a bits that allows to add new upgrade binaries & images /kinder/upgrade folder into the node image;
// those artifact will be used by the kinder do kubeadm-upgrade script
type upgradeBits struct {
	src string
}

var _ bits = &upgradeBits{}

func newUpgradeBits(arg string) bits {
	return &upgradeBits{
		src: arg,
	}
}

// Get implements bits.Getget
func (b *upgradeBits) Get(c *bitsContext) error {
	// ensure the dest path exists on host/inside the HostBitsPath
	dst := filepath.Join(c.HostBitsPath(), "upgrade")
	if err := os.Mkdir(dst, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}

	// Creates an extractor instance, that will read binaries & images required from upgrades from the src,
	// where source can be one of version/build-label/folder containing the  binaries & images,
	// and save it to the HostBitsPath/version
	e := extract.NewExtractor(
		b.src, dst,
		extract.WithVersionFolder(true),
	)

	// Extracts the binary bit
	_, err := e.Extract()
	return err
}

// Install implements bits.Install
func (b *upgradeBits) Install(c *bitsContext) error {

	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join(c.ContainerBitsPath(), "upgrade")

	// The dest path is /kinder/upgrades, a well known folder where kinder will
	// search when executing the upgrade procedure
	dest := filepath.Join("/kinder")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// copy artifacts in
	if err := c.RunInContainer("rsync", "-r", src, dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// make sure we own the binary
	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", filepath.Join("/kinder", "upgrade")); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
