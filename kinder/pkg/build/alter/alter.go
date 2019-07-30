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
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri"
	"k8s.io/kubeadm/kinder/pkg/extract"
	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
	kindexec "sigs.k8s.io/kind/pkg/exec"
	kindfs "sigs.k8s.io/kind/pkg/fs"
)

// DefaultBaseImage is the default base image used
const DefaultBaseImage = "kindest/node:latest"

// DefaultImage is the default name:tag for the alter image
const DefaultImage = DefaultBaseImage

// Context is used to alter the kind node image, and contains
// alter configuration
type Context struct {
	baseImage           string
	image               string
	initArtifactsSrc    string
	imageSrcs           []string
	imageNamePrefix     string
	upgradeArtifactsSrc string
	kubeadmSrc          string
	kubeletSrc          string
}

// Option is Context configuration option supplied to NewContext
type Option func(*Context)

// WithInitArtifacts configures a NewContext to include binaries & images for init
func WithInitArtifacts(src string) Option {
	return func(b *Context) {
		b.initArtifactsSrc = src
	}
}

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
	// initialize bits installers
	var bitsInstallers []bits.Installer

	if c.initArtifactsSrc != "" {
		bitsInstallers = append(bitsInstallers, bits.NewInitBits(c.initArtifactsSrc))
	}

	if c.kubeadmSrc != "" {
		bitsInstallers = append(bitsInstallers, bits.NewBinaryBits(c.kubeadmSrc, "kubeadm"))
	}
	if c.kubeletSrc != "" {
		bitsInstallers = append(bitsInstallers, bits.NewBinaryBits(c.kubeletSrc, "kubelet"))
	}

	if len(c.imageSrcs) > 0 {
		bitsInstallers = append(bitsInstallers, bits.NewImageBits(c.imageSrcs, c.imageNamePrefix))
	}

	if c.upgradeArtifactsSrc != "" {
		bitsInstallers = append(bitsInstallers, bits.NewUpgradeBits(c.upgradeArtifactsSrc))
	}

	// create tempdir to alter the image in
	alterDir, err := kindfs.TempDir("", "kinder-alter-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(alterDir)
	log.Infof("Altering node image in: %s", alterDir)

	// initialize the build context
	bc := bits.NewBuildContext(alterDir)

	// always create folder for storing bits output
	bitsDir := bc.HostBitsPath()
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}

	// populate the kubernetes artifacts first
	if err := c.prepareBits(bitsInstallers, bc); err != nil {
		return err
	}

	// then the perform the actual docker image alter
	return c.alterImage(bitsInstallers, bc)
}

func (c *Context) prepareBits(bitsInstallers []bits.Installer, bc *bits.BuildContext) error {
	log.Info("Preparing bits ...")

	var isAKubernetesImages = func(i string) bool {
		for _, ki := range extract.AllKubernetesImages {
			if i == ki {
				return true
			}
		}
		return false
	}

	for _, b := range bitsInstallers {
		// prepare bits
		bits, err := b.Prepare(bc)
		if err != nil {
			return errors.Wrap(err, "failed to copy alter bits")
		}

		// fix the bits in order to match kubeadm/kinder expectations
		// NB. this is done here so all the bits gets fixes, no matter of the source
		for k, v := range bits {
			// if the bit is one of the kubernetes images, we should ensure the repository/name matches kubeadm expectations
			if isAKubernetesImages(k) {
				if err := fixImageTar(v); err != nil {
					return errors.Wrap(err, "failed to fix bits")
				}
			}
		}
	}

	return nil
}

// fixImageTar ensure the repository/name matches kubeadm expectations
func fixImageTar(v string) error {
	log.Infof("fixing %s", v)

	// prepare to read the image tar
	f, err := os.Open(v)
	if err != nil {
		return err
	}
	defer f.Close()

	// read the image tar and write the fixed version on a string builder
	var w strings.Builder
	err = kinddocker.EditArchiveRepositories(f, &w, fixRepository)
	if err != nil {
		return err
	}
	f.Close()

	// override the image tar with the fixed version
	err = ioutil.WriteFile(v, []byte(w.String()), 0644)
	if err != nil {
		return err
	}

	return nil
}

// fixRepository drop the arch suffix from images to get the expected image;
// this is necessary for kubernetes v1.15+
// Nb. for < v1.12 it was requested to do the opposite, but it not necessary anymore
// because v.11 is already out of the kubeadm e2e test matrix
func fixRepository(repository string) string {
	archSuffix := "-amd64"

	if strings.HasSuffix(repository, archSuffix) {
		fixed := strings.TrimSuffix(repository, archSuffix)
		fmt.Println("fixed: " + repository + " -> " + fixed)
		repository = fixed
	}

	return repository
}

func (c *Context) alterImage(bitsInstallers []bits.Installer, bc *bits.BuildContext) error {
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
			kindexec.Command("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	// alter the image, tagged as tagImageAs, using the our tempdir as the context
	log.Debug("Starting image alter ...")

	// binds the BuildContext the the container
	bc.BindToContainer(containerID)

	// install the bits that are used to alter the image
	log.Info("Starting bits install ...")
	for _, b := range bitsInstallers {
		if err = b.Install(bc); err != nil {
			log.Errorf("Image build Failed! %v", err)
			return err
		}
	}

	runtime, err := status.InspectCRIinContainer(containerID)
	if err != nil {
		return errors.Wrap(err, "Error detecting CRI!")
	}
	log.Infof("Detected %s as container runtime", runtime)

	alterHelper, err := cri.NewAlterHelper(runtime)
	if err != nil {
		return err
	}

	log.Info("Pre loading images ...")
	if err := alterHelper.PreLoadInitImages(bc); err != nil {
		return errors.Wrapf(err, "Image build Failed! Failed to load images into %s", runtime)
	}

	log.Infof("Commit to %s ...", c.image)
	if err = alterHelper.Commit(containerID, c.image); err != nil {
		return errors.Wrap(err, "Image alter Failed! Failed to commit image")
	}

	log.Info("Image alter completed.")

	return nil
}

func (c *Context) createAlterContainer(bc *bits.BuildContext) (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = kinddocker.PullIfNotPresent(c.baseImage, 4)

	// define docker default args
	id = "kind-build-" + uuid.New().String()
	args := []string{
		"-d", // make the client exit while the container continues to run
		"-v", fmt.Sprintf("%s:%s", bc.HostBasePath(), bc.ContainerBasePath()),
		// the container should hang forever so we can exec in it
		"--entrypoint=sleep",
		"--name=" + id,
	}

	err = kinddocker.Run(
		c.baseImage,
		kinddocker.WithRunArgs(
			args...,
		),
		kinddocker.WithContainerArgs(
			"infinity", // sleep infinitely to keep the container around
		),
	)
	if err != nil {
		return id, errors.Wrap(err, "failed to create alter container")
	}
	return id, nil
}
