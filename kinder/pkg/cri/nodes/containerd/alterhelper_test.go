/*
Copyright 2025 The Kubernetes Authors.

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

package containerd

import (
	"testing"
)

func TestParseImageArchSuffix(t *testing.T) {
	cases := []struct {
		input string
		base  string
		arch  string
		tag   string
	}{
		{"registry.k8s.io/kube-apiserver-arm64:v1.34", "registry.k8s.io/kube-apiserver", "arm64", "v1.34"},
		{"registry.k8s.io/kube-apiserver-amd64:v1.34", "registry.k8s.io/kube-apiserver", "amd64", "v1.34"},
		{"registry.k8s.io/kube-apiserver:v1.34", "registry.k8s.io/kube-apiserver", "", "v1.34"},
		{"registry.k8s.io/kube-apiserver:dev", "registry.k8s.io/kube-apiserver", "", "dev"},
	}
	for _, c := range cases {
		base, arch, tag := parseImageArchSuffix(c.input)
		if base != c.base || arch != c.arch || tag != c.tag {
			t.Errorf("parseImageArchSuffix(%q) = (%q, %q, %q), want (%q, %q, %q)", c.input, base, arch, tag, c.base, c.arch, c.tag)
		}
	}
}
