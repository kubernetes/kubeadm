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

	"github.com/pkg/errors"

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
)

// infoAction implements an action for getting a summary of cluster infos
type infoAction struct{}

func init() {
	kcluster.RegisterAction("cluster-info", newInfoAction)
}

func newInfoAction() kcluster.Action {
	return &infoAction{}
}

// Tasks returns the list of action tasks for the infoAction
func (b *infoAction) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Cluster-info ⛵",
			TargetNodes: "@cp1",
			Run:         runInfo,
		},
	}
}

func runInfo(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	if err := kn.DebugCmd(
		"==> List nodes 🖥",
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "nodes", "-o=wide",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> List pods 📦",
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "--all-namespaces", "-o=wide",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> Check image versions for each pods 🐋",
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "--all-namespaces",
		"-o=jsonpath={range .items[*]}{\"\\n\"}{.metadata.name}{\" << \"}{range .spec.containers[*]}{.image}{\", \"}{end}{end}",
	); err != nil {
		return err
	}

	ip, err := kn.IP()
	if err != nil {
		return errors.Wrap(err, "Error getting node ip")
	}

	if kctx.ExternalEtcd() == nil {
		if err := kn.DebugCmd(
			"\n==> List etcd members 📦",
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "exec", "-n=kube-system", fmt.Sprintf("etcd-%s", kn.Name()),
			"--",
			"etcdctl", fmt.Sprintf("--endpoints=https://%s:2379", ip),
			"--ca-file=/etc/kubernetes/pki/etcd/ca.crt", "--cert-file=/etc/kubernetes/pki/etcd/peer.crt", "--key-file=/etc/kubernetes/pki/etcd/peer.key",
			"member", "list",
		); err != nil {
			return err
		}
	} else {
		fmt.Println("\n==> Using external etcd 📦")
	}

	return nil
}
