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

package config

import (
	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha3"
)

// NOTE: this is fork of the defaulting functions in the kind internal configuration
// adapted to be able to default the kind public types.

// ApplyClusterDefaults defaults a kind Cluster object
func ApplyClusterDefaults(c *v1alpha3.Cluster) {
	v1alpha3.SetDefaults_Cluster(c)
	for i := range c.Nodes {
		v1alpha3.SetDefaults_Node(&c.Nodes[i])
	}
}
