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
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/cmd"
	"k8s.io/kubeadm/kinder/pkg/cluster/cmd/colors"
	"k8s.io/kubeadm/kinder/pkg/constants"
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	ksigsyaml "sigs.k8s.io/yaml"
)

// commandMutator define a function that can mutate commands on a node.
// It is used to inject behaviours that should apply to all the command
// executed on a node, like e.g. DryRun
type commandMutator = func(*cmd.ProxyCmd) *cmd.ProxyCmd

// Node defines a K8s node running in a kinde(er) docker container or a container hosting
// one external dependency of the cluster, like etcd or the load balancer.
type Node struct {
	kindNode        kindnodes.Node
	cri             ContainerRuntime
	kubeadmVersion  *K8sVersion.Version
	etcdImage       string
	skip            bool
	commandMutators []commandMutator
}

// NodeSettings defines a set of settings that will be stored in the node and re-used
// by kinder during the node lifecyle.
//
// Storing value in the node is a specific necessity for kinder, because create nodes
// and actions for setting up a working cluster can happen at different time
// (while in kind everything happen within an atomic operation).
type NodeSettings struct {
	// NB Currently there are no persistent node settings used by kind, but we are preserving this feature for future changes
}

// NewNode returns a new kindnodes.Node wrapper
func NewNode(kindNode kindnodes.Node) (n *Node, err error) {
	_, err = kindNode.Role()
	if err != nil {
		return nil, err
	}

	return &Node{kindNode: kindNode}, nil
}

// Name returns the name of the node
func (n *Node) Name() string {
	return n.kindNode.Name()
}

// Role returns the role of the node
func (n *Node) Role() string {
	role, _ := n.kindNode.Role()
	return role
}

// IsControlPlane returns true if the node hosts a control plane instance
// NB. in single node clusters, control-plane nodes act also as a worker nodes
func (n *Node) IsControlPlane() bool {
	return n.Role() == constants.ControlPlaneNodeRoleValue
}

// IsWorker returns true if the node hosts a worker instance
func (n *Node) IsWorker() bool {
	return n.Role() == constants.WorkerNodeRoleValue
}

// IsExternalEtcd returns true if the node hosts an external etcd member
func (n *Node) IsExternalEtcd() bool {
	return n.Role() == constants.ExternalEtcdNodeRoleValue
}

// IsExternalLoadBalancer returns true if the node hosts an external load balancer
func (n *Node) IsExternalLoadBalancer() bool {
	return n.Role() == constants.ExternalLoadBalancerNodeRoleValue
}

// ProvisioningOrder returns the provisioning order for nodes, that
// should be defined according to the assigned Role; is is used to get consistent
// and repeatable ordering in the list of nodes
func (n *Node) provisioningOrder() int {
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

// Command returns a ProxyCmd that allows to run commands on the node
func (n *Node) Command(command string, args ...string) *cmd.ProxyCmd {
	// creates new ProxyCmd to run a command on a kind(er) node
	cmd := cmd.NewProxyCmd(n.Name(), command, args...)

	// applies command mutators
	for _, m := range n.commandMutators {
		cmd = m(cmd)
	}

	return cmd
}

// SkipActions marks the node to be skipped during actions.
func (n *Node) SkipActions() {
	n.skip = true
}

// DryRun instruct the node to dry run all the commands that will be executed on this node.
// DryRun differs from SkipRun, because in case of DryRun kinder prints all the details for running
// the command manually.
func (n *Node) DryRun() {
	if n.commandMutators == nil {
		n.commandMutators = []commandMutator{}
	}

	n.commandMutators = append(n.commandMutators,
		func(c *cmd.ProxyCmd) *cmd.ProxyCmd {
			return c.DryRun()
		},
	)
}

// Infof print an information message in the same format of commands on the node;
// the message is print after the prompt containing the kind (er) node name.
func (n *Node) Infof(message string, args ...interface{}) {
	node := colors.Prompt(fmt.Sprintf("%s:$ ", n.Name()))
	command := colors.Info(fmt.Sprintf(message, args...))
	fmt.Printf("\n%s%s\n", node, command)
}

// MustKubeadmVersion returns the kubeadm version installed on the node or panics
// if a valid kubeadm version can't be identified.
func (n *Node) MustKubeadmVersion() *K8sVersion.Version {
	_, err := n.KubeadmVersion()
	if err != nil {
		panic(err.Error())
	}
	return n.kubeadmVersion
}

// KubeadmVersion returns the kubeadm version installed on the node
func (n *Node) KubeadmVersion() (*K8sVersion.Version, error) {
	if n.kubeadmVersion != nil {
		return n.kubeadmVersion, nil
	}

	lines, err := cmd.NewProxyCmd(n.Name(), "kubeadm", "version", "-o=short").Silent().RunAndCapture()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubeadm version")
	}
	if len(lines) != 1 {
		return nil, errors.Errorf("kubeadm version should only be one line, got %d lines", len(lines))
	}
	n.kubeadmVersion, err = K8sVersion.ParseSemantic(lines[0])
	if err != nil {
		return nil, errors.Wrapf(err, "%q is not a valid kubeadm version", lines[0])
	}

	return n.kubeadmVersion, nil
}

// EtcdImage returns the etcdImage that should be used with the kubernetes version
// installed on this node
func (n *Node) EtcdImage() (string, error) {
	if n.etcdImage != "" {
		return n.etcdImage, nil
	}

	kubeVersion, err := n.KubeVersion()
	if err != nil {
		return "", err
	}

	lines, err := cmd.NewProxyCmd(
		n.Name(),
		"/bin/sh", "-c",
		fmt.Sprintf("kubeadm config images list --kubernetes-version=%s 2> /dev/null | grep etcd", kubeVersion),
	).Silent().RunAndCapture()
	if err != nil {
		return "", errors.Wrap(err, "failed to get the etcd image")
	}
	if len(lines) != 1 {
		return "", errors.Errorf("etcd version should only be one line, got %d lines", len(lines))
	}
	n.etcdImage = lines[0]

	return n.etcdImage, nil
}

const clusterSettingsPath = "/kinder/cluster-settings.yaml"

// WriteClusterSettings stores in the node a set of cluster-wide settings that will be re-used
// by kinder during the cluster lifecycle (after create)
func (n *Node) WriteClusterSettings(settings *ClusterSettings) error {
	s, err := ksigsyaml.Marshal(*settings)
	if err != nil {
		return errors.Wrapf(err, "failed to encode %s", clusterSettingsPath)
	}

	err = n.Command(
		"mkdir", "-p", filepath.Dir(clusterSettingsPath),
	).Silent().Run()
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", clusterSettingsPath)
	}

	err = n.Command(
		"cp", "/dev/stdin", clusterSettingsPath,
	).Stdin(
		strings.NewReader(string(s)),
	).Silent().Run()
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", clusterSettingsPath)
	}

	return nil
}

// ReadClusterSettings reads from the node a set of cluster-wide settings that
// are going to be re-used by kinder during the cluster lifecyle (after create)
func (n *Node) ReadClusterSettings() (*ClusterSettings, error) {
	lines, err := n.Command(
		"cat", clusterSettingsPath,
	).Silent().RunAndCapture()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", clusterSettingsPath)
	}

	var settings ClusterSettings
	err = ksigsyaml.Unmarshal([]byte(strings.Join(lines, "\n")), &settings)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s", clusterSettingsPath)
	}

	return &settings, nil
}

const nodeSettingsPath = "/kinder/node-settings.yaml"

// WriteNodeSettings stores in the node specific settings that will be re-used
// by kinder during the cluster lifecyle (after create)
func (n *Node) WriteNodeSettings(settings *NodeSettings) error {
	s, err := ksigsyaml.Marshal(*settings)
	if err != nil {
		return errors.Wrapf(err, "failed to encode %s", nodeSettingsPath)
	}

	err = n.Command(
		"mkdir", "-p", filepath.Dir(nodeSettingsPath),
	).Silent().Run()
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", nodeSettingsPath)
	}

	err = n.Command(
		"cp", "/dev/stdin", nodeSettingsPath,
	).Stdin(
		strings.NewReader(string(s)),
	).Silent().Run()
	if err != nil {
		return errors.Wrapf(err, "failed to write %s", nodeSettingsPath)
	}

	return nil
}

// ReadNodeSettings reads from the node specific settings that
// are going to be re-used by kinder during the cluster lifecyle (after create)
func (n *Node) ReadNodeSettings() (*NodeSettings, error) {
	lines, err := n.Command(
		"cat", nodeSettingsPath,
	).Silent().RunAndCapture()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", nodeSettingsPath)
	}

	var settings NodeSettings
	err = ksigsyaml.Unmarshal([]byte(strings.Join(lines, "\n")), &settings)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %s", nodeSettingsPath)
	}

	return &settings, nil
}

// CRI returns the ContainerRuntime installed on the node and that
// should be used by kubeadm for creating the K8s cluster
func (n *Node) CRI() (cri ContainerRuntime, err error) {
	if n.cri != "" {
		return n.cri, nil
	}

	n.cri, err = InspectCRIinContainer(n.Name())
	if err != nil {
		return "", err
	}

	return n.cri, nil
}

// Ports returns a specific port mapping for the node
// Node by convention use well known ports internally, while random port
// are used for making the `kind` cluster accessible from the host machine
func (n *Node) Ports(containerPort int32) (hostPort int32, err error) {
	return n.kindNode.Ports(containerPort)
}

// IP returns the IP address of the node
func (n *Node) IP() (ipv4 string, ipv6 string, err error) {
	return n.kindNode.IP()
}

// CopyFrom copies the source file on the node to dest on the host.
// Please note that this have limitations around symlinks.
func (n *Node) CopyFrom(source, dest string) error {
	return n.kindNode.CopyFrom(source, dest)
}

// CopyTo copies the source file on the host to dest on the node
func (n *Node) CopyTo(source, dest string) error {
	return n.kindNode.CopyTo(source, dest)
}

// KubeVersion returns the Kubernetes version installed on the node
func (n *Node) KubeVersion() (version string, err error) {
	return n.kindNode.KubeVersion()
}
