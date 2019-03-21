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

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	logutil "sigs.k8s.io/kind/pkg/log"

	kbuild "k8s.io/kubeadm/kinder/cmd/kinder/build"
	kcp "k8s.io/kubeadm/kinder/cmd/kinder/cp"
	kcreate "k8s.io/kubeadm/kinder/cmd/kinder/create"
	kdo "k8s.io/kubeadm/kinder/cmd/kinder/do"
	kexec "k8s.io/kubeadm/kinder/cmd/kinder/exec"
	kversion "k8s.io/kubeadm/kinder/cmd/kinder/version"
	"sigs.k8s.io/kind/cmd/kind/delete"
	"sigs.k8s.io/kind/cmd/kind/export"
	"sigs.k8s.io/kind/cmd/kind/get"
	"sigs.k8s.io/kind/cmd/kind/load"
)

const defaultLevel = logrus.WarnLevel

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
		Version:      kversion.Version,
	}
	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		defaultLevel.String(),
		"logrus log level "+logutil.LevelsString(),
	)

	// add kind top level subcommands re-used without changes
	cmd.AddCommand(delete.NewCommand())
	cmd.AddCommand(export.NewCommand())
	cmd.AddCommand(get.NewCommand())
	cmd.AddCommand(load.NewCommand())

	// add kind commands commands customized in kind
	cmd.AddCommand(kbuild.NewCommand())
	cmd.AddCommand(kcreate.NewCommand())
	cmd.AddCommand(kversion.NewCommand())

	// add kinder only commands
	cmd.AddCommand(kcp.NewCommand())
	cmd.AddCommand(kdo.NewCommand())
	cmd.AddCommand(kexec.NewCommand())

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
		ForceColors: logutil.IsTerminal(log.StandardLogger().Out),
	})
	if err := Run(); err != nil {
		os.Exit(1)
	}
}
