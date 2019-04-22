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

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/exec"
)

// waitNewControlPlaneNodeReady waits for a new control plane node reaching the target state after init/join
func waitNewControlPlaneNodeReady(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	fmt.Printf("==> Waiting for Node and control-plane Pods to become Ready (timeout %s) 📦\n", flags.Wait)
	if pass := waitFor(kctx, kn, flags.Wait,
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

// waitControlPlaneUpgraded waits for a control plane node reaching the target state after upgrade
func waitControlPlaneUpgraded(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	version := flags.UpgradeVersion.String()

	fmt.Printf("==> Waiting for control-plane Pods to restart with the new version (timeout %s) 📦\n", flags.Wait)
	if pass := waitFor(kctx, kn, flags.Wait,
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
func waitKubeletUpgraded(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	version := flags.UpgradeVersion.String()

	fmt.Printf("==> Waiting for Node to restart with the new version (timeout %s) 📦\n", flags.Wait)
	if pass := waitFor(kctx, kn, flags.Wait,
		nodeHasKubernetesVersion(version),
	); !pass {
		return errors.New("timeout: node did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitNewWorkerNodeReady waits for a new control plane node reaching the target state after join
func waitNewWorkerNodeReady(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	fmt.Printf("==> Waiting for Node to become Ready (timeout %s) 📦\n", flags.Wait)
	if pass := waitFor(kctx, kn, flags.Wait,
		nodeIsReady,
	); !pass {
		return errors.New("timeout: Node did not reach target state")
	}
	fmt.Println()
	return nil
}

// waitKubeletHasRBAC waits for the kubelet to have access to the expected config map
// please note that this is a temporary workaround for a problem we are observing on upgrades while
// executing node upgrades immediately after control-plane upgrade.
func waitKubeletHasRBAC(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	fmt.Printf("==> Waiting for kubelet RBAC validation - workaround (timeout %s) 📦\n", flags.Wait)
	if pass := waitFor(kctx, kn, flags.Wait,
		kubeletHasRBAC(flags.UpgradeVersion.Major(), flags.UpgradeVersion.Minor()),
	); !pass {
		return errors.New("timeout: Node did not reach target state")
	}
	fmt.Println()
	return nil
}

// try defines a function that test a condition to be waited for
type try func(*kcluster.KContext, *kcluster.KNode) bool

// waitFor implements the waiter core logic that is responsible for testing all the given contitions
// until are satisfied or a timeout are reached
func waitFor(kctx *kcluster.KContext, kn *kcluster.KNode, timeout time.Duration, conditions ...try) bool {
	// if timeout is 0 or no conditions are defined, exit fast
	if timeout == time.Duration(0) || len(conditions) == 0 {
		fmt.Println("Nothing to wait for, moving on")
		return true
	}

	// sets the timeout timer
	timer := time.NewTimer(timeout)

	// runs all the conditions in parallel
	pass := make(chan bool)
	for _, c := range conditions {
		// clone the condition func to make the closure point to right value
		// even after the for loop moves to the next condition
		x := c

		// run the condition in a go routine until it pass
		go func() {
			for {
				if x(kctx, kn) {
					pass <- true
					break
				}
				// add a little delay before retry
				time.Sleep(250 * time.Millisecond)
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
func nodeIsReady(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
	output := kubectlOutput(kctx.BootStrapControlPlane(),
		"get",
		"nodes",
		"--kubeconfig=/etc/kubernetes/admin.conf",
		// check for the selected node
		fmt.Sprintf("-l=kubernetes.io/hostname=%s", kn.Name()),
		// check for status.conditions type:Ready
		"-o=jsonpath='{.items..status.conditions[?(@.type == \"Ready\")].status}'",
	)
	if strings.Contains(output, "True") {
		fmt.Printf("Node %s is ready\n", kn.Name())
		return true
	}
	return false
}

// nodeHasKubernetesVersion implement a function that if a node is has the given Kubernetes version
func nodeHasKubernetesVersion(version string) func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
	return func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
		output := kubectlOutput(kctx.BootStrapControlPlane(),
			"get",
			"nodes",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			// check for the selected node
			fmt.Sprintf("-l=kubernetes.io/hostname=%s", kn.Name()),
			// check for the kubelet version
			"-o=jsonpath='{.items..status.nodeInfo.kubeletVersion}'",
		)
		if strings.Contains(output, version) {
			fmt.Printf("Node %s has Kubernetes version %s\n", kn.Name(), version)
			return true
		}
		return false
	}
}

// staticPodIsReady implement a function that test when a static pod is ready
func staticPodIsReady(pod string) func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
	return func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
		output := kubectlOutput(kctx.BootStrapControlPlane(),
			"get",
			"pods",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"-n=kube-system",
			// check for static pods existing on the selected node
			fmt.Sprintf("%s-%s", pod, kn.Name()),
			// check for status.conditions type:Ready
			"-o=jsonpath='{.status.conditions[?(@.type == \"Ready\")].status}'",
		)
		if strings.Contains(output, "True") {
			fmt.Printf("Pod %s-%s is ready\n", pod, kn.Name())
			return true
		}
		return false
	}
}

// staticPodHasVersion implement a function that if a static pod is has the given Kubernetes version
func staticPodHasVersion(pod, version string) func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
	return func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
		output := kubectlOutput(kctx.BootStrapControlPlane(),
			"get",
			"pods",
			"--kubeconfig=/etc/kubernetes/admin.conf",
			"-n=kube-system",
			// check for static pods existing on the selected node
			fmt.Sprintf("%s-%s", pod, kn.Name()),
			// check for the node image
			// NB. this assumes the Pod has only one container only
			// which is true for the control plane pods
			"-o=jsonpath='{.spec.containers[0].image}'",
		)
		if strings.Contains(output, version) {
			fmt.Printf("Pod %s-%s has Kubernetes version %s\n", pod, kn.Name(), version)
			return true
		}
		return false
	}
}

// kubeletHasRBAC is a test checking that kubelet has access to the expected config map
func kubeletHasRBAC(major, minor uint) func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
	return func(kctx *kcluster.KContext, kn *kcluster.KNode) bool {
		for i := 0; i < 5; i++ {
			output := kubectlOutput(kctx.BootStrapControlPlane(),
				"get",
				"cm",
				fmt.Sprintf("kubelet-config-%d.%d", major, minor),
				"--kubeconfig=/etc/kubernetes/kubelet.conf",
				"-n=kube-system",
				"-o=jsonpath='{.metadata.name}'",
			)
			if output != "" {
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
func kubectlOutput(kn *kcluster.KNode, args ...string) string {
	cmd := kn.Command(
		"kubectl",
		args...,
	)
	lines, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return ""
	}
	if len(lines) != 1 {
		return ""
	}

	return lines[0]
}
