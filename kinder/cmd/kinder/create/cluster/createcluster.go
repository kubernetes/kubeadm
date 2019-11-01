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
	configFlagName               = "config"
	controlPlaneNodesFlagName    = "control-plane-nodes"
	workerNodesFLagName          = "worker-nodes"
	kubeDNSFLagName              = "kube-dns"
	externalEtcdFlagName         = "external-etcd"
	externalLoadBalancerFlagName = "external-load-balancer"
)

type flagpole struct {
	Name                 string
	Config               string
	ImageName            string
	Workers              int
	ControlPlanes        int
	Retain               bool
	ExternalEtcd         bool
	ExternalLoadBalancer bool
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
		"cluster name")
	cmd.Flags().IntVar(
		&flags.ControlPlanes,
		controlPlaneNodesFlagName, 1,
		"number of control-plane nodes in the cluster",
	)
	cmd.Flags().IntVar(
		&flags.Workers,
		workerNodesFLagName, 0,
		"number of worker nodes in the cluster",
	)
	cmd.Flags().StringVar(
		&flags.ImageName,
		"image", "",
		"node docker image to use for booting the cluster",
	)
	cmd.Flags().BoolVar(
		&flags.Retain,
		"retain", false,
		"retain nodes for debugging when cluster creation fails",
	)
	cmd.Flags().BoolVar(
		&flags.ExternalEtcd,
		externalEtcdFlagName, false,
		"create an external etcd container and setup kubeadm for using it",
	)
	cmd.Flags().BoolVar(
		&flags.ExternalLoadBalancer,
		externalLoadBalancerFlagName, false,
		"add an external load balancer to the cluster (implicit if number of control-plane nodes>1)",
	)

	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	var err error

	if flags.ControlPlanes < 0 || flags.Workers < 0 {
		return errors.Errorf("flags --%s and --%s should not be a negative number", controlPlaneNodesFlagName, workerNodesFLagName)
	}

	// get a kinder cluster manager
	if err = manager.CreateCluster(
		flags.Name,
		manager.ControlPlanes(flags.ControlPlanes),
		manager.Workers(flags.Workers),
		manager.Image(flags.ImageName),
		manager.ExternalLoadBalancer(flags.ExternalLoadBalancer),
		manager.ExternalEtcd(flags.ExternalEtcd),
		manager.Retain(flags.Retain),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	return nil
}
