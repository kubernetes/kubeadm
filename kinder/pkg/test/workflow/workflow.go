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

package workflow

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

// Workflow represents a list of tasks to be executed during test workflow and related context
type Workflow struct {
	// Version of the workflow file
	// NB. We are enforcing version to be set only for future improvements of the config API
	// but currently there is no version management is in place.
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
	// one according to the precedence or the ARTIFACTS command line argument, env vars defined in the config
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
	Name string

	// Description of the task
	Description string

	// Dir allows to set the working directory for this tasks
	Dir string

	// Cmd to execute; it can be a literal or a template
	Cmd string

	// Args allows to set Cmd arguments; args can be a literal or a template
	Args []string

	// Force sets a task to be executed no matter of the result of the previous task.
	// This allows e.g. to define cleanup tasks to be always executed
	Force bool

	// Timeout for the current task, by default
	Timeout time.Duration
}

// NewWorkflow creates a new workflow as defined in a workflow file
func NewWorkflow(file string) (*Workflow, error) {
	// Checks if the worklow file exists
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

	// For each task
	for i, t := range w.Tasks {
		// if a task name is not defined, assign a default task name
		// otherwise prepend a prefix in order to get task logs ordered
		if t.Name == "" {
			t.Name = fmt.Sprintf("task-%d", i)
		} else {
			t.Name = fmt.Sprintf("task-%d-%s", i, t.Name)
		}

		// if a timeout is not defined, assign a default one
		// nb. we are assigning a fairly long timeout to avoid flakes in testgrid
		if t.Timeout == 0 {
			t.Timeout = time.Duration(5 * time.Minute)
		}

		// check if the task defines a cmd
		if t.Cmd == "" {
			return nil, errors.Errorf("invalid taskfile %s: task %q does not define a cmd", file, t.Name)
		}
	}

	// TODO: detect and resolve includes by expanding the top level Workflow

	return &w, nil
}

// Run executes a workflow
func (w *Workflow) Run(dryRun, verbose, exitOnError bool, artifacts string) (err error) {

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

	// Executes taskCmds
	for _, tcmd := range tcmds {
		fmt.Printf("# %s\n", tcmd.Name)
		fmt.Printf("%s\n\n", tcmd.CmdText)

		if !dryRun {
			err := taskCmdRunner.Run(tcmd, artifacts, verbose)
			if err != nil {
				fmt.Printf(" %v\n\n", err)

				if exitOnError {
					return nil
				}

				continue
			}

			fmt.Printf(" completed!\n\n")
		}
	}

	// If not dry running, prints task summary and dumps the junit_runner.xml file
	if !dryRun {
		taskCmdRunner.ReportSummary()

		if err := taskCmdRunner.DumpJUnitRunner(artifacts); err != nil {
			fmt.Printf("%v\n", err)
			return nil
		}
		fmt.Printf("see junit-runner.xml and task logs files for more details\n\n")
	}
	return nil
}
