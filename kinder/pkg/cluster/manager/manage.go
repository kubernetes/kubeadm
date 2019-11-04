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

package manager

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/manager/actions"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kexec "sigs.k8s.io/kind/pkg/exec"
)

// ClusterManager manages kind(er) clusters
type ClusterManager struct {
	*status.Cluster
}

// NewClusterManager returns a new cluster manager ready to manage
// a kind(er) cluster with a current status discovered by the actual containers nodes
func NewClusterManager(clusterName string) (c *ClusterManager, err error) {
	// Check if the cluster clusterName already exists
	known, err := kindcluster.IsKnown(clusterName)
	if err != nil {
		return nil, err
	}
	if !known {
		return nil, errors.Errorf("a cluster with the name %q does not exists", clusterName)
	}

	// Gets the all the cluster nodes from docker
	x, err := status.FromDocker(clusterName)
	if err != nil {
		return nil, err
	}

	// Validate the cluster has a consistent set of nodes
	if err := x.Validate(); err != nil {
		return nil, err
	}

	// Read the cluster setting saved by kinder at creation time
	if err := x.ReadSettings(); err != nil {
		return nil, err
	}

	return &ClusterManager{
		Cluster: x,
	}, nil
}

// DryRun instruct the cluster manager to dry run commands (without actually running them)
func (c *ClusterManager) DryRun() {
	for _, n := range c.Cluster.AllNodes() {
		n.DryRun()
	}
}

// OnlyNode instruct the cluster manager to run only commands on one node
func (c *ClusterManager) OnlyNode(node string) {
	for _, n := range c.Cluster.AllNodes() {
		if n.Name() != node {
			n.SkipActions()
		}
	}
}

// DoAction actions on kind(er) cluster
// Actions are repetitive, high level workflows composed
// by one or more lower level commands
func (c *ClusterManager) DoAction(action string, options ...actions.Option) error {
	log.Infof("Running action %s...", action)
	return actions.Run(c.Cluster, action, options...)
}

// ExecCommand is a topology aware wrapper of docker exec
func (c *ClusterManager) ExecCommand(nodeSelector string, args []string) error {
	nodes, err := c.SelectNodes(nodeSelector)
	if err != nil {
		return err
	}

	log.Infof("%d nodes selected as target for the command", len(nodes))
	for _, node := range nodes {
		fmt.Printf("🚀 Executing command on node %s 🚀\n", node.Name())

		cmdArgs := append([]string{"exec",
			node.Name(),
		}, args...)
		cmd := kexec.Command("docker", cmdArgs...)
		kexec.InheritOutput(cmd)
		err := cmd.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to execute command on node %s", node.Name())
		}
	}

	return nil
}

// CopyFile is a topology aware wrapper of docker cp
func (c *ClusterManager) CopyFile(source, target string) error {
	sourceNodes, sourcePath, err := c.ResolveNodesPath(source)
	if err != nil {
		return err
	}

	teargetNodes, targetPath, err := c.ResolveNodesPath(target)
	if err != nil {
		return err
	}

	if sourceNodes == nil && teargetNodes == nil {
		return errors.Errorf("at least one between source and target must be a node/nodes in the cluster")
	}

	if sourceNodes != nil {
		switch len(sourceNodes) {
		case 1:
			break // one source node selected: continue
		case 0:
			return errors.Errorf("no source node matches given criteria")
		default:
			return errors.Errorf("source can't be more than one node")
		}
	}

	if teargetNodes != nil && len(teargetNodes) == 0 {
		return errors.Errorf("no target node matches given criteria")
	}

	if sourceNodes != nil && teargetNodes != nil {
		// create tmp folder
		// cp locally
		return errors.Errorf("copy between nodes not implemented yet!")
	}

	if teargetNodes == nil {
		fmt.Printf("Copying from %s ...\n", sourceNodes[0].Name())
		sourceNodes[0].CopyFrom(sourcePath, targetPath)
	}

	for _, n := range teargetNodes {
		fmt.Printf("Copying to %s ...\n", n.Name())
		n.CopyTo(sourcePath, targetPath)
	}
	return nil
}
