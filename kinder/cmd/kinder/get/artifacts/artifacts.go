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

package artifacts

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

const (
	onlyKubeadmFlagName  = "only-kubeadm"
	onlyKubeletFlagName  = "only-kubelet"
	onlyBinariesFlagName = "only-binaries"
	onlyImagesFLagName   = "only-images"
)

type flagpole struct {
	OnlyKubeadm  bool
	OnlyKubelet  bool
	OnlyBinaries bool
	OnlyImages   bool
}

// NewCommand returns a new cobra.Command for exec
func NewCommand() *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args: cobra.RangeArgs(1, 2),
		Use: "artifacts [flags] KUBERNETES_VERSION [DESTINATION_PATH]\n\n" +
			"Args:\n" +
			"  KUBERNETES_VERSION is one of:\n" +
			"    release/LABEL    where label can be stable[-major[.minor]] or latest[-major[.minor]]\n" +
			"    ci/LABEL         where label can be latest[-major[.minor]]\n" +
			"    release/VERSION  where VERSION is a semantic version\n" +
			"    ci/VERSION       where VERSION is a semantic version\n" +
			"    VERSION          as shortcut to release/VERSION if build metadata are empty, else to ci/VERSION\n" +
			"    URL              an http or http server where release artifacts are available\n" +
			"    PATH             a local folder (file:// schema can be use to disambiguate release/ or ci/ folder)\n" +
			"  DESTINATION_PATH should be a local path; if missing the current path will be used",
		Aliases: []string{"build-artifacts", "release-artifacts", "ci-artifacts"},
		Short:   "Gets ci/release artifacts for a given Kubernetes version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(flags, cmd, args)
		},
	}

	cmd.Flags().BoolVar(
		&flags.OnlyKubeadm,
		onlyKubeadmFlagName, false,
		"Gets only the kubeadm binary (instead of all artifacts)",
	)
	cmd.Flags().BoolVar(&flags.OnlyKubelet,
		onlyKubeletFlagName, false,
		"Gets only the kubelet binary (instead of all artifacts)",
	)
	cmd.Flags().BoolVar(&flags.OnlyBinaries,
		onlyBinariesFlagName, false,
		"Gets only the kubeadm, kubelet, kubectl binaries (instead of all artifacts)",
	)
	cmd.Flags().BoolVar(&flags.OnlyImages,
		onlyImagesFLagName, false,
		"Gets only the kube-apiserver, kube-scheduler, kube-controller-manager and kube-proxy image tarballs (instead of all artifacts)",
	)

	return cmd
}

func runE(flags *flagpole, cmd *cobra.Command, args []string) error {
	//checks that mutually exclusive flags are not set at the same time
	exclusiveFlags := []string{onlyKubeadmFlagName, onlyKubeletFlagName, onlyBinariesFlagName, onlyImagesFLagName}
	if !checkExclusiveFlags(cmd.Flags(), exclusiveFlags) {
		return errors.Errorf("flags [%s] are mutually exclusive, please set only one of them", strings.Join(exclusiveFlags, ", "))
	}

	// retrive src and dst from arguments
	src := args[0]
	dst := ""
	if len(args) > 1 {
		dst = args[1]
	}

	// Build an artifact extractor customized with the command options
	e := extract.NewExtractor(src, dst,
		extract.OnlyKubeadm(flags.OnlyKubeadm),
		extract.OnlyKubelet(flags.OnlyKubelet),
		extract.OnlyKubernetesBinaries(flags.OnlyBinaries),
		extract.OnlyKubernetesImages(flags.OnlyImages),
	)

	// Extracts the artifacts from the source
	_, err := e.Extract()
	if err != nil {
		return errors.Wrapf(err, "failed to gets build artifacts for %s version", src)
	}

	return nil
}

func checkExclusiveFlags(flags *flag.FlagSet, exclusiveFlags []string) bool {
	n := 0
	for _, f := range exclusiveFlags {
		if flags.Changed(f) {
			n++
		}
	}
	return n <= 1
}
