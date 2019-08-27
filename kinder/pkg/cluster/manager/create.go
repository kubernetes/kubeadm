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

package manager

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/config"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	kindconcurrent "sigs.k8s.io/kind/pkg/concurrent"
	kindCRI "sigs.k8s.io/kind/pkg/container/cri"
	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
	kindlog "sigs.k8s.io/kind/pkg/log"
)

// CreateOptions holds all the options used at create time
type CreateOptions struct {
	externalLoadBalancer bool
	externalEtcd         bool
	retain               bool
}

// CreateOption is a configuration option supplied to Create
type CreateOption func(*CreateOptions)

// ExternalEtcd instruct create to add an external etcd to the cluster
func ExternalEtcd(externalEtcd bool) CreateOption {
	return func(c *CreateOptions) {
		c.externalEtcd = externalEtcd
	}
}

// ExternalLoadBalancer instruct create to add an external loadbalancer to the cluster.
// NB. this happens automatically when there are more than two control plane instances, but with this flag
// it is possible to override the default behaviour
func ExternalLoadBalancer(externalLoadBalancer bool) CreateOption {
	return func(c *CreateOptions) {
		c.externalLoadBalancer = externalLoadBalancer
	}
}

// Retain option instructs create cluster to preserve node in case of errors for debuggin pouposes
func Retain(retain bool) CreateOption {
	return func(c *CreateOptions) {
		c.retain = retain
	}
}

// CreateCluster creates a new kinder cluster
func CreateCluster(clusterName string, cfg *config.Cluster, options ...CreateOption) error {
	flags := &CreateOptions{}
	for _, o := range options {
		o(flags)
	}

	// Check if the cluster name already exists
	known, err := kindcluster.IsKnown(clusterName)
	if err != nil {
		return err
	}
	if known {
		return errors.Errorf("a cluster with the name %q already exists", clusterName)
	}

	status := kindlog.NewStatus(os.Stdout)
	status.MaybeWrapLogrus(log.StandardLogger())

	fmt.Printf("Creating cluster %q ...\n", clusterName)

	// attempt to explicitly pull the required node images if they doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	ensureNodeImages(status, cfg)

	// define the cluster label that identifies all the nodes in the cluster
	// NB. this should be consistent with kind
	clusterLabel := fmt.Sprintf("%s=%s", constants.ClusterLabelKey, clusterName)

	handleErr := func(err error) error {
		// In case of errors nodes are deleted (except if retain is explicitly set)
		if !flags.retain {
			ctx := kindcluster.NewContext(clusterName)
			ctx.Delete()
		}
		log.Error(err)

		return err
	}

	// Create node containers as defined in the kind config
	if err := createNodes(
		status,
		clusterName,
		clusterLabel,
		cfg,
		flags,
	); err != nil {
		return handleErr(err)
	}

	fmt.Println()
	fmt.Printf("Nodes creation complete. You can now continue creating a Kubernetes cluster using\n")
	fmt.Printf("kinder do, the kinder swiss knife 🚀!\n")

	return nil
}

func createNodes(spinner *kindlog.Status, clusterName string, clusterLabel string, cfg *config.Cluster, flags *CreateOptions) error {
	defer spinner.End(false)

	// compute the desired nodes, and inform the user that we are setting them up
	desiredNodes := nodesToCreate(clusterName, cfg, flags.externalLoadBalancer)
	numberOfNodes := len(desiredNodes)
	if flags.externalEtcd {
		numberOfNodes++
	}
	spinner.Start("Preparing nodes " + strings.Repeat("📦", numberOfNodes))

	// detect CRI runtime installed into images before actually creating nodes
	var createHelperMap = map[string]*cri.CreateHelper{}
	for _, n := range desiredNodes {
		if n.Role != constants.ExternalLoadBalancerNodeRoleValue {
			if _, ok := createHelperMap[n.Image]; ok {
				continue
			}

			runtime, err := status.InspectCRIinImage(n.Image)
			if err != nil {
				log.Errorf("Error detecting CRI for images %s! %v", n.Image, err)
				return err
			}
			log.Infof("Detected %s container runtime for image %s", runtime, n.Image)

			createHelper, err := cri.NewCreateHelper(runtime)
			if err != nil {
				log.Errorf("Error creating NewCreateHelper for CRI %s! %v", n.Image, err)
				return err
			}
			createHelperMap[n.Image] = createHelper
		}
	}

	// create all of the node containers, concurrently
	fns := []func() error{}
	for _, desiredNode := range desiredNodes {
		desiredNode := desiredNode // capture loop variable
		fns = append(fns, func() error {
			var createHelper *cri.CreateHelper
			if desiredNode.Role != constants.ExternalLoadBalancerNodeRoleValue {
				if _, ok := createHelperMap[desiredNode.Image]; !ok {
					return errors.Errorf("Unable to find create helper for image %s", desiredNode.Image)
				}
				createHelper = createHelperMap[desiredNode.Image]
			}

			// create the node into a container (~= docker run -d)
			return desiredNode.create(createHelper, clusterLabel)
		})
	}

	log.Info("Creating nodes...")
	if err := kindconcurrent.UntilError(fns); err != nil {
		return err
	}

	// add an external etcd if explicitly requested
	if flags.externalEtcd {
		log.Info("Getting required etcd image...")
		c, err := status.GetNodesFromDocker(clusterName)
		if err != nil {
			return err
		}

		etcdImage, err := c.BootstrapControlPlane().EtcdImage()
		if err != nil {
			return err
		}

		// attempt to explicitly pull the etcdImage if it doesn't exist locally
		// we don't care if this errors, we'll still try to run which also pulls
		_, _ = kinddocker.PullIfNotPresent(etcdImage, 4)

		log.Info("Creating external etcd...")
		CreateExternalEtcd(clusterName, etcdImage)
	}

	// get the cluster
	c, err := status.GetNodesFromDocker(clusterName)
	if err != nil {
		return err
	}

	// writes to the nodes the cluster settings that will be re-used by kinder during the cluster lifecyle.
	c.Settings = &status.ClusterSettings{
		IPFamily:                     status.ClusterIPFamily(cfg.Networking.IPFamily),
		APIServerPort:                cfg.Networking.APIServerPort,
		APIServerAddress:             cfg.Networking.APIServerAddress,
		PodSubnet:                    cfg.Networking.PodSubnet,
		ServiceSubnet:                cfg.Networking.ServiceSubnet,
		DisableDefaultCNI:            cfg.Networking.DisableDefaultCNI,
		KubeadmConfigPatches:         cfg.KubeadmConfigPatches,
		KubeadmConfigPatchesJSON6902: cfg.KubeadmConfigPatchesJSON6902,
	}
	if err := c.WriteSettings(); err != nil {
		return err
	}

	// writes to the nodes the node settings
	for _, n := range c.K8sNodes() {
		if err := n.WriteNodeSettings(&status.NodeSettings{}); err != nil {
			return err
		}
	}

	spinner.End(true)
	return nil
}

// nodeSpec describes a node to create purely from the container aspect
// this does not inlude eg starting kubernetes (see actions for that)
type nodeSpec struct {
	Name              string
	Role              string
	Image             string
	ExtraMounts       []kindCRI.Mount
	ExtraPortMappings []kindCRI.PortMapping
	APIServerPort     int32
	APIServerAddress  string
}

// nodesToCreate return the list of nodes to create for the cluster
func nodesToCreate(clusterName string, cfg *config.Cluster, externalLoadBalancer bool) []nodeSpec {
	desiredNodes := []nodeSpec{}

	// nodes are named based on the cluster name and their role, with a counter
	counter := make(map[string]int)
	nameMaker := func(role string) string {
		count := 1
		suffix := ""
		if v, ok := counter[role]; ok {
			count += v
			suffix = fmt.Sprintf("%d", count)
		}
		counter[role] = count
		return fmt.Sprintf("%s-%s%s", clusterName, role, suffix)
	}

	// prepare nodes explicitly defined in config
	for _, configNode := range cfg.Nodes {
		role := string(configNode.Role)

		desiredNode := nodeSpec{
			Name:              nameMaker(role),
			Image:             configNode.Image,
			Role:              role,
			ExtraMounts:       configNode.ExtraMounts,
			ExtraPortMappings: configNode.ExtraPortMappings,
		}

		// in case of control-plane nodes, inheriths network settings to be applied to the API servers
		if role == constants.ControlPlaneNodeRoleValue {
			desiredNode.APIServerPort = cfg.Networking.APIServerPort
			desiredNode.APIServerAddress = cfg.Networking.APIServerAddress
		}

		desiredNodes = append(desiredNodes, desiredNode)
	}

	// add an external load balancer if explicitly requested or if there are multiple control planes
	if externalLoadBalancer || counter[constants.ControlPlaneNodeRoleValue] > 1 {
		role := constants.ExternalLoadBalancerNodeRoleValue
		desiredNodes = append(desiredNodes, nodeSpec{
			Name:             nameMaker(role),
			Image:            constants.LoadBalancerImage,
			Role:             role,
			ExtraMounts:      []kindCRI.Mount{}, // There is no way to configure mounts for external load balancer
			APIServerAddress: cfg.Networking.APIServerAddress,
			APIServerPort:    cfg.Networking.APIServerPort,
		})

		// makes control-plane nodes internal
		for _, d := range desiredNodes {
			if d.Role == constants.ControlPlaneNodeRoleValue {
				d.APIServerPort = 0              // replaced with a random port
				d.APIServerAddress = "127.0.0.1" // only the LB needs to be non-local
			}
		}
	}

	return desiredNodes
}

func (d *nodeSpec) create(createHelper *cri.CreateHelper, clusterLabel string) error {
	switch d.Role {
	case constants.ExternalLoadBalancerNodeRoleValue:
		return CreateExternalLoadBalancerNode(d.Name, d.Image, clusterLabel, d.APIServerAddress, d.APIServerPort)
	case constants.ControlPlaneNodeRoleValue:
		return createHelper.CreateControlPlaneNode(d.Name, d.Image, clusterLabel, d.APIServerAddress, d.APIServerPort, d.ExtraMounts, d.ExtraPortMappings)
	case constants.WorkerNodeRoleValue:
		return createHelper.CreateWorkerNode(d.Name, d.Image, clusterLabel, d.ExtraMounts, d.ExtraPortMappings)
	}
	return errors.Errorf("unknown node role: %s", d.Role)
}

// CreateExternalLoadBalancerNode creates a docker container hosting an external loadbalancer
func CreateExternalLoadBalancerNode(name, image, clusterLabel, listenAddress string, port int32) error {
	_, err := kindnodes.CreateExternalLoadBalancerNode(name, image, clusterLabel, listenAddress, port)
	return err
}

// CreateExternalEtcd creates a docker container with an external etcd node
func CreateExternalEtcd(name, etcdImage string) error {
	// define name and labels mocking a kind external etcd node
	containerName := fmt.Sprintf("%s-%s", name, constants.ExternalEtcdNodeRoleValue)

	runArgs := []string{
		"-d",                        // run the container detached
		"--hostname", containerName, // make hostname match container name
		"--name", containerName, // ... and set the container name
		// label the node with the cluster ID
		"--label", fmt.Sprintf("%s=%s", constants.ClusterLabelKey, name),
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleKey, constants.ExternalEtcdNodeRoleValue),
	}

	// define a minimal etcd (insecure, single node, not exposed to the host machine)
	containerArgs := []string{
		"etcd",
		"--name", fmt.Sprintf("%s-etcd", name),
		"--advertise-client-urls", "http://127.0.0.1:2379",
		"--listen-client-urls", "http://0.0.0.0:2379",
	}

	// create the etcd container
	return kinddocker.Run(
		etcdImage,
		kinddocker.WithRunArgs(runArgs...),
		kinddocker.WithContainerArgs(containerArgs...),
	)
}

// ensureNodeImages ensures that the node images used by the create
// configuration are present
func ensureNodeImages(status *kindlog.Status, cfg *config.Cluster) {
	// pull each required image
	var images = map[string]interface{}{}
	for _, n := range cfg.Nodes {
		image := n.Image

		if _, ok := images[image]; ok {
			continue
		}

		// prints user friendly message
		if strings.Contains(image, "@sha256:") {
			image = strings.Split(image, "@sha256:")[0]
		}
		status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", image))

		// attempt to explicitly pull the image if it doesn't exist locally
		// we don't care if this errors, we'll still try to run which also pulls
		_, _ = kinddocker.PullIfNotPresent(image, 4)

		images[image] = nil
	}
}
