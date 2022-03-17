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

package cluster

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/pkg/cluster/manager"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

const (
	controlPlaneNodesFlagName = "control-plane-nodes"
	workerNodesFlagName       = "worker-nodes"
)

type flagpole struct {
	Name                 string
	ImageName            string
	ImageRepository      string
	Workers              int
	ControlPlanes        int
	Retain               bool
	ExternalEtcd         bool
	ExternalLoadBalancer bool
	Volumes              []string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a local Kubernetes cluster",
		Long:  "Creates a local Kubernetes cluster using Docker container 'nodes'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().StringVar(
		&flags.Name,
		"name", constants.DefaultClusterName,
		"cluster name",
	)
	cmd.Flags().IntVar(
		&flags.ControlPlanes,
		controlPlaneNodesFlagName, 1,
		"number of control-plane nodes in the cluster",
	)
	cmd.Flags().IntVar(
		&flags.Workers,
		workerNodesFlagName, 0,
		"number of worker nodes in the cluster",
	)
	cmd.Flags().StringVar(
		&flags.ImageName,
		"image", "",
		"node docker image to use for booting the cluster",
	)
	cmd.Flags().StringVar(
		&flags.ImageName,
		"image-repository", "registry.gcr.io",
		"set alternate repository for images",
	)
	cmd.Flags().BoolVar(
		&flags.Retain,
		"retain", false,
		"retain nodes for debugging when cluster creation fails",
	)
	cmd.Flags().BoolVar(
		&flags.ExternalEtcd,
		"external-etcd", false,
		"create an external etcd container and setup kubeadm for using it",
	)
	cmd.Flags().BoolVar(
		&flags.ExternalLoadBalancer,
		"external-load-balancer", false,
		"add an external load balancer to the cluster (implicit if number of control-plane nodes>1)",
	)
	cmd.Flags().StringSliceVar(
		&flags.Volumes,
		"volume", nil,
		"mount a volume on node containers",
	)

	cmd.MarkFlagRequired("image")

	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	var err error

	if flags.ControlPlanes < 0 || flags.Workers < 0 {
		return errors.Errorf("flags --%s and --%s should not be a negative number", controlPlaneNodesFlagName, workerNodesFlagName)
	}

	// get a kinder cluster manager
	if err = manager.CreateCluster(
		flags.Name,
		manager.ControlPlanes(flags.ControlPlanes),
		manager.Workers(flags.Workers),
		manager.Image(flags.ImageName),
		manager.ImageRepository(flags.ImageRepository),
		manager.ExternalLoadBalancer(flags.ExternalLoadBalancer),
		manager.ExternalEtcd(flags.ExternalEtcd),
		manager.Retain(flags.Retain),
		manager.Volumes(flags.Volumes),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
