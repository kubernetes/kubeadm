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

package actions

import (
	"fmt"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// CluterInfo actions prints the summary information about the cluster: list of nodes,
// list of pods, pods images, etcd members
func CluterInfo(c *status.Cluster) error {
	// commands are executed on the bootstrap control-plane
	cp1 := c.BootstrapControlPlane()

	if err := cp1.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "nodes", "-o=wide",
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "--all-namespaces", "-o=wide",
	).RunWithEcho(); err != nil {
		return err
	}

	if err := cp1.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "--all-namespaces",
		"-o=jsonpath={range .items[*]}{\"\\n\"}{.metadata.name}{\" << \"}{range .spec.containers[*]}{.image}{\", \"}{end}{end}",
	).RunWithEcho(); err != nil {
		return err
	}
	fmt.Println()

	if c.ExternalEtcd() == nil {
		// NB. before v1.13 local etcd is listening on localhost only; after v1.13
		// local etcd is listening on localhost and on the advertise address; we are
		// using localhost to accommodate both the use cases
		if err := cp1.Command(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "exec", "-n=kube-system", fmt.Sprintf("etcd-%s", c.BootstrapControlPlane().Name()),
			"--",
			"etcdctl", fmt.Sprintf("--endpoints=https://127.0.0.1:2379"),
			"--ca-file=/etc/kubernetes/pki/etcd/ca.crt", "--cert-file=/etc/kubernetes/pki/etcd/peer.crt", "--key-file=/etc/kubernetes/pki/etcd/peer.key",
			"member", "list",
		).RunWithEcho(); err != nil {
			return err
		}
	} else {
		fmt.Println("using external etcd")
	}

	return nil
}
