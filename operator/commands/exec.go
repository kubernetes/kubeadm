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

package commands

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
)

type cmd struct {
	command string
	args    []string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

func newCmd(command string, args ...string) *cmd {
	return &cmd{
		command: command,
		args:    args,
	}
}

func (c *cmd) Run() error {
	return c.runInnnerCommand()
}

func (c *cmd) RunWithEcho() error {
	c.stdout = os.Stderr
	c.stderr = os.Stdout
	return c.runInnnerCommand()
}

func (c *cmd) RunAndCapture() (lines []string, err error) {
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

func (c *cmd) Stdin(in io.Reader) *cmd {
	c.stdin = in
	return c
}

func (c *cmd) runInnnerCommand() error {
	cmd := exec.Command(c.command, c.args...)

	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}
	if c.stdout != nil {
		cmd.Stdout = c.stdout
	}
	if c.stderr != nil {
		cmd.Stderr = c.stderr
	}

	return cmd.Run()
}
