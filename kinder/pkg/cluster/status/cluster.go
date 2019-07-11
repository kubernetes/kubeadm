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

package status

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/constants"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kindkustomize "sigs.k8s.io/kind/pkg/kustomize"
)

// Cluster represents an existing kind(er) clusters
type Cluster struct {
	*kindcluster.Context
	Settings             *ClusterSettings
	allNodes             NodeList
	k8sNodes             NodeList
	controlPlanes        NodeList
	workers              NodeList
	externalEtcd         *Node
	externalLoadBalancer *Node
}

// ClusterSettings defines a set of settings that will be stored in the cluster and re-used
// by kinder during the cluster lifecyle.
//
// Storing value in the cluster is a specific necessity for kinder, because create nodes
// and actions for setting up a working cluster can happen at different time
// (while in kind everything happen within an atomic operation)
type ClusterSettings struct {
	// kind configuration settings that are used to configure the cluster when
	// generating the kubeadm config file.
	IPFamily                     ClusterIPFamily               `json:"ipFamily,omitempty"`
	APIServerPort                int32                         `json:"apiServerPort,omitempty"`
	APIServerAddress             string                        `json:"apiServerAddress,omitempty"`
	PodSubnet                    string                        `json:"podSubnet,omitempty"`
	ServiceSubnet                string                        `json:"serviceSubnet,omitempty"`
	KubeadmConfigPatches         []string                      `json:"kubeadmConfigPatches,omitempty"`
	KubeadmConfigPatchesJSON6902 []kindkustomize.PatchJSON6902 `json:"kubeadmConfigPatchesJson6902,omitempty"`

	// kind configuration settings that are used to disable installation of the default CNI pluging
	DisableDefaultCNI bool `json:"disableDefaultCNI,omitempty"`
}

// ClusterIPFamily defines cluster network IP family
type ClusterIPFamily string

const (
	// IPv4Family sets ClusterIPFamily to ipv4
	IPv4Family ClusterIPFamily = "ipv4"
	// IPv6Family sets ClusterIPFamily to ipv6
	IPv6Family ClusterIPFamily = "ipv6"
)

// GetNodesFromDocker returns a new cluster created by discovering
// and inspecting existing containers nodes
func GetNodesFromDocker(name string) (c *Cluster, err error) {
	// create a cluster context from current nodes
	ctx := kindcluster.NewContext(name)

	c = &Cluster{
		Context: ctx,
	}

	log.Debugf("Reading containers list for cluster %s", ctx.Name())
	nodes, err := ctx.ListNodes()
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		log.Debugf("Adding node %s to the cluster", n.Name())
		node, err := NewNode(n)
		if err != nil {
			return nil, err
		}

		if err = c.add(node); err != nil {
			return nil, err
		}
	}

	// ensures nodes are sorted consistently
	c.allNodes.Sort()
	c.k8sNodes.Sort()
	c.controlPlanes.Sort()
	c.workers.Sort()

	return c, nil
}

// Validate the cluster has a consistent set of nodes
func (c *Cluster) Validate() error {

	// There should be at least one control plane
	if c.BootstrapControlPlane() == nil {
		return errors.Errorf("please add at least one node with role %q", constants.ControlPlaneNodeRoleValue)
	}
	// There should be one load balancer if more than one control plane exists in the cluster
	if len(c.ControlPlanes()) > 1 && c.ExternalLoadBalancer() == nil {
		return errors.Errorf("please add a node with role %s because in the cluster there are more than one node with role %s",
			constants.ExternalLoadBalancerNodeRoleValue, constants.ControlPlaneNodeRoleValue)
	}

	return nil
}

// ReadSettings read cluster settings from a control-plane node
func (c *Cluster) ReadSettings() (err error) {
	log.Debug("Reading cluster settings...")
	c.Settings, err = c.BootstrapControlPlane().ReadClusterSettings()
	if err != nil {
		return errors.Wrapf(err, "failed to read cluster settings from node %s", c.BootstrapControlPlane().Name())
	}
	return nil
}

// WriteSettings writes cluster settings nodes
func (c *Cluster) WriteSettings() error {
	log.Debug("Writings cluster settings...")
	for _, n := range c.K8sNodes() {
		if err := n.WriteClusterSettings(c.Settings); err != nil {
			return errors.Wrapf(err, "failed to write cluster settings to node %s", n.Name())
		}
	}
	return nil
}

// add a Node to the Cluster, filling the derived list of Node by role
func (c *Cluster) add(node *Node) error {
	c.allNodes = append(c.allNodes, node)

	if node.IsControlPlane() || node.IsWorker() {
		c.k8sNodes = append(c.k8sNodes, node)
	}

	if node.IsControlPlane() {
		c.controlPlanes = append(c.controlPlanes, node)
	}

	if node.IsWorker() {
		c.workers = append(c.workers, node)
	}

	if node.IsExternalEtcd() {
		if c.externalEtcd != nil {
			return errors.Errorf("unable to add the node to the cluster. A cluster can not have more than one node with role %q", constants.ExternalEtcdNodeRoleValue)
		}
		c.externalEtcd = node
	}

	if node.IsExternalLoadBalancer() {
		if c.externalLoadBalancer != nil {
			return errors.Errorf("unable to add the node to the cluster. A cluster can not have more than one node role %q", constants.ExternalLoadBalancerNodeRoleValue)
		}
		c.externalLoadBalancer = node
	}

	return nil
}

// AllNodes returns all the nodes in the cluster (including K8s nodes, external loadbalancer and external etcd)
func (c *Cluster) AllNodes() NodeList {
	return c.allNodes
}

// K8sNodes returns all the nodes that hosts a Kubernetes nodes in the cluster (all nodes except external loadbalancer and external etcd)
func (c *Cluster) K8sNodes() NodeList {
	return c.k8sNodes
}

// ControlPlanes returns all the nodes with control-plane role
func (c *Cluster) ControlPlanes() NodeList {
	return c.controlPlanes
}

// BootstrapControlPlane returns the first node with control-plane role.
// This is the node where kubeadm init will be executed.
func (c *Cluster) BootstrapControlPlane() *Node {
	if len(c.controlPlanes) == 0 {
		return nil
	}
	return c.controlPlanes[0]
}

// SecondaryControlPlanes returns all the nodes with control-plane role
// except the BootstrapControlPlane node, if any,
func (c *Cluster) SecondaryControlPlanes() NodeList {
	if len(c.controlPlanes) <= 1 {
		return nil
	}
	return c.controlPlanes[1:]
}

// Workers returns all the nodes with Worker role, if any
func (c *Cluster) Workers() NodeList {
	return c.workers
}

// ExternalEtcd returns the node with external-etcd role, if defined
func (c *Cluster) ExternalEtcd() *Node {
	return c.externalEtcd
}

// ExternalLoadBalancer returns the node with external-load-balancer role, if defined
func (c *Cluster) ExternalLoadBalancer() *Node {
	return c.externalLoadBalancer
}

// ResolveNodesPath takes a "topology aware" path and resolve to one (or more) real paths.
//
// Topology aware paths are in the form [selector:]path, where a selector is a shortcut for
// a node or a set of nodes in the cluster. See SelectNodes
func (c *Cluster) ResolveNodesPath(nodesPath string) (nodes NodeList, path string, err error) {
	t := strings.Split(nodesPath, ":")
	switch len(t) {
	case 1:
		nodes = nil
		path = t[0]
	case 2:
		nodes, err = c.SelectNodes(t[0])
		if err != nil {
			return nil, "", err
		}
		path = t[1]
	default:
		return nil, "", errors.Errorf("invalid nodesPath %q", nodesPath)
	}

	return nodes, path, nil
}

// SelectNodes returns Nodes according to the given selector.
// a selector is a shortcut for a node or a set of nodes in the cluster.
func (c *Cluster) SelectNodes(nodeSelector string) (nodes NodeList, err error) {
	if strings.HasPrefix(nodeSelector, "@") {
		switch strings.ToLower(nodeSelector) {
		case "@all": // all the kubernetes nodes
			return c.K8sNodes(), nil
		case "@cp*": // all the control-plane nodes
			return c.ControlPlanes(), nil
		case "@cp1": // the bootstrap-control plane
			return toNodeList(c.BootstrapControlPlane()), nil
		case "@cpn":
			return c.SecondaryControlPlanes(), nil
		case "@w*":
			return c.Workers(), nil
		case "@lb":
			return toNodeList(c.ExternalLoadBalancer()), nil
		case "@etcd":
			return toNodeList(c.ExternalEtcd()), nil
		default:
			return nil, errors.Errorf("Invalid node selector %q. Use one of [@all, @cp*, @cp1, @cpn, @w*, @lb, @etcd]", nodeSelector)
		}
	}

	nodeName := fmt.Sprintf("%s-%s", c.Name(), nodeSelector)
	for _, n := range c.K8sNodes() {
		if strings.EqualFold(nodeName, n.Name()) {
			return toNodeList(n), nil
		}
	}

	return nil, nil
}

func toNodeList(node *Node) NodeList {
	if node != nil {
		return NodeList{node}
	}
	return nil
}
