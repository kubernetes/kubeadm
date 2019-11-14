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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/template"

	"github.com/pkg/errors"

	"k8s.io/kubeadm/kinder/pkg/extract"
)

// taskCmd defines a command that will execute the action defined in task action
// along with all the properties/setting that define how this command should behave
// in the workflow
type taskCmd struct {
	*Task
	Cmd     *exec.Cmd
	CmdText string
}

// taskCmdBuilder provide support for creating taskCmd, taking care of the context
// defined by Vars and Env variables
type taskCmdBuilder struct {
	env  map[string]string
	vars map[string]string
}

// newTaskCmdBuilder return a new taskCmdBuilder
func newTaskCmdBuilder(w *Workflow) (c *taskCmdBuilder, err error) {
	c = &taskCmdBuilder{
		env:  map[string]string{},
		vars: map[string]string{},
	}

	// loads OS environment variables into the taskCmdBuilder context
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		name := pair[0]
		value := pair[1]
		c.env[name] = value
	}

	// process vars defined in the workflow
	if w.Vars != nil {
		for n, v := range w.Vars {
			c.vars[n], err = c.expand(v)
			if err != nil {
				return nil, errors.Wrapf(err, "error expanding the %q var", n)
			}
		}
	}

	// process additional environment variables defined in the workflow
	if w.Env != nil {
		for n, v := range w.Env {
			c.env[n], err = c.expand(v)
			if err != nil {
				return nil, errors.Wrapf(err, "error expanding the %q env var", n)
			}
		}
	}

	return c, nil
}

// defines a list of custom utility functions that can be used in workflow templates
var funcMap = template.FuncMap{
	"resolve": extract.ResolveLabel, // e.g. used in templates >> stable: '{{ resolve "release/stable" }}' or {{ "ci/latest" | resolve }}
}

// expand takes a string that might contain a golang template and process it
// using Vars and Env variables as a context
func (c *taskCmdBuilder) expand(text string) (string, error) {
	templ, err := template.New("").Option("missingkey=error").Funcs(funcMap).Parse(text)
	if err != nil {
		return "", errors.Wrapf(err, "%q is not a valid expression", text)
	}

	var b bytes.Buffer
	if err = templ.Execute(&b, map[string]interface{}{
		"env":  c.env,
		"vars": c.vars,
	}); err != nil {
		return "", errors.Wrapf(err, "expression %q returned an error", text)
	}
	return b.String(), nil
}

// build creates a taskCmd
func (c *taskCmdBuilder) build(t *Task, verbose bool) (tcmd *taskCmd, err error) {
	// expand golang templates that might exists in the cmd and/or into the args
	t.Cmd, err = c.expand(t.Cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "error expanding cmd for task %q", t.Name)
	}
	for n, v := range t.Args {
		t.Args[n], err = c.expand(v)
		if err != nil {
			return nil, errors.Wrapf(err, "error expanding args[%d] for task %q", n, t.Name)
		}
	}

	// creates the command
	cmd := exec.Command(t.Cmd, t.Args...)

	// store a textual representation of the command to be used in logs/output
	cmdText := fmt.Sprintf("%s %s", t.Cmd, strings.Join(t.Args, " "))

	// set the working dir if different from the current one
	if t.Dir != "" {
		cmd.Dir = t.Dir
	}

	// set the environment variables for the command
	for k, v := range c.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// sets the command in order to have a gid that will allows to identify
	// all the child process eventually created by the testCmd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return &taskCmd{
		Task:    t,
		Cmd:     cmd,
		CmdText: cmdText,
	}, nil
}
