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
	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
)

// resetAction implements a developer friendly kubeadm reset workflow
type resetAction struct{}

func init() {
	kcluster.RegisterAction("kubeadm-reset", newResetAction)
}

func newResetAction() kcluster.Action {
	return &resetAction{}
}

// Tasks returns the list of action tasks for the resetAction
func (b *resetAction) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Destroy the Kubernetes cluster ⛵",
			TargetNodes: "@all",
			Run:         runReset,
		},
	}
}

func runReset(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	if err := kn.DebugCmd(
		"==> Kubeadm reset 🖥",
		"kubeadm", "reset", "--force",
	); err != nil {
		return err
	}

	return nil
}
