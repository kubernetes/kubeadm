/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"flag"
	"fmt"
	"os"

	versionutil "k8s.io/apimachinery/pkg/util/version"

	"k8s.io/kubeadm/kinder/ci/tools/update-workflows/pkg"
)

type versionValue struct {
	Version *versionutil.Version
}

func (v versionValue) String() string {
	if v.Version == nil {
		return ""
	}
	return v.Version.String()
}

func (v versionValue) Set(s string) error {
	newVer, err := versionutil.ParseGeneric(s)
	if err != nil {
		return err
	}
	*v.Version = *newVer
	return nil
}

func main() {
	// prepare flags
	settings := &pkg.Settings{}
	ver := versionValue{&versionutil.Version{}}
	flag.Var(&ver, "kubernetes-version", "Kubernetes version (e.g. v1.21.0)")
	flag.StringVar(&settings.PathConfig, "config", "", "config file")
	flag.StringVar(&settings.PathTestInfra, "path-test-infra", "", "path to the directory with test-infra kubeadm jobs")
	flag.StringVar(&settings.PathWorkflows, "path-workflows", "", "path to the directory with kinder workflows")
	flag.StringVar(&settings.ImageTestInfra, "image-test-infra", "", "image tag to use for test-infra jobs")
	flag.IntVar(&settings.SkewSize, "skew-size", 3, "number of prior Kubernetes minor version to be tested starting from --kubernetes-version (included)")
	flag.Parse()

	// check for flags with empty values
	flag.VisitAll(func(f *flag.Flag) {
		if len(f.Value.String()) != 0 {
			return
		}
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nerror: the flag --%s is required\n", f.Name)
		os.Exit(1)
	})

	// run
	settings.KubernetesVersion = ver.Version
	if err := pkg.Run(settings); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
