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
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/config/v1alpha3"
	"sigs.k8s.io/kind/pkg/util"
)

// NOTE: this is fork of the validator tests in the kind internal configuration
// adapted to be able to test the validators using the kind public type.

func TestClusterValidate(t *testing.T) {
	cases := []struct {
		Name         string
		Cluster      v1alpha3.Cluster
		ExpectErrors int
	}{
		{
			Name: "Defaulted",
			Cluster: func() v1alpha3.Cluster {
				c := v1alpha3.Cluster{}
				v1alpha3.SetDefaults_Cluster(&c)
				return c
			}(),
		},
		{
			Name: "bogus podSubnet",
			Cluster: func() v1alpha3.Cluster {
				c := v1alpha3.Cluster{}
				v1alpha3.SetDefaults_Cluster(&c)
				c.Networking.PodSubnet = "aa"
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "missing control-plane",
			Cluster: func() v1alpha3.Cluster {
				c := v1alpha3.Cluster{}
				v1alpha3.SetDefaults_Cluster(&c)
				c.Nodes = []v1alpha3.Node{}
				return c
			}(),
			ExpectErrors: 1,
		},
		{
			Name: "bogus node",
			Cluster: func() v1alpha3.Cluster {
				c := v1alpha3.Cluster{}
				v1alpha3.SetDefaults_Cluster(&c)
				n, n2 := v1alpha3.Node{}, v1alpha3.Node{}
				v1alpha3.SetDefaults_Node(&n)
				v1alpha3.SetDefaults_Node(&n2)
				n.Role = "bogus"
				c.Nodes = []v1alpha3.Node{n, n2}
				return c
			}(),
			ExpectErrors: 1,
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			err := ValidateCluster(&tc.Cluster)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t.Error("received no errors but expected errors for case")
				}
				return
			}
			// - not castable to *Errors, in which case we have the wrong error type ...
			configErrors, ok := err.(util.Errors)
			if !ok {
				t.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - ConfigErrors, in which case expect a certain number of errors
			errors := configErrors.Errors()
			if len(errors) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errors, len(errors))
			}
		})
	}
}

// TODO(fabriziopandini): ideally this should use scheme.Default, but this creates a circular dependency
// So the current solution is to mimic defaulting for the validation test
func newDefaultedNode(role v1alpha3.NodeRole) v1alpha3.Node {
	return v1alpha3.Node{
		Role:  role,
		Image: "myImage:latest",
	}
}

func TestNodeValidate(t *testing.T) {
	cases := []struct {
		TestName     string
		Node         v1alpha3.Node
		ExpectErrors int
	}{
		{
			TestName:     "Canonical node",
			Node:         newDefaultedNode(v1alpha3.ControlPlaneRole),
			ExpectErrors: 0,
		},
		{
			TestName: "Empty image field",
			Node: func() v1alpha3.Node {
				cfg := newDefaultedNode(v1alpha3.ControlPlaneRole)
				cfg.Image = ""
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Empty role field",
			Node: func() v1alpha3.Node {
				cfg := newDefaultedNode(v1alpha3.ControlPlaneRole)
				cfg.Role = ""
				return cfg
			}(),
			ExpectErrors: 1,
		},
		{
			TestName: "Unknown role field",
			Node: func() v1alpha3.Node {
				cfg := newDefaultedNode(v1alpha3.ControlPlaneRole)
				cfg.Role = "ssss"
				return cfg
			}(),
			ExpectErrors: 1,
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.TestName, func(t *testing.T) {
			t.Parallel()
			err := ValidateNode(&tc.Node)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil {
				if tc.ExpectErrors != 0 {
					t.Error("received no errors but expected errors for case")
				}
				return
			}
			// - not castable to *Errors, in which case we have the wrong error type ...
			configErrors, ok := err.(util.Errors)
			if !ok {
				t.Errorf("config.Validate should only return nil or ConfigErrors{...}, got: %v", err)
				return
			}
			// - ConfigErrors, in which case expect a certain number of errors
			errors := configErrors.Errors()
			if len(errors) != tc.ExpectErrors {
				t.Errorf("expected %d errors but got len(%v) = %d", tc.ExpectErrors, errors, len(errors))
			}
		})
	}
}
