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

package nodevariant

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/pkg/build/alter"
	kindebuildnode "sigs.k8s.io/kind/pkg/build/node"
)

type flagpole struct {
	Image            string
	BaseImage        string
	InitArtifacts    string
	ImageTars        []string
	ImageNamePrefix  string
	UpgradeArtifacts string
	Kubeadm          string
	Kubelet          string
	CRI              string
}

// NewCommand returns a new cobra.Command for building the node image
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:    cobra.NoArgs,
		Use:     "node-image-variant",
		Aliases: []string{"node-variant", "variant", "nv"},
		Short:   "build the node image variant",
		Long:    "build the variant for a node image by adding packages, images or replacing the kubeadm binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.Image, "image",
		kindebuildnode.DefaultImage,
		"name:tag of the resulting image to be built",
	)
	cmd.Flags().StringVar(
		&flags.BaseImage, "base-image",
		kindebuildnode.DefaultImage,
		"name:tag of the source image; this can be a kindest/base image or kindest/node image",
	)
	cmd.Flags().StringVar(
		&flags.InitArtifacts, "with-init-artifacts",
		"",
		"version/build-label/path to a folder with Kubernetes binaries & image tarballs to be used for the kubeadm init workflow",
	)
	cmd.Flags().StringSliceVar(
		&flags.ImageTars, "with-images",
		nil,
		"version/build-label/path to images tar or folder with images tars to be added to the images",
	)
	//TODO: remove this as soon CRI autodetection is implemented
	cmd.Flags().StringVar(
		&flags.CRI, "cri",
		"",
		"specifies the cri installed inside the base image",
	)
	cmd.Flags().StringVar(
		&flags.ImageNamePrefix, "image-name-prefix",
		"",
		"add a name prefix to images tars included in the image",
	)
	cmd.Flags().StringVar(
		&flags.UpgradeArtifacts, "with-upgrade-artifacts",
		"",
		"version/build-label/path to a folder with Kubernetes binaries & image tarballs to be used for testing the kubeadm-upgrade workflow",
	)
	cmd.Flags().StringVar(
		&flags.Kubeadm, "with-kubeadm",
		"",
		"override the kubeadm binary existing in the image with the given version/build-label/file or folder containing the kubeadm binary",
	)
	cmd.Flags().StringVar(
		&flags.Kubelet, "with-kubelet",
		"",
		"override the kubeadm binary existing in the image with the given version/build-label/file or folder containing the kubelet binary",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// check the cri flag
	var err error
	cri := flags.CRI
	if cri == "" {
		return errors.Wrap(err, "Please use the --cri flag to specify the container runtime installed inside the base image")
	}

	ctx, err := alter.NewContext(
		// base build options
		alter.WithBaseImage(flags.BaseImage),
		alter.WithCRI(cri),
		alter.WithImage(flags.Image),
		// bits to be added to the image
		alter.WithInitArtifacts(flags.InitArtifacts),
		alter.WithKubeadm(flags.Kubeadm),
		alter.WithKubelet(flags.Kubelet),
		alter.WithImageTars(flags.ImageTars),
		alter.WithUpgradeArtifacts(flags.UpgradeArtifacts),
		// bits options
		alter.WithImageNamePrefix(flags.ImageNamePrefix),
	)
	if err != nil {
		return errors.Wrap(err, "error creating alter context")
	}
	if err := ctx.Alter(); err != nil {
		return errors.Wrap(err, "error altering node image")
	}
	return nil
}
