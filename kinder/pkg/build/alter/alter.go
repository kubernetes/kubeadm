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
	"fmt"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kind/pkg/container/docker"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/fs"
)

// DefaultBaseImage is the default base image used
const DefaultBaseImage = "kindest/node:latest"

// DefaultImage is the default name:tag for the alter image
const DefaultImage = DefaultBaseImage

// AlterContainerLabelKey is applied to each altered container
const AlterContainerLabelKey = "io.k8s.sigs.kinder.alter"

// Context is used to alter the kind node image, and contains
// alter configuration
type Context struct {
	baseImage           string
	image               string
	imagePaths          []string
	upgradeBinariesPath string
	kubeadmPath         string
	bits                []bits
}

// Option is Context configuration option supplied to NewContext
type Option func(*Context)

// WithImage configures a NewContext to tag the built image with `image`
func WithImage(image string) Option {
	return func(b *Context) {
		b.image = image
	}
}

// WithBaseImage configures a NewContext to use `image` as the base image
func WithBaseImage(image string) Option {
	return func(b *Context) {
		b.baseImage = image
	}
}

// WithImageTars configures a NewContext to include additional images tars
func WithImageTars(paths []string) Option {
	return func(b *Context) {
		b.imagePaths = append(b.imagePaths, paths...)
	}
}

// WithUpgradeBinaries configures a NewContext to include binaries for upgrade
func WithUpgradeBinaries(upgradeBinariesPath string) Option {
	return func(b *Context) {
		b.upgradeBinariesPath = upgradeBinariesPath
	}
}

// WithKubeadm configures a NewContext to override the kubeadm binary
func WithKubeadm(path string) Option {
	return func(b *Context) {
		b.kubeadmPath = path
	}
}

// NewContext creates a new Context with default configuration,
// overridden by the options supplied in the order that they are supplied
func NewContext(options ...Option) (ctx *Context, err error) {
	// default options
	ctx = &Context{
		baseImage: DefaultBaseImage,
		image:     DefaultBaseImage,
	}
	// apply user options
	for _, option := range options {
		option(ctx)
	}

	// initialize bits
	if len(ctx.imagePaths) > 0 {
		ctx.bits = append(ctx.bits, newImageBits(ctx.imagePaths))
	}

	if ctx.upgradeBinariesPath != "" {
		ctx.bits = append(ctx.bits, newUpgradeBinaryBits(ctx.upgradeBinariesPath))
	}

	if ctx.kubeadmPath != "" {
		ctx.bits = append(ctx.bits, newKubeadmBits(ctx.kubeadmPath))
	}

	return ctx, nil
}

// Alter alters the cluster node image
func (c *Context) Alter() (err error) {
	// create tempdir to alter the image in
	alterDir, err := fs.TempDir("", "kinder-alter-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(alterDir)
	log.Infof("Altering node image in: %s", alterDir)

	// populate the kubernetes artifacts first
	if err := c.populateBits(alterDir); err != nil {
		return err
	}

	// then the perform the actual docker image alter
	return c.alterImage(alterDir)
}

func (c *Context) populateBits(alterDir string) error {
	log.Info("Starting populate bits ...")

	// always create bits dir
	bitsDir := path.Join(alterDir, "bits")
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}
	// copy all bits from their source path to where we will COPY them into
	// the dockerfile, see images/node/Dockerfile
	for _, bits := range c.bits {
		bitPaths := bits.Paths()
		for src, dest := range bitPaths {
			realDest := path.Join(bitsDir, dest)
			log.Debugf("Copying: %s to %s", src, dest)
			// NOTE: we use copy not copyfile because copy ensures the dest dir
			if err := fs.Copy(src, realDest); err != nil {
				return errors.Wrap(err, "failed to copy alter bits")
			}
		}
	}
	return nil
}

func (c *Context) alterImage(dir string) error {
	// alter the image, tagged as tagImageAs, using the our tempdir as the context
	log.Debug("Starting image alter ...")

	// create alter container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// install in the image
	containerID, err := c.createAlterContainer(dir)
	// ensure we will delete it
	if containerID != "" {
		defer func() {
			exec.Command("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	// install the kube bits
	log.Info("Starting bits install ...")
	ic := &installContext{
		basePath:    dir,
		containerID: containerID,
	}
	for _, bits := range c.bits {
		if err = bits.Install(ic); err != nil {
			log.Errorf("Image build Failed! %v", err)
			return err
		}
	}

	// Save the image changes to a new image
	cmd := exec.Command("docker", "commit", containerID, c.image)
	exec.InheritOutput(cmd)
	if err = cmd.Run(); err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	log.Info("Image alter completed.")
	return nil
}

func (c *Context) createAlterContainer(alterDir string) (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = docker.PullIfNotPresent(c.baseImage, 4)
	id, err = docker.Run(
		c.baseImage,
		docker.WithRunArgs(
			"-d", // make the client exit while the container continues to run
			// label the container to make them easier to track
			"--label", fmt.Sprintf("%s=%s", AlterContainerLabelKey, time.Now().Format(time.RFC3339Nano)),
			"-v", fmt.Sprintf("%s:/alter", alterDir),
			// the container should hang forever so we can exec in it
			"--entrypoint=sleep",
		),
		docker.WithContainerArgs(
			"infinity", // sleep infinitely to keep the container around
		),
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create alter container")
	}
	return id, nil
}
