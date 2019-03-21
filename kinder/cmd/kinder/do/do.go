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

// Package do implements the `do` command
package do

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/version"
	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster"
)

type flagpole struct {
	Name           string
	OnlyNode       string
	UsePhases      bool
	UpgradeVersion string
	CopyCerts      bool
}

// NewCommand returns a new cobra.Command for exec
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	actions := kcluster.KnownActions()
	cmd := &cobra.Command{
		Args: cobra.MinimumNArgs(1),
		Use: "do [flags] ACTION\n\n" +
			"Args:\n" +
			fmt.Sprintf("  ACTION is one of [%s]", strings.Join(actions, ", ")),
		Short: "Executes actions (tasks/sequence of commands) on one or more nodes in the local Kubernetes cluster",
		Long: "Action define a set of tasks/sequence of commands to be executed on a cluster. Usage of actions allows \n" +
			"automate repetitive operatitions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().StringVar(&flags.Name, "name", cluster.DefaultName, "cluster context name")
	cmd.Flags().StringVar(&flags.OnlyNode, "only-node", "", "exec the action only on the selected node")
	cmd.Flags().BoolVar(&flags.UsePhases, "use-phases", false, "use the kubeadm phases subcommands insted of the the kubeadm top-level commands")
	cmd.Flags().StringVar(&flags.UpgradeVersion, "upgrade-version", "", "defines the target upgrade version (it should match the version of upgrades binaries)")
	cmd.Flags().BoolVar(&flags.CopyCerts, "automatic-copy-certs", false, "use automatic copy certs instead of manual copy certs when joining new control-plane nodes")

	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	actionFlags := kcluster.ActionFlags{
		UsePhases: flags.UsePhases,
		CopyCerts: flags.CopyCerts,
	}

	//TODO: upgrade version mandatory for updates

	if flags.UpgradeVersion != "" {
		v, err := version.ParseSemantic(flags.UpgradeVersion)
		if err != nil {
			return err
		}
		actionFlags.UpgradeVersion = v
	}

	// Check if the cluster name already exists
	known, err := cluster.IsKnown(flags.Name)
	if err != nil {
		return err
	}
	if !known {
		return errors.Errorf("a cluster with the name %q does not exists", flags.Name)
	}

	// create a cluster context from current nodes
	ctx := cluster.NewContext(flags.Name)

	kcfg, err := kcluster.NewKContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster context")
	}

	err = kcfg.Do(args, actionFlags, flags.OnlyNode)
	if err != nil {
		return errors.Wrap(err, "failed to exec action on cluster nodes")
	}

	return nil
}
