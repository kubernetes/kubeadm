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

// Package test implements the `test` command
package test

import (
	"github.com/spf13/cobra"

	ke2e "k8s.io/kubeadm/kinder/cmd/kinder/test/e2e"
	ke2ekubeadm "k8s.io/kubeadm/kinder/cmd/kinder/test/e2ekubeadm"
)

// NewCommand returns a new cobra.Command for running E2E tests on cluster
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "test",
		Short: "Runs e2e test or e2e-kubeadm on a Kubernetes cluster",
		Long:  "Runs e2e test or e2e-kubeadm on a Kubernetes cluster",
	}
	cmd.AddCommand(ke2e.NewCommand())
	cmd.AddCommand(ke2ekubeadm.NewCommand())
	return cmd
}
