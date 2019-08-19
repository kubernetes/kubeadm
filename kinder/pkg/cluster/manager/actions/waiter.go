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
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// waitNewControlPlaneNodeReady waits for a new control plane node reaching the target state after init/join
func waitNewControlPlaneNodeReady(c *status.Cluster, n *status.Node, wait time.Duration) error {
	n.Infof("waiting for Node and control-plane Pods to become Ready (timeout %s)", wait)
	if pass := waitFor(c, n, wait,
		nodeIsReady,
		staticPodIsReady("kube-apiserver"),
		staticPodIsReady("kube-controller-manager"),
		staticPodIsReady("kube-scheduler"),
	); !pass {
		return errors.New("timeout: Node and control-plane did not reach target state")
	}
	fmt.Println()
	return nil
}

func waitForPodsRunning(c *status.Cluster, n *status.Node, wait time.Duration, label string, replicas int) error {
	if pass := waitFor(c, n, wait,
		podsAreRunning(n, label, replicas),
	); !pass {
		return errors.New("timeout: Node and control-plane did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitNewWorkerNodeReady waits for a new control plane node reaching the target state after join
func waitNewWorkerNodeReady(c *status.Cluster, n *status.Node, wait time.Duration) error {
	n.Infof("waiting for Node to become Ready (timeout %s)", wait)
	if pass := waitFor(c, n, wait,
		nodeIsReady,
	); !pass {
		return errors.New("timeout: Node did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitControlPlaneUpgraded waits for a control plane node reaching the target state after upgrade
func waitControlPlaneUpgraded(c *status.Cluster, n *status.Node, upgradeVersion *K8sVersion.Version, wait time.Duration) error {
	version := kubernetesVersionToImageTag(upgradeVersion.String())

	n.Infof("waiting for control-plane Pods to restart with the new version (timeout %s)", wait)
	if pass := waitFor(c, n, wait,
		staticPodHasVersion("kube-apiserver", version),
		staticPodHasVersion("kube-controller-manager", version),
		staticPodHasVersion("kube-scheduler", version),
	); !pass {
		return errors.New("timeout: control-plane did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitKubeletUpgraded waits for a node reaching the target state after upgrade
func waitKubeletUpgraded(c *status.Cluster, n *status.Node, upgradeVersion *K8sVersion.Version, wait time.Duration) error {
	version := upgradeVersion.String()

	n.Infof("waiting for node to restart with the new version (timeout %s)", wait)
	if pass := waitFor(c, n, wait,
		nodeHasKubernetesVersion(version),
	); !pass {
		return errors.New("timeout: node did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitKubeletHasRBAC waits for the kubelet to have access to the expected config map
// please note that this is a temporary workaround for a problem we are observing on upgrades while
// executing node upgrades immediately after control-plane upgrade.
func waitKubeletHasRBAC(c *status.Cluster, n *status.Node, upgradeVersion *K8sVersion.Version, wait time.Duration) error {
	n.Infof("waiting for kubelet RBAC validation - workaround (timeout %s)", wait)
	if pass := waitFor(c, n, wait,
		kubeletHasRBAC(upgradeVersion.Major(), upgradeVersion.Minor()),
	); !pass {
		return errors.New("timeout: Node did not reach target state")
	}
	fmt.Println()
	return nil
}

// try defines a function that test a condition to be waited for
type try func(*status.Cluster, *status.Node) bool

// waitFor implements the waiter core logic that is responsible for testing all the given contitions
// until are satisfied or a timeout are reached
func waitFor(c *status.Cluster, n *status.Node, timeout time.Duration, conditions ...try) bool {
	// if timeout is 0 or no conditions are defined, exit fast
	if timeout == time.Duration(0) {
		fmt.Println("Timeout set 0, skipping wait")
		return true
	}

	// sets the timeout timer
	timer := time.NewTimer(timeout)

	// runs all the conditions in parallel
	pass := make(chan bool)
	for _, wc := range conditions {
		// clone the condition func to make the closure point to right value
		// even after the for loop moves to the next condition
		x := wc

		// run the condition in a go routine until it pass
		go func() {
			// creates an arbitrary skew before starting a wait loop
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

			for {
				if x(c, n) {
					pass <- true
					break
				}
				// add a little delay + jitter before retry
				time.Sleep(1*time.Second + time.Duration(rand.Intn(500))*time.Millisecond)
			}
		}()
	}

	// wait for all the conditions to pass or for a timeout
	passed := 0
	for {
		select {
		case <-pass:
			passed++
			if passed == len(conditions) {
				return true
			}
		case <-timer.C:
			return false
		}
	}
}

// nodeIsReady implement a function that test when a node is ready
func nodeIsReady(c *status.Cluster, n *status.Node) bool {
	output := kubectlOutput(c.BootstrapControlPlane(),
		"get",
		"nodes",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		// check for the selected node
		fmt.Sprintf("-l=kubernetes.io/hostname=%s", n.Name()),
		// check for status.conditions type:Ready
		"-o=jsonpath='{.items..status.conditions[?(@.type == \"Ready\")].status}'",
	)
	if strings.Contains(output, "True") {
		fmt.Printf("Node %s is ready\n", n.Name())
		return true
	}
	return false
}

// nodeHasKubernetesVersion implement a function that if a node is has the given Kubernetes version
func nodeHasKubernetesVersion(version string) func(c *status.Cluster, n *status.Node) bool {
	return func(c *status.Cluster, n *status.Node) bool {
		output := kubectlOutput(c.BootstrapControlPlane(),
			"get",
			"nodes",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			// check for the selected node
			fmt.Sprintf("-l=kubernetes.io/hostname=%s", n.Name()),
			// check for the kubelet version
			"-o=jsonpath='{.items..status.nodeInfo.kubeletVersion}'",
		)
		if strings.Contains(output, version) {
			fmt.Printf("Node %s has Kubernetes version %s\n", n.Name(), version)
			return true
		}
		return false
	}
}

// staticPodIsReady implement a function that test when a static pod is ready
func staticPodIsReady(pod string) func(c *status.Cluster, n *status.Node) bool {
	return func(c *status.Cluster, n *status.Node) bool {
		output := kubectlOutput(c.BootstrapControlPlane(),
			"get",
			"pods",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"-n=kube-system",
			// check for static pods existing on the selected node
			fmt.Sprintf("%s-%s", pod, n.Name()),
			// check for status.conditions type:Ready
			"-o=jsonpath='{.status.conditions[?(@.type == \"Ready\")].status}'",
		)
		if strings.Contains(output, "True") {
			fmt.Printf("Pod %s-%s is ready\n", pod, n.Name())
			return true
		}
		return false
	}
}

func podsAreRunning(n *status.Node, label string, replicas int) func(c *status.Cluster, n *status.Node) bool {
	return func(c *status.Cluster, n *status.Node) bool {
		output := kubectlOutput(n,
			"get",
			"pods",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"-l", fmt.Sprintf("run=%s", label), "-o", "jsonpath='{.items[*].status.phase}'",
		)

		statuses := strings.Split(strings.Trim(output, "'"), " ")

		// if pod number not yet converged, wait
		if len(statuses) != replicas {
			return false
		}

		// check for pods status
		running := true
		for j := 0; j < replicas; j++ {
			if statuses[j] != "Running" {
				running = false
			}
		}

		if running {
			fmt.Printf("%d pods running!", replicas)
			return true
		}

		return false
	}
}

// staticPodHasVersion implement a function that if a static pod is has the given Kubernetes version
func staticPodHasVersion(pod, version string) func(c *status.Cluster, n *status.Node) bool {
	return func(c *status.Cluster, n *status.Node) bool {
		output := kubectlOutput(c.BootstrapControlPlane(),
			"get",
			"pods",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"-n=kube-system",
			// check for static pods existing on the selected node
			fmt.Sprintf("%s-%s", pod, n.Name()),
			// check for the node image
			// NB. this assumes the Pod has only one container only
			// which is true for the control plane pods
			"-o=jsonpath='{.spec.containers[0].image}'",
		)
		if strings.Contains(output, version) {
			fmt.Printf("Pod %s-%s has Kubernetes version %s\n", pod, n.Name(), version)
			return true
		}
		return false
	}
}

// kubeletHasRBAC is a test checking that kubelet has reliable access to kubelet-config-x.y and kube-proxy,
// where reliable = it have access for 5 seconds in a row.
//
// this test is a workaround meant to prevent errors like: configmaps "kube-proxy" is forbidden:
// User "system:node:kinder-upgrade-control-plane3" cannot get resource "configmaps" in API
// group "" in the namespace "kube-system": no relationship found between node "kinder-upgrade-control-plane3"
// and this object
//
// The real source of this errors during upgrades is still not clear, but it is probably related to
// the restarting of control-plane components after control-plane upgrade like e.g. the node authorizer
func kubeletHasRBAC(major, minor uint) func(c *status.Cluster, n *status.Node) bool {
	return func(c *status.Cluster, n *status.Node) bool {
		for i := 0; i < 5; i++ {
			output1 := kubectlOutput(c.BootstrapControlPlane(),
				"get",
				"cm",
				fmt.Sprintf("kubelet-config-%d.%d", major, minor),
				"--kubeconfig=/etc/kubernetes/kubelet.conf",
				"-n=kube-system",
				"-o=jsonpath='{.metadata.name}'",
			)
			output2 := kubectlOutput(c.BootstrapControlPlane(),
				"get",
				"cm",
				"kube-proxy",
				"--kubeconfig=/etc/kubernetes/kubelet.conf",
				"-n=kube-system",
				"-o=jsonpath='{.metadata.name}'",
			)
			if output1 != "" && output2 != "" {
				time.Sleep(1 * time.Second)
				continue
			}
			return false
		}

		fmt.Println("kubelet has access to expected config maps")
		return true
	}
}

// kubectlOutput implements a utility function that runs a kubectl command on the given node
// and captures the command output
func kubectlOutput(n *status.Node, args ...string) string {
	lines, err := n.Command(
		"kubectl",
		args...,
	).Silent().RunAndCapture()
	if err != nil {
		return ""
	}
	if len(lines) != 1 {
		return ""
	}

	return lines[0]
}

// kubernetesVersionToImageTag is helper function that replaces all
// non-allowed symbols in tag strings with underscores.
// Image tag can only contain lowercase and uppercase letters, digits,
// underscores, periods and dashes.
// Current usage is for CI images where all of symbols except '+' are valid,
// but function is for generic usage where input can't be always pre-validated.
func kubernetesVersionToImageTag(version string) string {
	allowed := regexp.MustCompile(`[^-a-zA-Z0-9_\.]`)
	return allowed.ReplaceAllString(version, "_")
}
