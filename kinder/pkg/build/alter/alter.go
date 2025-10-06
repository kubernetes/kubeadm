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
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/build/bits"
	"k8s.io/kubeadm/kinder/pkg/cluster/manager/actions/assets"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri/host"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes"
	"k8s.io/kubeadm/kinder/pkg/exec"
	"k8s.io/kubeadm/kinder/pkg/extract"
	kindfs "sigs.k8s.io/kind/pkg/fs"
)

// DefaultBaseImage is the default base image used
const DefaultBaseImage = "kindest/node:latest"

// DefaultImage is the default name:tag for the alter image
const DefaultImage = DefaultBaseImage

// Context is used to alter the kind node image, and contains
// alter configuration
type Context struct {
	baseImage               string
	image                   string
	initArtifactsSrc        string
	imageSrcs               []string
	imageNamePrefix         string
	upgradeArtifactsSrc     string
	kubeadmSrc              string
	kubeletSrc              string
	prePullAdditionalImages bool
	paths                   []string
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

// WithPrePullAdditionalImages configures a NewContext to pre-pull kubeadm additional required images
func WithPrePullAdditionalImages(pull bool) Option {
	return func(b *Context) {
		b.prePullAdditionalImages = pull
	}
}

// WithPath configures a NewContext to include a file/dir on the host
func WithPath(paths []string) Option {
	return func(b *Context) {
		b.paths = append(b.paths, paths...)
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
	// create tempdir to alter the image in
	alterDir, err := kindfs.TempDir("", "kinder-alter-image")
	if err != nil {
		return err
	}
	defer os.RemoveAll(alterDir)

	// initialize the build context
	bc := bits.NewBuildContext(alterDir)

	// always create folder for storing bits output
	bitsDir := bc.HostBitsPath()
	if err := os.Mkdir(bitsDir, 0777); err != nil {
		return errors.Wrap(err, "failed to make bits dir")
	}

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
		// If the upgrade artifacts source is the same as the init artifacts source,
		// avoid downloading artifacts again and just copy them.
		src := c.upgradeArtifactsSrc
		if src == c.initArtifactsSrc {
			src = filepath.Join(bc.HostBitsPath(), bits.InitBitsDir)
		}
		bitsInstallers = append(bitsInstallers, bits.NewUpgradeBits(src))
	}

	if len(c.paths) > 0 {
		bitsInstallers = append(bitsInstallers, bits.NewPathBits(c.paths))
	}

	log.Infof("Altering node image in: %s", alterDir)

	// populate the kubernetes artifacts first
	if err := c.prepareBits(bitsInstallers, bc); err != nil {
		return err
	}

	// then the perform the actual docker image alter
	return c.alterImage(bitsInstallers, bc)
}

func (c *Context) prepareBits(bitsInstallers []bits.Installer, bc *bits.BuildContext) error {
	log.Info("Preparing bits ...")

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
			if slices.Contains(extract.AllKubernetesImages, k) {
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
	err = host.EditArchiveRepositories(f, &w, fixRepository)
	if err != nil {
		return err
	}
	f.Close()

	// override the image tar with the fixed version
	err = os.WriteFile(v, []byte(w.String()), 0644)
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
	// get the container runtime from the base image
	runtime, err := status.InspectCRIinImage(c.baseImage)
	if err != nil {
		return errors.Wrap(err, "error detecting CRI!")
	}
	log.Infof("Detected %s as container runtime", runtime)

	alterHelper, err := nodes.NewAlterHelper(runtime)
	if err != nil {
		return err
	}

	// get the args for the alter container depending on the underlying CR
	runArgs, containerArgs := alterHelper.GetAlterContainerArgs()

	// create final alter container
	// NOTE: we are using docker run + docker commit so we can install
	// debians without permanently copying them into the image.
	// if docker gets proper squash support, we can rm them instead
	// This also allows the KubeBit implementations to perform programmatic
	// install in the image
	log.Debug("Starting alter container ...")
	containerID, err := c.createAlterContainer(bc, runArgs, containerArgs)
	// ensure we will delete it
	if containerID != "" {
		defer func() {
			exec.NewHostCmd("docker", "rm", "-f", "-v", containerID).Run()
		}()
	}
	if err != nil {
		log.Errorf("Image alter Failed! %v", err)
		return err
	}

	// alter the image, tagged as tagImageAs, using the our tempdir as the context
	log.Debug("Starting image alter ...")

	// binds the BuildContext the container
	bc.BindToContainer(containerID)

	// Make sure the /kind/images folder exists
	if err := bc.RunInContainer("mkdir", "-p", "/kind/images"); err != nil {
		return err
	}

	// install the bits that are used to alter the image
	log.Info("Starting bits install ...")
	for _, b := range bitsInstallers {
		if err = b.Install(bc); err != nil {
			log.Errorf("Image build Failed! %v", err)
			return err
		}
	}

	log.Info("Setup CRI ...")
	if err := alterHelper.SetupCRI(bc); err != nil {
		return errors.Wrapf(err, "image build Failed! Failed to setup %s", runtime)
	}

	log.Info("Start CRI ...")
	if err := alterHelper.StartCRI(bc); err != nil {
		return errors.Wrapf(err, "image build Failed! Failed to start %s", runtime)
	}

	log.Info("Pre-loading images ...")
	if err := alterHelper.PreLoadInitImages(bc, "/kind/images"); err != nil {
		return errors.Wrapf(err, "image build Failed! Failed to start %s", runtime)
	}

	if c.prePullAdditionalImages {
		log.Info("Pre-pulling additional images ...")

		// pull images required for init / join
		initPath := "/kind"
		images, err := alterHelper.GetImagesForKubeadmBinary(bc, filepath.Join(initPath, "bin", "kubeadm"))
		if err != nil {
			return err
		}

		// add the kindnet image
		images = append(images, assets.KindnetImage185)

		if err := pullImages(alterHelper, bc, images, filepath.Join(initPath, "images"), containerID); err != nil {
			return err
		}

		// pull images required for upgrade
		upgradePath := "/kinder/upgrade"
		// check if the version file for the upgrade artifacts is in place
		versionFile := filepath.Join(upgradePath, "version")
		version, err := bc.CombinedOutputLinesInContainer(
			"bash",
			"-c",
			"cat "+versionFile+" 2> /dev/null",
		)

		// don't return the error if the version file is missing
		if err == nil {
			if len(version) != 1 {
				return errors.Errorf("expected the version file %q to have 1 line, got: %v", versionFile, version)
			}

			// use the resulting upgrade path e.g. /kinder/upgrade/v1.19.0-alpha.3.36+8c4e3faed35411
			upgradeImages, err := alterHelper.GetImagesForKubeadmBinary(bc, filepath.Join(upgradePath, version[0], "kubeadm"))
			if err != nil {
				return err
			}

			if err := pullImages(alterHelper, bc, upgradeImages, filepath.Join(upgradePath, version[0]), containerID); err != nil {
				return err
			}
		}
	}

	log.Info("Stop CRI ...")
	if err := alterHelper.StopCRI(bc); err != nil {
		return errors.Wrapf(err, "image build Failed! Failed to stop %s", runtime)
	}

	log.Infof("Commit to %s ...", c.image)
	if err = alterHelper.Commit(containerID, c.image); err != nil {
		return errors.Wrap(err, "image alter Failed! Failed to commit image")
	}

	log.Info("Image alter completed.")

	return nil
}

func pullImages(alterHelper *nodes.AlterHelper, bc *bits.BuildContext, images []string, savePath, containerID string) error {
	tempDir, err := os.MkdirTemp("", "kinder-image-path")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	imageRegExp := regexp.MustCompile("[/:]")

	for _, image := range images {
		// Pull the image on the host
		if err := exec.NewHostCmd("docker", "pull", image).Run(); err != nil {
			return errors.Wrapf(err, "failed to pull image %q on the host", image)
		}

		// Create the path where the tar is going to be saved
		s := imageRegExp.Split(image, -1)
		if len(s) < 3 {
			return errors.Errorf("unsupported image URL: %s", image)
		}
		fileName := s[len(s)-2] + ".tar"
		hostPath := filepath.Join(tempDir, fileName)

		// Save the tar
		if err := exec.NewHostCmd("docker", "save", "-o="+hostPath, image).Run(); err != nil {
			return errors.Wrapf(err, "failed to save image %q to path %q", image, hostPath)
		}

		// Copy the tar to the container
		if err := exec.NewHostCmd("docker", "cp", hostPath, containerID+":"+savePath).Run(); err != nil {
			return errors.Wrapf(err, "failed to copy the file %q to container %q", image, containerID)
		}

		// Import the image in the runtime (containerd only, deletes the file from the container after import)
		if err := alterHelper.ImportImage(bc, filepath.Join(savePath, fileName)); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) createAlterContainer(bc *bits.BuildContext, runArgs, containerArgs []string) (id string, err error) {
	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = host.PullImage(c.baseImage, 4)

	// define docker default args
	id = "kind-build-" + uuid.New().String()
	args := []string{
		"-d", // make the client exit while the container continues to run
		"-v", fmt.Sprintf("%s:%s", bc.HostBasePath(), bc.ContainerBasePath()),
		"--name=" + id,
	}
	args = append(args, runArgs...)

	if err = host.Run(c.baseImage, args, containerArgs); err != nil {
		return id, errors.Wrap(err, "failed to create alter container")
	}
	return id, nil
}
