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
	versionutil "k8s.io/apimachinery/pkg/util/version"
)

const (
	latestVersion       = "latest"
	autogeneratedHeader = "# AUTOGENERATED by https://git.k8s.io/kubeadm/kinder/ci/tools/update-workflows"
)

// Settings holds additional settings from the user
type Settings struct {
	KubernetesVersion *versionutil.Version
	PathConfig        string
	PathTestInfra     string
	PathWorkflows     string
	ImageTestInfra    string
	SkewSize          int
}

type config struct {
	TargetVersion *versionutil.Version `json:"targetVersion,omitempty"`
	JobGroups     []jobGroup           `json:"jobGroups,omitempty"`
}

type jobGroup struct {
	Name                     string            `json:"name,omitempty"`
	MinimumKubernetesVersion string            `json:"minimumKubernetesVersion,omitempty"`
	TestInfraJobSpec         jobGroupTestInfra `json:"testInfraJobSpec,omitempty"`
	KinderWorkflowSpec       jobGroupWorkflows `json:"kinderWorkflowSpec,omitempty"`
	Jobs                     []job             `json:"jobs,omitempty"`
}

type jobGroupTestInfra struct {
	TargetFile string `json:"targetFile,omitempty"`
	Template   string `json:"template,omitempty"`
}

type jobGroupWorkflows struct {
	TargetFile      string   `json:"targetFile,omitempty"`
	Template        string   `json:"template,omitempty"`
	AdditionalFiles []string `json:"additionalFiles,omitempty"`
}

type job struct {
	InitVersion       string   `json:"initVersion,omitempty"`
	KubernetesVersion string   `json:"kubernetesVersion,omitempty"`
	KubeadmVersion    string   `json:"kubeadmVersion,omitempty"`
	KubeletVersion    string   `json:"kubeletVersion,omitempty"`
	SkipVersions      []string `json:"skipVersions,omitempty"`
}

type templateVars struct {
	KubernetesVersion    string
	KubeletVersion       string
	KubeadmVersion       string
	KubeadmConfigVersion string
	InitVersion          string

	TargetFile   string
	SkipVersions string

	TestInfraImage   string
	JobInterval      string
	AlertAnnotations string
	WorkflowFile     string
}
