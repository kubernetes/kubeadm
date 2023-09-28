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
	"fmt"
	"os"
	"testing"

	"path/filepath"
)

func TestGetCRISandboxImage(t *testing.T) {
	expectedSandboxImage := "registry.k8s.io/pause:3.7"

	data := fmt.Sprintf(`version = 2
[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "%s"
`, expectedSandboxImage)

	tempDir := t.TempDir()

	emptyFilePath := filepath.Join(tempDir, "empty.toml")
	if err := os.WriteFile(emptyFilePath, []byte(""), 0600); err != nil {
		t.Fatalf("couldn't write to file %s: %v", emptyFilePath, err)
	}

	defaultFilePath := filepath.Join(tempDir, "default.toml")
	if err := os.WriteFile(defaultFilePath, []byte(data), 0600); err != nil {
		t.Fatalf("couldn't write to file %s: %v", defaultFilePath, err)
	}

	var tests = []struct {
		name          string
		path          string
		expectedError bool
	}{
		{
			name:          "the containerd config file doesn't exist",
			path:          filepath.Join(tempDir, "invalid.toml"),
			expectedError: true,
		},
		{
			name:          "the containerd config file doesn't contain the sandbox_image field",
			path:          emptyFilePath,
			expectedError: true,
		},
		{
			name:          "the containerd config file contains the sandbox_image field",
			path:          defaultFilePath,
			expectedError: false,
		},
	}

	for _, rt := range tests {
		t.Run(rt.name, func(t *testing.T) {
			sandboxImage, err := GetCRISandboxImage(rt.path)
			if (err != nil) != rt.expectedError {
				t.Errorf("failed GetCRISandboxImage:\n\texpected error: %t\n\tactual error: %v", rt.expectedError, err)
			}

			if err == nil {
				if sandboxImage != expectedSandboxImage {
					t.Errorf("failed GetCRISandboxImage:\n\texpected sandbox image: %s\n\tactual sandbox image: %s", expectedSandboxImage, sandboxImage)
				}
			}
		})
	}
}

func TestSetCRISandboxImage(t *testing.T) {
	data := `version = 2
[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "registry.k8s.io/pause:3.7"
`
	tempDir := t.TempDir()

	emptyFilePath := filepath.Join(tempDir, "empty.toml")
	if err := os.WriteFile(emptyFilePath, []byte(""), 0600); err != nil {
		t.Fatalf("couldn't write to file %s: %v", emptyFilePath, err)
	}

	defaultFilePath := filepath.Join(tempDir, "default.toml")
	if err := os.WriteFile(defaultFilePath, []byte(data), 0600); err != nil {
		t.Fatalf("couldn't write to file %s: %v", defaultFilePath, err)
	}

	var tests = []struct {
		name          string
		path          string
		expectedError bool
	}{
		{
			name:          "the containerd config file doesn't exist",
			path:          filepath.Join(tempDir, "invalid.toml"),
			expectedError: true,
		},
		{
			name:          "the containerd config file doesn't contain the sandbox_image field",
			path:          emptyFilePath,
			expectedError: false,
		},
		{
			name:          "the containerd config file contains the sandbox_image field",
			path:          defaultFilePath,
			expectedError: false,
		},
	}

	expectedSandboxImage := "registry.k8s.io/pause:3.9"

	for _, rt := range tests {
		t.Run(rt.name, func(t *testing.T) {
			err := SetCRISandboxImage(rt.path, expectedSandboxImage)
			if (err != nil) != rt.expectedError {
				t.Errorf("failed SetCRISandboxImage:\n\texpected error: %t\n\tactual error: %v", rt.expectedError, err)
			}

			if err == nil {
				sandboxImage, err := GetCRISandboxImage(rt.path)
				if err != nil {
					t.Fatalf("failed to get sandbox image from config file %s: %v", rt.path, err)
				}

				if sandboxImage != expectedSandboxImage {
					t.Errorf("failed SetCRISandboxImage:\n\texpected sandbox image: %s\n\tactual sandbox image: %s", expectedSandboxImage, sandboxImage)
				}
			}
		})
	}
}
