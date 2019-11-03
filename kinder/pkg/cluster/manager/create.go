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
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	kindconcurrent "sigs.k8s.io/kind/pkg/concurrent"
	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
	kindlog "sigs.k8s.io/kind/pkg/log"
)

// CreateOptions holds all the options used at create time
type CreateOptions struct {
	controlPlanes        int
	workers              int
	image                string
	externalLoadBalancer bool
	externalEtcd         bool
	retain               bool
}

// CreateOption is a configuration option supplied to Create
type CreateOption func(*CreateOptions)

// ControlPlanes sets the number of control plane nodes for create
func ControlPlanes(controlPlanes int) CreateOption {
	return func(c *CreateOptions) {
		c.controlPlanes = controlPlanes
	}
}

// Workers sets the number of worker nodes for create
func Workers(workers int) CreateOption {
	return func(c *CreateOptions) {
		c.workers = workers
	}
}

// Image sets the image for create
func Image(image string) CreateOption {
	return func(c *CreateOptions) {
		c.image = image
	}
}

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
func CreateCluster(clusterName string, options ...CreateOption) error {
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

	// attempt to explicitly pull the required node image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	ensureNodeImage(status, flags.image)

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
		flags,
	); err != nil {
		return handleErr(err)
	}

	fmt.Println()
	fmt.Printf("Nodes creation complete. You can now continue creating a Kubernetes cluster using\n")
	fmt.Printf("kinder do, the kinder swiss knife 🚀!\n")

	return nil
}

func createNodes(spinner *kindlog.Status, clusterName string, clusterLabel string, flags *CreateOptions) error {
	defer spinner.End(false)

	// compute the desired nodes, and inform the user that we are setting them up
	desiredNodes := nodesToCreate(clusterName, flags)
	numberOfNodes := len(desiredNodes)
	if flags.externalEtcd {
		numberOfNodes++
	}
	spinner.Start("Preparing nodes " + strings.Repeat("📦", numberOfNodes))

	// detect CRI runtime installed into images before actually creating nodes
	runtime, err := status.InspectCRIinImage(flags.image)
	if err != nil {
		log.Errorf("Error detecting CRI for images %s! %v", flags.image, err)
		return err
	}
	log.Infof("Detected %s container runtime for image %s", runtime, flags.image)

	createHelper, err := cri.NewCreateHelper(runtime)
	if err != nil {
		log.Errorf("Error creating NewCreateHelper for CRI %s! %v", flags.image, err)
		return err
	}

	// create all of the node containers, concurrently
	fns := []func() error{}
	for _, desiredNode := range desiredNodes {
		desiredNode := desiredNode // capture loop variable
		fns = append(fns, func() error {
			switch desiredNode.Role {
			case constants.ExternalLoadBalancerNodeRoleValue:
				return CreateExternalLoadBalancerNode(desiredNode.Name, constants.LoadBalancerImage, clusterLabel)
			case constants.ControlPlaneNodeRoleValue:
				return createHelper.CreateControlPlaneNode(desiredNode.Name, flags.image, clusterLabel)
			case constants.WorkerNodeRoleValue:
				return createHelper.CreateWorkerNode(desiredNode.Name, flags.image, clusterLabel)
			default:
				return nil
			}
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

	// writes to the nodes the cluster settings that will be re-used by kinder during the cluster lifecycle.
	c.Settings = &status.ClusterSettings{
		IPFamily: status.IPv4Family, // support for ipv6 is still WIP
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
// this does not include eg starting kubernetes (see actions for that)
type nodeSpec struct {
	Name string
	Role string
}

// nodesToCreate return the list of nodes to create for the cluster
func nodesToCreate(clusterName string, flags *CreateOptions) []nodeSpec {
	var desiredNodes []nodeSpec

	// prepare nodes explicitly
	for n := 0; n < flags.controlPlanes; n++ {
		role := constants.ControlPlaneNodeRoleValue
		desiredNode := nodeSpec{
			Name: fmt.Sprintf("%s-%s-%d", clusterName, role, n+1),
			Role: role,
		}
		desiredNodes = append(desiredNodes, desiredNode)
	}
	for n := 0; n < flags.workers; n++ {
		role := constants.WorkerNodeRoleValue
		desiredNode := nodeSpec{
			Name: fmt.Sprintf("%s-%s-%d", clusterName, role, n+1),
			Role: role,
		}
		desiredNodes = append(desiredNodes, desiredNode)
	}

	// add an external load balancer if explicitly requested or if there are multiple control planes
	if flags.externalLoadBalancer || flags.controlPlanes > 1 {
		role := constants.ExternalLoadBalancerNodeRoleValue
		desiredNodes = append(desiredNodes, nodeSpec{
			Name: fmt.Sprintf("%s-%s", clusterName, role),
			Role: role,
		})
	}

	return desiredNodes
}

// CreateExternalLoadBalancerNode creates a docker container hosting an external loadbalancer
func CreateExternalLoadBalancerNode(name, image, clusterLabel string) error {
	_, err := kindnodes.CreateExternalLoadBalancerNode(name, image, clusterLabel, "127.0.0.1", 0)
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

// ensureNodeImage ensures that the node image used by the create is present
func ensureNodeImage(status *kindlog.Status, image string) {
	status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", image))

	// attempt to explicitly pull the image if it doesn't exist locally
	// we don't care if this errors, we'll still try to run which also pulls
	_, _ = kinddocker.PullIfNotPresent(image, 4)
}
