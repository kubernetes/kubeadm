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
	"time"

	"github.com/pkg/errors"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"
)

// TODO: make actions name costants, because they are used also in create

// action registry defines the list of available actions and the corresponding entry point.
var actionRegistry = map[string]func(*status.Cluster, *RunOptions) error{
	"loadbalancer": func(c *status.Cluster, flags *RunOptions) error {
		// Nb. this action is invoked automatically at create time, but it is possible
		// to invoke it separately as well
		return LoadBalancer(c)
	},
	"kubeadm-config": func(c *status.Cluster, flags *RunOptions) error {
		// Nb. this action is invoked automatically at create time, but it is possible
		// to invoke it separately as well
		return KubeadmConfig(c, flags.kubeDNS, flags.automaticCopyCerts)
	},
	"kubeadm-init": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmInit(c, flags.usePhases, flags.automaticCopyCerts, flags.wait, flags.vLevel)
	},
	"kubeadm-join": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmJoin(c, flags.usePhases, flags.automaticCopyCerts, flags.wait, flags.vLevel)
	},
	"kubeadm-upgrade": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmUpgrade(c, flags.upgradeVersion, flags.wait, flags.vLevel)
	},
	"kubeadm-reset": func(c *status.Cluster, flags *RunOptions) error {
		return KubeadmReset(c, flags.vLevel)
	},
	"copy-certs": func(c *status.Cluster, flags *RunOptions) error {
		return CopyCertificates(c)
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

// KubeDNS option instructs kubeadm config action to prepare the cluster for using kube-dns instead of CoreDNS
func KubeDNS(kubeDNS bool) Option {
	return func(r *RunOptions) {
		r.kubeDNS = kubeDNS
	}
}

// UsePhases option instructs kubeadm actions to use kubeadm phases when supported
func UsePhases(usePhases bool) Option {
	return func(r *RunOptions) {
		r.usePhases = usePhases
	}
}

// AutomaticCopyCerts option instructs kubeadm init/join actions to use automatic copy certs when initializing the cluster and
// when joining control-plane nodes
func AutomaticCopyCerts(automaticCopyCerts bool) Option {
	return func(r *RunOptions) {
		r.automaticCopyCerts = automaticCopyCerts
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

// VLevel option sets the number for the log level verbosity for the kubeadm commands
func VLevel(vLevel int) Option {
	return func(r *RunOptions) {
		r.vLevel = vLevel
	}
}

// RunOptions holds options supplied to actions.Run
type RunOptions struct {
	kubeDNS            bool
	usePhases          bool
	automaticCopyCerts bool
	wait               time.Duration
	upgradeVersion     *K8sVersion.Version
	vLevel             int
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
