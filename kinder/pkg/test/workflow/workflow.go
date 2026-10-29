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

/*
Package workflow implements a simple workflow manager aimed at automating kinder test workflows.

Workflows are YAML files containing either tasks or a group of child workflows.
Each task can be any command, including kinder commands invoked via CLI.
Child workflows execute sequentially with isolated variables and failure state;
a failed child does not prevent its siblings from running, but cancellation does.

Tasks will be executed in order; in case of errors the workflow will stop and the remaining tasks
will be skipped with the only exception of tasks specifically marked to be executed in any case
(e.g. cleanup tasks).
*/
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/yaml"
)

// Workflow defines either a sequence of tasks and its context, or a group of independent workflows.
type Workflow struct {
	// Version of the workflow file
	// NB. We are enforcing version to be set only for future improvements of the config API
	// but currently there is no version management in place.
	Version int

	// Summary provides an high level description of the test workflow
	Summary string

	// Vars defines a set of variables used for golang template expansion.
	// Variables are processed in order, and OS environment variables and already known Vars
	// can be used in templates for other vars.
	// Vars will be accessible as {{ .vars.KEY }}
	Vars map[string]string

	// Env defines a list of env variables to be passed to the workflow CMD (in addition to OS env variables);
	// Envs are processed in order, and OS environment variables, Vars and already known Vars
	// can be used in templates for other envs.
	// After all the envs are processed, an ARTIFACTS env variable is added by default, eventually overriding the existing
	// one according to the precedence of the ARTIFACTS command line argument, env vars defined in the config
	// and OS environment variables.
	// Env variables can be used for golang template expansion using {{ .env.KEY }}
	Env map[string]string

	// Tasks defines the list of tasks to be executed during test workflow
	Tasks Tasks

	// Workflows defines a sequential group of independent scenarios, mutually
	// exclusive with Tasks. A scenario failure does not skip its siblings.
	Workflows []*Scenario
}

// Scenario invokes a workflow in the same process with independent variables,
// task state and artifacts. Paths are relative to the containing workflow file.
type Scenario struct {
	Name     string
	Workflow string
	// Timeout bounds scenario execution. Forced cleanup tasks may run afterwards,
	// each bounded by its own task timeout. Zero means no scenario deadline.
	Timeout Duration

	child *Workflow
}

// Tasks represents a list of tasks to be executed during test workflow.
// Task are executed in order; if a task fails, timeouts or it is canceled by the user,
// following task are skipped (unless execution is explicitly forced on a specific task)
type Tasks []*Task

// Task represents a task to be executed as part of a test workflow
type Task struct {
	// Name of the task
	Name string

	// Description of the task
	Description string

	// Dir allows to set the working directory for this tasks
	Dir string

	// Cmd to execute; it can be a literal or a template
	Cmd string

	// Import defines a path of a workflow file to import into the current workflow
	Import string

	// Args allows to set Cmd arguments; args can be a literal or a template
	Args []string

	// Force sets a task to be executed no matter of the result of the previous task.
	// This allows e.g. to define cleanup tasks to be always executed
	Force bool

	// Timeout for the current task, 5m by default
	Timeout Duration

	// IgnoreError sets a task to be recorded as successful even if it is actually failed
	IgnoreError bool `yaml:"ignoreError"`
}

// Duration is a wrapper around time.Duration to satisfy the encoding/json Marshaller
// and Unmarshaller interfaces. This extends sigs.k8s.io/yaml to support JSON handling
// of time.Duration.
type Duration struct {
	time.Duration
}

// MarshalJSON marshals a Duration object to JSON data
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals JSON data to a Duration object
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	// Only float64 and string are supported
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// NewWorkflow creates a new workflow as defined in a workflow file
func NewWorkflow(file string) (*Workflow, error) {
	return loadWorkflow(file, map[string]bool{})
}

var scenarioNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func loadWorkflow(file string, active map[string]bool) (*Workflow, error) {
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid workflow file %s", file)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, err
	}
	if active[canonical] {
		return nil, errors.Errorf("cyclic workflow reference: %s", file)
	}
	active[canonical] = true
	defer delete(active, canonical)

	// Loads and unmarshal it
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading workflow file %s", file)
	}
	var w Workflow
	err = yaml.UnmarshalStrict(data, &w)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling workflow file %s", file)
	}

	// checks minimum requirements
	// - version is set and well know
	// - either tasks or child workflows exist

	if w.Version != 1 {
		return nil, errors.Errorf("invalid taskfile %s: version does not contain a supported value", file)
	}

	if (len(w.Tasks) == 0) == (len(w.Workflows) == 0) {
		return nil, errors.Errorf("invalid workflow %s: define either tasks or workflows", file)
	}
	if len(w.Workflows) > 0 {
		if len(w.Vars) > 0 || len(w.Env) > 0 {
			return nil, errors.Errorf("workflow group %s: define vars and env in each child workflow", file)
		}
		names := map[string]bool{}
		for _, scenario := range w.Workflows {
			if scenario == nil || !scenarioNamePattern.MatchString(scenario.Name) {
				return nil, errors.Errorf("workflow group %s: each scenario needs a safe, nonempty name (letters, digits, '_' or '-')", file)
			}
			if names[scenario.Name] {
				return nil, errors.Errorf("workflow group %s: duplicate scenario %q", file, scenario.Name)
			}
			names[scenario.Name] = true
			if scenario.Workflow == "" || scenario.Timeout.Duration < 0 {
				return nil, errors.Errorf("workflow group %s: scenario %q needs a workflow path and nonnegative timeout", file, scenario.Name)
			}
			childPath := scenario.Workflow
			if !filepath.IsAbs(childPath) {
				childPath = filepath.Join(filepath.Dir(file), childPath)
			}
			scenario.child, err = loadWorkflow(childPath, active)
			if err != nil {
				return nil, errors.Wrapf(err, "loading scenario %q", scenario.Name)
			}
		}
		return &w, nil
	}
	if w.Vars == nil {
		w.Vars = map[string]string{}
	}
	if w.Env == nil {
		w.Env = map[string]string{}
	}

	// Detect and resolve imports by expanding imported workflows into the top level workflow
	if err := w.expandImports(file, active); err != nil {
		return nil, err
	}

	// For each task
	for i, t := range w.Tasks {
		if t == nil {
			return nil, errors.Errorf("workflow %s: task #%d is null", file, i+1)
		}
		// if a task name is not defined, assign a default task name
		// otherwise prepend a prefix in order to get task logs ordered
		if t.Name == "" {
			t.Name = fmt.Sprintf("task-%02d", i)
		} else {
			t.Name = fmt.Sprintf("task-%02d-%s", i, t.Name)
		}

		// if a timeout is not defined, assign a default one
		// nb. we are assigning a fairly long timeout to avoid flakes in testgrid
		if t.Timeout.Duration == 0 {
			t.Timeout.Duration = time.Duration(5 * time.Minute)
		}

		if t.Timeout.Duration < 0 {
			return nil, errors.Errorf("task %q has a negative timeout", t.Name)
		}

		// check if the task defines a cmd
		if t.Cmd == "" {
			return nil, errors.Errorf("invalid taskfile %s: task %q does not define a cmd", file, t.Name)
		}
	}

	return &w, nil
}

// expandImports imports a secondary workflow into the top level Workflow
func (w *Workflow) expandImports(file string, active map[string]bool) error {
	tasks := w.Tasks
	w.Tasks = Tasks{}
	for i, t := range tasks {
		if t == nil {
			return errors.Errorf("workflow %s: task #%d is null", file, i+1)
		}
		// check if the task does not defines an import, preserve it as it is
		if t.Import == "" {
			w.Tasks = append(w.Tasks, t)
			continue
		}

		// otherwise it is an import task
		// ensure the import task does not have other settings
		if t.Dir != "" {
			return errors.Errorf("invalid workflow file %s: task #%d - dir setting can't be combined with import directive", file, i+1)
		}
		if t.Cmd != "" {
			return errors.Errorf("invalid workflow file %s: task #%d - cmd setting can't be combined with import directive", file, i+1)
		}
		if len(t.Args) != 0 {
			return errors.Errorf("invalid workflow file %s: task #%d - args setting can't be combined with import directive", file, i+1)
		}
		if t.Force {
			return errors.Errorf("invalid workflow file %s: task #%d - force setting can't be combined with import directive", file, i+1)
		}
		if t.Timeout.Duration != 0 {
			return errors.Errorf("invalid workflow file %s: task #%d - timeout setting can't be combined with import directive", file, i+1)
		}
		if t.IgnoreError {
			return errors.Errorf("invalid workflow file %s: task #%d - ignoreError setting can't be combined with import directive", file, i+1)
		}

		// reads the Import file
		// if path are relative, consider as a base path the folder where the importing file is located.
		path := t.Import
		if !filepath.IsAbs(path) {
			base := filepath.Dir(file)
			path = filepath.Join(base, path)
		}
		wx, err := loadWorkflow(path, active)
		if err != nil {
			return errors.Wrapf(err, "error importing workflow file %s", path)
		}

		if len(wx.Workflows) > 0 {
			return errors.Errorf("cannot import workflow group %s into tasks; use workflows instead", path)
		}

		// merge the vars from the import file into the parent file
		// in case of conflicts, vars in the parent file will shadow vars in the import file
		for k, v := range wx.Vars {
			if _, ok := w.Vars[k]; !ok {
				w.Vars[k] = v
				continue
			}
			log.Debugf("var %s in workflow file %s is shadowed by var %[1]s in parent workflow file %[3]s", k, path, file)
		}

		// merge the env vars from the import file into the parent file
		// in case of conflicts, env vars in the parent file will shadow env vars in the import file
		for k, v := range wx.Env {
			if _, ok := w.Env[k]; !ok {
				w.Env[k] = v
				continue
			}
			log.Debugf("env var %s in workflow file %s is shadowed by env var %[1]s in parent workflow file %[3]s", k, path, file)
		}

		// import all tasks from the import file into the parent file, removing task name prefix
		re := regexp.MustCompile(`^task\-\d{2}\-?`)
		for _, tx := range wx.Tasks {
			tx.Name = re.ReplaceAllString(tx.Name, "")
			w.Tasks = append(w.Tasks, tx)
		}
	}

	return nil
}
