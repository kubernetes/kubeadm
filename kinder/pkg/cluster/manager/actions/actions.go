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

/*
Package actions implements kinder actions executed by the Kinder cluster manager
*/
package actions

import (
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// action registry defines the list of available actions and the corresponding entry point.
var actionRegistry = map[string]func(*status.Cluster, *RunOptions) error{
	"loadbalancer": func(c *status.Cluster, flags *RunOptions) error {
		// Nb. this action is invoked automatically at kubeadm init/join time, but it is possible
		// to invoke it separately as well
		return LoadBalancer(c, c.ControlPlanes()...)
	},
	"kubeadm-config": func(c *status.Cluster, flags *RunOptions) error {
		// Nb. this action is invoked automatically at kubeadm init/join time, but it is possible
		// to invoke it separately as well
		return KubeadmConfig(c, flags.kubeadmConfigVersion, flags.copyCertsMode, flags.discoveryMode, flags.featureGate, flags.encryptionAlgorithm, flags.upgradeVersion, c.K8sNodes().EligibleForActions()...)
	},
	"kubeadm-init": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmInit(c, flags.usePhases, flags.copyCertsMode, flags.kubeadmConfigVersion, flags.patchesDir, flags.ignorePreflightErrors, flags.featureGate, flags.encryptionAlgorithm, flags.wait, flags.vLevel)
	},
	"kubeadm-join": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmJoin(c, flags.usePhases, flags.copyCertsMode, flags.discoveryMode, flags.kubeadmConfigVersion, flags.patchesDir, flags.ignorePreflightErrors, flags.wait, flags.vLevel)
	},
	"kubeadm-upgrade": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmUpgrade(c, flags.upgradeVersion, flags.patchesDir, flags.wait, flags.vLevel)
	},
	"kubeadm-reset": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmReset(c, flags.vLevel)
	},
	"copy-certs": func(c *status.Cluster, flags *RunOptions) error {
		return CopyCertificates(c)
	},
	"setup-external-ca": func(c *status.Cluster, flags *RunOptions) error {
		return SetupExternalCA(c, flags.vLevel)
	},
	"cluster-info": func(c *status.Cluster, flags *RunOptions) error {
		return CluterInfo(c)
	},
	"smoke-test": func(c *status.Cluster, flags *RunOptions) error {
		return SmokeTest(c, flags.wait)
	},
}

// KnownActions returns the list of known actions
func KnownActions() []string {
	names := []string{}
	for n := range actionRegistry {
		names = append(names, n)
	}
	sort.Strings(names)

	return names
}

// Option is configuration option supplied to actions.Run
type Option func(*RunOptions)

// UsePhases option instructs kubeadm actions to use kubeadm phases when supported
func UsePhases(usePhases bool) Option {
	return func(r *RunOptions) {
		r.usePhases = usePhases
	}
}

// CopyCerts option instructs kubeadm init/join actions to use use different methods for copying certs when initializing the cluster and
// when joining control-plane nodes
func CopyCerts(copyCertsMode CopyCertsMode) Option {
	return func(r *RunOptions) {
		r.copyCertsMode = copyCertsMode
	}
}

// Wait option instructs actions to use wait for cluster state (nodes, pods) to converge to the desired state
func Wait(wait time.Duration) Option {
	return func(r *RunOptions) {
		r.wait = wait
	}
}

// UpgradeVersion option instructs kubeadm actions to use wait for cluster state (nodes, pods) to converge to the desired state
func UpgradeVersion(upgradeVersion *K8sVersion.Version) Option {
	return func(r *RunOptions) {
		r.upgradeVersion = upgradeVersion
	}
}

// Discovery option instructs kubeadm join to use a specific discovery mode
func Discovery(discoveryMode DiscoveryMode) Option {
	return func(r *RunOptions) {
		r.discoveryMode = discoveryMode
	}
}

// VLevel option sets the number for the log level verbosity for the kubeadm commands
func VLevel(vLevel int) Option {
	return func(r *RunOptions) {
		r.vLevel = vLevel
	}
}

// PatchesDir option sets the patches dir for the kubeadm commands
func PatchesDir(patchesDir string) Option {
	return func(r *RunOptions) {
		r.patchesDir = patchesDir
	}
}

// IgnorePreflightErrors sets which errors to ignore during kubeadm preflight
func IgnorePreflightErrors(ignorePreflightErrors string) Option {
	return func(r *RunOptions) {
		r.ignorePreflightErrors = ignorePreflightErrors
	}
}

// KubeadmConfigVersion option sets the kubeadm config version for the kubeadm commands
func KubeadmConfigVersion(kubeadmConfigVersion string) Option {
	return func(r *RunOptions) {
		r.kubeadmConfigVersion = kubeadmConfigVersion
	}
}

// FeatureGate option sets a single kubeadm feature-gate for the kubeadm commands
func FeatureGate(featureGate string) Option {
	return func(r *RunOptions) {
		// We remove the leading and trailing double or single quotes because the
		// feature-gate could be set as
		// --kubeadm-feature-gate="RootlessControlPlane=true" or
		// --kubeadm-feature-gate='RootlessControlPlane=true', so the value
		// of featureGate string would be "\"RootlessControlPlane=true"\" or "'RootlessControlPlane=true'" respectively.
		// Once we trim the value double or single quotes the value will be "RootlessControlPlane=true".
		trimmedFeatureGate := strings.Trim(featureGate, "\"'")
		r.featureGate = trimmedFeatureGate
	}
}

// EncryptionAlgorithm option sets the EncryptionAlgorithm during cluster creation
func EncryptionAlgorithm(algorithm string) Option {
	return func(r *RunOptions) {
		r.encryptionAlgorithm = algorithm
	}
}

// RunOptions holds options supplied to actions.Run
type RunOptions struct {
	usePhases             bool
	copyCertsMode         CopyCertsMode
	discoveryMode         DiscoveryMode
	wait                  time.Duration
	upgradeVersion        *K8sVersion.Version
	vLevel                int
	patchesDir            string
	ignorePreflightErrors string
	kubeadmConfigVersion  string
	featureGate           string
	encryptionAlgorithm   string
}

// DiscoveryMode defines discovery mode supported by kubeadm join
type DiscoveryMode string

const (
	// TokenDiscovery for kubeadm join
	TokenDiscovery = DiscoveryMode("token")

	// FileDiscoveryWithoutCredentials for kubeadm join
	FileDiscoveryWithoutCredentials = DiscoveryMode("file")

	// FileDiscoveryWithToken for kubeadm join
	FileDiscoveryWithToken = DiscoveryMode("file-with-token")

	// FileDiscoveryWithEmbeddedClientCerts for kubeadm join
	FileDiscoveryWithEmbeddedClientCerts = DiscoveryMode("file-with-embedded-client-certificates")

	// FileDiscoveryWithExternalClientCerts for kubeadm join
	FileDiscoveryWithExternalClientCerts = DiscoveryMode("file-with-external-client-certificates")
)

// KnownDiscoveryMode returns the list of known DiscoveryMode
func KnownDiscoveryMode() []string {
	return []string{
		string(TokenDiscovery),
		string(FileDiscoveryWithoutCredentials),
		string(FileDiscoveryWithToken),
		string(FileDiscoveryWithEmbeddedClientCerts),
		string(FileDiscoveryWithExternalClientCerts),
	}
}

// ValidateDiscoveryMode validates a DiscoveryMode
func ValidateDiscoveryMode(t DiscoveryMode) error {
	switch t {
	case TokenDiscovery:
	case FileDiscoveryWithoutCredentials:
	case FileDiscoveryWithToken:
	case FileDiscoveryWithEmbeddedClientCerts:
	case FileDiscoveryWithExternalClientCerts:
	default:
		return errors.Errorf("invalid discovery mode. Use one of %s", KnownDiscoveryMode())
	}
	return nil
}

// CopyCertsMode defines the mode used to copy certs on kubeadm join
type CopyCertsMode string

const (
	// CopyCertsModeNone results in no certs being copied
	CopyCertsModeNone = CopyCertsMode("none")

	// CopyCertsModeManual manually copies certs
	CopyCertsModeManual = CopyCertsMode("manual")

	// CopyCertsModeAuto copies certs using the --upload-certs / --certificate-key functionality
	CopyCertsModeAuto = CopyCertsMode("auto")
)

// KnownCopyCertsMode returns the list of known CopyCertsMode
func KnownCopyCertsMode() []string {
	return []string{
		string(CopyCertsModeNone),
		string(CopyCertsModeManual),
		string(CopyCertsModeAuto),
	}
}

// ValidateCopyCertsMode validates a CopyCertsMode
func ValidateCopyCertsMode(t CopyCertsMode) error {
	switch t {
	case CopyCertsModeNone:
	case CopyCertsModeManual:
	case CopyCertsModeAuto:
	default:
		return errors.Errorf("invalid copy-certs mode. Use one of %s", KnownCopyCertsMode())
	}
	return nil
}

// Run executes one action
func Run(c *status.Cluster, action string, options ...Option) error {
	flags := &RunOptions{}
	for _, o := range options {
		o(flags)
	}

	if a, ok := actionRegistry[action]; ok {
		return a(c, flags)
	}

	return errors.Errorf("%s is not a valid action name. Use one of %s", action, KnownActions())
}
