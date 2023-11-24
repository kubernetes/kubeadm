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

package build

import (
	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/cmd/kinder/build/baseimage"
	"k8s.io/kubeadm/kinder/cmd/kinder/build/nodevariant"
)

// NewCommand returns a new cobra.Command for building
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "build",
		Short: "Build one of [base-image, node-image-variant]",
		Long: "The 'build' command is used for creating images necessary for a Kubernetes cluster created with kinder.\n" +
			"It has two primary subcommands:\n" +
			"1. 'base-image': This command is used to build the base image for nodes in the cluster.\n" +
			"The base image includes all the necessary dependencies and configurations required for Kubernetes nodes.\n" +
			"2. 'node-image-variant': This command is used to build different variants of the node images based\n" +
			"on the base image. These variants may include different Kubernetes versions, CNI plugins,\n" +
			"or any other variations as needed for testing different Kubernetes features and behaviors.\n",
	}
	// add subcommands
	cmd.AddCommand(baseimage.NewCommand())
	cmd.AddCommand(nodevariant.NewCommand())
	return cmd
}
