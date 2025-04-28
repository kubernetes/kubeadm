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
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

const etcKubernetes = "/etc/kubernetes"

// CopyCertificates actions automate the manual copy of
// certificates from the bootstrap control-plane to the secondary control-plane nodes
func CopyCertificates(c *status.Cluster) error {
	for _, n := range c.SecondaryControlPlanes() {
		if err := copyCertificatesToNode(c, n); err != nil {
			return err
		}
	}
	return nil
}

// copyCertificatesToNode copies certificate files from the bootstrap node to another node
func copyCertificatesToNode(c *status.Cluster, n *status.Node) error {
	// define the list of necessary cluster certificates
	fileNames := []string{
		"ca.crt", "ca.key",
		"front-proxy-ca.crt", "front-proxy-ca.key",
		"sa.pub", "sa.key",
	}
	if c.ExternalEtcd() == nil {
		fileNames = append(fileNames, "etcd/ca.crt", "etcd/ca.key")
	}

	// caKeys includes the list of keys that we throw warnings for, if missing
	caKeys := []string{"ca.key", "front-proxy-ca.key", "etcd/ca.key"}

	if err := copyBootstrapEtcKubernetesFilesToNode(c, n, "pki", fileNames, caKeys); err != nil {
		return err
	}
	return nil
}

// copyCAToNode copies the root CA cert and key to a node
func copyCAToNode(c *status.Cluster, n *status.Node) error {
	fileNames := []string{"ca.crt", "ca.key"}
	if err := copyBootstrapEtcKubernetesFilesToNode(c, n, "pki", fileNames, []string{}); err != nil {
		return err
	}
	return nil
}

// copyKubeconfigFilesToNode copies kubeconfig files from the bootstrap node to another node
func copyKubeconfigFilesToNode(c *status.Cluster, n *status.Node) error {
	fileNames := []string{
		"admin.conf",
		"controller-manager.conf",
		"scheduler.conf",
	}

	if err := copyBootstrapEtcKubernetesFilesToNode(c, n, "", fileNames, []string{}); err != nil {
		return err
	}
	return nil
}

// copyBootstrapEtcKubernetesFilesToNode is an utility function that can copy files from the bootstrap node's
// /etc/kubernetes directory to a the same directory on a node
func copyBootstrapEtcKubernetesFilesToNode(c *status.Cluster, n *status.Node, basePath string, fileNames, filesToWarn []string) error {
	n.Infof("Importing cluster certificates from %s", c.BootstrapControlPlane().Name())

	// creates the folder tree for pre-loading necessary cluster certificates and kubeconfig files
	// on the joining node
	if err := n.Command("mkdir", "-p", etcKubernetes+"/pki/etcd").Silent().Run(); err != nil {
		return errors.Wrap(err, "failed to create pki folder")
	}

	// copies certificates from the bootstrap control plane node to the joining node
	for _, fileName := range fileNames {
		fmt.Printf("%s\n", fileName)

		// sets the path of the certificate into a node
		containerPath := filepath.Join(etcKubernetes, basePath, fileName)

		// copies from bootstrap control plane node to tmp area
		lines, err := c.BootstrapControlPlane().Command(
			"cat", containerPath,
		).Silent().RunAndCapture()
		if err != nil {
			// assume the file is missing; check if this file should cause a warning
			// instead of erroring out (e.g. missing ca.key)
			if !slices.Contains(filesToWarn, fileName) {
				return errors.Wrapf(err, "failed to read file %s from %s", fileName, c.BootstrapControlPlane().Name())
			}
			fmt.Printf("Missing file %s on node %s\n", fileName, c.BootstrapControlPlane().Name())
			continue
		}
		// copies from tmp area to joining node
		if err := n.WriteFile(containerPath, []byte(strings.Join(lines, "\n"))); err != nil {
			return err
		}
	}

	return nil
}
