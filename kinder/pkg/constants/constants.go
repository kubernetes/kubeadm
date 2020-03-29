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

package constants

import (
	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	kindinternalkubeadm "k8s.io/kubeadm/kinder/third_party/kind/kubeadm"
	kindinternalloadbalancer "k8s.io/kubeadm/kinder/third_party/kind/loadbalancer"
	kindconstants "sigs.k8s.io/kind/pkg/cluster/constants"
)

// constants inherited from kind.
// those values are replicated here with the goal to keep under strict control kind dependencies
const (
	// DefaultNodeImage is the default name:tag for a base image
	DefaultBaseImage = "kindest/base:latest"

	// DefaultNodeImage is the default name:tag for a node image
	DefaultNodeImage = "kindest/node:latest"

	// ControlPlaneNodeRoleValue identifies a node that hosts a Kubernetes
	// control-plane.
	//
	// NOTE: in single node clusters, control-plane nodes act as worker nodes
	ControlPlaneNodeRoleValue string = kindconstants.ControlPlaneNodeRoleValue

	// WorkerNodeRoleValue identifies a node that hosts a Kubernetes worker
	WorkerNodeRoleValue string = kindconstants.WorkerNodeRoleValue

	// ExternalLoadBalancerNodeRoleValue identifies a node that hosts an
	// external load balancer for the API server in HA configurations.
	//
	// Please note that `kind` nodes (containers) hosting external load balancer are not kubernetes nodes
	ExternalLoadBalancerNodeRoleValue string = kindconstants.ExternalLoadBalancerNodeRoleValue

	// ExternalEtcdNodeRoleValue identifies a node that hosts an external-etcd
	// instance.
	//
	// WARNING: this node type is not yet implemented in kind! (in kinder it is implemented)
	//
	// Please note that `kind` nodes (containers) hosting external etcd are not kubernetes nodes
	ExternalEtcdNodeRoleValue string = kindconstants.ExternalEtcdNodeRoleValue

	// DefaultClusterName is the default cluster name
	DefaultClusterName = kindconstants.DefaultClusterName

	// ClusterLabelKey is applied to each "node" docker container for identification
	ClusterLabelKey = kindconstants.ClusterLabelKey

	// NodeRoleKey is applied to each "node" docker container for categorization
	// of nodes by role
	NodeRoleKey = kindconstants.NodeRoleKey

	// PodSubnet defines the default pod subnet used by kind
	// TODO: send a PR to define this value in a kind constant (currently it is not)
	PodSubnet = "10.244.0.0/16"

	// KubeadmConfigPath defines the path to the kubeadm config file in the K8s nodes
	// TODO: send a PR to define this value in a kind constant (currently it is not)
	KubeadmConfigPath = "/kind/kubeadm.conf"

	// KubeadmIgnorePreflightErrorsFlag holds the default list of preflight errors to skip
	// on "kubeadm init" and "kubeadm join"
	KubeadmIgnorePreflightErrorsFlag = "--ignore-preflight-errors=Swap,SystemVerification,FileContent--proc-sys-net-bridge-bridge-nf-call-iptables"

	// APIServerPort is the expected default APIServerPort on the control plane node(s)
	// https://kubernetes.io/docs/reference/access-authn-authz/controlling-access/#api-server-ports-and-ips
	APIServerPort = kindinternalkubeadm.APIServerPort

	// Token defines a dummy, well known token for automating TLS bootstrap process
	Token = kindinternalkubeadm.Token

	// ControlPlanePort defines the port where the control plane is listening on the load balancer node
	ControlPlanePort = kindinternalloadbalancer.ControlPlanePort

	// LoadBalancerImage defines the loadbalancer image:tag
	LoadBalancerImage = kindinternalloadbalancer.Image

	// ConfigPath defines the path to the config file in the load balancer node
	LoadBalancerConfigPath = kindinternalloadbalancer.ConfigPath
)

// constants used by the ClusterManager / inside actions
const (
	// CertificateKey defines a dummy, well known CertificateKey for automating automatic copy certs process
	// const CertificateKey = "d02db674b27811f4508bf8a5fa19fbe060921340552f13c15c9feb05aaa96824"
	CertificateKey = "0123456789012345678901234567890123456789012345678901234567890123"

	// DiscoveryFile defines the path to a discovery file stored on nodes
	DiscoveryFile = "/kinder/discovery.conf"

	// KustomizeDir defines the path to patches stored on node
	KustomizeDir = "/kinder/kustomize"
)

// kubernetes releases, used for branching code according to K8s release or kubeadm release version
var (
	// V1_13 minor version
	V1_13 = K8sVersion.MustParseSemantic("v1.13.0-0")

	// V1_14 minor version
	V1_14 = K8sVersion.MustParseSemantic("v1.14.0-0")

	// V1.15 minor version
	V1_15 = K8sVersion.MustParseSemantic("v1.15.0-0")

	// V1.16 minor version
	V1_16 = K8sVersion.MustParseSemantic("v1.16.0-0")

	// V1.17 minor version
	V1_17 = K8sVersion.MustParseSemantic("v1.17.0-0")

	// V1.18 minor version
	V1_18 = K8sVersion.MustParseSemantic("v1.18.0-0")
)

// other constants
const (
	// KinderVersion is the kinder CLI version
	KinderVersion = "0.1.0-alpha.3"
)
