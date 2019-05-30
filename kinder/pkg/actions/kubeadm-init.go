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

// initAction implements a developer friendly kubeadm init workflow
type initAction struct{}

func init() {
	kcluster.RegisterAction("kubeadm-init", newInitAction)
}

func newInitAction() kcluster.Action {
	return &initAction{}
}

// Tasks returns the list of action tasks for the initAction
func (b *initAction) Tasks() []kcluster.Task {
	return []kcluster.Task{
		{
			Description: "Starting Kubernetes using kubeadm init (this may take a minute) ☸",
			TargetNodes: "@cp1",
			Run: func(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
				switch flags.UsePhases {
				case true:
					return runInitPhases(kctx, kn, flags)
				default:
					return runInit(kctx, kn, flags)
				}
			},
		},
	}
}

func runInit(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	initArgs := []string{
		"init",
		"--ignore-preflight-errors=all",
		"--config=/kind/kubeadm.conf",
	}
	if flags.CopyCerts {
		// automatic copy certs is supported starting from v1.14
		if err := atLeastKubeadm(kn, "v1.14.0-0"); err != nil {
			return errors.Wrapf(err, "--automatic-copy-certs can't be used")
		}

		// before v1.15, upload-certs require the experimental prefix
		uploadCertsFlag := "--upload-certs"
		if err := atLeastKubeadm(kn, "v1.15.0-0"); err != nil {
			uploadCertsFlag = "--experimental-upload-certs"
		}
		initArgs = append(initArgs,
			uploadCertsFlag,
			fmt.Sprintf("--certificate-key=%s", CertificateKey),
		)
	}

	if err := kn.DebugCmd(
		"==> kubeadm init 🚀",
		"kubeadm", initArgs...,
	); err != nil {
		return err
	}

	if err := postInit(
		kctx, kn, flags,
	); err != nil {
		return err
	}

	return nil
}

func runInitPhases(kctx *kcluster.KContext, kn *kcluster.KNode, flags kcluster.ActionFlags) error {
	if err := kn.DebugCmd(
		"==> kubeadm init phase preflight 🚀",
		"kubeadm", "init", "phase", "preflight", "--ignore-preflight-errors=all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase kubelet-start 🚀",
		"kubeadm", "init", "phase", "kubelet-start", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase certs all 🚀",
		"kubeadm", "init", "phase", "certs", "all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase kubeconfig all 🚀",
		"kubeadm", "init", "phase", "kubeconfig", "all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase control-plane all 🚀",
		"kubeadm", "init", "phase", "control-plane", "all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase etcd local 🚀",
		"kubeadm", "init", "phase", "etcd", "local", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> wait for kube-api server 🗻",
		"/bin/bash", "-c", //use shell to get $(...) resolved into the container
		fmt.Sprintf("while [[ \"$(curl -k https://localhost:%d/healthz -s -o /dev/null -w ''%%{http_code}'')\" != \"200\" ]]; do echo -n \".\"; sleep 1; done", APIServerPort),
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase upload-config all 🚀",
		"kubeadm", "init", "phase", "upload-config", "all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if flags.CopyCerts {
		if err := atLeastKubeadm(kn, "v1.14.0-0"); err != nil {
			return errors.Wrapf(err, "--automatic-copy-certs can't be used")
		}

		uploadCertsFlag := "--upload-certs"
		if err := atLeastKubeadm(kn, "v1.15.0-0"); err != nil {
			uploadCertsFlag = "--experimental-upload-certs"
		}

		if err := kn.DebugCmd(
			"==> kubeadm init phase upload-certs 🚀",
			"kubeadm", "init", "phase", "upload-certs", "--config=/kind/kubeadm.conf",
			uploadCertsFlag, fmt.Sprintf("--certificate-key=%s", CertificateKey),
		); err != nil {
			return err
		}
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase mark-control-plane 🚀",
		"kubeadm", "init", "phase", "mark-control-plane", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase bootstrap-token 🚀",
		"kubeadm", "init", "phase", "bootstrap-token", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := kn.DebugCmd(
		"==> kubeadm init phase addon all 🚀",
		"kubeadm", "init", "phase", "addon", "all", "--config=/kind/kubeadm.conf",
	); err != nil {
		return err
	}

	if err := postInit(
		kctx, kn, flags,
	); err != nil {
		return err
	}

	return nil
}
