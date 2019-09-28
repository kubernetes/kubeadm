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

package workflow

import (
	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/pkg/test/workflow"
)

type flagpole struct {
	DryRun      bool
	Verbose     bool
	ExitOnError bool
}

// NewCommand returns a new cobra.Command for e2e-kubeadm
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use: "workflow [flags] CONFIG [ARTIFACTS]\n\n" +
			"Args:\n" +
			"  CONFIG is the path of a workflow config file\n" +
			"  ARTIFACTS is the path to the directory where to store ARTIFACTS\n",
		Short: "Runs test workflow",
		Args:  cobra.RangeArgs(1, 2),
		// TODO: add a long description
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().BoolVar(
		&flags.DryRun,
		"dry-run", false,
		"only prints workflow commands, without executing them",
	)
	cmd.Flags().BoolVar(
		&flags.Verbose,
		"verbose", false,
		"redirect command output to stdout",
	)
	cmd.Flags().BoolVar(
		&flags.ExitOnError,
		"exit-on-task-error", false,
		"exit after first task failed",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {

	// retrieve config and artifacts from arguments
	config := args[0]
	artifacts := ""
	if len(args) > 1 {
		artifacts = args[1]
	}

	w, err := workflow.NewWorkflow(config)
	if err != nil {
		return err
	}

	return w.Run(flags.DryRun, flags.Verbose, flags.ExitOnError, artifacts)
}
