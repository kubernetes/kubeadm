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

package test

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// GinkgoFlags defines a type for handling flag/values pairs to be passed to the ginkgo test runner
type GinkgoFlags map[string]string

// NewGinkgoFlags returns a new GinkgoFlags struct created by parsing the space-separated list of arguments
func NewGinkgoFlags(flagString string) (GinkgoFlags, error) {
	ginkgoFlags, err := parseFlagsString(flagString)
	if err != nil {
		return nil, err
	}
	return ginkgoFlags, nil
}

// AddFocusRegex allows to add a new regex to pass to ginkgo with the --focus flag.
// In case the flag is already set, the new regex is appended (existing or new)
func (g GinkgoFlags) AddFocusRegex(val string) {
	g.mergeRegex("focus", val)
}

// AddSkipRegex allows to add a new regex to pass to ginkgo with the --skip flag.
// In case the flag is already set, the new regex is appended (existing or new)
func (g GinkgoFlags) AddSkipRegex(val string) {
	g.mergeRegex("skip", val)
}

func (g GinkgoFlags) mergeRegex(key, val string) {
	if exp, ok := g[key]; !ok {
		g[key] = val
	} else {
		g[key] = fmt.Sprintf("%s|%s", exp, val)
	}
}

// SuiteFlags defines a type for handling flag/values pairs to be passed to the test suite
type SuiteFlags map[string]string

// NewSuiteFlags returns a new SuiteFlags struct created by parsing the space-separated list of arguments
func NewSuiteFlags(flagString string) (SuiteFlags, error) {
	testFlags, err := parseFlagsString(flagString)
	if err != nil {
		return nil, err
	}
	return testFlags, nil
}

// parseFlagsString parse the space-separated list of arguments
func parseFlagsString(flagString string) (flags map[string]string, err error) {
	flags = make(map[string]string)
	if flagString == "" {
		return
	}
	// splits the space-separated list and parse all the --key=value argument
	for _, arg := range strings.Split(flagString, " ") {
		key, val, err := parseFlagString(arg)
		if err != nil {
			return nil, errors.Errorf("flag %q could not be parsed correctly: %v", arg, err)
		}

		flags[key] = val
	}
	return flags, nil
}

// parseFlagString parse a --key=value argument in the space-separated list of arguments
func parseFlagString(arg string) (string, string, error) {
	if !strings.HasPrefix(arg, "--") {
		return "", "", errors.New("the argument should start with '--'")
	}
	if !strings.Contains(arg, "=") {
		return "", "", errors.New("the argument should have a '=' between the flag name and the value")
	}
	// Remove the starting --
	arg = strings.TrimPrefix(arg, "--")
	// Split the string on =. Return only two substrings, since we want only key/value, but the value can include '=' as well
	keyvalSlice := strings.SplitN(arg, "=", 2)

	// Make sure both a key and value is present
	if len(keyvalSlice) != 2 {
		return "", "", errors.New("the argument must have both a key and a value")
	}
	if len(keyvalSlice[0]) == 0 {
		return "", "", errors.New("the argument must have a key")
	}

	return keyvalSlice[0], keyvalSlice[1], nil
}
