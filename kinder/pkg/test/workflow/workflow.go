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

Workflows must be defined in a yaml file containing list of tasks; each task
can be any command, thus including also any kinder commands invoked via CLI.

Tasks will be executed in order; in case of errors the workflow will stop and the remaining tasks
will be skipped with the only exception of tasks specifically marked to be executed in any case
(e.g. cleanup tasks).
*/
package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"
)

// Workflow represents a list of tasks to be executed during test workflow and related context
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
	var v interface{}
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
	// Checks if the workflow file exists
	if _, err := os.Stat(file); err != nil {
		return nil, errors.Errorf("invalid workflow file: %s does not exist", file)
	}

	// Loads and unmarshal it
	data, err := ioutil.ReadFile(file)
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
	// - at least one task exists

	if w.Version != 1 {
		return nil, errors.Errorf("invalid taskfile %s: version does not contain a supported value", file)
	}

	if len(w.Tasks) == 0 {
		return nil, errors.Errorf("invalid taskfile %s: at least one task should be defined", file)
	}

	// Detect and resolve imports by expanding imported workflows into the top level workflow
	if err := w.expandImports(file); err != nil {
		return nil, err
	}

	// For each task
	for i, t := range w.Tasks {
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

		// check if the task defines a cmd
		if t.Cmd == "" {
			return nil, errors.Errorf("invalid taskfile %s: task %q does not define a cmd", file, t.Name)
		}
	}

	return &w, nil
}

// expandImports imports a secondary workflow into the top level Workflow
func (w *Workflow) expandImports(file string) error {
	tasks := w.Tasks
	w.Tasks = Tasks{}
	for i, t := range tasks {
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
		// TODO: implement a check for avoiding circular imports
		path := t.Import
		if !filepath.IsAbs(path) {
			base := filepath.Dir(file)
			path = filepath.Join(base, path)
		}
		wx, err := NewWorkflow(path)
		if err != nil {
			return errors.Wrapf(err, "error importing workflow file %s", path)
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

// Run executes a workflow
func (w *Workflow) Run(out io.Writer, dryRun, verbose, exitOnError bool, artifacts string) (err error) {

	// get a new taskCmdBuilder, responsible for creating taskCmd commands
	taskCmdBuilder, err := newTaskCmdBuilder(w)
	if err != nil {
		return err
	}

	// if artifact folder is not provided as input argument check
	// 1. ARTIFACTS env var from the workflow file
	// 2. ARTIFACTS OS env var
	// Otherwise generate an artifact folder (or dummy placeholder in case of dry running)

	if artifacts == "" {
		artifacts = taskCmdBuilder.env["ARTIFACTS"]
	}

	if artifacts == "" {
		artifacts = os.Getenv("ARTIFACTS")
	}

	if artifacts == "" {
		if !dryRun {
			dir, err := os.Getwd()
			if err != nil {
				return errors.Wrapf(err, "error getting current directory")
			}

			artifacts, err = ioutil.TempDir(dir, "kinder-test-workflow")
			if err != nil {
				return errors.Wrapf(err, "error creating artifact folder")
			}
		} else {
			artifacts = "<tmp-folder>"
		}
	}

	//TODO: ensure artifact folder exist and can be written

	// adds a new env variable indicating where test artifacts should be stored
	// to make this value available for cmd and args expansion
	taskCmdBuilder.env["ARTIFACTS"] = artifacts

	// Gets a taskCmdRunner, responsible for executing taskCmd,
	// handling failure, cancellation, timeouts and for generating or collecting
	// all the workflow artifacts (junit_runner.xml, task logs, etc)
	taskCmdRunner := newTaskCmdRunner()

	// Process all tasks, exploding golang templates for cmd and args
	// and create the corresponding taskCmd
	// Nb. we are splitting this step from actual execution of task for ensuring
	// that all the formal error are detected before starting any real activity
	var tcmds []*taskCmd
	for _, t := range w.Tasks {

		tcmd, err := taskCmdBuilder.build(t, verbose)
		if err != nil {
			return err
		}

		tcmds = append(tcmds, tcmd)
	}

	foundError := false
	// Executes taskCmds
	for _, tcmd := range tcmds {
		fmt.Fprintf(out, "# %s\n", tcmd.Name)
		fmt.Fprintf(out, "%s\n\n", tcmd.CmdText)

		if !dryRun {
			err := taskCmdRunner.Run(tcmd, artifacts, verbose)
			if err != nil {
				foundError = true
				fmt.Fprintf(out, " %v\n\n", err)

				if exitOnError {
					return err
				}

				continue
			}

			fmt.Fprintf(out, " completed!\n\n")
		}
	}

	// If not dry running, prints task summary and dumps the junit_runner.xml file
	if !dryRun {
		taskCmdRunner.ReportSummary()

		if err := taskCmdRunner.DumpJUnitRunner(artifacts); err != nil {
			fmt.Fprintf(out, "%v\n", err)
			return err
		}
		fmt.Fprintf(out, "see junit-runner.xml and task logs files for more details\n\n")
	}

	if foundError {
		return errors.New("failed executing the workflow")
	}
	return nil
}
