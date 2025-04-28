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
	"maps"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// imageBits defines a bit installer that allows to add new images tarball in the /kind/images folder into the node image;
// those images will be automatically loaded into docker when the container/the node will start
type imageBits struct {
	srcs       []string
	namePrefix string
}

var _ Installer = &imageBits{}

// NewImageBits returns a new imageBits
func NewImageBits(args []string, namePrefix string) Installer {
	return &imageBits{
		srcs:       args,
		namePrefix: namePrefix,
	}
}

// Get implements Installer.Get
func (b *imageBits) Prepare(c *BuildContext) (map[string]string, error) {
	// ensure the dest path exists on host/inside the HostBitsPath
	dst := filepath.Join(c.HostBitsPath(), "images")
	if err := os.Mkdir(dst, 0777); err != nil {
		return nil, errors.Wrap(err, "failed to make bits dir")
	}

	// for each of the given sources
	allImages := map[string]string{}
	for _, src := range b.srcs {
		// Creates an extractor instance, that will read the binary bit from the src,
		// that can be one of version/build-label/file or folder containing the binary,
		// and save it to the dest path (inside HostBitsPath)
		e := extract.NewExtractor(
			src, dst,
			extract.OnlyKubernetesImages(true),
			extract.WithNamePrefix(b.namePrefix),
		)

		// if the source is a local repository
		if extract.GetSourceType(src) == extract.LocalRepositorySource {
			// sets the extractor for importing all image tarballs existing in the local repository,
			// not only the kubernetes ones (this will allow to use this function for loading other images)
			e.SetFiles(extract.AllImagesPattern)
		}

		// Extracts the image tarballs bit
		images, err := e.Extract()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract %s", src)
		}

		// keeps track of the image tarballs bit extracted from the source
		maps.Copy(allImages, images)
	}

	return allImages, nil
}

// Install implements bits.Install
func (b *imageBits) Install(c *BuildContext) error {

	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join(c.ContainerBitsPath(), "images")

	// The dest path is /kind/images, a well known folder where kind(er) will
	// search for pre-loaded images during `kind(er) create`
	dest := filepath.Join("/kind")

	// copy artifacts in
	if err := c.RunInContainer("rsync", "-r", src, dest); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	// make sure we own the tarballs
	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", filepath.Join("/kind", "images")); err != nil {
		log.Errorf("Image alter failed! %v", err)
		return err
	}

	return nil
}
