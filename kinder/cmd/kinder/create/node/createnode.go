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

// Package cluster implements the `create cluster` command
// Nb. re-implemented in Kinder in order to add the --install-kubernetes flag
package cluster

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster"
)

type flagpole struct {
	Name      string
	ImageName string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(role string) *cobra.Command {
	roleNode := fmt.Sprintf("%s-node", role)

	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   roleNode,
		Short: fmt.Sprintf("Creates a %s for a Kubernetes cluster", roleNode),
		Long:  fmt.Sprintf("Creates a %s for a local Kubernetes cluster using Docker container 'nodes'", roleNode),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, role, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	return cmd
}

func runE(flags *flagpole, role string, cmd *cobra.Command, args []string) error {
	//TODO: fail il image name is empty

	// Check if the cluster name already exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if !known {
		return errors.Errorf("a cluster with the name %q does not exists", flags.Name)
	}

	// create a cluster context from current nodes
	ctx := cluster.NewContext(flags.Name)

	kcfg, err := kcluster.NewKContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster context")
	}

	fmt.Printf("Creating %s node in cluster %s ...\n", role, flags.Name)
	err = kcfg.CreateNode(role, flags.ImageName)
	if err != nil {
		return errors.Wrap(err, "failed to create node")
	}

	return nil
}
