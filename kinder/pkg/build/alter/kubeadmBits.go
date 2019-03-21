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

//TODO: use const for paths and filenames

// kubeadmBits implements Bits for the overriding the kubeadm binary into the node image
type kubeadmBits struct {
	src string
}

var _ bits = &kubeadmBits{}

// newKubeadmBits returns a new bits backed by the kubeadm binary
func newKubeadmBits(arg string) bits {
	return &kubeadmBits{
		src: arg,
	}
}

// Paths implements bits.Paths
func (b *kubeadmBits) Paths() map[string]string {
	var paths = map[string]string{}

	// gets the  src descriptor
	info, err := os.Stat(b.src)
	if err != nil {
		log.Warningf("Error getting file descriptor for %q: %v", b.src, err)
	}

	// check if the file is a valid kubeadm file (if not discard)
	if !(filepath.Base(b.src) == "kubeadm" && info.Mode().IsRegular()) {
		log.Warningf("Image file %q is not a valid kubeadm binary file. Removed from kubeadmBits", b.src)
	}

	// Add to the path list; the dest path is a kubeadm file into the alterDir
	dest := "kubeadm"
	paths[b.src] = dest
	log.Debugf("kubeadmBits %s added to paths", b.src)

	return paths
}

// Install implements bits.Install
func (b *kubeadmBits) Install(ic *installContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join("/alter", "bits", "kubeadm")

	// The dest path is /usr/bin/kubeadm, the location of the kubeadm binary
	// installed in the node image during kind(er) build node-image
	dest := filepath.Join("/usr", "bin", "kubeadm")

	// copy artifacts in
	if err := ic.Run("cp", src, dest); err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	// make sure we own the packages
	// TODO: someday we might need a different user ...
	if err := ic.Run("chown", "-R", "root:root", "/usr/bin/kubeadm"); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
