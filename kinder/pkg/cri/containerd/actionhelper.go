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

package containerd

import (
	"fmt"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// PreLoadUpgradeImages preload images required by kubeadm-upgrade into the containerd runtime that exists inside a kind(er) node
func PreLoadUpgradeImages(n *status.Node, srcFolder string) error {
	// NB. this code is an extract from "sigs.k8s.io/kind/pkg/build/node"
	return n.Command(
		"bash", "-c",
		`find `+fmt.Sprintf("%s", srcFolder)+` -name *.tar -print0 | xargs -0 -n 1 -P $(nproc) ctr --namespace=k8s.io images import --no-unpack && rm -rf `+fmt.Sprintf("%s", srcFolder)+`/*.tar`,
	).Silent().Run()
}
