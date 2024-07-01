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

	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes"
	"k8s.io/kubeadm/kinder/pkg/kubeadm"
)

// kubeadmConfigOptionsall stores all the kinder flags that impact on the kubeadm config generation
type kubeadmConfigOptions struct {
	configVersion string
	copyCertsMode CopyCertsMode
	discoveryMode DiscoveryMode
}

// KubeadmInitConfig action writes the InitConfiguration into /kind/kubeadm.conf file on all the K8s nodes in the cluster.
// Please note that this action is automatically executed at create time, but it is possible
// to invoke it separately as well.
func KubeadmInitConfig(c *status.Cluster, kubeadmConfigVersion string, copyCertsMode CopyCertsMode, featureGate, encryptionAlgorithm string, nodes ...*status.Node) error {
	// defaults everything not relevant for the Init Config
	return KubeadmConfig(c, kubeadmConfigVersion, copyCertsMode, TokenDiscovery, featureGate, encryptionAlgorithm, nil, nodes...)
}

// KubeadmJoinConfig action writes the JoinConfiguration into /kind/kubeadm.conf file on all the K8s nodes in the cluster.
// Please note that this action is automatically executed at create time, but it is possible
// to invoke it separately as well.
func KubeadmJoinConfig(c *status.Cluster, kubeadmConfigVersion string, copyCertsMode CopyCertsMode, discoveryMode DiscoveryMode, nodes ...*status.Node) error {
	// defaults everything not relevant for the join Config
	return KubeadmConfig(c, kubeadmConfigVersion, copyCertsMode, discoveryMode, "", "", nil, nodes...)
}

// KubeadmUpgradeConfig action writes the UpgradeConfiguration into /kind/kubeadm.conf file on all the K8s nodes in the cluster.
func KubeadmUpgradeConfig(c *status.Cluster, upgradeVersion *version.Version, nodes ...*status.Node) error {
	return KubeadmConfig(c, "", "", "", "", "", upgradeVersion, nodes...)
}

// KubeadmResetConfig action writes the UpgradeConfiguration into /kind/kubeadm.conf file on all the K8s nodes in the cluster.
func KubeadmResetConfig(c *status.Cluster, nodes ...*status.Node) error {
	return KubeadmConfig(c, "", "", "", "", "", nil, nodes...)
}

// KubeadmConfig action writes the /kind/kubeadm.conf file on all the K8s nodes in the cluster.
// Please note that this action is automatically executed at create time, but it is possible
// to invoke it separately as well.
func KubeadmConfig(c *status.Cluster, kubeadmConfigVersion string, copyCertsMode CopyCertsMode, discoveryMode DiscoveryMode, featureGate, encryptionAlgorithm string, upgradeVersion *version.Version, nodes ...*status.Node) error {
	cp1 := c.BootstrapControlPlane()

	// get installed kubernetes version from the node image
	kubeVersion, err := cp1.KubeVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}

	// gets the IP of the bootstrap control plane node
	controlPlaneIP, controlPlaneIPV6, err := c.BootstrapControlPlane().IP()
	if err != nil {
		return errors.Wrapf(err, "failed to get IP for node: %s", c.BootstrapControlPlane().Name())
	}

	// get the control plane endpoint, in case the cluster has an external load balancer in
	// front of the control-plane nodes
	controlPlaneEndpoint, controlPlaneEndpointIPv6, ControlPlanePort, err := getControlPlaneAddress(c)
	if err != nil {
		return err
	}

	// configure the right protocol addresses
	if c.Settings.IPFamily == status.IPv6Family {
		controlPlaneIP = controlPlaneIPV6
		controlPlaneEndpoint = controlPlaneEndpointIPv6
	}

	featureGateName := ""
	featureGateValue := ""
	if len(featureGate) > 0 {
		split := strings.Split(featureGate, "=")
		if len(split) != 2 {
			return errors.New("feature gate must be formatted as 'key=value'")
		}
		featureGateName = split[0]
		featureGateValue = split[1]
	}

	if copyCertsMode == "" {
		copyCertsMode = CopyCertsModeAuto
	}

	if discoveryMode == "" {
		discoveryMode = TokenDiscovery
	}

	// Use a placeholder upgrade version for non-upgrade actions.
	if upgradeVersion == nil {
		upgradeVersion = version.MustParseSemantic("v1.0.0")
	}

	// create configData with all the configurations supported by the kubeadm config template implemented in kind
	configData := kubeadm.ConfigData{
		ClusterName:          c.Name(),
		KubernetesVersion:    kubeVersion,
		ControlPlaneEndpoint: fmt.Sprintf("%s:%d", controlPlaneEndpoint, ControlPlanePort),
		APIBindPort:          constants.APIServerPort,
		APIServerAddress:     controlPlaneIP,
		Token:                constants.Token,
		PodSubnet:            "192.168.0.0/16", // default for kindnet
		ControlPlane:         true,
		IPv6:                 c.Settings.IPFamily == status.IPv6Family,
		FeatureGateName:      featureGateName,
		FeatureGateValue:     featureGateValue,
		EncryptionAlgorithm:  encryptionAlgorithm,
		UpgradeVersion:       fmt.Sprintf("v%s", upgradeVersion.String()),
	}

	// create configOptions with all the kinder flags that impact on the kubeadm config generation
	configOptions := kubeadmConfigOptions{
		configVersion: kubeadmConfigVersion,
		copyCertsMode: copyCertsMode,
		discoveryMode: discoveryMode,
	}

	// writs the kubeadm config file on all the K8s nodes.
	for _, node := range nodes {
		if err := writeKubeadmConfig(c, node, configData, configOptions); err != nil {
			return err
		}
	}

	return nil
}

// getControlPlaneAddress return the join address that is the control plane endpoint in case the cluster has
// an external load balancer in front of the control-plane nodes, otherwise the address of the
// bootstrap control plane node.
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
	n.Infof("Preparing %s", constants.KubeadmConfigPath)

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

	log.Debug("generating config...")
	if log.GetLevel() == log.DebugLevel {
		fmt.Print(kubeadmConfig)
	}

	// copy the config to the node
	if err := n.WriteFile(constants.KubeadmConfigPath, []byte(kubeadmConfig)); err != nil {
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
	log.Debugf("kubeadm version %s", kubeadmVersion)

	kubeadmConfigVersion := options.configVersion
	if len(kubeadmConfigVersion) == 0 {
		kubeadmConfigVersion = kubeadm.GetKubeadmConfigVersion(kubeadmVersion)
	}
	log.Debugf("using kubeadm config version %s", kubeadmConfigVersion)

	// generate the "raw config", using the kubeadm config template provided by kind
	rawconfig, err := kubeadm.Config(kubeadmConfigVersion, data)
	if err != nil {
		return "", err
	}

	// apply all the kinder specific settings using patches
	var patches = []string{}
	var jsonPatches = []kubeadm.PatchJSON6902{}

	// add patches for instructing kubeadm to use the CRI runtime engine  installed on a node
	// NB. this is a no-op in case of containerd, because it is already the default in the raw config
	// TODO: currently we are always specifying the CRI kubeadm should use; it will be nice in the future to
	// have the possibility to test the kubeadm CRI autodetection
	nodeCRI, err := n.CRI()
	if err != nil {
		return "", err
	}

	criConfigHelper, err := nodes.NewConfigHelper(nodeCRI)
	if err != nil {
		return "", err
	}

	criPatches, err := criConfigHelper.GetKubeadmConfigPatches(kubeadmConfigVersion, data.ControlPlane)
	if err != nil {
		return "", err
	}

	patches = append(patches, criPatches...)

	// if requested automatic copy certs and the node is a controlplane node,
	// add patches for adding the certificateKey value
	// NB. this is a no-op in case of kubeadm config API older than v1beta2, because
	// this feature was not supported before (the --certificate-key flag should be used instead)
	if options.copyCertsMode == CopyCertsModeAuto && n.IsControlPlane() {
		automaticCopyCertsPatches, err := kubeadm.GetAutomaticCopyCertsPatches(kubeadmConfigVersion)
		if err != nil {
			return "", err
		}

		patches = append(patches, automaticCopyCertsPatches...)
	}

	// add patches directory to the config
	patchesDirectoryPatches, err := kubeadm.GetPatchesDirectoryPatches(kubeadmConfigVersion)
	// skip if kubeadm config version is not v1beta3
	if err == nil {
		patches = append(patches, patchesDirectoryPatches...)
	}

	// if requested to use file discovery and not the first control-plane, add patches for using file discovery
	if options.discoveryMode != TokenDiscovery && !(n == c.BootstrapControlPlane()) {
		// remove token from config
		removeTokenPatch, err := kubeadm.GetRemoveTokenPatch(kubeadmConfigVersion)
		if err != nil {
			return "", err
		}
		jsonPatches = append(jsonPatches, removeTokenPatch)

		// create the discovery file on the node
		// NB. this requires that kubeadm init is already completed on the BootstrapControlPlane in order
		// to have CAs and admin.conf already in place
		if err := createDiscoveryFile(c, n, options.discoveryMode); err != nil {
			return "", errors.Wrapf(err, "failed to generate a discovery file. Please ensure that kubeadm-init is already completed")
		}

		// add discovery file path to the config
		fileDiscoveryPatch, err := kubeadm.GetFileDiscoveryPatch(kubeadmConfigVersion)
		if err != nil {
			return "", err
		}
		patches = append(patches, fileDiscoveryPatch)

		// if the file discovery does not contains the authorization credentials, add tls discovery token
		if options.discoveryMode == FileDiscoveryWithoutCredentials {
			tlsBootstrapPatch, err := kubeadm.GetTLSBootstrapPatch(kubeadmConfigVersion)
			if err != nil {
				return "", err
			}
			patches = append(patches, tlsBootstrapPatch)
		}
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

		externalEtcdPatch, err := kubeadm.GetExternalEtcdPatch(kubeadmConfigVersion, externalEtcdIP)
		if err != nil {
			return "", err
		}

		patches = append(patches, externalEtcdPatch)
	}

	// encryption algorithm
	if len(data.EncryptionAlgorithm) > 0 {
		encryptionAlgorithmPatch, err := kubeadm.GetEncryptionAlgorithmPatch(kubeadmConfigVersion, data.EncryptionAlgorithm)
		if err != nil {
			return "", err
		}
		patches = append(patches, encryptionAlgorithmPatch)
	}

	// apply patches
	patched, err := kubeadm.Build(rawconfig, patches, jsonPatches)
	if err != nil {
		return "", err
	}

	// Select the objects that are relevant for a specific node;
	// if the node is the bootstrap control plane, then all the objects used as init time
	if n == c.BootstrapControlPlane() {
		return selectYamlFramentByKind(patched,
			"ClusterConfiguration",
			"InitConfiguration",
			"UpgradeConfiguration",
			"ResetConfiguration",
			"KubeletConfiguration",
			"KubeProxyConfiguration"), nil
	}

	// otherwise select only the JoinConfiguration
	return selectYamlFramentByKind(patched,
		"JoinConfiguration",
		"UpgradeCOnfiguration",
		"ResetConfiguration",
	), nil
}

func createDiscoveryFile(c *status.Cluster, n *status.Node, discoveryMode DiscoveryMode) error {
	// the discovery file is a kubeaconfig file, so for sake of semplicity in setting up this test,
	// we are using the admin.conf file created by kubeadm on the bootstrap control plane node
	// as a starting point (e.g. it already contains the necessary server address/server certificate)
	// IMPORTANT. Don't do this in production, admin.conf contains cluster-admin credentials.
	lines, err := c.BootstrapControlPlane().Command(
		"cat", "/etc/kubernetes/admin.conf",
	).Silent().RunAndCapture()
	if err != nil {
		return errors.Wrapf(err, "failed to read /etc/kubernetes/admin.conf from %s", c.BootstrapControlPlane().Name())
	}
	if len(lines) == 0 {
		return errors.Errorf("failed to read /etc/kubernetes/admin.conf from %s", c.BootstrapControlPlane().Name())
	}

	configBytes := []byte(strings.Join(lines, "\n"))
	config, err := clientcmd.Load(configBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to parse /etc/kubernetes/admin.conf from %s", c.BootstrapControlPlane().Name())
	}

	// tweak admin.conf into a discovery file that comply the expected Discovery Mode variant
	user := config.Contexts[config.CurrentContext].AuthInfo
	authInfo := config.AuthInfos[user]

	switch discoveryMode {
	case FileDiscoveryWithoutCredentials:
		// Nuke X509 credentials embedded in the admin.conf file
		authInfo.ClientKeyData = []byte{}
		authInfo.ClientCertificateData = []byte{}
	case FileDiscoveryWithToken:
		// Nuke X509 credentials embedded in the admin.conf file
		authInfo.ClientKeyData = []byte{}
		authInfo.ClientCertificateData = []byte{}
		// Add a token
		authInfo.Token = constants.Token
	case FileDiscoveryWithEmbeddedClientCerts:
		// This is NOP, because admin.conf already contains embedded client certs
	case FileDiscoveryWithExternalClientCerts:
		// Save the client certificate key embedded in admin.conf into an external file and update authinfo accordingly
		keyFile := "/kinder/discovery-client-key.pem"
		if err := n.WriteFile(keyFile, authInfo.ClientKeyData); err != nil {
			return err
		}
		authInfo.ClientKeyData = []byte{}
		authInfo.ClientKey = keyFile

		// Save the client certificate embedded in admin.conf into an external file and update authinfo accordingly
		certFile := "/kinder/discovery-client-cert.pem"
		if err := n.WriteFile(certFile, authInfo.ClientCertificateData); err != nil {
			return err
		}
		authInfo.ClientCertificateData = []byte{}
		authInfo.ClientCertificate = certFile
	}

	// writes the discovery file to the joining node
	configBytes, err = clientcmd.Write(*config)
	if err != nil {
		return errors.Wrapf(err, "failed to encode %s", constants.DiscoveryFile)
	}
	if err := n.WriteFile(constants.DiscoveryFile, configBytes); err != nil {
		return err
	}

	log.Debugf("generated discovery file:\n%s", string(configBytes))

	return nil
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
