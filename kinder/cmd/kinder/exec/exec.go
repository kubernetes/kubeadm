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

// Package exec implements the `exec` command
package exec

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for exec
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.MinimumNArgs(2),
		Use: "exec [flags] NODE_NAME|NODE_SELECTOR -- COMMAND [ARG...]>\n\n" +
			"Args:\n" +
			"  NODE_NAME is the container name without the cluster name prefix\n" +
			"  NODE_SELECTOR can be one of:\n" +
			"    @all 	all the control-plane and worker nodes \n" +
			"    @cp* 	all the control-plane nodes \n" +
			"    @cp1 	the bootstrap-control plane node \n" +
			"    @cpN 	the secondary master nodes \n" +
			"    @w* 	all the worker nodes\n" +
			"    @lb 	the external load balancer\n" +
			"    @etcd 	the external etcd",
		Short: "Executes command on one or more nodes in the local Kubernetes cluster",
		Long:  "Exec is a \"topology aware\" wrapper on docker exec, allowing to run command on one or more nodes in the local Kubernetes cluster\n",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
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

	err = kcfg.Exec(args[0], args[1:])
	if err != nil {
		return errors.Wrap(err, "failed to exec command on cluster nodes")
	}

	return nil
}
