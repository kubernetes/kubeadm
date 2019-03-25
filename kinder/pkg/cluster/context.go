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
	"fmt"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/exec"
)

// KContext is used to create / manipulate kubernetes-in-docker clusters
// See: NewContext()
type KContext struct {
	*cluster.Context
	kubernetesNodes      KNodes
	controlPlanes        KNodes
	workers              KNodes
	externalEtcd         *KNode
	externalLoadBalancer *KNode
}

// NewKContext returns a new cluster management context; The context
// is initialized by discovering the actual containers nodes
func NewKContext(ctx *cluster.Context) (c *KContext, err error) {
	c = &KContext{
		Context: ctx,
	}

	nodes, err := ctx.ListNodes()
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		node, err := NewKNode(n)
		if err != nil {
			return nil, err
		}

		if err = c.add(node); err != nil {
			return nil, err
		}
	}

	// There should be at least one control plane
	if c.BootStrapControlPlane() == nil {
		return nil, errors.Errorf("please add at least one node with role %q", constants.ControlPlaneNodeRoleValue)
	}
	// There should be one load balancer if more than one control plane exists in the cluster
	if len(c.ControlPlanes()) > 1 && c.ExternalLoadBalancer() == nil {
		return nil, errors.Errorf("please add a node with role %s because in the cluster there are more than one node with role %s",
			constants.ExternalLoadBalancerNodeRoleValue, constants.ControlPlaneNodeRoleValue)
	}

	return c, nil
}

// add a KNode to the KContext, filling the derived list of KNode by role
func (c *KContext) add(node *KNode) error {
	if node.IsControlPlane() || node.IsWorker() {
		c.kubernetesNodes = append(c.kubernetesNodes, node)
		c.kubernetesNodes.Sort()
	}

	if node.IsControlPlane() {
		c.controlPlanes = append(c.controlPlanes, node)
		c.controlPlanes.Sort()
	}

	if node.IsWorker() {
		c.workers = append(c.workers, node)
		c.workers.Sort()
	}

	if node.IsExternalEtcd() {
		if c.externalEtcd != nil {
			return errors.Errorf("invalid config. there are two nodes with role %q", constants.ExternalEtcdNodeRoleValue)
		}
		c.externalEtcd = node
	}

	if node.IsExternalLoadBalancer() {
		if c.externalLoadBalancer != nil {
			return errors.Errorf("invalid config. there are two nodes with role %q", constants.ExternalLoadBalancerNodeRoleValue)
		}
		c.externalLoadBalancer = node
	}

	return nil
}

// CreateNode create a new node of the given role
func (c *KContext) CreateNode(role string, image string) error {
	clusterLabel := fmt.Sprintf("%s=%s", constants.ClusterLabelKey, c.Name())
	switch role {
	case constants.WorkerNodeRoleValue:
		n := len(c.workers) + 1
		name := fmt.Sprintf("%s-%s%d", c.Name(), role, n)
		_, err := nodes.CreateWorkerNode(name, image, clusterLabel, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create worker node")
		}
		return nil
	case constants.ControlPlaneNodeRoleValue:
		//this is currently super hacky, looking for a better solution
		if c.externalLoadBalancer == nil {
			return errors.Errorf("Unable to create a new control plane node in a cluster without a load balancer")
		}
		n := len(c.controlPlanes) + 1
		name := fmt.Sprintf("%s-%s%d", c.Name(), role, n)
		node, err := nodes.CreateControlPlaneNode(name, image, clusterLabel, "127.0.0.1", 6443, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create control-plane node")
		}

		ip, err := node.IP()
		if err != nil {
			return errors.Wrap(err, "failed to get new control-plane node ip")
		}

		if err := c.ExternalLoadBalancer().Command("bin/bash", "-c",
			fmt.Sprintf("`echo \"    server %s %s:6443 check\" >> /kind/haproxy.cfg`", name, ip),
		).Run(); err != nil {
			return errors.Wrap(err, "failed to update load balancer config")
		}

		if err := c.ExternalLoadBalancer().Command("docker", "kill", "-s", "HUP", "haproxy").Run(); err != nil { //this assumes ha proxy having a well know name
			return errors.Wrap(err, "failed to reload load balancer config")
		}

		return nil
	}

	return errors.Errorf("creation of new %s nodes not supported", role)
}

// KubernetesNodes returns all the Kubernetes nodes in the cluster
func (c *KContext) KubernetesNodes() KNodes {
	return c.kubernetesNodes
}

// ControlPlanes returns all the nodes with control-plane role
func (c *KContext) ControlPlanes() KNodes {
	return c.controlPlanes
}

// BootStrapControlPlane returns the first node with control-plane role
// This is the node where kubeadm init will be executed.
func (c *KContext) BootStrapControlPlane() *KNode {
	if len(c.controlPlanes) == 0 {
		return nil
	}
	return c.controlPlanes[0]
}

// SecondaryControlPlanes returns all the nodes with control-plane role
// except the BootStrapControlPlane node, if any,
func (c *KContext) SecondaryControlPlanes() KNodes {
	if len(c.controlPlanes) <= 1 {
		return nil
	}
	return c.controlPlanes[1:]
}

// Workers returns all the nodes with Worker role, if any
func (c *KContext) Workers() KNodes {
	return c.workers
}

// ExternalEtcd returns the node with external-etcd role, if defined
func (c *KContext) ExternalEtcd() *KNode {
	return c.externalEtcd
}

// ExternalLoadBalancer returns the node with external-load-balancer role, if defined
func (c *KContext) ExternalLoadBalancer() *KNode {
	return c.externalLoadBalancer
}

//TODO: Refactor how we are exposing this flags
type ActionFlags struct {
	UsePhases      bool
	UpgradeVersion *version.Version
	CopyCerts      bool
}

// Do actions on kubernetes-in-docker cluster
// Actions are repetitive, high level abstractions/workflows composed
// by one or more lower level tasks, that automatically adapt to the
// current cluster topology
func (c *KContext) Do(actions []string, flags ActionFlags, onlyNode string) error {
	// Create an ExecutionPlan that applies the given actions to the
	// topology defined in the config
	executionPlan, err := newExecutionPlan(c, actions)
	if err != nil {
		return err
	}

	// Executes all the selected action
	for _, plannedTask := range executionPlan {
		if onlyNode != "" {
			onlyNodeName := fmt.Sprintf("%s-%s", c.Name(), onlyNode)
			if !strings.EqualFold(onlyNodeName, plannedTask.Node.Name()) {
				continue
			}
		}

		fmt.Printf("[%s] %s\n\n", plannedTask.Node.Name(), plannedTask.Task.Description)
		err := plannedTask.Task.Run(c, plannedTask.Node, flags)
		if err != nil {
			// in case of error, the execution plan is halted
			log.Error(err)
			return err
		}
	}

	return nil
}

// Exec is a topology aware wrapper of docker exec
func (c *KContext) Exec(nodeSelector string, args []string) error {
	nodes, err := c.selectNodes(nodeSelector)
	if err != nil {
		return err
	}

	log.Infof("%d nodes selected as target for the command", len(nodes))
	for _, node := range nodes {
		fmt.Printf("🚀 Executing command on node %s 🚀\n", node.Name())

		cmdArgs := append([]string{"exec",
			node.Name(),
		}, args...)
		cmd := exec.Command("docker", cmdArgs...)
		exec.InheritOutput(cmd)
		err := cmd.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to execute command on node %s", node.Name())
		}
	}

	return nil
}

// Copy is a topology aware wrapper of docker cp
func (c *KContext) Copy(source, target string) error {
	sourceNodes, sourcePath, err := c.resolveNodesPath(source)
	if err != nil {
		return err
	}

	teargetNodes, targetPath, err := c.resolveNodesPath(target)
	if err != nil {
		return err
	}

	if sourceNodes == nil && teargetNodes == nil {
		return errors.Errorf("at least one between source and target must be a node/nodes in the cluster")
	}

	if sourceNodes != nil {
		switch len(sourceNodes) {
		case 1:
			break // one source node selected: continue
		case 0:
			return errors.Errorf("no source node matches given criteria")
		default:
			return errors.Errorf("source can't be more than one node")
		}
	}

	if teargetNodes != nil && len(teargetNodes) == 0 {
		return errors.Errorf("no target node matches given criteria")
	}

	if sourceNodes != nil && teargetNodes != nil {
		// create tmp folder
		// cp locally
		return errors.Errorf("copy between nodes not implemented yet!")
	}

	if teargetNodes == nil {
		fmt.Printf("Copying from %s ...\n", sourceNodes[0].Name())
		sourceNodes[0].CopyFrom(sourcePath, targetPath)
	}

	for _, n := range teargetNodes {
		fmt.Printf("Copying to %s ...\n", n.Name())
		n.CopyTo(sourcePath, targetPath)
	}
	return nil
}

// resolveNodesPath takes a "topology aware" path and resolve to one (or more) real paths
func (c *KContext) resolveNodesPath(nodesPath string) (nodes KNodes, path string, err error) {
	t := strings.Split(nodesPath, ":")
	switch len(t) {
	case 1:
		nodes = nil
		path = t[0]
	case 2:
		nodes, err = c.selectNodes(t[0])
		if err != nil {
			return nil, "", err
		}
		path = t[1]
	default:
		return nil, "", errors.Errorf("invalid nodesPath %q", nodesPath)
	}

	return nodes, path, nil
}

// selectNodes returns KNodes according to the given selector
func (c *KContext) selectNodes(nodeSelector string) (nodes KNodes, err error) {
	if strings.HasPrefix(nodeSelector, "@") {
		switch strings.ToLower(nodeSelector) {
		case "@all": // all the kubernetes nodes
			return c.KubernetesNodes(), nil
		case "@cp*": // all the control-plane nodes
			return c.ControlPlanes(), nil
		case "@cp1": // the bootstrap-control plane
			return toKNodes(c.BootStrapControlPlane()), nil
		case "@cpn":
			return c.SecondaryControlPlanes(), nil
		case "@w*":
			return c.Workers(), nil
		case "@lb":
			return toKNodes(c.ExternalLoadBalancer()), nil
		case "@etcd":
			return toKNodes(c.ExternalEtcd()), nil
		default:
			return nil, errors.Errorf("Invalid node selector %q. Use one of [@all, @cp*, @cp1, @cpn, @w*, @lb, @etcd]", nodeSelector)
		}
	}

	nodeName := fmt.Sprintf("%s-%s", c.Name(), nodeSelector)
	for _, n := range c.KubernetesNodes() {
		if strings.EqualFold(nodeName, n.Name()) {
			return toKNodes(n), nil
		}
	}

	return nil, nil
}

func toKNodes(node *KNode) KNodes {
	if node != nil {
		return KNodes{node}
	}
	return nil
}
