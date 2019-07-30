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
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri"
	"k8s.io/kubeadm/kinder/pkg/kubeadm"
	kindkustomize "sigs.k8s.io/kind/pkg/kustomize"
)

// kubeadmConfigOptionsall stores all the kinder flags that impact on the kubeadm config generation
type kubeadmConfigOptions struct {
	kubeDNS            bool
	automaticCopyCerts bool
}

// KubeadmConfig action writes the /kind/kubeadm.conf file on all the K8s nodes in the cluster.
// Please note that this action is automatically executed at create time, but it is possible
// to invoke it separately as well.
func KubeadmConfig(c *status.Cluster, kubeDNS bool, automaticCopyCerts bool) error {
	cp1 := c.BootstrapControlPlane()

	// get installed kubernetes version from the node image
	kubeVersion, err := cp1.KubeVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}

	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	controlPlaneEndpoint, controlPlaneEndpointIPv6, ControlPlanePort, err := getControlPlaneAddress(c)
	if err != nil {
		return err
	}

	// configure the right protocol addresses
	if c.Settings.IPFamily == status.IPv6Family {
		controlPlaneEndpoint = controlPlaneEndpointIPv6
	}

	// create configData with all the configurations supported by the kubeadm config template implemented in kind
	configData := kubeadm.ConfigData{
		ClusterName:          c.Name(),
		KubernetesVersion:    kubeVersion,
		ControlPlaneEndpoint: fmt.Sprintf("%s:%d", controlPlaneEndpoint, ControlPlanePort),
		APIBindPort:          constants.APIServerPort,
		APIServerAddress:     c.Settings.APIServerAddress,
		Token:                constants.Token,
		PodSubnet:            c.Settings.PodSubnet,
		ServiceSubnet:        c.Settings.ServiceSubnet,
		ControlPlane:         true,
		IPv6:                 c.Settings.IPFamily == status.IPv6Family,
	}

	// create configOptions with all the kinder flags that impact on the kubeadm config generation
	configOptions := kubeadmConfigOptions{
		kubeDNS:            kubeDNS,
		automaticCopyCerts: automaticCopyCerts,
	}

	// writs the kubeadm config file on all the K8s nodes.
	for _, node := range c.K8sNodes() {
		if err := writeKubeadmConfig(c, node, configData, configOptions); err != nil {
			return err
		}
	}

	return nil
}

// getControlPlaneAddress return the join address that is the control plane endpoint in case the cluster has
// an external load balancer in front of the control-plane nodes, otherwise the address of the
// boostrap control plane node.
func getControlPlaneAddress(c *status.Cluster) (string, string, int, error) {
	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	if c.ExternalLoadBalancer() != nil {
		// gets the IP of the load balancer
		loadBalancerIP, loadBalancerIPV6, err := c.ExternalLoadBalancer().IP()
		if err != nil {
			return "", "", 0, errors.Wrapf(err, "failed to get IP for node: %s", c.ExternalLoadBalancer().Name())
		}

		return loadBalancerIP, loadBalancerIPV6, constants.ControlPlanePort, nil
	}

	// gets the IP of the bootstrap control plane node
	controlPlaneIP, controlPlaneIPV6, err := c.BootstrapControlPlane().IP()
	if err != nil {
		return "", "", 0, errors.Wrapf(err, "failed to get IP for node: %s", c.BootstrapControlPlane().Name())
	}

	return controlPlaneIP, controlPlaneIPV6, constants.APIServerPort, nil
}

// writeKubeadmConfig writes the /kind/kubeadm.conf file on a node
func writeKubeadmConfig(c *status.Cluster, n *status.Node, data kubeadm.ConfigData, options kubeadmConfigOptions) error {
	log.Debugf("Writing kubeadm config on %s...", n.Name())

	// Amends the ConfigData struct with node specific settings

	// control plane/worker role
	data.ControlPlane = n.IsControlPlane()

	// the node address
	nodeAddress, nodeAddressIPv6, err := n.IP()
	if err != nil {
		return errors.Wrap(err, "failed to get IP for node")
	}

	data.NodeAddress = nodeAddress
	if c.Settings.IPFamily == status.IPv6Family {
		data.NodeAddress = nodeAddressIPv6
	}

	// Gets the kubeadm config customize for this node
	kubeadmConfig, err := getKubeadmConfig(c, n, data, options)
	if err != nil {
		return errors.Wrap(err, "failed to generate kubeadm config content")
	}

	log.Debugf("generated config:\n%s", kubeadmConfig)

	// copy the config to the node
	err = n.Command("cp", "/dev/stdin", constants.KubeadmConfigPath).
		Stdin(strings.NewReader(kubeadmConfig)).
		Silent().
		Run()

	if err != nil {
		return errors.Wrapf(err, "failed to write the kubeadm config to node %s", n.Name())
	}

	return nil
}

// getKubeadmConfig generates the kubeadm config customized for a specific node
func getKubeadmConfig(c *status.Cluster, n *status.Node, data kubeadm.ConfigData, options kubeadmConfigOptions) (string, error) {
	kubeadmVersion, err := n.KubeadmVersion()
	if err != nil {
		return "", err
	}

	// generate the "raw config", using the kubeadm config template provided by kind
	rawconfig, err := kubeadm.Config(kubeadmVersion, data)
	if err != nil {
		return "", err
	}

	// apply all the kinder specific settings using patches
	var patches = []string{}
	var jsonPatches = []kindkustomize.PatchJSON6902{}

	// add patches for instructing kubeadm to use the CRI runtime engine  installed on a node
	// NB. this is a no-op in case of containerd, because it is already the default in the raw config
	// TODO: currently we are always specifying the CRI kubeadm should use; it will be nice in the future to
	// have the possibility to test the kubeadm CRI autodetection
	nodeCRI, err := n.CRI()
	if err != nil {
		return "", err
	}

	criConfigHelper, err := cri.NewConfigHelper(nodeCRI)
	if err != nil {
		return "", err
	}

	criPatches, err := criConfigHelper.GetKubeadmConfigPatches(kubeadmVersion)
	if err != nil {
		return "", err
	}

	patches = append(patches, criPatches...)

	// if requested automatic copy certs and the node is a controlplane node,
	// add patches for adding the certificateKey value
	// NB. this is a no-op in case of kubeadm config API older than v1beta2, because
	// this feature was not supported before (the --certificate-key flag should be used instead)
	if options.automaticCopyCerts && n.IsControlPlane() {
		automaticCopyCertsPatches, err := kubeadm.GetAutomaticCopyCertsPatches(kubeadmVersion)
		if err != nil {
			return "", err
		}

		patches = append(patches, automaticCopyCertsPatches...)
	}

	// if requested automatic copy certs, add patches for using kube-dns addon instead of coreDNS
	if options.kubeDNS {
		kubeDNSPatch, err := kubeadm.GetKubeDNSPatch(kubeadmVersion)
		if err != nil {
			return "", err
		}
		patches = append(patches, kubeDNSPatch)
	}

	// if the cluster is using an external etcd node, add patches for configuring access
	// to external etcd cluster
	if c.ExternalEtcd() != nil {
		externalEtcdIP, externalEtcdIPV6, err := c.ExternalEtcd().IP()
		if err != nil {
			return "", errors.Wrapf(err, "failed to get IP for node: %s", c.ExternalEtcd().Name())
		}

		// configure the right protocol addresses
		if c.Settings.IPFamily == status.IPv6Family {
			externalEtcdIP = externalEtcdIPV6
		}

		externalEtcdPatch, err := kubeadm.GetExternalEtcdPatch(kubeadmVersion, externalEtcdIP)
		if err != nil {
			return "", err
		}

		patches = append(patches, externalEtcdPatch)
	}

	// if the cluster is using default PodSubnet, add the external calico patch
	// (otherwise use the value provided in the kind config at create time)
	if c.Settings.PodSubnet == constants.PodSubnet {
		calicoPatch, err := kubeadm.GetCalicoPatch(kubeadmVersion)
		if err != nil {
			return "", err
		}
		patches = append(patches, calicoPatch)
	}

	// After kinder patches, we append the patches from the kind config eventually
	// provided by the user at create time.
	// NB. patches from cluster settings MUST go after default patches in order to
	// allow user to kustomize everything, overriding also kinder default kustomizations
	// if necessary
	patches = append(patches, c.Settings.KubeadmConfigPatches...)
	jsonPatches = append(jsonPatches, c.Settings.KubeadmConfigPatchesJSON6902...)

	// fix all the patches to have name metadata matching the generated config
	patches, jsonPatches = setPatchNames(patches, jsonPatches)

	// apply patches
	patched, err := kindkustomize.Build([]string{rawconfig}, patches, jsonPatches)
	if err != nil {
		return "", err
	}

	// remove metadata info from the kubeadm config template provided by kind;
	// those info are not part of the kubeadm config API, but are necessary for Kustomize to work
	patched = removeMetadata(patched)

	// Select the objects that are relevant for a specific node;
	// if the node is the bootstrap control plane, then all the objects used as init time
	if n == c.BootstrapControlPlane() {
		return selectYamlFramentByKind(patched,
			"ClusterConfiguration",
			"InitConfiguration",
			"KubeletConfiguration",
			"KubeProxyConfiguration"), nil
	}

	// otherwise select only the JoinConfiguration
	return selectYamlFramentByKind(patched, "JoinConfiguration"), nil
}

// objectName is the name every generated object will have
// I.E. `metadata:\nname: config`
const objectName = "config"

// setPatchNames sets the targeted object name on every patch to be the fixed
// name we use when generating config objects (we have one of each type, all of
// which have the same fixed name)
func setPatchNames(patches []string, jsonPatches []kindkustomize.PatchJSON6902) ([]string, []kindkustomize.PatchJSON6902) {
	fixedPatches := make([]string, len(patches))
	fixedJSONPatches := make([]kindkustomize.PatchJSON6902, len(jsonPatches))
	for i, patch := range patches {
		// insert the generated name metadata
		fixedPatches[i] = fmt.Sprintf("metadata:\nname: %s\n%s", objectName, patch)
	}
	for i, patch := range jsonPatches {
		// insert the generated name metadata
		patch.Name = objectName
		fixedJSONPatches[i] = patch
	}
	return fixedPatches, fixedJSONPatches
}

// removeMetadata trims out the metadata.name we put in the config for kustomize matching,
// kubeadm will complain about this otherwise
func removeMetadata(kustomized string) string {
	return strings.Replace(
		kustomized,
		`metadata:
  name: config
`,
		"",
		-1,
	)
}

const yamlSeparator = "---\n"

// selectYamlFramentByKind selects yaml fragments of a specific list of kinds
func selectYamlFramentByKind(rawconfig string, kind ...string) string {
	yamls := strings.Split(rawconfig, yamlSeparator)

	config := []string{}
	for _, k := range kind {
		for _, y := range yamls {
			if strings.Contains(y, fmt.Sprintf("\nkind: %s\n", k)) {
				config = append(config, y)
			}
		}
	}

	return strings.Join(config, yamlSeparator)
}
