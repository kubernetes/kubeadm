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
Package kubeadm contains all the logic for creating kubeadm config patches to be
used during cluster creation.

Those logic are dependent to the K8s initVersion, and developed under the assumption
that there will be no version skew between K8s version and the kubeadm version.
*/
package kubeadm
