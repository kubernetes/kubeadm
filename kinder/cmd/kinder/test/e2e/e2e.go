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

package e2e

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"

	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/test/e2e"
)

type flagpole struct {
	KubeRoot            string
	Parallel            bool
	TestGridConformance bool
	GinkgoFlags         string
	TestFlags           string
	Name                string
	kubeconfig          string
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

	cmd.Flags().StringVar(
		&flags.KubeRoot,
		"kube-root", "",
		"Path to the Kubernetes source directory (if empty, the path is autodetected)",
	)
	cmd.Flags().BoolVar(
		&flags.TestGridConformance,
		"conformance", true,
		"if set, instruct ginkgo for running only tests required for conformance in testgrid",
	)
	cmd.Flags().BoolVar(&flags.Parallel,
		"parallel", false,
		"if set, instruct ginkgo for running tests in parallel",
	)
	cmd.Flags().StringVar(&flags.GinkgoFlags,
		"ginkgo-flags", "",
		"Space-separated list of arguments to pass to ginkgo test runner",
	)
	cmd.Flags().StringVar(&flags.TestFlags,
		"test-flags", "",
		"Space-separated list of arguments to pass to e2e_kubeadm test",
	)

	cmd.Flags().StringVar(&flags.Name,
		"name", constants.DefaultClusterName,
		"cluster name",
	)
	cmd.Flags().StringVar(&flags.kubeconfig,
		"kubeconfig", "",
		"The kubeconfig file to use when talking to the cluster. If the flag is not set, this value will be set to the location of the kubeconfig for the kind cluster pointed by name",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	// Create a map with the flag/values to pass to the ginkgo test runner
	ginkgoFlags, err := e2e.NewGinkgoFlags(flags.GinkgoFlags)
	if err != nil {
		return err
	}

	// if --conformance is set, adds well know flag/values for instructing ginkgo for running only tests required for conformance in testgrid
	if flags.TestGridConformance {
		// instruct ginkgo to run only Conformance test as defined in [Display Conformance Tests with Testgrid]
		// (https://git.k8s.io/test-infra/testgrid/conformance)

		// see https://git.k8s.io/community/contributors/devel/sig-testing/e2e-tests.md
		// for description of test labels
		ginkgoFlags.AddFocusRegex(regexp.QuoteMeta("[Conformance]"))
		ginkgoFlags.AddSkipRegex("Aggregator|Alpha|Kubectl|\\[(Disruptive|Feature:[^\\]]+|Flaky)\\]")
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
	testFlags, err := e2e.NewSuiteFlags(flags.TestFlags)
	if err != nil {
		return err
	}

	// if not explicitly set, gets the kubeconfig file for the selected kind cluster
	if flags.kubeconfig == "" {
		// Check if the cluster name already exists
		known, err := status.IsKnown(flags.Name)
		if err != nil {
			return err
		}
		if !known {
			return errors.Errorf("a cluster with the name %q does not exists", flags.Name)
		}

		// Gets the kubeconfig file for the cluster name
		flags.kubeconfig = status.KubeConfigPath(flags.Name)
	}
	// instruct e2e.test to use the kubeconfig file (if not already set into test-flags)
	if _, ok := testFlags["kubeconfig"]; !ok {
		testFlags["kubeconfig"] = flags.kubeconfig
	}

	// instruct e2e.test to do not ssh into nodes and dump logs (if not already set into test-flags)
	if _, ok := testFlags["disable-log-dump"]; !ok {
		testFlags["disable-log-dump"] = "true"
	}

	// creates a NewKubernetesTestRunner with the desired options and run it
	testRunner, err := e2e.NewKubernetesTestRunner(
		e2e.KubeRoot(flags.KubeRoot),
		e2e.WithGinkgoFlags(ginkgoFlags),
		e2e.WithSuiteFlags(testFlags),
	)
	if err != nil {
		return errors.Wrapf(err, "failed create test runner")
	}
	return testRunner.Run()
}
