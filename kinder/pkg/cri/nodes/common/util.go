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

package common

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/exec"
)

// BaseRunArgs computes docker arguments that apply to all containers
func BaseRunArgs(cluster, name, role string) ([]string, error) {
	// standard arguments all nodes containers need, computed once
	args := []string{
		"run",
		"--detach", // run the container detached
		"--tty",    // allocate a tty for entrypoint logs
		// label the node with the cluster ID
		"--label", fmt.Sprintf("%s=%s", constants.ClusterLabelKey, cluster),
		"--label", fmt.Sprintf("%s=%s", constants.DeprecatedClusterLabelKey, cluster),
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleLabelKey, role),
		"--label", fmt.Sprintf("%s=%s", constants.DeprecatedNodeRoleLabelKey, role),
	}

	// TODO: enable IPv6 if necessary
	// args = append(args, "--sysctl=net.ipv6.conf.all.disable_ipv6=0", "--sysctl=net.ipv6.conf.all.forwarding=1")

	// pass proxy environment variables
	proxyEnv, err := getProxyEnvs()
	if err != nil {
		return nil, errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// handle hosts that have user namespace remapping enabled
	if UsernsRemap() {
		args = append(args, "--userns=host")
	}
	return args, nil
}

// UsernsRemap checks if userns-remap is enabled in dockerd
func UsernsRemap() bool {
	cmd := exec.NewHostCmd("docker", "info", "--format", "'{{json .SecurityOptions}}'")
	lines, err := cmd.RunAndCapture()
	if err != nil {
		return false
	}
	if len(lines) > 0 {
		if strings.Contains(lines[0], "name=userns") {
			return true
		}
	}
	return false
}

const (
	defaultNetwork = "bridge"
	httpProxy      = "HTTP_PROXY"
	httpsProxy     = "HTTPS_PROXY"
	noProxy        = "NO_PROXY"
)

func getProxyEnvs() (map[string]string, error) {
	envs := make(map[string]string)
	for _, name := range []string{httpProxy, httpsProxy, noProxy} {
		val := os.Getenv(name)
		if val == "" {
			val = os.Getenv(strings.ToLower(name))
		}
		if val != "" {
			envs[name] = val
			envs[strings.ToLower(name)] = val
		}
	}
	// Specifically add the cluster subnets to NO_PROXY if we are using a proxy
	if len(envs) > 0 {
		noProxyVal := envs[noProxy]
		if noProxyVal != "" {
			noProxyVal += ","
		}
		//TODO: noProxy += cfg.Networking.ServiceSubnet + "," + cfg.Networking.PodSubnet
		envs[noProxy] = noProxyVal
		envs[strings.ToLower(noProxy)] = noProxyVal
	}

	// Specifically add the docker network subnets to NO_PROXY if we are using a proxy
	if len(envs) > 0 {
		// Docker default bridge network is named "bridge" (https://docs.docker.com/network/bridge/#use-the-default-bridge-network)
		subnets, err := getSubnets(defaultNetwork)
		if err != nil {
			return nil, err
		}
		noProxyList := strings.Join(append(subnets, envs[noProxy]), ",")
		envs[noProxy] = noProxyList
		envs[strings.ToLower(noProxy)] = noProxyList
	}

	return envs, nil
}

func getSubnets(networkName string) ([]string, error) {
	format := `{{range (index (index . "IPAM") "Config")}}{{index . "Subnet"}} {{end}}`
	cmd := exec.NewHostCmd("docker", "network", "inspect", "-f", format, networkName)
	lines, err := cmd.RunAndCapture()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get subnets")
	}
	return strings.Split(strings.TrimSpace(lines[0]), " "), nil
}

// RunArgsForNode computes docker run arguments that apply to containers that should host K8s nodes
func RunArgsForNode(role string, volumes []string, args []string) ([]string, error) {
	args = append(args,
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		// runtime temporary storage
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// runtime persistent storage
		// this ensures that E.G. pods, logs etc. are not on the container
		// filesystem, which is not only better for performance, but allows
		// running kind in kind for "party tricks"
		// (please don't depend on doing this though!)
		"--volume", "/var",
		// some k8s things want to read /lib/modules
		"--volume", "/lib/modules:/lib/modules:ro",
	)

	for _, v := range volumes {
		args = append(args, "--volume", v)
	}

	if role == constants.ControlPlaneNodeRoleValue {
		// API server port mapping
		hostPort, err := getPort()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get host port for the API server address")
		}
		args = append(args, fmt.Sprintf("--publish=%d:%d/TCP", hostPort, constants.ControlPlanePort))
	}

	return args, nil
}

// helper used to get a free TCP port for the API server
func getPort() (int32, error) {
	dummyListener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer dummyListener.Close()
	port := dummyListener.Addr().(*net.TCPAddr).Port
	return int32(port), nil
}

// RunArgsForExternalLoadBalancer computes docker run arguments that apply to containers that should host external load balancers
func RunArgsForExternalLoadBalancer(args []string) ([]string, error) {
	// load balancer port mapping
	hostPort, err := getPort()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get host port for the load balancer endpoint")
	}
	args = append(args, fmt.Sprintf("--publish=%d:%d/TCP", hostPort, constants.ControlPlanePort))

	return args, nil
}

// RunArgsForExternalEtcd computes docker run arguments that apply to containers that should host external etcd members
func RunArgsForExternalEtcd(args []string) []string {
	return args
}

// ContainerArgsForExternalEtcd computes arguments to pass to the external etcd container's entry point
func ContainerArgsForExternalEtcd(name string, args []string) []string {
	args = append(args,
		// define a minimal etcd (insecure, single node, not exposed to the host machine)
		"etcd",
		"--name", fmt.Sprintf("%s-etcd", name),
		"--advertise-client-urls", "http://127.0.0.1:2379",
		"--listen-client-urls", "http://0.0.0.0:2379",
	)

	return args
}

// TryUntil implements an helper that calls `try()` in a loop until the deadline `until`
// has passed or `try()`returns true, returns whether try ever returned true
func TryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}
