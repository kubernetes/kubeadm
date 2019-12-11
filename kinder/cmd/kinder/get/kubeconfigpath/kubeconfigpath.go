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

package kubeconfigpath

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of nodes in a cluster
func NewCommand() *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kubeconfig-path",
		Short: "Prints the default kubeconfig path for the kind cluster by --name",
		Long:  "Prints the default kubeconfig path for the kind cluster by --name",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(status.KubeConfigPath(flags.Name))
			return nil
		},
	}

	cmd.Flags().StringVar(
		&flags.Name,
		"name", constants.DefaultClusterName, "cluster name",
	)
	return cmd
}
