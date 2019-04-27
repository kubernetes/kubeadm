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
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

// taskCmdRunner defines all the info of a runner responsible for executing as
// sequence of taskCmd, handling failure, cancellation, timeouts and for generating
// and/or collecting all the workflow artifacts (junit_runner.xml, task logs, etc)
type taskCmdRunner struct {
	start    time.Time
	suite    junitTestSuite
	failed   bool
	canceled bool
	timedOut bool
}

// junitTestSuite implements junit TestSuite standard object
type junitTestSuite struct {
	XMLName  xml.Name `xml:"testsuite"`
	Failures int      `xml:"failures,attr"`
	Tests    int      `xml:"tests,attr"`
	Time     float64  `xml:"time,attr"`
	Cases    []junitTestCase
}

// junitTestCase implements junit TestCase standard object
type junitTestCase struct {
	XMLName   xml.Name `xml:"testcase"`
	ClassName string   `xml:"classname,attr"`
	Name      string   `xml:"name,attr"`
	Time      float64  `xml:"time,attr"`
	Failure   string   `xml:"failure,omitempty"`
	Skipped   string   `xml:"skipped,omitempty"`
}

// newTaskCmdRunner returns a new taskCmdRunner
func newTaskCmdRunner() *taskCmdRunner {
	return &taskCmdRunner{
		start: time.Now(),
		suite: junitTestSuite{},
	}
}

// Run a taskCmd
func (c *taskCmdRunner) Run(t *taskCmd, artifacts string, verbose bool) error {
	start := time.Now()

	// unless the cmd execution is forced, check if the taskCmd should be skipped because one of
	// the previous taskCmd failed, timedOut or was canceled.
	// if this is the case record test case as skipped and exits with error
	if !t.Force {
		if c.failed {
			return c.registerTestCase(t.Name, withSkipped("skipping because a predecessor task failed"))
		}
		if c.timedOut {
			return c.registerTestCase(t.Name, withSkipped("skipping because a predecessor task timed-out"))
		}
		if c.canceled {
			return c.registerTestCase(t.Name, withSkipped("skipping because task workflow was canceled by the user"))
		}
	}

	// creates a channel for handling command cancellation
	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGINT, syscall.SIGTERM)

	// sets Stdout and Stderr for the command.
	// please note that the command output will go on files by default,
	// and it will be echoed on video only if specifically requested
	taskLog := filepath.Join(artifacts, fmt.Sprintf("%s-log.txt", t.Name))
	writer, err := os.Create(taskLog)
	if err != nil {
		return errors.Wrapf(err, "error creating %q log file", taskLog)
	}

	t.Cmd.Stdout = writer
	t.Cmd.Stderr = writer

	if verbose {
		t.Cmd.Stdout = io.MultiWriter(writer, os.Stdout)
		t.Cmd.Stderr = io.MultiWriter(writer, os.Stderr)
	}

	// outputs a command overview before executing it
	writer.WriteString(fmt.Sprintf("%s\n", strings.Repeat("-", 80)))
	writer.WriteString(fmt.Sprintf("%s\n", t.Name))
	if t.Description != "" {
		writer.WriteString(fmt.Sprintf("%s\n", t.Description))
	}
	writer.WriteString(fmt.Sprintf("command : %s\n", t.CmdText))
	writer.WriteString(fmt.Sprintf("timeout : %s\n", t.Timeout))
	writer.WriteString(fmt.Sprintf("force   : %v\n", t.Force))
	writer.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("-", 80)))

	// starts the command
	if err := t.Cmd.Start(); err != nil {
		// keeps track of this failure type to block execution of following TestCmd
		c.failed = true

		// record test case timeout and exits with error
		return c.registerTestCase(t.Name, withFailure(err.Error()), withDuration(time.Since(start)))
	}

	// starts a go ruting responsible for waiting the command completes
	result := make(chan error, 1)
	go func() {
		result <- t.Cmd.Wait()
	}()

	// Wait for one of:
	// - the command completes
	// - the command is canceled
	// - the timeout is reached
	select {
	case err := <-result:
		// if the command completed without an error or if we are ignoring errors, record the test case success and exit
		if err == nil || t.IgnoreError {
			// record test case timeout as success
			return c.registerTestCase(t.Name,
				withDuration(time.Since(start)),
			)
		}
		// keeps track of this failure type to block execution of following TestCmd
		c.failed = true

		// cleanup command process and its child, if any
		cleanup(t.Cmd)

		// otherwise record test case failure and exits with error
		return c.registerTestCase(t.Name,
			withFailure(err.Error()),
			withDuration(time.Since(start)),
		)

	case <-cancel:
		// keeps track of this failure type to block execution of following TestCmd
		c.canceled = true

		// cleanup command process and its child, if any
		cleanup(t.Cmd)

		// record test case cancellation and exits with error
		return c.registerTestCase(t.Name,
			withFailure("task was canceled by the user"),
			withDuration(time.Since(start)),
		)

	case <-time.After(t.Timeout):
		// keeps track of this failure type to block execution of following TestCmd
		c.timedOut = true

		// cleanup command process and its child, if any
		cleanup(t.Cmd)

		// record test case timeout and exits with error
		return c.registerTestCase(t.Name,
			withFailure(fmt.Sprintf("timeout. task did not completed in less than %s as expected", t.Timeout)),
			withDuration(time.Since(start)),
		)
	}
}

// ReportSummary prints a summary of executed task
func (c *taskCmdRunner) ReportSummary() {
	total := c.suite.Tests
	skipped := 0
	for _, t := range c.suite.Cases {
		if t.Skipped != "" {
			skipped++
		}
	}
	run := total - skipped
	failures := c.suite.Failures
	passed := run - failures

	fmt.Printf("Ran %d of %d tasks in %.3f seconds\n", run, total, c.suite.Time)
	if failures > 0 {
		fmt.Printf("FAIL! -- %d tasks Passed | %d Failed | %d Skipped\n\n", passed, failures, skipped)
		return
	}
	fmt.Printf("SUCCESS! -- %d tasks Passed | %d Failed | %d Skipped\n\n", passed, failures, skipped)
}

// DumpJUnitRunner writes a report of executed tasks as a junit file
func (c *taskCmdRunner) DumpJUnitRunner(artifacts string) error {
	// sets test suite duration
	c.suite.Time = time.Since(c.start).Seconds()

	// marshal test suite into the junit_runner.xml file
	out, err := xml.MarshalIndent(&c.suite, "", "    ")
	if err != nil {
		return errors.Wrapf(err, "error marshaling test suite results")
	}
	file := filepath.Join(artifacts, "junit_runner.xml")
	f, err := os.Create(file)
	if err != nil {
		return errors.Wrapf(err, "error creating %s", file)
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return errors.Wrapf(err, "error writing XML header to %s", file)
	}
	if _, err := f.Write(out); err != nil {
		return errors.Wrapf(err, "error writing XML data to %s", file)
	}

	return nil
}

type testCaseOption func(*junitTestCase)

func withDuration(duration time.Duration) testCaseOption {
	return func(t *junitTestCase) {
		t.Time = duration.Seconds()
	}
}
func withFailure(message string) testCaseOption {
	return func(t *junitTestCase) {
		t.Failure = message
	}
}

func withSkipped(message string) testCaseOption {
	return func(t *junitTestCase) {
		t.Skipped = message
	}
}

// registerTestCase register task output as a test case result
func (c *taskCmdRunner) registerTestCase(name string, options ...testCaseOption) error {
	tc := &junitTestCase{
		ClassName: "kinder.test.workflow",
		Name:      name,
	}

	for _, option := range options {
		option(tc)
	}

	c.suite.Cases = append(c.suite.Cases, *tc)
	c.suite.Tests++
	if tc.Failure != "" {
		c.suite.Failures++
		return errors.New(tc.Failure)
	}

	if tc.Skipped != "" {
		return errors.New(tc.Skipped)
	}

	return nil
}

// cleanup tries to ensure a cmdtask is properly closed
func cleanup(cmd *exec.Cmd) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	/* temporary disabled to better investigate test-grid failures
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		cmd.Process.Kill()
	}

	if err := syscall.Kill(-pgid, syscall.SIGABRT); err == nil {
		return
	}

	syscall.Kill(-pgid, syscall.SIGTERM)
	*/

	cmd.Process.Kill()
}
