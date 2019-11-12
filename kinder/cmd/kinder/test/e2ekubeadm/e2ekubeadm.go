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

package e2ekubeadm

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubeadm/kinder/pkg/cluster/status"

	"k8s.io/kubeadm/kinder/pkg/constants"
	"k8s.io/kubeadm/kinder/pkg/test/e2e"
)

type flagpole struct {
	KubeRoot    string
	SingleNode  bool
	CopyCerts   bool
	GinkgoFlags string
	TestFlags   string
	Name        string
	kubeconfig  string
}

// NewCommand returns a new cobra.Command for e2e-kubeadm
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:   "e2e-kubeadm",
		Short: "Runs kubeadm e2e tests",
		// TODO: add a long description
		// TODO: adde examples with flags usage and report-dir
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().StringVar(
		&flags.KubeRoot,
		"kube-root", "", "Path to the Kubernetes source directory (if empty, the path is autodetected)",
	)
	cmd.Flags().BoolVar(
		&flags.SingleNode,
		"single-node", false,
		"if set, skips tests labeled with [multi-node]",
	)
	cmd.Flags().BoolVar(
		&flags.CopyCerts,
		"automatic-copy-certs", false,
		"if set, adds tests labeled with [copy-cert] for validating alpha feature automatic-copy-certs",
	)
	cmd.Flags().StringVar(&flags.GinkgoFlags,
		"ginkgo-flags", "",
		"Space-separated list of arguments to pass to Ginkgo test runner",
	)
	cmd.Flags().StringVar(
		&flags.TestFlags,
		"test-flags", "",
		"Space-separated list of arguments to pass to node e2e test",
	)

	cmd.Flags().StringVar(
		&flags.Name,
		"name", constants.DefaultClusterName,
		"cluster name",
	)
	cmd.Flags().StringVar(
		&flags.kubeconfig,
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

	// if --conformance is set, adds well know flag/values for instructing ginkgo to skip tests labeled with [multi-node]
	if flags.SingleNode {
		ginkgoFlags.AddSkipRegex(regexp.QuoteMeta("[multi-node]"))
	}

	// if --automatic-copy-certs is set, adds well know flag/values for instructing ginkgo to skip tests labeled with [copy-cert]
	if !flags.CopyCerts {
		ginkgoFlags.AddSkipRegex(regexp.QuoteMeta("[copy-certs]"))
	}

	// Create a map with the flag/values to pass to e2e-kubeadm.test
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
	// instruct e2e-kubeadm.test to use the kubeconfig file (if not already set into test-flags)
	if _, ok := testFlags["kubeconfig"]; !ok {
		testFlags["kubeconfig"] = flags.kubeconfig
	}

	// creates a KubeadmTestRunner with the desired options and run it
	testRunner, err := e2e.NewKubeadmTestRunner(
		e2e.KubeRoot(flags.KubeRoot),
		e2e.WithGinkgoFlags(ginkgoFlags),
		e2e.WithSuiteFlags(testFlags),
	)
	if err != nil {
		return errors.Wrapf(err, "failed create test runner")
	}
	return testRunner.Run()
}
