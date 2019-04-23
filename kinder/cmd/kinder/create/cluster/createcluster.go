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
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/config"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha2"
	"sigs.k8s.io/kind/pkg/cluster/create"
	"sigs.k8s.io/kind/pkg/util"
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
	Workers              int32
	ControlPlanes        int32
	KubeDNS              bool
	Retain               bool
	Wait                 time.Duration
	SetupKubernetes      bool
	ExternalLoadBalancer bool
	ExternalEtcd         bool
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
	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.Config, configFlagName, "", "path to a kind config file")
	cmd.Flags().Int32Var(&flags.ControlPlanes, controlPlaneNodesFlagName, 1, "number of control-plane nodes in the cluster")
	cmd.Flags().Int32Var(&flags.Workers, workerNodesFLagName, 0, "number of worker nodes in the cluster")
	cmd.Flags().StringVar(&flags.ImageName, "image", "", "node docker image to use for booting the cluster")
	cmd.Flags().BoolVar(&flags.Retain, "retain", false, "retain nodes for debugging when cluster creation fails")
	cmd.Flags().DurationVar(&flags.Wait, "wait", time.Duration(0), "Wait for control plane node to be ready (default 0s)")
	cmd.Flags().BoolVar(&flags.SetupKubernetes, "setup-kubernetes", false, "setup Kubernetes on cluster nodes")
	cmd.Flags().BoolVar(&flags.KubeDNS, kubeDNSFLagName, false, "setup kubeadm for installing kube-dns instead of CoreDNS")
	cmd.Flags().BoolVar(&flags.ExternalEtcd, externalEtcdFlagName, false, "create an external etcd and setup kubeadm for using it")
	cmd.Flags().BoolVar(&flags.ExternalLoadBalancer, externalLoadBalancerFlagName, false, "add an external load balancer to the cluster (implicit if number of control-plane nodes>1)")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// refactor this...
	if cmd.Flags().Changed(configFlagName) && (cmd.Flags().Changed(controlPlaneNodesFlagName) ||
		cmd.Flags().Changed(workerNodesFLagName) ||
		cmd.Flags().Changed(kubeDNSFLagName) ||
		cmd.Flags().Changed(externalEtcdFlagName) ||
		cmd.Flags().Changed(externalLoadBalancerFlagName)) {
		return errors.Errorf("flag --%s can't be used in combination with --%s flags", configFlagName, strings.Join([]string{controlPlaneNodesFlagName, workerNodesFLagName, kubeDNSFLagName, externalEtcdFlagName, externalLoadBalancerFlagName}, ","))
	}

	if flags.ControlPlanes < 0 || flags.Workers < 0 {
		return errors.Errorf("flags --%s and --%s should not be a negative number", controlPlaneNodesFlagName, workerNodesFLagName)
	}

	// Check if the cluster name already exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if known {
		return errors.Errorf("a cluster with the name %q already exists", flags.Name)
	}

	//TODO: this should go away as soon as kind will support etcd nodes
	var externalEtcdIP string
	if flags.ExternalEtcd {
		fmt.Printf("Creating external etcd for the cluster %q ...\n", flags.Name)

		var err error
		externalEtcdIP, err = kcluster.CreateExternalEtcd(flags.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster")
		}
	}

	cfg := NewConfig(flags.ControlPlanes, flags.Workers, flags.KubeDNS, externalEtcdIP, flags.ExternalLoadBalancer)
	// override the config with the one from file, if specified
	if flags.Config != "" {
		// load the config
		cfg, err := encoding.Load(flags.Config)
		if err != nil {
			return errors.Wrap(err, "error loading config")
		}

		// validate the config
		err = cfg.Validate()
		if err != nil {
			log.Error("Invalid configuration!")
			configErrors := err.(*util.Errors)
			for _, problem := range configErrors.Errors() {
				log.Error(problem)
			}
			return errors.New("aborting due to invalid configuration")
		}
	}

	// create a cluster context and create the cluster
	ctx := cluster.NewContext(flags.Name)
	if flags.ImageName != "" {
		// Apply image override to all the Nodes defined in Config
		// TODO(Fabrizio Pandini): this should be reconsidered when implementing
		//     https://github.com/kubernetes-sigs/kind/issues/133
		for i := range cfg.Nodes {
			cfg.Nodes[i].Image = flags.ImageName
		}

		err := cfg.Validate()
		if err != nil {
			log.Errorf("Invalid flags, configuration failed validation: %v", err)
			return errors.New("aborting due to invalid configuration")
		}
	}

	fmt.Printf("Creating cluster %q ...\n", flags.Name)
	if err = ctx.Create(cfg,
		create.Retain(flags.Retain),
		create.WaitForReady(flags.Wait),
		create.SetupKubernetes(flags.SetupKubernetes),
	); err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	fmt.Printf("\nYou can also use kinder commands:\n\n")
	fmt.Printf("- kinder do, the kinder swiss knife 🚀!\n")
	fmt.Printf("- kinder exec, a \"topology aware\" wrapper on docker exec\n")
	fmt.Printf("- kinder cp, a \"topology aware\" wrapper on docker cp\n")

	return nil
}

// NewConfig returns the default config according to requested number of control-plane
// and worker nodes
func NewConfig(controlPlanes, workers int32, kubeDNS bool, externalEtcdIP string, externalLoadBalancer bool) *config.Cluster {
	var latestPublicConfig = &v1alpha2.Config{}

	// create default config according to requested number of control-plane and worker nodes
	// adds the control-plane node(s)
	controlPlaneNodes := v1alpha2.Node{Role: v1alpha2.ControlPlaneRole, Replicas: &controlPlanes}

	controlPlaneNodes.KubeadmConfigPatches = []string{}
	if kubeDNS {
		controlPlaneNodes.KubeadmConfigPatches = append(controlPlaneNodes.KubeadmConfigPatches, kubeDNSPatch)
	}
	if externalEtcdIP != "" {
		controlPlaneNodes.KubeadmConfigPatches = append(controlPlaneNodes.KubeadmConfigPatches, fmt.Sprintf(externalEtcdPatch, externalEtcdIP))
	}
	// enable Calico
	controlPlaneNodes.KubeadmConfigPatches = append(controlPlaneNodes.KubeadmConfigPatches, calicoPatch)

	latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, controlPlaneNodes)
	// if requester or more than one control-plane node(s), add an external load balancer
	if externalLoadBalancer || controlPlanes > 1 {
		latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, v1alpha2.Node{Role: v1alpha2.ExternalLoadBalancerRole})
	}

	// adds the worker node(s), if any
	if workers > 0 {
		latestPublicConfig.Nodes = append(latestPublicConfig.Nodes, v1alpha2.Node{Role: v1alpha2.WorkerRole, Replicas: &workers})
	}

	// apply defaults
	encoding.Scheme.Default(latestPublicConfig)

	// converts to internal config
	var cfg = &config.Cluster{}
	encoding.Scheme.Convert(latestPublicConfig, cfg, nil)

	// unmarshal the file content into a `kind` Config
	return cfg
}

const kubeDNSPatch = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
dns:
  type: "kube-dns"`

const externalEtcdPatch = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
etcd:
  external:
    endpoints:
    - http://%s:2379`

const calicoPatch = `apiVersion: kubeadm.k8s.io/v1beta1
kind: ClusterConfiguration
metadata:
  name: config
networking:
  podSubnet: "192.168.0.0/16"`
