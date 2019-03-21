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
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

//TODO: use const for paths

// imageBits implements Bits for the copying into the node image additional image tars
type imageBits struct {
	srcs []string
}

var _ bits = &imageBits{}

func newImageBits(args []string) bits {
	return &imageBits{
		srcs: args,
	}
}

// Paths implements bits.Paths
func (b *imageBits) Paths() map[string]string {
	var paths = map[string]string{}

	// for each of the given path
	for _, src := range b.srcs {
		// gets the  src descriptor
		info, err := os.Stat(src)
		if err != nil {
			log.Warningf("Error getting file descriptor for %q: %v", src, err)
		}

		// if src is a Directory
		if info.IsDir() {
			// gets all the entries in the folder
			entries, err := ioutil.ReadDir(src)
			if err != nil {
				log.Warningf("Error getting directory content for %q: %v", src, err)
			}

			// for each entry in the folder
			for _, entry := range entries {
				// check if the file is a valid tar file (if not discard)
				name := entry.Name()
				if !(filepath.Ext(name) == ".tar" && entry.Mode().IsRegular()) {
					log.Warningf("Image file %q is not a valid .tar file. Removed from imageBits", name)
					continue
				}

				// Add to the path list; the dest path is a subfolder into the alterDir
				entrySrc := filepath.Join(src, name)
				entryDest := filepath.Join("images", name)
				paths[entrySrc] = entryDest
				log.Debugf("imageBits %s added to paths", entrySrc)
			}
			continue
		}

		// check if the file is a valid tar file (if not discard)
		if !(filepath.Ext(src) == ".tar" && info.Mode().IsRegular()) {
			log.Warningf("Image file %q is not a valid .tar file. Removed from imageBits", src)
		}

		// Add to the path list; the dest path is a subfolder into the alterDir
		dest := filepath.Join("images", filepath.Base(src))
		paths[src] = dest
		log.Debugf("imageBits %s added to paths", src)
	}

	return paths
}

// Install implements bits.Install
func (b *imageBits) Install(ic *installContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join("/alter", "bits", "images")

	// The dest path is /kind/images, a well known folder where kind(er) will
	// search for pre-loaded images during `kind(er) create`
	dest := filepath.Join("/kind")

	// copy artifacts in
	if err := ic.Run("rsync", "-r", src, dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// make sure we own the tarballs
	// TODO: someday we might need a different user ...
	if err := ic.Run("chown", "-R", "root:root", filepath.Join("/kind", "images")); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
