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

package cluster

import (
	"fmt"
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/constants"
)

func newTestNode(name string, role string) *KNode {
	return &KNode{name: name, role: role}
}

func (ns *KNodes) Names() string {
	var s = []string{}
	for _, n := range *ns {
		s = append(s, n.Name())
	}
	return fmt.Sprintf("[%s]", strings.Join(s, ", "))
}

var defaultNodes = KNodes{
	newTestNode("test-cp1", constants.ControlPlaneNodeRoleValue),
}

var haNodes = KNodes{
	newTestNode("test-lb", constants.ExternalLoadBalancerNodeRoleValue),
	newTestNode("test-cp1", constants.ControlPlaneNodeRoleValue),
	newTestNode("test-cp2", constants.ControlPlaneNodeRoleValue),
	newTestNode("test-cp3", constants.ControlPlaneNodeRoleValue),
	newTestNode("test-w1", constants.WorkerNodeRoleValue),
	newTestNode("test-w2", constants.WorkerNodeRoleValue),
}

func newTestCluster(name string, nodes KNodes) (c *KContext) {
	c = &KContext{
		Context: cluster.NewContext(name),
	}
	for _, n := range nodes {
		c.add(n)
	}
	return c
}

func TestSelectNodes(t *testing.T) {
	cases := []struct {
		TestName     string
		Nodes        KNodes
		NodeSelector string
		ExpectNodes  string
		ExpectError  bool
	}{

		{
			TestName:     "all on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@all",
			ExpectNodes:  "[test-cp1]",
		},
		{
			TestName:     "lb selector on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@lb",
			ExpectNodes:  "[]",
		},
		{
			TestName:     "cp* selector on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@cp*",
			ExpectNodes:  "[test-cp1]",
		},
		{
			TestName:     "cp1 selector on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@cp1",
			ExpectNodes:  "[test-cp1]",
		},
		{
			TestName:     "cpN selector on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@cpN",
			ExpectNodes:  "[]",
		},
		{
			TestName:     "w* selector on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "@w*",
			ExpectNodes:  "[]",
		},
		{
			TestName:     "select by node name on default cluster",
			Nodes:        defaultNodes,
			NodeSelector: "cp1",
			ExpectNodes:  "[test-cp1]",
		},

		{
			TestName:     "all on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@all",
			ExpectNodes:  "[test-cp1, test-cp2, test-cp3, test-w1, test-w2]",
		},
		{
			TestName:     "lb selector on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@lb",
			ExpectNodes:  "[test-lb]",
		},
		{
			TestName:     "cp* selector on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@cp*",
			ExpectNodes:  "[test-cp1, test-cp2, test-cp3]",
		},
		{
			TestName:     "cp1 selector on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@cp1",
			ExpectNodes:  "[test-cp1]",
		},
		{
			TestName:     "cpN selector on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@cpN",
			ExpectNodes:  "[test-cp2, test-cp3]",
		},
		{
			TestName:     "w* selector on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "@w*",
			ExpectNodes:  "[test-w1, test-w2]",
		},
		{
			TestName:     "select by node name on ha cluster",
			Nodes:        haNodes,
			NodeSelector: "cp1",
			ExpectNodes:  "[test-cp1]",
		},

		{
			TestName:     "node selectors are case insensitive",
			Nodes:        haNodes,
			NodeSelector: "@ALL",
			ExpectNodes:  "[test-cp1, test-cp2, test-cp3, test-w1, test-w2]",
		},
		{
			TestName:     "invalid selector",
			Nodes:        defaultNodes,
			NodeSelector: "@invalid",
			ExpectError:  true,
		},
		{
			TestName:    "node does not exists",
			Nodes:       defaultNodes,
			ExpectNodes: "[]",
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			var testCluster = newTestCluster("test", c.Nodes)

			n, err := testCluster.selectNodes(c.NodeSelector)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while adding nodes: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t.Fatalf("unexpected lack or error while adding nodes")
				}
			}

			if n.Names() != c.ExpectNodes {
				t.Errorf("saw %q as nodes, expected %q", n.Names(), c.ExpectNodes)
			}
		})
	}
}

func TestResolveNodesPath(t *testing.T) {
	var testCluster = newTestCluster("test", haNodes)

	cases := []struct {
		TestName    string
		NodesPath   string
		ExpectNodes string
		ExpectPath  string
		ExpectError bool
	}{
		{
			TestName:    "path without node (local path)",
			NodesPath:   "path",
			ExpectNodes: "[]",
			ExpectPath:  "path",
		},
		{
			TestName:    "nodeSelector path",
			NodesPath:   "@all:path",
			ExpectNodes: "[test-cp1, test-cp2, test-cp3, test-w1, test-w2]",
			ExpectPath:  "path",
		},
		{
			TestName:    "nodeName path",
			NodesPath:   "cp1:path",
			ExpectNodes: "[test-cp1]",
			ExpectPath:  "path",
		},
		{
			TestName:    "node-that-does-not-exists path",
			NodesPath:   "node-that-does-not-exists:path",
			ExpectNodes: "[]",
			ExpectPath:  "path",
		},
		{
			TestName:    "invalid node path",
			NodesPath:   "@all:path:invalid",
			ExpectError: true,
		},
		{
			TestName:    "invalid-node selector path",
			NodesPath:   "@invalid:path",
			ExpectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t *testing.T) {
			nodes, path, err := testCluster.resolveNodesPath(c.NodesPath)
			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err != nil {
				if !c.ExpectError {
					t.Fatalf("unexpected error while resolving nodes and path: %v", err)
				}
				return
			}
			// - not nil, in which case we should expect errors or fail
			if err == nil {
				if c.ExpectError {
					t.Fatalf("unexpected lack or error while resolving nodes and path")
				}
			}

			if nodes.Names() != c.ExpectNodes {
				t.Errorf("saw %q as nodes, expected %q", nodes.Names(), c.ExpectNodes)
			}

			if path != c.ExpectPath {
				t.Errorf("saw %q as path, expected %q", path, c.ExpectPath)
			}
		})
	}
}
