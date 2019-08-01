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
Package cmd provide the ProxyCmd utility that can be used for running commands on a kind(er) node.

Each ProxyCmd is bound to the node where the inner command should be executed, and by default
the command is printed to stdout before execution; to enable colorized print of the command text, that can help in debugging,
please set the KINDER_COLORS environment variable to ON.

By default, when the command is run it does not print any output generated during execution.

See Silent, Stdin, RunWithEcho, RunAndCapture, Skip and DryRun for possible variations to the default behavior.
*/
package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"k8s.io/kubeadm/kinder/pkg/cluster/cmd/colors"
)

// ProxyCmd allows to run a command on a kind(er) node
type ProxyCmd struct {
	node    string
	command string
	args    []string
	silent  bool
	dryRun  bool
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

// NewProxyCmd returns a new ProxyCmd to run a command on a kind(er) node
func NewProxyCmd(node, command string, args ...string) *ProxyCmd {
	return &ProxyCmd{
		node:    node,
		command: command,
		args:    args,
		silent:  false,
		dryRun:  false,
	}
}

// Run execute the inner command on a kind(er) node
func (c *ProxyCmd) Run() error {
	return c.runInnnerCommand()
}

// RunWithEcho execute the inner command on a kind(er) node and echoes the command output to screen
func (c *ProxyCmd) RunWithEcho() error {
	c.stdout = os.Stderr
	c.stderr = os.Stdout
	return c.runInnnerCommand()
}

// RunAndCapture executes the inner command on a kind(er) node and return the output captured during execution
func (c *ProxyCmd) RunAndCapture() (lines []string, err error) {
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
func (c *ProxyCmd) Stdin(in io.Reader) *ProxyCmd {
	c.stdin = in
	return c
}

// Silent instructs the proxy command to not the command text to stdout before execution
func (c *ProxyCmd) Silent() *ProxyCmd {
	c.silent = true
	return c
}

// DryRun instruct the proxy command to print the inner command text instead of running it.
func (c *ProxyCmd) DryRun() *ProxyCmd {
	c.dryRun = true
	return c
}

func (c *ProxyCmd) runInnnerCommand() error {
	// define the proxy command used to pass the command to the node container
	command := "docker"

	// prepare the args
	args := []string{
		"exec",
		// "--privileged"
	}

	// if it is requested to pipe data to the command itself, instruct docker exec to Keep STDIN open even if not attached
	if c.stdin != nil {
		args = append(args, "-i")
	}

	// add args for defining the target node container and the command to be executed
	args = append(
		args,
		c.node,
		c.command,
	)

	// adds the args for the command to be executed
	args = append(
		args,
		c.args...,
	)

	// create the proxy commands
	cmd := exec.Command(command, args...)

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

	// if not silent, prints the screen echo for the command to be executed
	if !c.silent {
		prompt := colors.Prompt(fmt.Sprintf("%s:$ ", c.node))
		command := colors.Command(fmt.Sprintf("%s %s", c.command, strings.Join(c.args, " ")))
		fmt.Printf("\n%s%s\n", prompt, command)
	}

	// if we are dry running, eventually print the proxy command and then exit
	if c.dryRun {
		log.Debugf("Dry-running: %v", cmd.Args)
		return nil
	}

	// eventually print the proxy command, and then run the command to be executed
	log.Debugf("Running: %v", cmd.Args)
	return cmd.Run()
}
