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
	"reflect"
	"testing"
)

func TestAppendEtcdctlCertArgs(t *testing.T) {
	tests := []struct {
		name          string
		inputImage    string
		inputArgs     []string
		expectedArgs  []string
		expectedError bool
	}{
		{
			name:         "valid: version 3.4",
			inputImage:   "etcd:3.4.0",
			expectedArgs: etcdCertArgsNew,
		},
		{
			name:         "valid: version 4.0",
			inputImage:   "etcd:4.0.0",
			expectedArgs: etcdCertArgsNew,
		},
		{
			name:         "valid: old version",
			inputImage:   "etcd:3.3.17",
			expectedArgs: etcdCertArgsOld,
		},
		{
			name:         "valid: append to existing args",
			inputImage:   "etcd:3.4.0",
			inputArgs:    []string{"foo"},
			expectedArgs: append([]string{"foo"}, etcdCertArgsNew...),
		},
		{
			name:          "invalid: image string missing ':'",
			inputImage:    "3.4.0",
			expectedError: true,
		},
		{
			name:          "invalid: image tag is not semver",
			inputImage:    "etcd:111",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.inputArgs
			err := appendEtcdctlCertArgs(test.inputImage, &args)
			if (err != nil) != test.expectedError {
				t.Fatalf("expected error: %v, found %v, error: %v", test.expectedError, err != nil, err)
			}
			if test.expectedError {
				return
			}
			if !reflect.DeepEqual(args, test.expectedArgs) {
				t.Fatalf("expected args: %v, found %v", test.expectedArgs, args)
			}
		})
	}
}
