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

package do

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/cluster/manager"
	"k8s.io/kubeadm/kinder/pkg/cluster/manager/actions"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

type flagpole struct {
	Name                  string
	UsePhases             bool
	UpgradeVersion        string
	CopyCerts             string
	Discovery             string
	OnlyNode              string
	DryRun                bool
	VLevel                int
	PatchesDir            string
	Wait                  time.Duration
	IgnorePreflightErrors string
	KubeadmConfigVersion  string
}

// NewCommand returns a new cobra.Command for exec
func NewCommand() *cobra.Command {
	flags := &flagpole{
		Discovery: string(actions.TokenDiscovery),
	}
	cmd := &cobra.Command{
		Args: cobra.ExactArgs(1),
		Use: "do [flags] ACTION\n\n" +
			"Args:\n" +
			fmt.Sprintf("  ACTION is one of %s", actions.KnownActions()),
		Short: "Executes actions (tasks/sequence of commands) on a cluster",
		Long: "Action define a set of tasks/sequence of commands to be executed on a cluster. Usage of actions allows \n" +
			"to automate repetitive operations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().StringVar(
		&flags.Name,
		"name", constants.DefaultClusterName, "cluster name",
	)
	cmd.Flags().StringVar(&flags.OnlyNode,
		"only-node",
		"", "exec the action only on the selected node",
	)
	cmd.Flags().BoolVar(
		&flags.DryRun,
		"dry-run", false,
		"only prints workflow commands, without executing them",
	)
	cmd.Flags().BoolVar(
		&flags.UsePhases, "use-phases",
		false, "use the kubeadm phases subcommands instead of the kubeadm top-level commands",
	)
	cmd.Flags().StringVar(
		&flags.UpgradeVersion,
		"upgrade-version", "",
		"defines the target upgrade version (it should match the version of upgrades binaries)",
	)
	cmd.Flags().StringVar(
		&flags.CopyCerts,
		"copy-certs", string(actions.CopyCertsModeManual),
		fmt.Sprintf("mode to copy certs when joining new control-plane nodes; use one of %s", actions.KnownCopyCertsMode()),
	)
	cmd.Flags().StringVar(
		&flags.Discovery,
		"discovery-mode", flags.Discovery,
		fmt.Sprintf("the discovery mode to be used for join; use one of %s", actions.KnownDiscoveryMode()),
	)
	cmd.Flags().DurationVar(
		&flags.Wait,
		"wait", time.Duration(5*time.Minute),
		"Wait for cluster state to converge after action",
	)
	cmd.Flags().IntVarP(
		&flags.VLevel,
		"kubeadm-verbosity", "v", 0,
		"Number for the log level verbosity for the kubeadm commands",
	)
	cmd.Flags().StringVar(
		&flags.PatchesDir,
		"patches", flags.PatchesDir,
		"the patches directory to be used for init, join and upgrade",
	)
	cmd.Flags().StringVar(
		&flags.IgnorePreflightErrors,
		"ignore-preflight-errors", constants.KubeadmIgnorePreflightErrors,
		"list of kubeadm preflight errors to skip",
	)
	cmd.Flags().StringVar(
		&flags.KubeadmConfigVersion,
		"kubeadm-config-version", flags.KubeadmConfigVersion,
		"the kubeadm config version to be used for init, join and upgrade. "+
			"If not set, kubeadm will automatically choose the kubeadm config version "+
			"according to the Kubernetes version in use",
	)
	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) (err error) {
	// validate UpgradeVersion flag
	var upgradeVersion *K8sVersion.Version
	if flags.UpgradeVersion != "" {
		upgradeVersion, err = K8sVersion.ParseSemantic(flags.UpgradeVersion)
		if err != nil {
			return err
		}
	}

	discovery := actions.DiscoveryMode(strings.ToLower(flags.Discovery))
	if err := actions.ValidateDiscoveryMode(discovery); err != nil {
		return err
	}

	copyCerts := actions.CopyCertsMode(strings.ToLower(flags.CopyCerts))
	if err := actions.ValidateCopyCertsMode(copyCerts); err != nil {
		return err
	}

	// get a kinder cluster manager
	o, err := manager.NewClusterManager(flags.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to create a kinder cluster manager for %s", flags.Name)
	}

	// eventually, instruct the cluster manager to run only commands on one node
	if flags.OnlyNode != "" {
		o.OnlyNode(flags.OnlyNode)
	}

	// eventually, instruct the cluster manager to dry run commands (without actually running them)
	if flags.DryRun {
		o.DryRun()

		flags.Wait = 0
	}

	// executed the requested action
	action := args[0]
	err = o.DoAction(action,
		actions.UsePhases(flags.UsePhases),
		actions.CopyCerts(copyCerts),
		actions.Discovery(discovery),
		actions.Wait(flags.Wait),
		actions.UpgradeVersion(upgradeVersion),
		actions.VLevel(flags.VLevel),
		actions.PatchesDir(flags.PatchesDir),
		actions.IgnorePreflightErrors(flags.IgnorePreflightErrors),
		actions.KubeadmConfigVersion(flags.KubeadmConfigVersion),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to exec action %s", action)
	}

	return nil
}
