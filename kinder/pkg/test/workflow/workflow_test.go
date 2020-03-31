/*
Copyright 2020 The Kubernetes Authors.

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

package workflow

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestDurationJSON(t *testing.T) {
	// Marshal cases
	testCasesMarshal := []struct {
		name           string
		input          *Duration
		expectedOutput []byte
		expectedError  bool
	}{
		{
			name: "valid string output",
			input: &Duration{
				time.Duration(time.Minute * 5),
			},
			expectedOutput: []byte(`"5m0s"`),
		},
		{
			name:           "valid 0s on empty input",
			input:          &Duration{},
			expectedOutput: []byte(`"0s"`),
		},
	}
	for _, tc := range testCasesMarshal {
		t.Run("marshal: "+tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.input)
			if (err != nil) != tc.expectedError {
				t.Errorf("expected error %v, got %v, error: %v", tc.expectedError, err != nil, err)
			}
			if err != nil {
				return
			}
			if !bytes.Equal(b, tc.expectedOutput) {
				t.Errorf("expected output %s, got %s", tc.expectedOutput, b)
			}
		})
	}

	// Unmarshal cases
	testCasesUnmarshal := []struct {
		name           string
		input          []byte
		expectedOutput *Duration
		expectedError  bool
	}{
		{
			name:  "valid string input",
			input: []byte(`"5m0s"`),
			expectedOutput: &Duration{
				time.Duration(time.Minute * 5),
			},
		},
		{
			name:  "valid float64 input",
			input: []byte(`240000000000`),
			expectedOutput: &Duration{
				time.Duration(time.Minute * 4),
			},
		},
		{
			name:          "invalid foo input",
			input:         []byte(`foo`),
			expectedError: true,
		},
	}
	for _, tc := range testCasesUnmarshal {
		t.Run("unmarshal: "+tc.name, func(t *testing.T) {
			d := &Duration{}
			err := json.Unmarshal(tc.input, d)
			if (err != nil) != tc.expectedError {
				t.Errorf("expected error %v, got %v, error: %v", tc.expectedError, err != nil, err)
			}
			if err != nil {
				return
			}
			if *tc.expectedOutput != *d {
				t.Errorf("expected output %#v, got %#v", *tc.expectedOutput, *d)
			}
		})
	}
}
