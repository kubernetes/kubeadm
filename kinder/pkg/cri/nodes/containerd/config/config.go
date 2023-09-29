/*
Copyright 2023 The Kubernetes Authors.

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

package config

import (
	"os"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

var (
	// DefaultConfigPath is the default location of the containerd config file.
	DefaultConfigPath = "/etc/containerd/config.toml"
)

var (
	sandboxImageFieldPath = []string{"plugins", "io.containerd.grpc.v1.cri", "sandbox_image"}
)

// GetCRISandboxImage returns the sandbox image defined in the containerd config file.
func GetCRISandboxImage(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	tree, err := toml.LoadFile(path)
	if err != nil {
		return "", err
	}

	if !tree.HasPath(sandboxImageFieldPath) {
		return "", errors.Errorf("the field %v doesn't exist in the config file %s", sandboxImageFieldPath, path)
	}

	return tree.GetPath(sandboxImageFieldPath).(string), nil
}

// SetCRISandboxImage sets the sandbox image field of the containerd config file to the specified value.
func SetCRISandboxImage(path string, sandboxImage string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	tree, err := toml.LoadFile(path)
	if err != nil {
		return err
	}

	tree.SetPath(sandboxImageFieldPath, sandboxImage)

	data, err := tree.ToTomlString()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(data), 0666); err != nil {
		return errors.Errorf("failed to write to config file %s, error: %v", path, err)
	}

	return nil
}
