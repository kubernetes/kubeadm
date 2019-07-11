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

package loadbalancer

import (
	kindinternalloadbalancer "k8s.io/kubeadm/kinder/third_party/kind/loadbalancer"
)

// ConfigData is supplied to the load balancer config template, with values populated by the cluster package
//
// NB. this is an alias to a kind internal type from "sigs.k8s.io/kind/pkg/cluster/internal/loadbalancer" package forked under third_party folder;
// always prefer using this alias instead of the internal type.
type ConfigData kindinternalloadbalancer.ConfigData

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
//
// NB. this ia proxy to a kind internal function from "sigs.k8s.io/kind/pkg/cluster/internal/create/actions/config"
// package forked under third_party folder; always prefer using this proxy instead of the internal func.
func Config(data ConfigData) (config string, err error) {
	internalData := kindinternalloadbalancer.ConfigData(data)
	return kindinternalloadbalancer.Config(&internalData)
}
