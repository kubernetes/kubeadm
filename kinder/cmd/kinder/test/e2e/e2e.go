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

// Package e2e implements the `e2e` command
package e2e

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	ktest "k8s.io/kubeadm/kinder/pkg/test"
)

type flagpole struct {
	KubeRoot            string
	Parallel            bool
	TestGridConformance bool
	GinkgoFlags         string
	TestFlags           string
}

// NewCommand returns a new cobra.Command for e2e
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "e2e",
		Short: "Runs Kubernetes e2e tests",
		// TODO: add a long description
		// TODO: adde examples with flags usage and report-dir
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().StringVar(&flags.KubeRoot, "kube-root", "", "Path to the Kubernetes source directory (if empty, the path is autodetected)")
	cmd.Flags().BoolVar(&flags.TestGridConformance, "conformance", true, "if set, instruct ginkgo for running only tests required for conformance in testgrid")
	cmd.Flags().BoolVar(&flags.Parallel, "parallel", false, "if set, instruct ginkgo for running tests in parallel")
	cmd.Flags().StringVar(&flags.GinkgoFlags, "ginkgo-flags", "", "Space-separated list of arguments to pass to ginkgo test runner")
	cmd.Flags().StringVar(&flags.TestFlags, "test-flags", "", "Space-separated list of arguments to pass to e2e_kubeadm test")
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// Create a map with the flag/values to pass to the ginkgo test runner
	ginkgoFlags, err := ktest.NewGinkgoFlags(flags.GinkgoFlags)
	if err != nil {
		return err
	}

	// if --conformance is set, adds well know flag/values for instructing ginkgo for running only tests required for conformance in testgrid
	if flags.TestGridConformance {
		// instruct ginkgo to run only Conformance test as defined in [Display Conformance Tests with Testgrid]
		// (https://github.com/kubernetes/test-infra/tree/master/testgrid/conformance)

		// see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/e2e-tests.md
		// for description of test labels
		ginkgoFlags.AddFocusRegex(regexp.QuoteMeta("[Conformance]"))
		ginkgoFlags.AddSkipRegex("Alpha|Kubectl|\\[(Disruptive|Feature:[^\\]]+|Flaky)\\]")
	}

	// if --parallel is set, adds well know flag/values for instructing ginkgo for running tests in parallel
	if flags.Parallel {
		// please note that this spin-up a default number of test runners (runtime.NumCPU() if runtime.NumCPU() <= 4, otherwise it is runtime.NumCPU() - 1);
		// if you want to control the level of parallelism, you can use --ginkgo-flags "--nodes=25"
		// see https://onsi.github.io/ginkgo/#parallel-specs for more info
		ginkgoFlags["p"] = "true"
		ginkgoFlags.AddSkipRegex(regexp.QuoteMeta("[Serial]"))
	}

	// Create a map with the flag/values to pass to the e2e_kubeadm.test binary
	testFlags, err := ktest.NewSuiteFlags(flags.TestFlags)
	if err != nil {
		return err
	}

	// instruct e2e.test to do not ssh into nodes and dump logs
	testFlags["disable-log-dump"] = "true"

	// creates a NewKubernetesTestRunner with the desired options and run it
	testRunner, err := ktest.NewKubernetesTestRunner(
		ktest.KubeRoot(flags.KubeRoot),
		ktest.WithGinkgoFlags(ginkgoFlags),
		ktest.WithSuiteFlags(testFlags),
	)
	if err != nil {
		return errors.Wrapf(err, "failed create test runner")
	}
	return testRunner.Run()
}
