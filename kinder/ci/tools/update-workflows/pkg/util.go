/*
Copyright 2021 The Kubernetes Authors.

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

package pkg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	versionutil "k8s.io/apimachinery/pkg/util/version"
)

func versionWithSkew(ver *versionutil.Version, str string) (string, error) {
	if str == latestVersion {
		return str, nil
	}
	if str == "" {
		return fmt.Sprintf("%v.%v", ver.Major(), ver.Minor()), nil
	}
	errStr := fmt.Sprintf("version skew modifiers must be equal to 'latest' or an integer"+
		"(e.g. '0', '+1', '-1'), got %q", str)
	n, err := strconv.Atoi(str)
	if err != nil {
		return "", errors.Wrap(err, errStr)
	}
	// TODO support skewing with MAJOR version eventually when k8s v2 is out
	return fmt.Sprintf("%v.%v", ver.Major(), int(ver.Minor())+n), nil
}

func versionWithSkewInt(ver *versionutil.Version, n int) (*versionutil.Version, error) {
	vstr, err := versionWithSkew(ver, fmt.Sprintf("%d", n))
	if err != nil {
		return nil, err
	}
	return versionutil.MustParseGeneric(vstr), nil
}

func parseSkipVersions(ver *versionutil.Version, vers []string) (string, error) {
	var err error
	for i, v := range vers {
		vers[i], err = versionWithSkew(ver, v)
		if err != nil {
			return "", err
		}
	}
	return strings.Join(vers, "|"), nil
}

func dashVer(ver string) string {
	return strings.ReplaceAll(ver, ".", "-")
}

func ciLabelFor(ver string) string {
	if ver == latestVersion {
		return ver
	}
	return fmt.Sprintf("%s-%s", latestVersion, ver)
}

func branchFor(ver string) string {
	if ver == latestVersion {
		return "master" // TODO: change to main when kubernetes/kubernetes uses main as the default branch
	}
	return fmt.Sprintf("release-%s", ver)
}

func imageVer(ver string) string {
	if ver == latestVersion {
		return "master" // TODO: change to main when kubernetes/test-infra uses main as the default branch
	}
	return ver
}

func sigReleaseVer(ver string) string {
	return imageVer(ver)
}

func skipVersion(oldestVer, minVer *versionutil.Version, kubernetesVersion string) bool {
	if kubernetesVersion == latestVersion {
		return false
	}
	// skip if the global skew version is newer than the k8s version in a job
	if comp, _ := oldestVer.Compare(kubernetesVersion); comp > 0 {
		log.Infof("global skew version %s is newer than %s",
			oldestVer.String(), kubernetesVersion)
		return true
	}

	// skip if the minimum k8s version is newer than the k8s version in a job
	if minVer != nil {
		if comp, _ := minVer.Compare(kubernetesVersion); comp > 0 {
			log.Infof("MinimumKubernetesVersion %s is newer than %s",
				minVer.String(), kubernetesVersion)
			return true
		}
	}
	return false
}

func updateJobVersions(ver *versionutil.Version, job *job) error {
	var err error
	// if the kubelet and kubeadm versions are not specified they would be set to the k8s version
	if len(job.KubernetesVersion) == 0 {
		return errors.New("KubernetesVersion cannot be empty")
	}
	job.KubernetesVersion, err = versionWithSkew(ver, job.KubernetesVersion)
	if err != nil {
		return err
	}

	if len(job.KubeadmVersion) == 0 {
		job.KubeadmVersion = job.KubernetesVersion
	} else {
		job.KubeadmVersion, err = versionWithSkew(ver, job.KubeadmVersion)
		if err != nil {
			return err
		}
	}

	if len(job.KubeletVersion) == 0 {
		job.KubeletVersion = job.KubernetesVersion
	} else {
		job.KubeletVersion, err = versionWithSkew(ver, job.KubeletVersion)
		if err != nil {
			return err
		}
	}

	if len(job.InitVersion) == 0 {
		job.InitVersion = job.KubernetesVersion
	} else {
		job.InitVersion, err = versionWithSkew(ver, job.InitVersion)
		if err != nil {
			return err
		}
	}

	if len(job.UpgradeVersion) == 0 {
		job.UpgradeVersion = job.KubernetesVersion
	} else {
		job.UpgradeVersion, err = versionWithSkew(ver, job.UpgradeVersion)
		if err != nil {
			return err
		}
	}

	return nil
}
