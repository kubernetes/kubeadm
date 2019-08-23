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

package docker

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/constants"
	kindnodes "sigs.k8s.io/kind/pkg/cluster/nodes"
	kindCRI "sigs.k8s.io/kind/pkg/container/cri"
	kinddocker "sigs.k8s.io/kind/pkg/container/docker"
	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// CreateControlPlaneNode creates a kind(er) contol-plane node that uses docker runtime internally
func CreateControlPlaneNode(name, image, clusterLabel, listenAddress string, port int32, mounts []kindCRI.Mount, portMappings []kindCRI.PortMapping) error {
	// gets a random host port for the API server
	if port == 0 {
		p, err := getPort()
		if err != nil {
			return errors.Wrap(err, "failed to get port for API server")
		}
		port = p
	}

	// add api server port mapping
	portMappingsWithAPIServer := append(portMappings, kindCRI.PortMapping{
		ListenAddress: listenAddress,
		HostPort:      port,
		ContainerPort: constants.APIServerPort,
	})
	return createNode(
		name, image, clusterLabel, constants.ControlPlaneNodeRoleValue, mounts, portMappingsWithAPIServer,
		// publish selected port for the API server
		"--expose", fmt.Sprintf("%d", port),
	)
}

// CreateWorkerNode creates a kind(er) worker node node that uses the docker runtime internally
func CreateWorkerNode(name, image, clusterLabel string, mounts []kindCRI.Mount, portMappings []kindCRI.PortMapping) error {
	return createNode(name, image, clusterLabel, constants.WorkerNodeRoleValue, mounts, portMappings)
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

// createNode `docker run`s the node image, note that due to
// images/node/entrypoint being the entrypoint, this container will
// effectively be paused until we call actuallyStartNode(...)
func createNode(name, image, clusterLabel, role string, mounts []kindCRI.Mount, portMappings []kindCRI.PortMapping, extraArgs ...string) error {
	runArgs := []string{
		"-d", // run the container detached
		"-t", // allocate a tty for entrypoint logs
		// running containers in a container requires privileged
		// NOTE: we could try to replicate this with --cap-add, and use less
		// privileges, but this flag also changes some mounts that are necessary
		// including some ones docker would otherwise do by default.
		// for now this is what we want. in the future we may revisit this.
		"--privileged",
		"--security-opt", "seccomp=unconfined", // also ignore seccomp
		"--tmpfs", "/tmp", // various things depend on working /tmp
		"--tmpfs", "/run", // systemd wants a writable /run
		// some k8s things want /lib/modules
		"-v", "/lib/modules:/lib/modules:ro",
		"--hostname", name, // make hostname match container name
		"--name", name, // ... and set the container name
		// label the node with the cluster ID
		"--label", clusterLabel,
		// label the node with the role ID
		"--label", fmt.Sprintf("%s=%s", constants.NodeRoleKey, role),
		// explicitly set the entrypoint
		"--entrypoint=/usr/local/bin/entrypoint",
	}

	// pass proxy environment variables to be used by node's docker daemon
	proxyDetails, err := getProxyDetails()
	if err != nil || proxyDetails == nil {
		return errors.Wrap(err, "proxy setup error")
	}
	for key, val := range proxyDetails.Envs {
		runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// adds node specific args
	runArgs = append(runArgs, extraArgs...)

	if kinddocker.UsernsRemap() {
		// We need this argument in order to make this command work
		// in systems that have userns-remap enabled on the docker daemon
		runArgs = append(runArgs, "--userns=host")
	}

	err = kinddocker.Run(
		image,
		kinddocker.WithRunArgs(runArgs...),
		kinddocker.WithContainerArgs(
			// explicitly pass the entrypoint argument
			"/sbin/init",
		),
		kinddocker.WithMounts(mounts),
		kinddocker.WithPortMappings(portMappings),
	)
	if err != nil {
		return err
	}

	handle := kindnodes.FromName(name)

	// Deletes the machine-id embedded in the node image and regenerate a new one.
	// This is necessary because both kubelet and other components like weave net
	// use machine-id internally to distinguish nodes.
	if err := handle.Command("rm", "-f", "/etc/machine-id").Run(); err != nil {
		return errors.Wrap(err, "machine-id-setup error")
	}

	if err := handle.Command("systemd-machine-id-setup").Run(); err != nil {
		return errors.Wrap(err, "machine-id-setup error")
	}

	// we need to change a few mounts once we have the container
	// we'd do this ahead of time if we could, but --privileged implies things
	// that don't seem to be configurable, and we need that flag
	if err := fixMounts(handle); err != nil {
		return err
	}

	// signal the node container entrypoint to continue booting into systemd
	if err := signalStart(handle); err != nil {
		return err
	}

	// wait for docker to be ready
	if !waitForDocker(handle, time.Now().Add(time.Second*30)) {
		return errors.Errorf("timed out waiting for docker to be ready on node %s", handle.Name())
	}

	// load the docker image artifacts into the docker daemon
	loadImages(handle)

	return nil
}

// proxyDetails contains proxy settings discovered on the host
type proxyDetails struct {
	Envs map[string]string
	// future proxy details here
}

const (
	// Docker default bridge network is named "bridge" (https://docs.docker.com/network/bridge/#use-the-default-bridge-network)
	defaultNetwork = "bridge"
	httpProxy      = "HTTP_PROXY"
	httpsProxy     = "HTTPS_PROXY"
	noProxy        = "NO_PROXY"
)

// getProxyDetails returns a struct with the host environment proxy settings
// that should be passed to the nodes
func getProxyDetails() (*proxyDetails, error) {
	var proxyEnvs = []string{httpProxy, httpsProxy, noProxy}
	var val string
	var details proxyDetails
	details.Envs = make(map[string]string)

	proxySupport := false

	for _, name := range proxyEnvs {
		val = os.Getenv(name)
		if val != "" {
			proxySupport = true
			details.Envs[name] = val
			details.Envs[strings.ToLower(name)] = val
		} else {
			val = os.Getenv(strings.ToLower(name))
			if val != "" {
				proxySupport = true
				details.Envs[name] = val
				details.Envs[strings.ToLower(name)] = val
			}
		}
	}

	// Specifically add the docker network subnets to NO_PROXY if we are using proxies
	if proxySupport {
		subnets, err := getSubnets(defaultNetwork)
		if err != nil {
			return nil, err
		}
		noProxyList := strings.Join(append(subnets, details.Envs[noProxy]), ",")
		details.Envs[noProxy] = noProxyList
		details.Envs[strings.ToLower(noProxy)] = noProxyList
	}

	return &details, nil
}

// getSubnets returns a slice of subnets for a specified network
func getSubnets(networkName string) ([]string, error) {
	format := `{{range (index (index . "IPAM") "Config")}}{{index . "Subnet"}} {{end}}`
	lines, err := kinddocker.NetworkInspect([]string{networkName}, format)
	if err != nil {
		return nil, err
	}
	return strings.Split(lines[0], " "), nil
}

// fixMounts will correct mounts in the node container to meet the right
// sharing and permissions for systemd and Docker / Kubernetes
func fixMounts(n *kindnodes.Node) error {
	// Check if userns-remap is enabled
	if kinddocker.UsernsRemap() {
		// The binary /bin/mount should be owned by root:root in order to execute
		// the following mount commands
		if err := n.Command("chown", "root:root", "/bin/mount").Run(); err != nil {
			return err
		}
		// The binary /bin/mount should have the setuid bit
		if err := n.Command("chmod", "-s", "/bin/mount").Run(); err != nil {
			return err
		}
	}

	// systemd-in-a-container should have read only /sys
	// https://www.freedesktop.org/wiki/Software/systemd/ContainerInterface/
	// however, we need other things from `docker run --privileged` ...
	// and this flag also happens to make /sys rw, amongst other things
	if err := n.Command("mount", "-o", "remount,ro", "/sys").Run(); err != nil {
		return err
	}
	// kubernetes needs shared mount propagation
	if err := n.Command("mount", "--make-shared", "/").Run(); err != nil {
		return err
	}
	if err := n.Command("mount", "--make-shared", "/run").Run(); err != nil {
		return err
	}
	if err := n.Command("mount", "--make-shared", "/var/lib/docker").Run(); err != nil {
		return err
	}
	return nil
}

// signalStart sends SIGUSR1 to the node, which signals our entrypoint to boot
// see images/node/entrypoint
func signalStart(n *kindnodes.Node) error {
	return kinddocker.Kill("SIGUSR1", n.Name())
}

// waitForDocker waits for Docker to be ready on the node
// it returns true on success, and false on a timeout
func waitForDocker(n *kindnodes.Node, until time.Time) bool {
	return tryUntil(until, func() bool {
		cmd := n.Command("systemctl", "is-active", "docker")
		out, err := kindexec.CombinedOutputLines(cmd)
		if err != nil {
			return false
		}
		return len(out) == 1 && out[0] == "active"
	})
}

// helper that calls `try()`` in a loop until the deadline `until`
// has passed or `try()`returns true, returns wether try ever returned true
func tryUntil(until time.Time, try func() bool) bool {
	for until.After(time.Now()) {
		if try() {
			return true
		}
	}
	return false
}

// loadImages loads image tarballs stored on the node into docker on the node
func loadImages(n *kindnodes.Node) {
	// load images cached on the node into docker
	if err := n.Command(
		"/bin/bash", "-c",
		// use xargs to load images in parallel
		`find /kind/images -name *.tar -print0 | xargs -0 -n 1 -P $(nproc) docker load -i`,
	).Run(); err != nil {
		log.Warningf("Failed to preload docker images: %v", err)
		return
	}

	// if this fails, we don't care yet, but try to get the kubernetes version
	// and see if we can skip retagging for amd64
	// if this fails, we can just assume some unknown version and re-tag
	// in a future release of kind, we can probably drop v1.11 support
	// and remove the logic below this comment entirely
	if rawVersion, err := n.KubeVersion(); err == nil {
		if ver, err := K8sVersion.ParseGeneric(rawVersion); err == nil {
			if !ver.LessThan(K8sVersion.MustParseSemantic("v1.12.0")) {
				return
			}
		}
	}

	// for older releases, we need the images to have the arch in their name
	// bazel built images were missing these, newer releases do not use them
	// for any builds ...
	// retag images that are missing -amd64 as image:tag -> image-amd64:tag
	if err := n.Command(
		"/bin/bash", "-c",
		`docker images --format='{{.Repository}}:{{.Tag}}' | grep -v amd64 | xargs -L 1 -I '{}' /bin/bash -c 'docker tag "{}" "$(echo "{}" | sed s/:/-amd64:/)"'`,
	).Run(); err != nil {
		log.Warningf("Failed to re-tag docker images: %v", err)
	}
}
