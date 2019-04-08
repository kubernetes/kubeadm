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
	imageSrcs           []string
	imageNamePrefix     string
	upgradeArtifactsSrc string
	kubeadmSrc          string
	kubeletSrc          string
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
func WithImageTars(srcs []string) Option {
	return func(b *Context) {
		b.imageSrcs = append(b.imageSrcs, srcs...)
	}
}

// WithImageNamePrefix configures a NewContext to add a name prefix to included images tars
func WithImageNamePrefix(namePrefix string) Option {
	return func(b *Context) {
		b.imageNamePrefix = namePrefix
	}
}

// WithUpgradeArtifacts configures a NewContext to include binaries & images for upgrade
func WithUpgradeArtifacts(src string) Option {
	return func(b *Context) {
		b.upgradeArtifactsSrc = src
	}
}

// WithKubeadm configures a NewContext to override the kubeadm binary
func WithKubeadm(src string) Option {
	return func(b *Context) {
		b.kubeadmSrc = src
	}
}

// WithKubelet configures a NewContext to override the kubelet binary
func WithKubelet(src string) Option {
	return func(b *Context) {
		b.kubeletSrc = src
	}
}

// NewContext creates a new Context with default configuration,
// overridden by the options supplied in the order that they are supplied
func NewContext(options ...Option) (ctx *Context, err error) {
	// default options
	ctx = &Context{}

	// apply user options
	for _, option := range options {
		option(ctx)
	}

	return ctx, nil
}

// Alter alters the cluster node image
func (c *Context) Alter() (err error) {
	// initialize bits
	var bits []bits

	if c.kubeadmSrc != "" {
		bits = append(bits, newBinaryBits(c.kubeadmSrc, "kubeadm"))
	}
	if c.kubeletSrc != "" {
		bits = append(bits, newBinaryBits(c.kubeletSrc, "kubelet"))
	}

	if len(c.imageSrcs) > 0 {
		bits = append(bits, newImageBits(c.imageSrcs, c.imageNamePrefix))
	}

	if c.upgradeArtifactsSrc != "" {
		bits = append(bits, newUpgradeBits(c.upgradeArtifactsSrc))
	}

	// create tempdir to alter the image in
	alterDir, err := fs.TempDir("", "kinder-alter-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(alterDir)
	log.Infof("Altering node image in: %s", alterDir)

	// initialize the bits working context
	bc := &bitsContext{
		hostBasePath: alterDir,
	}

	// always create folder for storing bits output
	bitsDir := bc.HostBitsPath()
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}

	// populate the kubernetes artifacts first
	if err := c.populateBits(bits, bc); err != nil {
		return err
	}

	// then the perform the actual docker image alter
	return c.alterImage(bits, bc)
}

func (c *Context) populateBits(bits []bits, bc *bitsContext) error {
	log.Info("Starting populate bits ...")

	for _, b := range bits {
		if err := b.Get(bc); err != nil {
			return errors.Wrap(err, "failed to copy alter bits")
		}
	}
	return nil
}

func (c *Context) alterImage(bits []bits, bc *bitsContext) error {
	// alter the image, tagged as tagImageAs, using the our tempdir as the context
	log.Debug("Starting image alter ...")

	// create alter container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// install in the image
	containerID, err := c.createAlterContainer(bc)
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

	// binds the bitsContext the the container
	bc.BindToContainer(containerID)

	// install the bits that are used to alter the image
	log.Info("Starting bits install ...")
	for _, b := range bits {
		if err = b.Install(bc); err != nil {
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

func (c *Context) createAlterContainer(bc *bitsContext) (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = docker.PullIfNotPresent(c.baseImage, 4)
	id, err = docker.Run(
		c.baseImage,
		docker.WithRunArgs(
			"-d", // make the client exit while the container continues to run
			// label the container to make them easier to track
			"--label", fmt.Sprintf("%s=%s", AlterContainerLabelKey, time.Now().Format(time.RFC3339Nano)),
			"-v", fmt.Sprintf("%s:%s", bc.HostBasePath(), bc.ContainerBasePath()),
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
