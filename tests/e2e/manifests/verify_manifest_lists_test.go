/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/version"
)

func TestSemVerListSort(t *testing.T) {
	tests := []struct {
		name   string
		input  VersionList
		output VersionList
	}{
		{
			name: "valid: list is sorted correctly [1]",
			input: VersionList{
				version.MustParseSemantic("1.12.2-alpha.0"),
				version.MustParseSemantic("1.12.2-alpha.3"),
				version.MustParseSemantic("1.12.2-beta.2"),
				version.MustParseSemantic("1.12.3"),
				version.MustParseSemantic("1.12.2-rc.4"),
				version.MustParseSemantic("1.12.2"),
			},
			output: VersionList{
				version.MustParseSemantic("1.12.3"),
				version.MustParseSemantic("1.12.2"),
				version.MustParseSemantic("1.12.2-rc.4"),
				version.MustParseSemantic("1.12.2-beta.2"),
				version.MustParseSemantic("1.12.2-alpha.3"),
				version.MustParseSemantic("1.12.2-alpha.0"),
			},
		},
		{
			name: "valid: list is sorted correctly [2]",
			input: VersionList{
				version.MustParseSemantic("1.12.7"),
				version.MustParseSemantic("1.12.2-beta.2"),
				version.MustParseSemantic("1.9.1-alpha.3"),
				version.MustParseSemantic("1.12.2-beta.4"),
				version.MustParseSemantic("1.8.2-rc.4"),
				version.MustParseSemantic("1.11.2-rc.1"),
			},
			output: VersionList{
				version.MustParseSemantic("1.12.7"),
				version.MustParseSemantic("1.12.2-beta.4"),
				version.MustParseSemantic("1.12.2-beta.2"),
				version.MustParseSemantic("1.11.2-rc.1"),
				version.MustParseSemantic("1.9.1-alpha.3"),
				version.MustParseSemantic("1.8.2-rc.4"),
			},
		},
		{
			name: "valid: list is sorted correctly [3]",
			input: VersionList{
				version.MustParseSemantic("2.1.7"),
				version.MustParseSemantic("2.2.1"),
				version.MustParseSemantic("4.0.1"),
				version.MustParseSemantic("4.1.0"),
			},
			output: VersionList{
				version.MustParseSemantic("4.1.0"),
				version.MustParseSemantic("4.0.1"),
				version.MustParseSemantic("2.2.1"),
				version.MustParseSemantic("2.1.7"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input.sort()
			for i, v := range test.output {
				res, err := v.Compare(test.input[i].String())
				if err != nil {
					t.Fatalf("fatal error for input #%d; error: %v", i, err)
				}
				if res != 0 {
					t.Fatalf("element %d does not match: expected: %s, got: %s\n", i, v.String(), test.input[i].String())
				}
			}
		})
	}
}

func TestFilterVersions(t *testing.T) {
	tests := []struct {
		name   string
		input  VersionList
		output VersionList
	}{
		{
			name: "valid: list is filtered correctly [1]",
			input: VersionList{
				version.MustParseSemantic("1.12.0-alpha.0"),
				version.MustParseSemantic("1.12.0-alpha.3"),
				version.MustParseSemantic("1.12.0"),
				version.MustParseSemantic("1.12.0-rc.3"),
				version.MustParseSemantic("1.11.0-beta.2"),
			},
			output: VersionList{
				version.MustParseSemantic("1.12.0"),
				version.MustParseSemantic("1.12.0-rc.3"),
			},
		},
		{
			name: "valid: list is filtered correctly [2]",
			input: VersionList{
				version.MustParseSemantic("1.12.3-alpha.0"),
				version.MustParseSemantic("1.12.3-beta.1"),
				version.MustParseSemantic("1.11.11-rc.3"),
				version.MustParseSemantic("1.12.11-beta.2"),
			},
			output: VersionList{
				version.MustParseSemantic("1.12.11-beta.2"),
				version.MustParseSemantic("1.12.3-beta.1"),
				version.MustParseSemantic("1.12.3-alpha.0"),
			},
		},
		{
			name: "valid: result is an empty list",
			input: VersionList{
				version.MustParseSemantic("1.12.0-beta.2"),
				version.MustParseSemantic("1.12.0-alpha.0"),
				version.MustParseSemantic("1.11.0"),
				version.MustParseSemantic("1.10.0"),
				version.MustParseSemantic("1.9.0"),
			},
			output: VersionList{},
		},
		{
			name:   "valid: empty input list",
			input:  VersionList{},
			output: VersionList{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := filterVersions(test.input)
			if err != nil {
				t.Fatalf("fatal error for input: %v", err)
			}
			if len(output) != len(test.output) {
				t.Fatalf("filtered list is of different length: expected: %d, got: %d\n", len(test.output), len(output))
			}
			for i, v := range test.output {
				res, err := v.Compare(output[i].String())
				if err != nil {
					t.Fatalf("fatal error for input #%d; error: %v", i, err)
				}
				if res != 0 {
					t.Fatalf("element %d does not match: expected: %s, got: %s\n", i, v.String(), test.input[i].String())
				}
			}
		})
	}
}
