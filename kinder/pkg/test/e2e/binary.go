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

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	kindexec "sigs.k8s.io/kind/pkg/exec"
)

// getBinary tries to find a binary, and if not exist it builds it
func getOrBuildBinary(kubeRoot, name, target string) (string, error) {
	path, err := findBinary(kubeRoot, name)
	if err != nil {
		return "", errors.Errorf("error finding %s binary", name)
	}

	if path == "" {
		log.Debugf("%s binary not found, triggering build", name)

		cmd := kindexec.Command("make", "-C", kubeRoot, fmt.Sprintf("WHAT=%s", target))
		kindexec.InheritOutput(cmd)
		err = cmd.Run()
		if err != nil {
			return "", errors.Errorf("error building %s binary - %s target", name, target)
		}
	}

	path, err = findBinary(kubeRoot, name)
	if err != nil {
		return "", errors.Errorf("error finding build output for %s binary", name)
	}

	if path != "" {
		log.Infof("using %s binary: %s", name, path)
		return path, nil
	}

	return "", errors.Errorf("unable to find build output for %s binary", name)
}

// findKubeRoot attempts to locate a kubernetes checkout
func findKubeRoot() (root string, err error) {
	goroot := os.Getenv("GOPATH")
	if goroot == "" {
		cmd := kindexec.Command("go", "env", "GOPATH")
		lines, err := kindexec.CombinedOutputLines(cmd)
		if err != nil {
			return "", err
		}
		goroot = lines[0]
		if goroot == "" {
			return "", errors.New("unable to get GOPATH env variable. Please provide Kubernetes path source using the --kube-root flag")
		}
	}

	kubeRoot := filepath.Join(goroot, "src", "k8s.io", "kubernetes")
	if _, err := os.Stat(kubeRoot); os.IsNotExist(err) {
		return "", errors.New("$GOPATH/src/k8s.io/kubernetes does not exists. Please provide Kubernetes path source using the --kube-root flag")
	}

	if !maybeKubeRoot(kubeRoot) {
		return "", errors.New("$GOPATH/src/k8s.io/kubernetes does not seems a valid Kubernetes source folder. Please provide Kubernetes source path using the --kube-root flag")
	}
	return kubeRoot, nil
}

// maybeKubeRoot returns true if the dir looks plausibly like a kubernetes source directory
func maybeKubeRoot(dir string) bool {
	// TODO: consider adding other sanity checks
	return dir != ""
}

// findBinary finds a file by name, from a list of well-known output locations
// When multiple matches are found, the most recent will be returned
// Based on kube::util::find-binary from kubernetes/kubernetes
func findBinary(kubeRoot string, name string) (string, error) {

	locations := []string{
		filepath.Join(kubeRoot, "_output", "bin", name),
		filepath.Join(kubeRoot, "_output", "dockerized", "bin", name),
		filepath.Join(kubeRoot, "_output", "local", "bin", name),
		filepath.Join(kubeRoot, "platforms", runtime.GOOS, runtime.GOARCH, name),
	}

	bazelBin := filepath.Join(kubeRoot, "bazel-bin")
	bazelBinExists := true
	if _, err := os.Stat(bazelBin); os.IsNotExist(err) {
		bazelBinExists = false
	}

	if bazelBinExists {
		err := filepath.Walk(bazelBin, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "error walking %s tree", bazelBin)
			}
			if info.Name() != name {
				return nil
			}
			if !strings.Contains(path, runtime.GOOS+"_"+runtime.GOARCH) {
				return nil
			}
			locations = append(locations, path)
			return nil
		})
		if err != nil {
			return "", err
		}
	}

	newestLocation := ""
	var newestModTime time.Time
	for _, loc := range locations {
		stat, err := os.Stat(loc)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", errors.Wrapf(err, "error accessing %s location", loc)
		}
		if newestLocation == "" || stat.ModTime().After(newestModTime) {
			newestModTime = stat.ModTime()
			newestLocation = loc
		}
	}

	if newestLocation == "" {
		log.Debugf("could not find %s binary, looked in %s", name, locations)
	}

	return newestLocation, nil
}
