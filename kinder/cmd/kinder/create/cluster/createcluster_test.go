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

package cluster

import (
	"strings"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/config"
)

func TestNewConfig(t *testing.T) {
	cases := []struct {
		TestName             string
		initVersion          string
		controlPlanes        int32
		workers              int32
		kubeDNS              bool
		externalEtcdIP       string
		expectedPatchVersion string
	}{
		{
			TestName:             "Default",
			initVersion:          "v1.15.0",
			controlPlanes:        1,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta2",
		},
		{
			TestName:             "More workers",
			initVersion:          "v1.15.0",
			controlPlanes:        1,
			workers:              2,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta2",
		},
		{
			TestName:             "More control-planes",
			initVersion:          "v1.15.0",
			controlPlanes:        2,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta2",
		},
		{
			TestName:             "Kube dns",
			initVersion:          "v1.15.0",
			controlPlanes:        1,
			kubeDNS:              true,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta2",
		},
		{
			TestName:             "External etcd",
			initVersion:          "v1.15.0",
			controlPlanes:        1,
			externalEtcdIP:       "https://1.2.3.4:5678",
			expectedPatchVersion: "kubeadm.k8s.io/v1beta2",
		},
		{
			TestName:             "initVersion v1.14",
			initVersion:          "v1.14.0",
			controlPlanes:        1,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta1",
		},
		{
			TestName:             "initVersion v1.13",
			initVersion:          "v1.13.0",
			controlPlanes:        1,
			expectedPatchVersion: "kubeadm.k8s.io/v1beta1",
		},
		{
			TestName:             "initVersion v1.12",
			initVersion:          "v1.12.0",
			controlPlanes:        1,
			expectedPatchVersion: "kubeadm.k8s.io/v1alpha3",
		},
		{
			TestName:             "initVersion v1.11",
			initVersion:          "v1.11.0",
			controlPlanes:        1,
			expectedPatchVersion: "kubeadm.k8s.io/v1alpha2",
		},
	}

	for _, c := range cases {
		t.Run(c.TestName, func(t2 *testing.T) {
			cfg, err := NewConfig(c.initVersion, c.controlPlanes, c.workers, c.kubeDNS, false, c.externalEtcdIP)
			if err != nil {
				t.Errorf("NewConfig returned error: %v", err)
				return
			}

			var controlPlanes, workers int32
			for _, n := range cfg.Nodes {
				switch n.Role {
				case config.ControlPlaneRole:
					controlPlanes++
				case config.WorkerRole:
					workers++
				}
			}

			if c.controlPlanes != controlPlanes {
				t.Errorf("expected %d control-plane nodes, saw %d", c.controlPlanes, controlPlanes)
			}

			if c.workers != workers {
				t.Errorf("expected %d workers nodes, saw %d", c.workers, workers)
			}

			var kubeDNSPatch, externalEtcdPatch, calicoPatch bool
			for _, p := range cfg.KubeadmConfigPatches {
				if !strings.Contains(p, c.expectedPatchVersion) {
					t.Errorf("NewConfig does not have expected version %s: saw %s", c.expectedPatchVersion, p)
					return
				}

				if strings.Contains(p, "podSubnet: \"192.168.0.0/16\"") {
					calicoPatch = true
				}
				if strings.Contains(p, "etcd:") && strings.Contains(p, "external:") {
					externalEtcdPatch = true
				}
				if strings.Contains(p, "kube-dns") || strings.Contains(p, "CoreDNS: false") {
					kubeDNSPatch = true
				}
			}

			if !calicoPatch {
				t.Error("expected calico patch missing")
			}

			if c.kubeDNS && !kubeDNSPatch {
				t.Error("expected kube-dns patch missing")
			}

			if c.externalEtcdIP != "" && !externalEtcdPatch {
				t.Error("expected external-etcd patch missing")
			}
		})
	}
}
