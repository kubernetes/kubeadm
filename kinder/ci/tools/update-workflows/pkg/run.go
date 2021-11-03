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
	"io/ioutil"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/yaml"
)

// Run runs the main tool logic
func Run(settings *Settings) error {
	log.Infof("using k8s version: %s", settings.KubernetesVersion.String())

	// parse config
	log.Infof("reading config from path: %s", settings.PathConfig)
	configBytes, err := ioutil.ReadFile(settings.PathConfig)
	if err != nil {
		return err
	}
	config := &config{}
	if err := yaml.UnmarshalStrict(configBytes, &config); err != nil {
		return errors.Wrapf(err, "cannot parse config")
	}

	for _, j := range config.JobGroups {
		if err := processjobGroup(settings, &j); err != nil {
			return err
		}
	}
	return nil
}

func processjobGroup(settings *Settings, cfg *jobGroup) error {
	log.Infof("processing JobGroup %#v", cfg)

	oldestVer, err := versionWithSkewInt(settings.KubernetesVersion, -settings.SkewSize)
	if err != nil {
		return errors.Wrapf(err, "could not parse KubernetesVersion - SkewSize")
	}

	var minVer *versionutil.Version
	if len(cfg.MinimumKubernetesVersion) != 0 {
		minVer, err = versionutil.ParseGeneric(cfg.MinimumKubernetesVersion)
		if err != nil {
			return errors.Wrap(err, "could not parse minimumKubernetesVersion")
		}
	}

	log.Infof("oldest supported version in the skew is %s", oldestVer.String())

	// go through the jobs in this group and parse version skew values like 'latest', '+1' etc
	// and update them in place.
	for i := range cfg.Jobs {
		log.Infof("processing version skew modifiers in Job %d", i)
		if err := updateJobVersions(settings.KubernetesVersion, &cfg.Jobs[i]); err != nil {
			return errors.Wrapf(err, "could not update versions for Job index %d in JobGroup %q", i, cfg.Name)
		}
		log.Infof("resulted Job object: %#v", cfg.Jobs[i])
	}

	// process workflows
	if err := processWorkflows(settings, cfg, oldestVer, minVer); err != nil {
		return err
	}

	// process testinfra
	if err := processTestInfra(settings, cfg, oldestVer, minVer); err != nil {
		return err
	}

	return nil
}
