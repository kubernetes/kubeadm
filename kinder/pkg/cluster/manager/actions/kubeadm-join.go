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
	"time"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/constants"
)

// KubeadmJoin executes the kubeadm join workflow both for control-plane nodes and
// worker nodes
func KubeadmJoin(c *status.Cluster, usePhases, automaticCopyCerts bool, wait time.Duration, vLevel int) (err error) {
	if err := joinControlPlanes(c, usePhases, automaticCopyCerts, wait, vLevel); err != nil {
		return err
	}

	if err := joinWorkers(c, usePhases, wait, vLevel); err != nil {
		return err
	}
	return nil
}

func joinControlPlanes(c *status.Cluster, usePhases, automaticCopyCerts bool, wait time.Duration, vLevel int) (err error) {
	for _, cp2 := range c.SecondaryControlPlanes() {
		// automatic copy certs is supported starting from v1.14
		if automaticCopyCerts && !cp2.MustKubeadmVersion().AtLeast(constants.V1_14) {
			return errors.New("--automatic-copy-certs can't be used with kubeadm older than v1.14")
		}

		// if not automatic copy certs, simulate manual copy
		if !automaticCopyCerts {
			if err := copyCertificatesToNode(c, cp2); err != nil {
				return err
			}
		}

		// checks pre-loaded images available on the node (this will report missing images, if any)
		kubeVersion, err := cp2.KubeVersion()
		if err != nil {
			return err
		}

		if err := checkImagesForVersion(cp2, kubeVersion); err != nil {
			return err
		}

		// executes the kubeadm join control-plane workflow
		if usePhases {
			err = kubeadmJoinControlPlaneWithPhases(cp2, automaticCopyCerts, vLevel)
		} else {
			err = kubeadmJoinControlPlane(cp2, automaticCopyCerts, vLevel)
		}
		if err != nil {
			return err
		}

		if err := waitNewControlPlaneNodeReady(c, cp2, wait); err != nil {
			return err
		}
	}
	return nil
}

func kubeadmJoinControlPlane(cp *status.Node, automaticCopyCerts bool, vLevel int) (err error) {
	joinArgs := []string{
		"join",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
		"--ignore-preflight-errors=all", // this is required because some check does not pass in kind;
		//TODO: change from all > exact list of checks
	}
	if automaticCopyCerts {
		// if before v1.15, add certificate key flag (for >= 15, certificate key is passed via the config file)
		if cp.MustKubeadmVersion().LessThan(constants.V1_15) {
			joinArgs = append(joinArgs,
				fmt.Sprintf("--certificate-key=%s", constants.CertificateKey),
			)
		}
	}

	if err := cp.Command(
		"kubeadm", joinArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	return nil
}

func kubeadmJoinControlPlaneWithPhases(cp *status.Node, automaticCopyCerts bool, vLevel int) (err error) {
	// kubeadm join phase preflight
	preflightArgs := []string{
		"join", "phase", "preflight",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
		"--ignore-preflight-errors=all", // this is required because some check does not pass in kind;
		//TODO: change from all > exact list of checks
	}
	if automaticCopyCerts {
		// if before v1.15, add certificate key flag (for >= 15, certificate key is passed via the config file)
		if cp.MustKubeadmVersion().LessThan(constants.V1_15) {
			preflightArgs = append(preflightArgs,
				fmt.Sprintf("--certificate-key=%s", constants.CertificateKey),
			)
		}
	}

	if err := cp.Command(
		"kubeadm", preflightArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	// kubeadm join phase control-plane-prepare
	prepareArgs := []string{
		"join", "phase", "control-plane-prepare", "all",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
	}
	if automaticCopyCerts {
		// if before v1.15, add certificate key flag (for >= 15, certificate key is passed via the config file)
		if cp.MustKubeadmVersion().LessThan(constants.V1_15) {
			prepareArgs = append(prepareArgs,
				fmt.Sprintf("--certificate-key=%s", constants.CertificateKey),
			)
		}
	}

	if err := cp.Command(
		"kubeadm", prepareArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	// kubeadm join phase kubelet-start
	if err := cp.Command(
		"kubeadm", "join", "phase", "kubelet-start",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	// kubeadm join phase control-plane-join
	controlPlaneArgs := []string{
		"join", "phase", "control-plane-join", "all",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
	}
	if automaticCopyCerts {
		// if before v1.15, add certificate key flag (for >= 15, certificate key is passed via the config file)
		if cp.MustKubeadmVersion().LessThan(constants.V1_15) {
			controlPlaneArgs = append(controlPlaneArgs,
				fmt.Sprintf("--certificate-key=%s", constants.CertificateKey),
			)
		}
	}

	if err := cp.Command(
		"kubeadm", controlPlaneArgs...,
	).RunWithEcho(); err != nil {
		return err
	}

	return nil
}

func joinWorkers(c *status.Cluster, usePhases bool, wait time.Duration, vLevel int) (err error) {
	for _, w := range c.Workers() {
		if usePhases && !w.MustKubeadmVersion().AtLeast(constants.V1_14) {
			return errors.New("--automatic-copy-certs can't be used with kubeadm older than v1.14")
		}

		// checks pre-loaded images available on the node (this will report missing images, if any)
		kubeVersion, err := w.KubeVersion()
		if err != nil {
			return err
		}

		if err := checkImagesForVersion(w, kubeVersion); err != nil {
			return err
		}

		// executes the kubeadm join workflow
		if usePhases {
			err = kubeadmJoinWorkerWithPhases(w, vLevel)
		} else {
			err = kubeadmJoinWorker(w, vLevel)
		}
		if err != nil {
			return err
		}

		if err := waitNewWorkerNodeReady(c, w, wait); err != nil {
			return err
		}
	}
	return nil
}

func kubeadmJoinWorker(w *status.Node, vLevel int) (err error) {
	if err := w.Command(
		"kubeadm", "join",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
		"--ignore-preflight-errors=all", // this is required because some check does not pass in kind;
		//TODO: change from all > exact list of checks
	).RunWithEcho(); err != nil {
		return err
	}

	return nil
}

func kubeadmJoinWorkerWithPhases(w *status.Node, vLevel int) (err error) {
	// kubeadm join phase preflight
	if err := w.Command(
		"kubeadm", "join", "phase", "preflight",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
		"--ignore-preflight-errors=all", // this is required because some check does not pass in kind;
		//TODO: change from all > exact list of checks
	).RunWithEcho(); err != nil {
		return err
	}

	// NB. kubeadm join phase control-plane-prepare should not be executed when joining a worker node

	// kubeadm join phase kubelet-start
	if err := w.Command(
		"kubeadm", "join", "phase", "kubelet-start",
		fmt.Sprintf("--config=%s", constants.KubeadmConfigPath),
		fmt.Sprintf("--v=%d", vLevel),
	).RunWithEcho(); err != nil {
		return err
	}

	// NB. kubeadm join phase control-plane-join should not be executed when joining a worker node

	return nil
}
