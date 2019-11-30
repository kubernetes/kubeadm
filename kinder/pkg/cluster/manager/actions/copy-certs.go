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
	"strings"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

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

// copyCertificatesToNode automate copy of certificates from the bootstrap control-plane node to the target node
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

	n.Infof("Importing cluster certificates from %s", c.BootstrapControlPlane().Name())

	// creates the folder tree for pre-loading necessary cluster certificates into the node
	// on the joining node
	if err := n.Command("mkdir", "-p", "/etc/kubernetes/pki/etcd").Silent().Run(); err != nil {
		return errors.Wrap(err, "failed to create pki folder")
	}

	// copies certificates from the bootstrap control plane node to the joining node
	for _, fileName := range fileNames {
		fmt.Printf("%s\n", fileName)

		// sets the path of the certificate into a node
		containerPath := filepath.Join("/etc/kubernetes/pki", fileName)

		// copies from bootstrap control plane node to tmp area
		lines, err := c.BootstrapControlPlane().Command(
			"cat", containerPath,
		).Silent().RunAndCapture()
		if err != nil {
			return errors.Wrapf(err, "failed to read certificate %s from %s", fileName, c.BootstrapControlPlane().Name())
		}
		// copies from tmp area to joining node
		if err := n.WriteFile(containerPath, []byte(strings.Join(lines, "\n"))); err != nil {
			return err
		}
	}

	return nil
}
