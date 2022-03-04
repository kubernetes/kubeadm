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

package exec

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// HostCmd allows to run a command on the host
// By default, when the command is run it does not print any output generated during execution.
// See Silent, Stdin, RunWithEcho, RunAndCapture, Skip and DryRun for possible variations to the default behavior.
type HostCmd struct {
	command string
	args    []string
	env     []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

// NewHostCmd returns a new HostCmd to run a command on a host
func NewHostCmd(command string, args ...string) *HostCmd {
	return &HostCmd{
		command: command,
		args:    args,
	}
}

// Run execute the inner command on a kind(er) node
func (c *HostCmd) Run() error {
	return c.runInnnerCommand()
}

// RunWithEcho execute the inner command on a kind(er) node and echoes the command output to screen
func (c *HostCmd) RunWithEcho() error {
	c.stdout = os.Stderr
	c.stderr = os.Stdout
	return c.runInnnerCommand()
}

// RunAndCapture executes the inner command on a kind(er) node and return the output captured during execution
func (c *HostCmd) RunAndCapture() (lines []string, err error) {
	var buff bytes.Buffer
	c.stdout = &buff
	c.stderr = &buff
	err = c.runInnnerCommand()

	scanner := bufio.NewScanner(&buff)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())

	}
	return lines, err
}

// Stdin sets an io.Reader to be used for streaming data in input to the inner command
func (c *HostCmd) Stdin(in io.Reader) *HostCmd {
	c.stdin = in
	return c
}

// SetEnv sets env variables to be used when running the inner command
func (c *HostCmd) SetEnv(env ...string) *HostCmd {
	c.env = env
	return c
}

func (c *HostCmd) runInnnerCommand() error {
	// create the commands
	cmd := exec.Command(c.command, c.args...)

	// redirects flows if requested
	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}
	if c.stdout != nil {
		cmd.Stdout = c.stdout
	}
	if c.stderr != nil {
		cmd.Stderr = c.stderr
	}

	if len(c.env) > 0 {
		cmd.Env = c.env
	}

	// eventually print the proxy command, and then run the command to be executed
	log.Debugf("Running: %s", strings.Join(cmd.Args, " "))
	return cmd.Run()
}
