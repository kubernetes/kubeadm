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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/version"

	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/fs"
)

// copyKubeConfigToHost copies the admin.conf file to the host,
// taking care of replacing the server address with localhost:port
func copyKubeConfigToHost(kctx *kcluster.KContext, kn *kcluster.KNode) error {

	//TODO: use host port from external lb in case it exists
	hostPort, err := kn.Ports(6443)
	if err != nil {
		return errors.Wrap(err, "failed to get api server port mapping")
	}

	kubeConfigPath := kctx.KubeConfigPath()
	if err := kn.WriteKubeConfig(kubeConfigPath, hostPort); err != nil {
		return errors.Wrap(err, "failed to get kubeconfig from node")
	}

	return nil
}

// executes postInit tasks, including copying the admin.conf file to the host,
// installing the CNI plugin, and eventually remove the master taint
func postInit(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {

	if err := copyKubeConfigToHost(
		kctx, kn,
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> install cni 🗻",
		"/bin/sh", "-c", //use shell to get $(...) resolved into the container
		"kubectl apply --kubeconfig=/etc/kubernetes/admin.conf -f https://docs.projectcalico.org/v3.8/manifests/calico.yaml",
	); err != nil {
		return err
	}

	if len(kctx.Workers()) == 0 {
		if err := kn.DebugCmd(
			"==> remove master taint 🗻",
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "taint", "nodes", "--all", "node-role.kubernetes.io/master-",
		); err != nil {
			return err
		}
	}

	/*TODO
	// add the default storage class
	if err := addDefaultStorageClass(node); err != nil {
		return errors.Wrap(err, "failed to add default storage class")
	}
	*/

	if err := waitNewControlPlaneNodeReady(kctx, kn, flags); err != nil {
		return err
	}

	fmt.Printf(
		"Cluster creation complete. You can now use the cluster with:\n\n"+

			"export KUBECONFIG=\"$(kind get kubeconfig-path --name=%q)\"\n"+
			"kubectl cluster-info\n",
		kctx.Name(),
	)

	return nil
}

// getJoinAddress return the join address that is the control plane endpoint in case the cluster has
// an external load balancer in front of the control-plane nodes, otherwise the address of the
// boostrap control plane node.
func getJoinAddress(kctx *kcluster.KContext) (string, error) {
	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	if kctx.ExternalLoadBalancer() != nil {
		// gets the IP of the load balancer
		loadBalancerIP, err := kctx.ExternalLoadBalancer().IP()
		if err != nil {
			return "", errors.Wrapf(err, "failed to get IP for node: %s", kctx.ExternalLoadBalancer().Name())
		}

		return fmt.Sprintf("%s:%d", loadBalancerIP, ControlPlanePort), nil
	}

	// gets the IP of the bootstrap control plane node
	controlPlaneIP, err := kctx.BootStrapControlPlane().IP()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get IP for node: %s", kctx.BootStrapControlPlane().Name())
	}

	return fmt.Sprintf("%s:%d", controlPlaneIP, APIServerPort), nil
}

// Copy certs from the bootstrap master to the current node
func doManualCopyCerts(kctx *kcluster.KContext, kn *kcluster.KNode) error {
	fmt.Printf("==> copy certificate\n")

	// creates the folder tree for pre-loading necessary cluster certificates
	// on the joining node
	if err := kn.Command("mkdir", "-p", "/etc/kubernetes/pki/etcd").Run(); err != nil {
		return errors.Wrap(err, "failed to create pki folder")
	}

	// define the list of necessary cluster certificates
	fileNames := []string{
		"ca.crt", "ca.key",
		"front-proxy-ca.crt", "front-proxy-ca.key",
		"sa.pub", "sa.key",
	}
	if kctx.ExternalEtcd() == nil {
		fileNames = append(fileNames, "etcd/ca.crt", "etcd/ca.key")
	}

	// creates a temporary folder on the host that should acts as a transit area
	// for moving necessary cluster certificates
	tmpDir, err := fs.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = os.MkdirAll(filepath.Join(tmpDir, "/etcd"), os.ModePerm)
	if err != nil {
		return err
	}

	// copies certificates from the bootstrap control plane node to the joining node
	for _, fileName := range fileNames {
		fmt.Printf("%s\n", fileName)

		// sets the path of the certificate into a node
		containerPath := filepath.Join("/etc/kubernetes/pki", fileName)
		// set the path of the certificate into the tmp area on the host
		tmpPath := filepath.Join(tmpDir, fileName)
		// copies from bootstrap control plane node to tmp area
		if err := kctx.BootStrapControlPlane().CopyFrom(containerPath, tmpPath); err != nil {
			return errors.Wrapf(err, "failed to copy certificate %s", fileName)
		}
		// copies from tmp area to joining node
		if err := kn.CopyTo(tmpPath, containerPath); err != nil {
			return errors.Wrapf(err, "failed to copy certificate %s", fileName)
		}
	}

	fmt.Println()

	return nil
}

// Copy certs from the bootstrap master to the current node
func atLeastKubeadm(kn *kcluster.KNode, v string) error {
	kubeadmVersion, err := kn.KubeadmVersion()
	if err != nil {
		return err
	}

	vS, err := version.ParseSemantic(v)
	if err != nil {
		return err
	}

	if !kubeadmVersion.AtLeast(vS) {
		return errors.Errorf("At least kubeadm version %s is required on node %s, currently %q", v, kn.Name(), kubeadmVersion)
	}

	return nil
}

func waitForPodsRunning(kn *kcluster.KNode, label string, replicas int) error {
	for i := 0; i < 10; i++ {
		fmt.Printf(".")
		time.Sleep(time.Duration(i) * time.Second)

		if lines, err := kn.CombinedOutputLines(
			"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "-l", fmt.Sprintf("run=%s", label), "-o", "jsonpath='{.items[*].status.phase}'",
		); err == nil {
			if len(lines) != 1 {
				return errors.New("Error checking pod status")
			}

			statuses := strings.Split(strings.Trim(lines[0], "'"), " ")

			// if pod number not yet converged, wait
			if len(statuses) != replicas {
				continue
			}

			// check for pods status
			running := true
			for j := 0; j < replicas; j++ {
				if statuses[j] != "Running" {
					running = false
				}
			}

			if running {
				fmt.Printf("%d pods running!\n\n", replicas)
				return nil
			}
		}
	}

	return errors.New("Pod not yet started :-(")
}

func getNodePort(kn *kcluster.KNode, svc string) (string, error) {
	lines, err := kn.CombinedOutputLines(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "svc", svc, "--output=jsonpath='{range .spec.ports[0]}{.nodePort}'",
	)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get node port")
	}
	if len(lines) != 1 {
		return "", errors.New("failed to parse node port")
	}

	return strings.Trim(lines[0], "'"), nil
}

func checkNodePort(kctx *kcluster.KContext, port string) error {
	for _, n := range kctx.KubernetesNodes() {
		fmt.Printf("checking node port %s on node %s...", port, n.Name())

		ip, err := n.IP()
		if err != nil {
			return err
		}

		lines, err := n.CombinedOutputLines(
			"curl", "-I", fmt.Sprintf("http://%s:%s", ip, port),
		)

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

	fmt.Printf("\n")
	return nil
}

func getPodName(kn *kcluster.KNode, label string) (string, error) {
	lines, err := kn.CombinedOutputLines(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf", "get", "pods", "-l", fmt.Sprintf("run=%s", label), "-o", "jsonpath='{.items[0].metadata.name}'",
	)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get pod name")
	}
	if len(lines) != 1 {
		return "", errors.New("failed to parse pod name")
	}

	return strings.Trim(lines[0], "'"), nil
}
