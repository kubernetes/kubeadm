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

package alter

import (
	"fmt"

	"github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/container/docker"
)

// Labels are used to encode metadata on the kind(er) images.
// This allows to simplify UX at create or init time, because some information
// can be retrieved from label instead of being provided with flags.

// alterContainerLabelKey is applied to each altered container
const alterContainerLabelKey = "io.k8s.sigs.kinder.alter"

// alterVersionLabelKey is applied to each altered container with a well known init version.
// This metadata is used e.g. to choose which kubeadm config version should be used at create time
const alterVersionLabelKey = "io.k8s.sigs.kinder.initVersion"

// GetImageVersion return the version number the image is tagged with, if any
func GetImageVersion(image string) (string, error) {
	return getImageLabel(image, alterVersionLabelKey)
}

func getImageLabel(image, key string) (string, error) {
	lines, err := docker.Inspect(image, fmt.Sprintf("{{index .Config.Labels %q}}", key))
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %q label", key)
	}
	if len(lines) != 1 {
		return "", errors.Errorf("%q label should only be one line, got %d lines", key, len(lines))
	}

	return lines[0], nil
}
