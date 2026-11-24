/*
Copyright 2026 The Kubernetes Authors.

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
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

// Run executes a workflow or scenario group. One cancellation context covers the
// entire tree, including the intervals between commands and child workflows.
func (w *Workflow) Run(out io.Writer, dryRun, verbose, exitOnError bool, artifacts string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return w.runContext(ctx, out, dryRun, verbose, exitOnError, artifacts)
}

func (w *Workflow) runContext(ctx context.Context, out io.Writer, dryRun, verbose, exitOnError bool, artifacts string) error {
	plan, err := w.prepare(artifacts, dryRun, "")
	if err != nil {
		return err
	}
	return plan.run(ctx, out, dryRun, verbose, exitOnError)
}

// preparedWorkflow validates the whole tree before any command is executed.
// Each leaf builds its own environment; parent Vars/Env are never merged into it.
type preparedWorkflow struct {
	artifacts string
	scope     string
	commands  []*taskCmd
	scenarios []preparedScenario
}

type preparedScenario struct {
	name     string
	timeout  time.Duration
	workflow *preparedWorkflow
}

func (w *Workflow) prepare(artifacts string, dryRun bool, scope string) (*preparedWorkflow, error) {
	builder, err := newTaskCmdBuilder(w)
	if err != nil {
		return nil, err
	}
	if artifacts == "" {
		artifacts = builder.env["ARTIFACTS"]
	}
	if artifacts == "" {
		if dryRun {
			artifacts = "<tmp-folder>"
		} else {
			artifacts, err = os.MkdirTemp(".", "kinder-test-workflow")
			if err != nil {
				return nil, err
			}
		}
	}
	// Commands with a different working directory still need an absolute path.
	if !dryRun {
		artifacts, err = filepath.Abs(artifacts)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(artifacts, 0755); err != nil {
			return nil, err
		}
	}
	builder.env["ARTIFACTS"] = artifacts
	plan := &preparedWorkflow{artifacts: artifacts, scope: scope}
	for _, task := range w.Tasks {
		cmd, err := builder.build(task, false)
		if err != nil {
			return nil, err
		}
		plan.commands = append(plan.commands, cmd)
	}
	for _, scenario := range w.Workflows {
		childScope := scenario.Name
		if scope != "" {
			childScope = scope + "." + childScope
		}
		child, err := scenario.child.prepare(filepath.Join(artifacts, scenario.Name), dryRun, childScope)
		if err != nil {
			return nil, errors.Wrapf(err, "preparing scenario %q", scenario.Name)
		}
		plan.scenarios = append(plan.scenarios, preparedScenario{scenario.Name, scenario.Timeout.Duration, child})
	}
	return plan, nil
}

func (p *preparedWorkflow) run(ctx context.Context, out io.Writer, dryRun, verbose, exitOnError bool) error {
	runner := newTaskCmdRunner()
	if p.scope != "" {
		runner.suite.Name += "." + p.scope
	}
	foundError := false
	for _, scenario := range p.scenarios {
		fmt.Fprintf(out, "# workflow %s\n", scenario.name)
		if !dryRun && (ctx.Err() != nil || (foundError && exitOnError)) {
			reason := "skipping because a predecessor scenario failed"
			if ctx.Err() != nil {
				reason = "skipping because workflow was canceled or its deadline expired"
			}
			_ = runner.registerTestCase(scenario.name, withSkipped(reason))
			foundError = true
			continue
		}
		childCtx := ctx
		cancel := func() {}
		if !dryRun && scenario.timeout > 0 {
			childCtx, cancel = context.WithTimeout(ctx, scenario.timeout)
		}
		start := time.Now()
		err := scenario.workflow.run(childCtx, out, dryRun, verbose, exitOnError)
		// A deadline that expires during forced cleanup must still fail the scenario.
		if err == nil {
			err = childCtx.Err()
		}
		cancel()
		if dryRun {
			if err != nil {
				return err
			}
			continue
		}
		options := []testCaseOption{withDuration(time.Since(start))}
		if err != nil {
			options = append(options, withFailure(err.Error()))
			foundError = true
		}
		_ = runner.registerTestCase(scenario.name, options...)
	}
	for _, cmd := range p.commands {
		fmt.Fprintf(out, "# %s\n%s\n\n", cmd.Name, cmd.CmdText)
		if dryRun {
			continue
		}
		// Forced tasks are cleanup: they may run after failure or cancellation,
		// with their own finite timeout. No new scenario starts after cancellation.
		if err := runner.Run(ctx, cmd, p.artifacts, verbose); err != nil {
			foundError = true
			fmt.Fprintf(out, " %v\n\n", err)
		} else {
			fmt.Fprint(out, " completed!\n\n")
		}
	}
	if dryRun {
		return nil
	}
	if err := runner.DumpJUnitRunner(p.artifacts); err != nil {
		return err
	}
	runner.ReportSummary()
	fmt.Fprintf(out, "see %s and task logs for more details\n\n", filepath.Join(p.artifacts, "junit_runner.xml"))
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if foundError {
		return errors.New("failed executing the workflow")
	}
	return nil
}
