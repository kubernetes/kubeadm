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

	log "github.com/sirupsen/logrus"
)

// upgradeBinaryBits implements Bits for the copying into the node image debian packages for upgrades
type upgradeBinaryBits struct {
	src string
}

var _ bits = &upgradeBinaryBits{}

func newUpgradeBinaryBits(arg string) bits {
	return &upgradeBinaryBits{
		src: arg,
	}
}

// Paths implements bits.Paths
func (b *upgradeBinaryBits) Paths() map[string]string {
	var paths = map[string]string{}

	addPathForBinary := func(binary string) {
		// gets the  src descriptor
		src := filepath.Join(b.src, binary)
		info, err := os.Stat(src)
		if err != nil {
			log.Warningf("Error getting file descriptor for %q: %v", src, err)
		}

		// check if the file is a valid file (if not error)
		if !(info.Mode().IsRegular()) {
			log.Warningf("%q is not a valid binary file. Removed from upgradeBinaryBits", src)
		}

		// Add to the path list; the dest path is subfolder into the alterDir
		dest := filepath.Join("upgrade", binary)
		paths[src] = dest
		log.Debugf("upgradeBinaryBits %s added to paths", src)
	}

	addPathForBinary("kubeadm")
	addPathForBinary("kubelet")
	addPathForBinary("kubectl")

	return paths
}

// Install implements bits.Install
func (b *upgradeBinaryBits) Install(ic *installContext) error {

	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join("/alter", "bits", "upgrade")

	// The dest path is /kinder/upgrades, a well known folder where kinder will
	// search when executing the upgrade procedure
	dest := filepath.Join("/kinder")

	// create dest folder
	if err := ic.Run("mkdir", "-p", dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// copy artifacts in
	if err := ic.Run("rsync", "-r", src, dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// make sure we own the binary
	// TODO: someday we might need a different user ...
	if err := ic.Run("chown", "-R", "root:root", filepath.Join("/kinder", "upgrade")); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// make sure the binary are executable
	// TODO: someday we might need a different user ...
	if err := ic.Run("chmod", "-R", "+x", filepath.Join("/kinder", "upgrade")); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
