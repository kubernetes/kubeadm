/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// pathBits defines a bit installer that allows to copy a file or dir (recursively) from host machine into the node image
type pathBits struct {
	paths []string
}

var _ Installer = &pathBits{}

// NewPathBits returns a new path Installer
func NewPathBits(paths []string) Installer {
	return &pathBits{
		paths: paths,
	}
}

// Prepare implements bits.Prepare
func (b *pathBits) Prepare(c *BuildContext) (map[string]string, error) {

	// ensure the staging dest path exists on host at HostBitsPath
	dstDir := filepath.Join(c.HostBitsPath(), "files")
	if err := os.Mkdir(dstDir, 0777); err != nil {
		return nil, errors.Wrap(err, "failed to make bits dir")
	}

	for idx, path := range b.paths {

		pathSrcDst := strings.Split(path, ":")

		if len(pathSrcDst) != 2 {
			return nil, errors.New(fmt.Sprintf("invalid value %q for --with-path option, has to be in 'srcPath:destPath' format", path))
		}

		if !filepath.IsAbs(pathSrcDst[1]) {
			return nil, errors.New(fmt.Sprintf("invalid value %q for --with-path option, destPath has to be an absolute path", path))
		}

		// Creates an extractor instance, that will read the path bits from src,
		// and save it to the staging dest path (inside HostBitsPath)
		// Append index to file name to handle duplicate file names
		e := extract.NewExtractor(
			pathSrcDst[0], dstDir,
			extract.WithNameOverride(fmt.Sprintf("%s_%d", filepath.Base(pathSrcDst[0]), idx)),
			extract.WithVersionFile(false),
		)

		// if the src path is a directory, then setting files like this gives us the desired effect of extracting the
		// directory recursively, otherwise only the files inside the directory are extracted
		// this feels a little hacky, but this way we are reusing existing extraction logic in extract.go
		if info, err := os.Stat(pathSrcDst[0]); err == nil && info.IsDir() {
			e.SetFiles([]string{""})
		}

		_, err := e.Extract()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract %s", pathSrcDst[0])
		}

	}

	return nil, nil
}

// Install implements bits.Install
func (b *pathBits) Install(c *BuildContext) error {

	for idx, path := range b.paths {
		pathSrcDst := strings.Split(path, ":")

		// The src path is a subfolder into the alterDir, that is mounted in the container as /alter
		src := filepath.Join(c.ContainerBitsPath(), "files", fmt.Sprintf("%s_%d", filepath.Base(pathSrcDst[0]), idx))

		// ensure parent directories exist
		if err := c.RunInContainer("mkdir", "-p", filepath.Dir(pathSrcDst[1])); err != nil {
			log.Errorf("Image alter Failed! %v", err)
			return err
		}

		// copy file or directory
		if err := c.RunInContainer("cp", "-r", src, pathSrcDst[1]); err != nil {
			log.Errorf("Image alter Failed! %v", err)
			return err
		}

		// make sure we own the packages
		// TODO: someday we might need a different user ...
		if err := c.RunInContainer("chown", "-R", "root:root", pathSrcDst[1]); err != nil {
			log.Errorf("Image alter failed! %v", err)
			return err
		}

	}

	return nil
}
