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
	"sort"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/util/version"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"
)

// KNode implements a test friendly wrapper on nodes.Node
type KNode struct {
	nodes.Node

	// local properties use to avoid access to the real nodes.Node during tests
	// TODO: move local properties in test file
	name string
	role string
}

// NewKNode returns a new nodes.Node wrapper
func NewKNode(node nodes.Node) (n *KNode, err error) {
	_, err = node.Role()
	if err != nil {
		return nil, err
	}

	return &KNode{Node: node}, nil
}

// Name returns the name of the node
func (n *KNode) Name() string {
	// if local name is set, use to avoid access to the real nodes.Node during tests
	if n.name != "" {
		return n.name
	}
	return n.Node.String()
}

// Role returns the role of the node
func (n *KNode) Role() string {
	// if local role is set, use to avoid access to the real nodes.Node during tests
	if n.role != "" {
		return n.role
	}
	role, _ := n.Node.Role()
	return role
}

// IsControlPlane returns true if the node hosts a control plane instance
// NB. in single node clusters, control-plane nodes act also as a worker nodes
func (n *KNode) IsControlPlane() bool {
	return n.Role() == constants.ControlPlaneNodeRoleValue
}

// IsWorker returns true if the node hosts a worker instance
func (n *KNode) IsWorker() bool {
	return n.Role() == constants.WorkerNodeRoleValue
}

// IsExternalEtcd returns true if the node hosts an external etcd member
func (n *KNode) IsExternalEtcd() bool {
	return n.Role() == constants.ExternalEtcdNodeRoleValue
}

// IsExternalLoadBalancer returns true if the node hosts an external load balancer
func (n *KNode) IsExternalLoadBalancer() bool {
	return n.Role() == constants.ExternalLoadBalancerNodeRoleValue
}

// ProvisioningOrder returns the provisioning order for nodes, that
// should be defined according to the assigned NodeRole
func (n *KNode) ProvisioningOrder() int {
	switch n.Role() {
	// External dependencies should be provisioned first; we are defining an arbitrary
	// precedence between etcd and load balancer in order to get predictable/repeatable results
	case constants.ExternalEtcdNodeRoleValue:
		return 1
	case constants.ExternalLoadBalancerNodeRoleValue:
		return 2
	// Then control plane nodes
	case constants.ControlPlaneNodeRoleValue:
		return 3
	// Finally workers
	case constants.WorkerNodeRoleValue:
		return 4
	default:
		return 99
	}
}

// DebugCmd executes a command on a node and prints the command output on the screen
func (n *KNode) DebugCmd(message string, command string, args ...string) error {
	fmt.Println(message)
	fmt.Println()
	fmt.Printf("%s %s\n\n", command, strings.Join(args, " "))
	cmd := n.Command(command, args...)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error executing %s", message)
	}
	fmt.Println()

	return nil
}

// CombinedOutputLines returns the combined output lines from stdout and stderr
func (n *KNode) CombinedOutputLines(command string, args ...string) (lines []string, err error) {
	cmd := n.Command(command, args...)
	return exec.CombinedOutputLines(cmd)
}

// KubeadmVersion returns the kubeadm version installed on the node
func (n *KNode) KubeadmVersion() (*version.Version, error) {
	// NB. we are not caching version, because it can change e.g. after upgrades
	cmd := n.Command("kubeadm", "version", "-o=short")
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubeadm version")
	}
	if len(lines) != 1 {
		return nil, errors.Errorf("kubeadm version should only be one line, got %d lines", len(lines))
	}
	kubeadmVersion, err := version.ParseSemantic(lines[0])
	if err != nil {
		return nil, errors.Wrapf(err, "%q is not a valid kubeadm version", lines[0])
	}

	return kubeadmVersion, nil
}

// KNodes defines a list of nodes.Node wrapper
type KNodes []*KNode

// Sort the list of nodes.Node wrapper by node provisioning order and by name
func (l KNodes) Sort() {
	sort.Slice(l, func(i, j int) bool {
		return l[i].ProvisioningOrder() < l[j].ProvisioningOrder() ||
			(l[i].ProvisioningOrder() == l[j].ProvisioningOrder() && l[i].Name() < l[j].Name())
	})
}
