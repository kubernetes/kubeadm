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

package actions

import (
	"fmt"

	"github.com/pkg/errors"
	kcluster "k8s.io/kubeadm/kinder/pkg/cluster"
)

// joinAction implements a developer friendly kubeadm join workflow
type joinAction struct{}

func init() {
	kcluster.RegisterAction("kubeadm-join", newJoinAction)
}

func newJoinAction() kcluster.Action {
	return &joinAction{}
}

// Tasks returns the list of action tasks for the joinAction
func (b *joinAction) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Joining control-plane node to Kubernetes ☸",
			TargetNodes: "@cpN",
			Run: func(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
				switch flags.UsePhases {
				case true:
					return runJoinControlPlanePhases(kctx, kn, flags)
				default:
					return runJoinControlPlane(kctx, kn, flags)
				}
			},
		},
		{
			Description: "Joining worker node to Kubernetes ☸",
			TargetNodes: "@w*",
			Run: func(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
				switch flags.UsePhases {
				case true:
					return runJoinWorkersPhases(kctx, kn, flags)
				default:
					return runJoinWorkers(kctx, kn, flags)
				}
			},
		},
	}
}

func runJoinWorkers(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	// get the join address
	joinAddress, err := getJoinAddress(kctx)
	if err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm join worker 🚀",
		"kubeadm", "join", joinAddress, "--token", Token, "--discovery-token-unsafe-skip-ca-verification", "--ignore-preflight-errors=all",
	); err != nil {
		return err
	}

	return nil
}

func runJoinWorkersPhases(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	// join phases are supported starting from v1.14
	if err := atLeastKubeadm(kn, "v1.14.0-0"); err != nil {
		return errors.Wrapf(err, "join phases can't be used")
	}

	// get the join address
	joinAddress, err := getJoinAddress(kctx)
	if err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm join phase preflight 🚀",
		"kubeadm", "join", "phase", "preflight", joinAddress, "--token", Token, "--discovery-token-unsafe-skip-ca-verification", "--ignore-preflight-errors=all",
	); err != nil {
		return err
	}

	// NB. Test control-plane-prepare does not execute actions when joining a worker node
	//if err := kn.DebugCmd(
	//	"==> kubeadm join phase control-plane-prepare 🚀",
	//	"kubeadm", "join", "phase", "control-plane-prepare", "all", joinAddress, "--discovery-token", Token, "--discovery-token-unsafe-skip-ca-verification",
	//); err != nil {
	//	return err
	//}

	if err := kn.DebugCmd(
		"==> kubeadm join phase kubelet-start 🚀",
		"kubeadm", "join", "phase", "kubelet-start", joinAddress, "--discovery-token", Token, "--discovery-token-unsafe-skip-ca-verification",
	); err != nil {
		return err
	}

	// NB. Test control-plane-join does not execute actions when joining a worker node
	//if err := kn.DebugCmd(
	//	"==> kubeadm join phase control-plane-join all 🚀",
	//	"kubeadm", "join", "phase", "control-plane-join", "all",
	//); err != nil {
	//	return err
	//}

	return nil
}

func runJoinControlPlane(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	// automatic copy certs is supported starting from v1.14
	if flags.CopyCerts {
		if err := atLeastKubeadm(kn, "v1.14.0-0"); err != nil {
			return errors.Wrapf(err, "--automatic-copy-certs can't be used")
		}
	}

	// if not automatic copy certs, simulate manual copy
	if !flags.CopyCerts {
		err := doManualCopyCerts(kctx, kn)
		if err != nil {
			return err
		}
	}

	// get the join address
	joinAddress, err := getJoinAddress(kctx)
	if err != nil {
		return err
	}

	joinArgs := []string{
		"join", joinAddress, "--experimental-control-plane", "--token", Token, "--discovery-token-unsafe-skip-ca-verification", "--ignore-preflight-errors=all",
	}
	if flags.CopyCerts {
		joinArgs = append(joinArgs,
			fmt.Sprintf("--certificate-key=%s", CertificateKey),
		)
	}
	if err := kn.DebugCmd(
		"==> kubeadm join control plane 🚀",
		"kubeadm", joinArgs...,
	); err != nil {
		return err
	}

	return nil
}

func runJoinControlPlanePhases(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	// join phases are supported starting from v1.14
	if err := atLeastKubeadm(kn, "v1.14.0-0"); err != nil {
		return errors.Wrapf(err, "join phases can't be used")
	}

	// if not automatic copy certs, simulate manual copy
	if !flags.CopyCerts {
		err := doManualCopyCerts(kctx, kn)
		if err != nil {
			return err
		}
	}

	// get the join address
	joinAddress, err := getJoinAddress(kctx)
	if err != nil {
		return err
	}

	preflightArgs := []string{
		"join", "phase", "preflight", joinAddress, "--experimental-control-plane", "--token", Token, "--discovery-token-unsafe-skip-ca-verification", "--ignore-preflight-errors=all",
	}
	if flags.CopyCerts {
		preflightArgs = append(preflightArgs,
			fmt.Sprintf("--certificate-key=%s", CertificateKey),
		)
	}
	if err := kn.DebugCmd(
		"==> kubeadm join phase preflight 🚀",
		"kubeadm", preflightArgs...,
	); err != nil {
		return err
	}

	prepareArgs := []string{
		"join", "phase", "control-plane-prepare", "all", joinAddress, "--experimental-control-plane", "--discovery-token", Token, "--discovery-token-unsafe-skip-ca-verification",
	}
	if flags.CopyCerts {
		prepareArgs = append(prepareArgs,
			fmt.Sprintf("--certificate-key=%s", CertificateKey),
		)
	}
	if err := kn.DebugCmd(
		"==> kubeadm join phase control-plane-prepare 🚀",
		"kubeadm", prepareArgs...,
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm join phase kubelet-start 🚀",
		"kubeadm", "join", "phase", "kubelet-start", joinAddress, "--discovery-token", Token, "--discovery-token-unsafe-skip-ca-verification",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm join phase control-plane-join all 🚀",
		"kubeadm", "join", "phase", "control-plane-join", "all", "--experimental-control-plane",
	); err != nil {
		return err
	}

	return nil
}
