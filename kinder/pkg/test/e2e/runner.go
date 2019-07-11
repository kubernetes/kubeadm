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
Package e2e implements support for running kubeadm e2e tests or kubernetes e2e test.

It takes care of building upstream test suites if necessary, and provides "sane" defaults
for simplifying test invokation.
*/
package e2e

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// Option is an Runner configuration option supplied to NewRunner
type Option func(*Runner)

// KubeRoot option sets the kubernetes checkout folder
func KubeRoot(kubeRoot string) Option {
	return func(r *Runner) {
		r.kubeRoot = kubeRoot
	}
}

// WithGinkgoFlags option sets flags for ginkgo
func WithGinkgoFlags(ginkgoFlags GinkgoFlags) Option {
	return func(r *Runner) {
		r.ginkgoFlags = ginkgoFlags
	}
}

// WithSuiteFlags option sets flags for the test program
func WithSuiteFlags(suiteFlags SuiteFlags) Option {
	return func(r *Runner) {
		r.suiteFlags = suiteFlags
	}
}

// Runner defines attributes for a Kubernetes artifact extractor
type Runner struct {
	testBinary     string
	makeBinaryGoal string
	kubeRoot       string
	ginkgoFlags    GinkgoFlags
	suiteFlags     SuiteFlags
}

// NewKubernetesTestRunner returns a new E2E (Kubernetes) test runner configured with the given options
func NewKubernetesTestRunner(options ...Option) (runner *Runner, err error) {
	return newTestRunner("e2e.test", "test/e2e/e2e.test", options...)
}

// NewKubeadmTestRunner returns a new E2E kubeadm test runner configured with the given options
func NewKubeadmTestRunner(options ...Option) (runner *Runner, err error) {
	return newTestRunner("e2e_kubeadm.test", "test/e2e_kubeadm/e2e_kubeadm.test", options...)
}

// newTestRunner returns a new test runner - using ginkgo and the given test binary - configured with the given options
func newTestRunner(testBinary, makeBinaryGoal string, options ...Option) (runner *Runner, err error) {
	runner = &Runner{
		testBinary:     testBinary,
		makeBinaryGoal: makeBinaryGoal,
	}

	// apply user options
	for _, option := range options {
		option(runner)
	}

	// sets kubeRoot if not provided by the user
	if runner.kubeRoot == "" {
		runner.kubeRoot, err = findKubeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		log.Infof("using Kubernetes checkout in %s folder", runner.kubeRoot)
	}

	return runner, nil
}

// Run executes tests as defined by the selected runner options.
// it takes care of building ginkgo and upstream test suites if necessary,
func (r *Runner) Run() error {
	// find a ginkgo binary or build it if it not exists
	ginkgoBinary, err := getOrBuildBinary(r.kubeRoot, "ginkgo", "vendor/github.com/onsi/ginkgo/ginkgo")
	if err != nil {
		return err
	}

	// find the binary with the test suites to be executes or build it if it not exists
	testBinary, err := getOrBuildBinary(r.kubeRoot, r.testBinary, r.makeBinaryGoal)
	if err != nil {
		return err
	}

	// prepare args to be passed to ginkgo test runner:
	// ginkgo [ginkgo-flags] [test-suite-binary] -- [test-suite-flags]
	var args []string
	for k, v := range r.ginkgoFlags {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}
	args = append(args, testBinary, "--")
	for k, v := range r.suiteFlags {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	log.Debugf("invoking ginkgo with followiong args: %s", args)

	// executes the command.
	// TODO: switch to an executor that supports timeout/cancellation
	cmd := kindexec.Command(ginkgoBinary, args...)
	kindexec.InheritOutput(cmd)
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "error running test")
	}

	fmt.Println(args)

	return nil
}
