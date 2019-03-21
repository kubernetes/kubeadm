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

// manualCopyCerts implements copy of certs from bootstrap control-plane to secondary control-planes
type manualCopyCerts struct{}

func init() {
	kcluster.RegisterAction("manual-copy-certs", newManualCopyCerts)
}

func newManualCopyCerts() kcluster.Action {
	return &manualCopyCerts{}
}

// Tasks returns the list of action tasks for the manualCopyCerts
func (b *manualCopyCerts) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Joining control-plane node to Kubernetes ☸",
			TargetNodes: "@cpN",
			Run:         runManualCopyCerts,
		},
	}
}

func runManualCopyCerts(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	err := doManualCopyCerts(kctx, kn)
	if err != nil {
		return err
	}

	return nil
}
