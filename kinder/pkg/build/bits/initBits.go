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
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// initBits defines a bit installer that allows to add Kubernetes binaries & images to the node image;
// those artifact will be used by the kinder do kubeadm-init script
type initBits struct {
	src string
}

var _ Installer = &initBits{}

// NewInitBits returns a new initBits
func NewInitBits(arg string) Installer {
	return &initBits{
		src: arg,
	}
}

// Get implements Installer.Get
func (b *initBits) Prepare(c *BuildContext) (map[string]string, error) {
	// ensure the dest path exists on host/inside the HostBitsPath
	dst := filepath.Join(c.HostBitsPath(), "init")
	if err := os.Mkdir(dst, 0777); err != nil {
		return nil, errors.Wrap(err, "failed to make bits dir")
	}

	// Creates an extractor instance, that will read binaries & images required for init from the src,
	// where source can be one of version/build-label/folder containing the  binaries & images,
	// and save it to the dst folder
	e := extract.NewExtractor(
		b.src, dst,
	)

	// Extracts the binaries & images
	return e.Extract()
}

// Install implements Installer.Install
func (b *initBits) Install(c *BuildContext) error {

	// install Kubernetes version file for init
	if err := installInitVersionFile(c); err != nil {
		return err
	}

	// install Kubernetes images for init
	if err := installInitImages(c); err != nil {
		return err
	}

	// install Kubernetes binaries for init
	if err := installInitBinaries(c); err != nil {
		return err
	}

	// configure kubelet service with the kubeadm drop in file
	if err := configureKubelet(c); err != nil {
		return err
	}

	return nil
}

func installInitVersionFile(c *BuildContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join(c.ContainerBitsPath(), "init")

	// The dest path for Kubernetes version file is /kind, a well known folder where kind(er) will
	// search for this file when looking for image metadata
	dst := filepath.Join("/kind")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dst); err != nil {
		log.Errorf("failed to create %s folder into the image! %v", dst, err)
		return err
	}

	// copy image tarballs artifacts into the image
	srcVersion := filepath.Join(src, "version")
	dstVersion := filepath.Join(dst, "version")
	log.Infof("Adding %s file to the image", dstVersion)
	if err := c.RunInContainer("cp", srcVersion, dstVersion); err != nil {
		log.Errorf("failed to copy %s into the image! %v", src, err)
		return err
	}

	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", dstVersion); err != nil {
		log.Errorf("failed to set ownership on %s! %v", dstVersion, err)
		return err
	}

	return nil
}

func installInitImages(c *BuildContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	hsrc := filepath.Join(c.HostBitsPath(), "init")
	csrc := filepath.Join(c.ContainerBitsPath(), "init")

	// The dest path for Kubernetes images is /kind/images, a well known folder where kind(er) will
	// search for pre-loaded images during `kind(er) create`
	dst := filepath.Join("/kind", "images")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dst); err != nil {
		log.Errorf("failed to create %s folder into the image! %v", dst, err)
		return err
	}

	// copy image tarballs artifacts into the image
	pattern := filepath.Join(hsrc, "*.tar")
	images, err := filepath.Glob(pattern)
	log.Debugf("Searching %s", pattern)
	log.Debugf("Found %s", images)
	if err != nil {
		return errors.Wrapf(err, "failed to search for images in %s", hsrc)
	}
	for _, image := range images {
		srcImage := filepath.Join(csrc, filepath.Base(image))
		dstImage := filepath.Join(dst, filepath.Base(image))
		log.Infof("Adding %s image tarball to the image", dstImage)
		if err := c.RunInContainer("cp", srcImage, dstImage); err != nil {
			log.Errorf("failed to copy %s into the image! %v", srcImage, err)
			return err
		}

		// TODO: someday we might need a different user ...
		if err := c.RunInContainer("chown", "-R", "root:root", dstImage); err != nil {
			log.Errorf("failed to set ownership on %s! %v", dstImage, err)
			return err
		}
	}

	return nil
}

func installInitBinaries(c *BuildContext) error {
	// The src path is a subfolder into the alterDir, that is mounted in the
	// container as /alter
	src := filepath.Join(c.ContainerBitsPath(), "init")

	// The destination path for the Kubernetes binaries is /kind/bin, a well known folder where kind adds Kubernetes binaries.
	dst := filepath.Join("/kind", "bin")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dst); err != nil {
		log.Errorf("failed to create %s folder into the image! %v", dst, err)
		return err
	}

	// copy binary artifacts into the image and symlink the kubernetes binaries into $PATH
	binaries := []string{"kubeadm", "kubelet", "kubectl"}
	for _, binary := range binaries {
		srcBinary := filepath.Join(src, binary)
		dstBinary := filepath.Join(dst, binary)
		lnBinary := filepath.Join("/usr", "bin", binary)
		log.Infof("Adding %s binary to the image and symlink to %s", dstBinary, lnBinary)
		if err := c.RunInContainer("cp", srcBinary, dstBinary); err != nil {
			log.Errorf("failed to copy %s into the image! %v", src, err)
			return err
		}

		// TODO: someday we might need a different user ...
		if err := c.RunInContainer("chown", "-R", "root:root", dstBinary); err != nil {
			log.Errorf("failed to set ownership on %s! %v", dstBinary, err)
			return err
		}

		if err := c.RunInContainer("ln", "-sf", dstBinary, lnBinary); err != nil {
			return errors.Wrapf(err, "failed to symlink %s", dstBinary)
		}
	}

	return nil
}

func configureKubelet(c *BuildContext) error {
	// files for the kubelet.service is created on the flight directly into the alter filesystem
	hsrc := filepath.Join(c.HostBitsPath(), "systemd")
	if err := os.MkdirAll(hsrc, 0777); err != nil {
		log.Errorf("failed to create %s folder! %v", hsrc, err)
		return err
	}
	csrc := filepath.Join(c.ContainerBitsPath(), "systemd")

	// The destination path for the kubelet.service file is /kind/systemd,
	dst := filepath.Join("/kind", "systemd")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dst); err != nil {
		log.Errorf("failed to create %s folder into the image! %v", dst, err)
		return err
	}

	// write the kubelet.service file
	hsrcFile := filepath.Join(hsrc, "kubelet.service")
	csrcFile := filepath.Join(csrc, "kubelet.service")
	dstFile := filepath.Join(dst, "kubelet.service")
	log.Infof("Adding %s to the image and enabling the kubelet service", dstFile)
	if err := os.WriteFile(hsrcFile, kubeletService, 0644); err != nil {
		log.Errorf("failed to create %s file into the image! %v", dstFile, err)
		return err
	}

	if err := c.RunInContainer("cp", csrcFile, dstFile); err != nil {
		log.Errorf("failed to copy %s into the image! %v", csrcFile, err)
		return err
	}

	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", dstFile); err != nil {
		log.Errorf("failed to set ownership on %s! %v", dstFile, err)
		return err
	}

	// enable the kubelet service
	if err := c.RunInContainer("systemctl", "enable", dstFile); err != nil {
		return errors.Wrap(err, "failed to enable kubelet service")
	}

	// The destination path for the kubeadm dropin file is /etc/systemd/system/kubelet.service.d/
	dst = filepath.Join("/etc", "systemd", "system", "kubelet.service.d")

	// create dest folder
	if err := c.RunInContainer("mkdir", "-p", dst); err != nil {
		log.Errorf("failed to create %s folder into the image! %v", dst, err)
		return err
	}

	// write the 10-kubeadm.conf file
	hsrcFile = filepath.Join(hsrc, "10-kubeadm.conf")
	csrcFile = filepath.Join(csrc, "10-kubeadm.conf")
	dstFile = filepath.Join(dst, "10-kubeadm.conf")
	log.Infof("Adding %s to the image", dstFile)
	if err := os.WriteFile(hsrcFile, kubeadmDropIn, 0644); err != nil {
		log.Errorf("failed to create %s file into the image! %v", dstFile, err)
		return err
	}

	if err := c.RunInContainer("cp", csrcFile, dstFile); err != nil {
		log.Errorf("failed to copy %s into the image! %v", csrcFile, err)
		return err
	}

	// TODO: someday we might need a different user ...
	if err := c.RunInContainer("chown", "-R", "root:root", dstFile); err != nil {
		log.Errorf("failed to set ownership on %s! %v", dstFile, err)
		return err
	}

	return nil
}

// from k/k build/debs/kubelet.service
var kubeletService = []byte(`
[Unit]
Description=kubelet: The Kubernetes Node Installer
Documentation=http://kubernetes.io/docs/

[Service]
ExecStart=/usr/bin/kubelet
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`)

// from k/k build/debs/10-kubeadm.conf
var kubeadmDropIn = []byte(`
# Note: This dropin only works with kubeadm and kubelet v1.11+
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_CONFIG_ARGS=--config=/var/lib/kubelet/config.yaml"
# This is a file that "kubeadm init" and "kubeadm join" generates at runtime, populating the KUBELET_KUBEADM_ARGS variable dynamically
EnvironmentFile=-/var/lib/kubelet/kubeadm-flags.env
# This is a file that the user can use for overrides of the kubelet args as a last resort. Preferably, the user should use
# the .NodeRegistration.KubeletExtraArgs object in the configuration files instead. KUBELET_EXTRA_ARGS should be sourced from this file.
EnvironmentFile=-/etc/default/kubelet
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_CONFIG_ARGS $KUBELET_KUBEADM_ARGS $KUBELET_EXTRA_ARGS
`)
