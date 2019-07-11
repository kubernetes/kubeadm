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
	"strings"
	"time"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// SmokeTest actions execute a set of simple test checking proper functioning of
// deployments, services/type node port, kubectl logs & exec & DNS resolution
func SmokeTest(c *status.Cluster, wait time.Duration) error {
	// test are executed on the bootstrap control-plane
	cp1 := c.BootstrapControlPlane()

	// cleanups garbage from previous test
	cleanupSmokeTest(cp1)

	// Test deployments
	cp1.Infof("test deployments")

	if err := cp1.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"run", "nginx", "--image=nginx:1.15.9-alpine", "--image-pull-policy=IfNotPresent",
	).RunWithEcho(); err != nil {
		return err
	}

	if err := waitForPodsRunning(c, cp1, wait, "nginx", 1); err != nil {
		return err
	}

	// Test service type NodePort
	cp1.Infof("service type NodePort")

	if err := cp1.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"expose", "deployment", "nginx", "--port=80", "--type=NodePort",
	).Run(); err != nil {
		return err
	}

	nodePort, err := getNodePort(cp1, "nginx")
	if err != nil {
		return err
	}

	err = checkNodePort(c, nodePort)
	if err != nil {
		return err
	}

	podName, err := getPodName(cp1, "nginx")
	if err != nil {
		return err
	}

	// Test kubectl logs
	cp1.Infof("test kubectl logs")

	lines, err := cp1.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "logs", podName,
	).RunAndCapture()
	if err != nil {
		return errors.Wrapf(err, "failed to run kubectl logs")
	}
	fmt.Printf("%d logs lines returned\n", len(lines))

	// Test kubectl exec
	cp1.Infof("test kubectl exec")

	lines, err = cp1.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "exec", podName, "--", "nslookup", "kubernetes",
	).RunAndCapture()
	if err != nil {
		return errors.Wrapf(err, "failed to run kubectl exec")
	}
	fmt.Printf("%d output lines returned\n", len(lines))

	// Test DNS resolution
	cp1.Infof("test DNS resolution")

	if len(lines) < 3 || !strings.Contains(lines[3], "kubernetes.default.svc.cluster.local") {
		return errors.Wrapf(err, "dns resolution error")
	}
	fmt.Printf("kubernetes service answers to %s\n", lines[3])

	// cleanups and print final message
	cleanupSmokeTest(cp1)
	fmt.Printf("\nSmoke test passed!\n")

	return nil
}

func cleanupSmokeTest(cp1 *status.Node) {
	cp1.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "deployments/nginx",
	).Silent().Run()

	cp1.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "service/nginx",
	).Silent().Run()
}

func getNodePort(n *status.Node, svc string) (string, error) {
	lines, err := n.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "svc", svc, "--output=jsonpath='{range .spec.ports[0]}{.nodePort}'",
	).Silent().RunAndCapture()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get node port")
	}
	if len(lines) != 1 {
		return "", errors.New("failed to parse node port")
	}

	return strings.Trim(lines[0], "'"), nil
}

func checkNodePort(c *status.Cluster, port string) error {
	for _, n := range c.K8sNodes() {
		fmt.Printf("checking node port %s on node %s...", port, n.Name())

		//TODO: test IPV6
		ip, _, err := n.IP()
		if err != nil {
			return err
		}

		lines, err := n.Command(
			"curl", "-Is", fmt.Sprintf("http://%s:%s", ip, port),
		).Silent().RunAndCapture()

		if err != nil {
			return errors.Wrapf(err, "error checking node port")
		}

		if len(lines) < 1 {
			return errors.Wrapf(err, "error checking node port. invalid answer")
		}

		if strings.Trim(lines[0], "\n\r") == "HTTP/1.1 200 OK" {
			fmt.Printf("pass!\n")
			continue
		}

		return errors.Errorf("node port %s on node %s doesn't works", port, n.Name())
	}

	return nil
}

func getPodName(n *status.Node, label string) (string, error) {
	lines, err := n.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		"get", "pods", "-l", fmt.Sprintf("run=%s", label), "-o", "jsonpath='{.items[0].metadata.name}'",
	).Silent().RunAndCapture()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get pod name")
	}
	if len(lines) != 1 {
		return "", errors.New("failed to parse pod name")
	}

	return strings.Trim(lines[0], "'"), nil
}
