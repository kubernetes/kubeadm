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

// Package kinder implements the root kinder cobra command, and the cli Main()
package kinder

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/kubeadm/kinder/cmd/kinder/build"
	"k8s.io/kubeadm/kinder/cmd/kinder/cp"
	"k8s.io/kubeadm/kinder/cmd/kinder/create"
	"k8s.io/kubeadm/kinder/cmd/kinder/do"
	"k8s.io/kubeadm/kinder/cmd/kinder/exec"
	"k8s.io/kubeadm/kinder/cmd/kinder/get"
	"k8s.io/kubeadm/kinder/cmd/kinder/test"
	"k8s.io/kubeadm/kinder/cmd/kinder/version"
	"k8s.io/kubeadm/kinder/pkg/constants"
	kinddelete "sigs.k8s.io/kind/cmd/kind/delete"
	kindexport "sigs.k8s.io/kind/cmd/kind/export"
	kindlog "sigs.k8s.io/kind/pkg/log"
)

const defaultLevel = log.WarnLevel

// Flags for the kinder command
type Flags struct {
	LogLevel string
}

// NewCommand returns a new cobra.Command implementing the root command for kinder
func NewCommand() *cobra.Command {
	flags := &Flags{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "kinder",
		Short: "kinder is an example of kind(https://github.com/kubernetes-sigs/kind) used as a library",
		Long: "   _  ___         _\n" +
			"  | |/ (_)_ _  __| |___ _ _\n" +
			"  | ' <| | ' \\/ _` / -_) '_|\n" +
			"  |_|\\_\\_|_||_\\__,_\\___|_|\n\n" +
			"kinder is an example of kind(https://github.com/kubernetes-sigs/kind) used as a library.\n\n" +
			"All the kind commands will be available in kinder, side by side with additional commands \n" +
			"designed for helping kubeadm contributors.\n\n" +
			"kinder is still a work in progress. Test It, Break It, Send feedback!",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
		SilenceUsage: true,
		Version:      constants.KinderVersion,
	}
	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		defaultLevel.String(),
		"logrus log level "+kindlog.LevelsString(),
	)

	// add kind top level subcommands re-used without changes
	cmd.AddCommand(kinddelete.NewCommand())
	cmd.AddCommand(kindexport.NewCommand())

	// add kind commands commands customized in kind
	cmd.AddCommand(build.NewCommand())
	cmd.AddCommand(create.NewCommand())
	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(get.NewCommand())

	// add kinder only commands
	cmd.AddCommand(cp.NewCommand())
	cmd.AddCommand(do.NewCommand())
	cmd.AddCommand(exec.NewCommand())
	cmd.AddCommand(test.NewCommand())

	return cmd
}

func runE(flags *Flags, cmd *cobra.Command, args []string) error {
	level := defaultLevel
	parsed, err := log.ParseLevel(flags.LogLevel)
	if err != nil {
		log.Warnf("Invalid log level '%s', defaulting to '%s'", flags.LogLevel, level)
	} else {
		level = parsed
	}
	log.SetLevel(level)
	return nil
}

// Run runs the `kind` root command
func Run() error {
	return NewCommand().Execute()
}

// Main wraps Run and sets the log formatter
func Main() {
	// let's explicitly set stdout
	log.SetOutput(os.Stdout)
	// this formatter is the default, but the timestamps output aren't
	// particularly useful, they're relative to the command start
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		// we force colors because this only forces over the isTerminal check
		// and this will not be accurately checkable later on when we wrap
		// the logger output with our logutil.StatusFriendlyWriter
		ForceColors: kindlog.IsTerminal(log.StandardLogger().Out),
	})
	if err := Run(); err != nil {
		os.Exit(1)
	}
}
