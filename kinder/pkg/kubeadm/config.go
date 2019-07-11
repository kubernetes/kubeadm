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

package kubeadm

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/kubeadm/kinder/pkg/constants"
	kindinternalkubeadm "k8s.io/kubeadm/kinder/third_party/kind/kubeadm"
)

// ConfigData is supplied to the kubeadm config template, with values populated by the cluster package
//
// NB. this is an alias to a kind internal type from "sigs.k8s.io/kind/pkg/cluster/internal/kubeadm" package forked under third_party folder;
// always prefer using this alias instead of the internal type.
type ConfigData kindinternalkubeadm.ConfigData

// Config returns a kubeadm generated using the the config API version corresponding
// to the kubeadmVersion and with the customizable settings based on data
func Config(kubeadmVersion *K8sVersion.Version, data ConfigData) (config string, err error) {
	// gets the config version corresponding to a kubeadm version
	kubeadmConfigVersion, err := getKubeadmConfigVersion(kubeadmVersion)
	if err != nil {
		return "", err
	}

	// select the patches for the kubeadm config version
	log.Debugf("Preparing kubeadm config %s (kubeadm version %s)", kubeadmConfigVersion, kubeadmVersion)
	var templateSource string
	switch kubeadmConfigVersion {
	case "v1beta2":
		templateSource = kindinternalkubeadm.ConfigTemplateBetaV2
	case "v1beta1":
		templateSource = kindinternalkubeadm.ConfigTemplateBetaV1
	case "v1alpha3":
		templateSource = kindinternalkubeadm.ConfigTemplateAlphaV3
	case "v1alpha2":
		templateSource = kindinternalkubeadm.ConfigTemplateAlphaV2
	default:
		return "", errors.Errorf("unknown kubeadm config version: %s", kubeadmConfigVersion)
	}

	t, err := template.New("kubeadm-config").Parse(templateSource)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}

	// derive any automatic fields if not supplied
	// NB. this requires to cast back to the internal kind ConfigData type forked under third_party folder
	internalData := kindinternalkubeadm.ConfigData(data)
	internalData.Derive()

	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, internalData)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}

	return buff.String(), nil
}

// getKubeadmConfigVersion returns the kubeadm config version corresponding to a Kubernetes kubeadmVersion
func getKubeadmConfigVersion(kubeadmVersion *K8sVersion.Version) (string, error) {
	// returns the corresponding config version
	// nb v1alpha1 (that is Kubernetes v1.10.0) is out of support
	if kubeadmVersion.LessThan(constants.V1_12) {
		return "v1alpha2", nil
	}

	if kubeadmVersion.LessThan(constants.V1_13) {
		return "v1alpha3", nil
	}

	if kubeadmVersion.LessThan(constants.V1_15) {
		return "v1beta1", nil
	}

	return "v1beta2", nil
}
