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

/*
Package kubeadm contains all the logic for creating kubeadm config and the kubeadm
config patches to be used during cluster creation.

Having direct control on kubeadm config is a specific necessity for kinder, because
create nodes supports different CRI while kind supports only containerd;
additionally, in kinder all the actions for setting up a working cluster can happen
at different time while in kind everything - from create to a working K8s cluster -
happens within an atomic operation, create.

Another difference from kind, is that kinder support skew from kubeadm version and
K8s version, and as a consequence it was necessary to ensure that the code in
this package is dependent on the kubeadm version installed on nodes.

Nevertheless, the core config used by kinder is a fork from "sigs.k8s.io/kind/pkg/cluster/internal/kubeadm";
all the kinder specific settings are applied as kustomize patches.
*/
package kubeadm
