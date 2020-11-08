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

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/cri/host"
	"k8s.io/kubeadm/kinder/pkg/loadbalancer"
)

// LoadBalancer action writes the loadbalancer configuration file on the load balancer node.
// Please note that this action is automatically executed at create time, but it is possible
// to invoke it separately as well.
func LoadBalancer(c *status.Cluster, nodes ...*status.Node) error {
	// identify external load balancer node
	lb := c.ExternalLoadBalancer()

	// if there's no loadbalancer we're done
	if lb == nil {
		return nil
	}

	ipv6 := (c.Settings.IPFamily == status.IPv6Family)

	// collect info about the existing controlplane nodes
	lb.Infof("Updating load balancer configuration with %d control plane backends", len(nodes))

	var backendServers = map[string]string{}
	for _, n := range nodes {
		controlPlaneIPv4, controlPlaneIPv6, err := n.IP()
		if err != nil {
			return errors.Wrapf(err, "failed to get IP for node %s", n.Name())
		}
		if controlPlaneIPv4 != "" && !ipv6 {
			backendServers[n.Name()] = fmt.Sprintf("%s:%d", controlPlaneIPv4, constants.APIServerPort)
		}
		if controlPlaneIPv6 != "" && ipv6 {
			backendServers[n.Name()] = fmt.Sprintf("[%s]:%d", controlPlaneIPv6, constants.APIServerPort)
		}
	}

	// create loadbalancer config data
	loadbalancerConfig, err := loadbalancer.Config(&loadbalancer.ConfigData{
		ControlPlanePort: constants.ControlPlanePort,
		BackendServers:   backendServers,
		IPv6:             ipv6,
	})
	if err != nil {
		return errors.Wrap(err, "failed to generate loadbalancer config data")
	}

	// create loadbalancer config on the node
	log.Debugf("Writing loadbalancer config on %s...", lb.Name())

	if err := lb.WriteFile(constants.LoadBalancerConfigPath, []byte(loadbalancerConfig)); err != nil {
		return errors.Wrap(err, "failed to copy loadbalancer config to node")
	}

	// reload the config
	if err := host.SendSignal("SIGHUP", lb.Name()); err != nil {
		return errors.Wrap(err, "failed to reload loadbalancer")
	}

	return nil
}
