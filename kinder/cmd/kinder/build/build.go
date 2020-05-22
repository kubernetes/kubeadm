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
		Args: cobra.NoArgs,
		// TODO(bentheelder): more detailed usage
		Use:   "build",
		Short: "Build one of [base-image, node-image-variant]",
		Long:  "Build the base node image (base-image) or node image variants (node-image-variant)",
	}
	// add subcommands
	cmd.AddCommand(baseimage.NewCommand())
	cmd.AddCommand(nodevariant.NewCommand())
	return cmd
}
