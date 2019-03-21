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

	"github.com/pkg/errors"
	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
)

// smokeTest implements a quick test about the proper functioning of a Kubernetes cluster
type smokeTest struct{}

func init() {
	kcluster.RegisterAction("smoke-test", newSmokeTest)
}

func newSmokeTest() kcluster.Action {
	return &smokeTest{}
}

// Tasks returns the list of action tasks for the smokeTest
func (b *smokeTest) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Joining control-plane node to Kubernetes ☸",
			TargetNodes: "@cp1",
			Run:         runSmokeTest,
		},
	}
}

func runSmokeTest(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {

	kn.Command("kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "delete", "deployments/nginx").Run()
	kn.Command("kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "delete", "service/nginx").Run()

	// Test deployments
	if err := kn.DebugCmd(
		"==> Test deployments 🖥",
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "run", "nginx", "--image=nginx:1.15.9-alpine", "--image-pull-policy=IfNotPresent",
	); err != nil {
		return err
	}

	if err := waitForPodsRunning(kn, "nginx", 1); err != nil {
		return err
	}

	// Test service type NodePort
	if err := kn.DebugCmd(
		"==> service type NodePort 🖥",
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "expose", "deployment", "nginx", "--port=80", "--type=NodePort",
	); err != nil {
		return err
	}

	nodePort, err := getNodePort(kn, "nginx")
	if err != nil {
		return err
	}

	err = checkNodePort(kctx, nodePort)
	if err != nil {
		return err
	}

	podName, err := getPodName(kn, "nginx")
	if err != nil {
		return err
	}

	fmt.Printf("==> Test kubectl logs 🖥\n\n")
	lines, err := kn.CombinedOutputLines(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "logs", podName,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to run kubectl logs")
	}
	fmt.Printf("%d logs lines returned\n\n", len(lines))

	fmt.Printf("==> Test kubectl exec 🖥\n\n")
	lines, err = kn.CombinedOutputLines(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "exec", podName, "--", "nslookup", "kubernetes",
	)
	if err != nil {
		return errors.Wrapf(err, "failed to run kubectl exec")
	}
	fmt.Printf("%d output lines returned\n\n", len(lines))

	fmt.Printf("==> Test DNS resolution 🖥\n\n")
	if len(lines) < 3 || !strings.Contains(lines[3], "kubernetes.default.svc.cluster.local") {
		return errors.Wrapf(err, "dns resolution error")
	}
	fmt.Printf("kubernetes service answers to %s\n\n", lines[3])

	kn.Command("kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "delete", "deployments/nginx").Run()
	kn.Command("kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "delete", "service/nginx").Run()

	fmt.Printf("==> Smoke test passed!\n\n")

	return nil
}
