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
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/status"
	"k8s.io/kubeadm/kinder/pkg/cri/nodes"
)

// checkImagesForVersion pre-loaded images available on the node (this will report missing images, if any)
func checkImagesForVersion(n *status.Node, version string) error {
	n.Infof("Checking pre-loaded images")

	imageListCmd := fmt.Sprintf("kubeadm config images list --kubernetes-version=%s 2>/dev/null", version)

	// gets the list of images kubeadm is going to use
	expected, err := n.Command(
		"bash", "-c", imageListCmd,
	).Silent().RunAndCapture()
	if err != nil {
		return errors.Wrapf(err, "failed to read expected images for version %s from %s", version, n.Name())
	}
	log.Debugf("List of images kubeadm is going to use %s\n", expected)

	// gets the list of images already pre-loaded in the node
	nodeCRI, err := n.CRI()
	if err != nil {
		return err
	}

	actionHelper, err := nodes.NewActionHelper(nodeCRI)
	if err != nil {
		return err
	}

	current, err := actionHelper.GetImages(n)
	if err != nil {
		return err
	}
	log.Debugf("List of images already pre-loaded in the node %s\n", current)

	// Compare expected and current image and report result
	var currentMap = map[string]string{}
	for _, c := range current {
		currentMap[c] = c
	}

	var missing = []string{}
	for _, e := range expected {
		if _, ok := currentMap[e]; ok {
			continue
		}

		missing = append(missing, e)
	}

	if len(missing) > 0 {
		fmt.Printf("Some of the required images are not pre-loaded into the container runtime:\n%s\n", strings.Join(missing, "\n"))
		return nil
	}

	fmt.Println("All the requested images are already pre-loaded into the container runtime")
	return nil
}
