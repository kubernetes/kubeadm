/*
Copyright 2020 The Kubernetes Authors.

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

package actions

import (
	"fmt"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// SetupExternalCA setups certificates and kubeconfig files to be able to create a cluster without CA keys.
func SetupExternalCA(c *status.Cluster, vLevel int) error {
	fmt.Println("Setuping external CA for the cluster...")

	// gets the IP of the load balancer
	loadBalancerIP, _, err := c.ExternalLoadBalancer().IP()
	if err != nil {
		return errors.Wrapf(err, "failed to get IP for node: %s", c.ExternalLoadBalancer().Name())
	}

	// generate certs on the primary node
	// ensure that "localhost" is included for the SANs for the kube-apiserver serving certificate,
	// so that the server is accessible from the kubeconfig on the host
	if err := c.BootstrapControlPlane().Command(
		"/bin/sh", "-c",
		fmt.Sprintf("kubeadm init phase certs all --control-plane-endpoint=%s --apiserver-cert-extra-sans=localhost --v=%d",
			loadBalancerIP,
			vLevel),
	).RunWithEcho(); err != nil {
		return errors.Wrapf(err, "could not generate certs on node: %s", c.BootstrapControlPlane().Name())
	}

	// generate kubeconfig files on the primary node
	if err := c.BootstrapControlPlane().Command(
		"/bin/sh", "-c",
		fmt.Sprintf("kubeadm init phase kubeconfig all --control-plane-endpoint=%s --v=%d",
			loadBalancerIP,
			vLevel),
	).RunWithEcho(); err != nil {
		return errors.Wrapf(err, "could not generate kubeconfig files on node: %s", c.BootstrapControlPlane().Name())
	}

	// Create a function to generate a kubelet.conf using the CA on a node.
	// This is required since without a CA key in the cluster, there is no authority
	// to sign the CSRs for new joining kubelets. Normally users should install
	// an external signer or manage the kubelet.conf files manually.
	generateKubeletConf := func(n *status.Node) error {
		if err := n.Command(
			"/bin/sh", "-c",
			fmt.Sprintf("kubeadm init phase kubeconfig kubelet --control-plane-endpoint=%s --v=%d",
				loadBalancerIP,
				vLevel),
		).RunWithEcho(); err != nil {
			return errors.Wrapf(err, "could not generate a kubelet.conf on node: %s", n.Name())
		}
		return nil
	}

	generateKubeletConfWorker := func(n *status.Node) error {
		if err := n.Command(
			"/bin/sh", "-c",
			fmt.Sprintf("kubeadm init phase kubeconfig kubelet --control-plane-endpoint=%s --apiserver-advertise-address=%s --v=%d",
				loadBalancerIP, loadBalancerIP,
				vLevel),
		).RunWithEcho(); err != nil {
			return errors.Wrapf(err, "could not generate a kubelet.conf on node: %s", n.Name())
		}
		return nil
	}

	// iterate secondary CP nodes
	for _, n := range c.SecondaryControlPlanes() {
		// copy the shared kubeconfig files
		if err := copyKubeconfigFilesToNode(c, n); err != nil {
			return errors.Wrapf(err, "could not copy kubeconfig files to node %s", n.Name())
		}

		// copy the shared certs, including the CA cert and key
		if err := copyCertificatesToNode(c, n); err != nil {
			return errors.Wrapf(err, "could not copy certs to node %s", n.Name())
		}

		// generate remaining certificates
		if err := n.Command(
			"/bin/sh", "-c",
			fmt.Sprintf("kubeadm init phase certs all --control-plane-endpoint=%s --apiserver-cert-extra-sans=localhost --v=%d",
				loadBalancerIP,
				vLevel),
		).RunWithEcho(); err != nil {
			return errors.Wrapf(err, "could not generate remaining certificates on node: %s", n.Name())
		}

		// generate kubelet.conf
		if err := generateKubeletConf(n); err != nil {
			return err
		}
	}

	// iterate all workers
	for _, n := range c.Workers() {
		// copy the CA cert and key
		if err := copyCAToNode(c, n); err != nil {
			return err
		}

		// generate kubelet.conf
		if err := generateKubeletConfWorker(n); err != nil {
			return err
		}
	}

	// delete the ca.key from all nodes
	for _, n := range c.AllNodes() {
		if err := n.Command("rm", "-f", "/etc/kubernetes/pki/ca.key").Run(); err != nil {
			return errors.Wrapf(err, "could not delete ca.key on node: %s", n.Name())
		}
	}

	return nil
}
