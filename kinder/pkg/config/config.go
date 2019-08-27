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
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kindv1alpha3 "sigs.k8s.io/kind/pkg/apis/config/v1alpha3"
	"sigs.k8s.io/yaml"

	kindinternalconfig "k8s.io/kubeadm/kinder/third_party/kind/config"
)

// The purpose of this package to wrap the public kind types and allow validation and defaulting.
// The rest of kinder should import this package instead of the kind public type package.

// Cluster is a type alias of the kind Cluster type
type Cluster kindv1alpha3.Cluster

// Node is a type alias of the kind Node type
type Node kindv1alpha3.Node

// PatchJSON6902 is a type alias of the kind PatchJSON6902 type
type PatchJSON6902 kindv1alpha3.PatchJSON6902

// ConvertPatchJSON6902List converts a []kindv1alpha3.PatchJSON6902 to []PatchJSON6902
func ConvertPatchJSON6902List(plist []kindv1alpha3.PatchJSON6902) (converted []PatchJSON6902) {
	for _, p := range plist {
		converted = append(converted, PatchJSON6902(p))
	}
	return
}

// NewConfig returns the default config according to requested number of control-plane and worker nodes
func NewConfig(controlPlanes, workers int, image string) (*Cluster, error) {
	var cfg = &kindv1alpha3.Cluster{
		Nodes: []kindv1alpha3.Node{},
	}

	// adds the control-plane node(s)
	for i := 0; i < controlPlanes; i++ {
		cfg.Nodes = append(cfg.Nodes, kindv1alpha3.Node{Role: kindv1alpha3.ControlPlaneRole, Image: image})
	}

	// adds the worker node(s), if any
	for i := 0; i < workers; i++ {
		cfg.Nodes = append(cfg.Nodes, kindv1alpha3.Node{Role: kindv1alpha3.WorkerRole, Image: image})
	}

	// apply defaults and validate
	applyClusterDefaults(cfg)
	if err := kindinternalconfig.ValidateCluster(cfg); err != nil {
		return nil, err
	}
	return (*Cluster)(cfg), nil
}

// LoadConfig reads the file at path and attempts to convert into a `kind` configuration object
// If path == "" then the default config is returned
// If path == "-" then reads from stdin
func LoadConfig(path, imageName string) (*Cluster, error) {
	var cfg = &kindv1alpha3.Cluster{}
	var err error
	var contents []byte

	if path == "-" {
		// read in stdin
		contents, err = ioutil.ReadAll(os.Stdin)
	} else {
		// read in file
		contents, err = ioutil.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	if err = yaml.UnmarshalStrict(contents, cfg); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling kind config")
	}

	// apply image
	for i := range cfg.Nodes {
		cfg.Nodes[i].Image = imageName
	}

	// apply defaults and validate
	applyClusterDefaults(cfg)
	if err := kindinternalconfig.ValidateCluster(cfg); err != nil {
		return nil, err
	}
	return (*Cluster)(cfg), nil
}

// applyClusterDefaults defaults a kind Cluster object
func applyClusterDefaults(c *kindv1alpha3.Cluster) {
	kindv1alpha3.SetDefaults_Cluster(c)
	for i := range c.Nodes {
		kindv1alpha3.SetDefaults_Node(&c.Nodes[i])
	}
}
