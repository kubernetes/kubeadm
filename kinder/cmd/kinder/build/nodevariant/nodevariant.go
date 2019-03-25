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
	"sigs.k8s.io/kind/pkg/build/node"
)

type flagpole struct {
	Image           string
	BaseImage       string
	ImageTars       []string
	UpgradeBinaries string
	Kubeadm         string
}

// NewCommand returns a new cobra.Command for building the node image
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "node-variant",
		Short: "build the node image variant",
		Long:  "build the variant for a node image by adding packages, images or replacing the kubeadm binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(
		&flags.Image, "image",
		node.DefaultImage,
		"name:tag of the resulting image to be built",
	)
	cmd.Flags().StringVar(
		&flags.BaseImage, "base-image",
		node.DefaultImage,
		"name:tag of the base image to use for the build",
	)
	cmd.Flags().StringSliceVar(
		&flags.ImageTars, "with-images",
		nil,
		"images tar or folder with images tars to be added to the images",
	)
	cmd.Flags().StringVar(
		&flags.UpgradeBinaries, "with-upgrade-binaries",
		"",
		"path to a folder with kubernetes binaries [kubelet, kubeadm, kubectl] to be used for testing the kubeadm-upgrade workflow",
	)
	cmd.Flags().StringVar(
		&flags.Kubeadm, "with-kubeadm",
		"",
		"override the kubeadm binary existing in the image with the given file",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	ctx, err := alter.NewContext(
		// base build options
		alter.WithImage(flags.Image),
		alter.WithBaseImage(flags.BaseImage),
		// bits to be added to the image
		alter.WithImageTars(flags.ImageTars),
		alter.WithUpgradeBinaries(flags.UpgradeBinaries),
		alter.WithKubeadm(flags.Kubeadm),
	)
	if err != nil {
		return errors.Wrap(err, "error creating alter context")
	}
	if err := ctx.Alter(); err != nil {
		return errors.Wrap(err, "error altering node image")
	}
	return nil
}
