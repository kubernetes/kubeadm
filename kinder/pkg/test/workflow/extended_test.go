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
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func writeWorkflowFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func loadTestWorkflow(t *testing.T, file string) *Workflow {
	t.Helper()
	w, err := NewWorkflow(file)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func readReport(t *testing.T, artifacts string) junitTestSuite {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(artifacts, "junit_runner.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var suite junitTestSuite
	if err := xml.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	return suite
}

func requireFile(t *testing.T, path, content string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != content {
		t.Fatalf("%s: got %q, error %v, want %q", path, data, err, content)
	}
}

func requireAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected file %s (stat error %v)", path, err)
	}
}

// Execute real child workflows through the generated group's native references.
// No mock kinder executable, Docker, or network is involved.
func TestExtendedWorkflow(t *testing.T) {
	source, err := os.ReadFile("../../../ci/workflows/extended-latest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, failed := range []string{"", "upgrade-etcd-rollback", "upgrade-no-addon-config-maps", "both"} {
		t.Run("failed="+failed, func(t *testing.T) {
			dir := t.TempDir()
			file := writeWorkflowFile(t, dir, "extended.yaml", string(source))
			t.Setenv("WORKFLOW_TEST_EQUALS", "a=b=c")
			for _, name := range []string{"upgrade-etcd-rollback", "upgrade-no-addon-config-maps"} {
				code := 0
				if failed == name || failed == "both" {
					code = 1
				}
				writeWorkflowFile(t, dir, name+"-latest.yaml", fmt.Sprintf(`version: 1
vars:
  marker: %s
env:
  SCENARIO: '{{ .vars.marker }}'
tasks:
- name: run
  cmd: sh
  args:
  - -ec
  - |
    test "$WORKFLOW_TEST_EQUALS" = 'a=b=c'
    printf '%%s' "$SCENARIO" > "$ARTIFACTS/ran"
    exit %d
- name: dependent
  cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/dependent"']
- name: cleanup
  cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/cleaned"']
  force: true
`, name, code))
			}
			w := loadTestWorkflow(t, file)
			// Dry-running must not mutate templates or create artifacts.
			dryArtifacts := filepath.Join(dir, "dry")
			if err := w.Run(io.Discard, true, false, false, dryArtifacts); err != nil {
				t.Fatal(err)
			}
			requireAbsent(t, dryArtifacts)
			// Run the same loaded object twice with distinct artifact roots.
			for i := 0; i < 2; i++ {
				artifacts := filepath.Join(dir, fmt.Sprintf("artifacts %d", i))
				err := w.Run(io.Discard, false, false, false, artifacts)
				if (err != nil) != (failed != "") {
					t.Fatalf("Run() error = %v", err)
				}
				report := readReport(t, artifacts)
				wantFailures := 0
				for _, tc := range report.Cases {
					childFailed := failed == tc.Name || failed == "both"
					if childFailed {
						wantFailures++
					}
					if (tc.Failure != "") != childFailed || tc.Skipped != "" {
						t.Fatalf("unexpected scenario result: %+v", tc)
					}
					childArtifacts := filepath.Join(artifacts, tc.Name)
					requireFile(t, filepath.Join(childArtifacts, "ran"), tc.Name)
					requireFile(t, filepath.Join(childArtifacts, "cleaned"), "")
					child := readReport(t, childArtifacts)
					for _, task := range child.Cases {
						if task.ClassName != "kinder.test.workflow."+tc.Name {
							t.Fatalf("missing scenario namespace: %+v", task)
						}
					}
					if child.Tests != 3 || len(child.Cases) != 3 {
						t.Fatalf("unexpected child report: %+v", child)
					}
					if childFailed {
						requireAbsent(t, filepath.Join(childArtifacts, "dependent"))
						if child.Failures != 1 || child.Cases[1].Skipped == "" {
							t.Fatalf("lost child failure: %+v", child)
						}
					} else {
						requireFile(t, filepath.Join(childArtifacts, "dependent"), "")
						if child.Failures != 0 {
							t.Fatalf("unexpected child failure: %+v", child)
						}
					}
				}
				if report.Tests != 2 || len(report.Cases) != 2 || report.Failures != wantFailures {
					t.Fatalf("unexpected group report: %+v", report)
				}
			}
		})
	}
}

func TestWorkflowGroupValidation(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"mixed", "tasks: [{cmd: 'true'}]\nworkflows: [{name: child, workflow: leaf.yaml}]"},
		{"empty", "tasks: []"},
		{"null-child", "workflows: [null]"},
		{"null-task", "tasks: [null]"},
		{"duplicate", "workflows: [{name: child, workflow: leaf.yaml}, {name: child, workflow: leaf.yaml}]"},
		{"unsafe-name", "workflows: [{name: ../escape, workflow: leaf.yaml}]"},
		{"missing-file", "workflows: [{name: child, workflow: missing.yaml}]"},
		{"missing-path", "workflows: [{name: child}]"},
		{"negative-timeout", "workflows: [{name: child, workflow: leaf.yaml, timeout: -1s}]"},
		{"group-env", "env: {A: b}\nworkflows: [{name: child, workflow: leaf.yaml}]"},
		{"group-vars", "vars: {A: b}\nworkflows: [{name: child, workflow: leaf.yaml}]"},
		{"force-scenario", "workflows: [{name: child, workflow: leaf.yaml, force: true}]"},
		{"cycle", "workflows: [{name: child, workflow: group.yaml}]"},
		{"import-cycle", "tasks: [{import: group.yaml}]"},
		{"mixed-cycle", "workflows: [{name: child, workflow: import-parent.yaml}]"},
		{"import-group", "tasks: [{import: other-group.yaml}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeWorkflowFile(t, dir, "leaf.yaml", "version: 1\ntasks: [{cmd: 'true'}]\n")
			writeWorkflowFile(t, dir, "import-parent.yaml", "version: 1\ntasks: [{import: group.yaml}]\n")
			writeWorkflowFile(t, dir, "other-group.yaml", "version: 1\nworkflows: [{name: child, workflow: leaf.yaml}]\n")
			file := writeWorkflowFile(t, dir, "group.yaml", "version: 1\n"+tc.body+"\n")
			if _, err := NewWorkflow(file); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
	t.Run("symlink-cycle", func(t *testing.T) {
		dir := t.TempDir()
		file := writeWorkflowFile(t, dir, "group.yaml", "version: 1\nworkflows: [{name: child, workflow: alias.yaml}]\n")
		if err := os.Symlink(file, filepath.Join(dir, "alias.yaml")); err != nil {
			t.Fatal(err)
		}
		if _, err := NewWorkflow(file); err == nil {
			t.Fatal("symlink cycle accepted")
		}
	})
}

func TestWorkflowGroupPreflight(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowFile(t, dir, "first.yaml", `version: 1
tasks:
- cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/ran"']
`)
	writeWorkflowFile(t, dir, "bad.yaml", "version: 1\ntasks: [{cmd: '{{ .vars.missing }}'}]\n")
	w := loadTestWorkflow(t, writeWorkflowFile(t, dir, "group.yaml", `version: 1
workflows:
- name: first
  workflow: first.yaml
- name: bad
  workflow: bad.yaml
`))
	for _, dry := range []bool{true, false} {
		artifacts := t.TempDir()
		if err := w.Run(io.Discard, dry, false, false, artifacts); err == nil {
			t.Fatal("invalid child template accepted")
		}
		requireAbsent(t, filepath.Join(artifacts, "first", "ran"))
	}
}

func TestWorkflowNestedGroupsAndImports(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowFile(t, dir, "nested/tasks.yaml", `version: 1
vars: {value: imported}
tasks:
- cmd: sh
  args: [-ec, 'printf "%s" "{{ .vars.value }}" > "$ARTIFACTS/value"']
`)
	writeWorkflowFile(t, dir, "nested/leaf.yaml", "version: 1\nvars: {value: overridden}\ntasks: [{import: tasks.yaml}]\n")
	writeWorkflowFile(t, dir, "nested/group.yaml", "version: 1\nworkflows: [{name: inner, workflow: leaf.yaml}]\n")
	// Referencing the same workflow twice is allowed: only active cycles fail.
	w := loadTestWorkflow(t, writeWorkflowFile(t, dir, "group.yaml", "version: 1\nworkflows: [{name: a, workflow: nested/group.yaml}, {name: b, workflow: nested/group.yaml}]\n"))
	artifacts := t.TempDir()
	if err := w.Run(io.Discard, false, false, false, artifacts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a", "b"} {
		requireFile(t, filepath.Join(artifacts, name, "inner", "value"), "overridden")
		if r := readReport(t, filepath.Join(artifacts, name)); r.Tests != 1 || r.Failures != 0 {
			t.Fatalf("nested report: %+v", r)
		}
	}
}

// Lifecycle fixture: a task starts a real long-lived process in its task group.
func lifecycleWorkflow(t *testing.T, mode string) (*Workflow, string) {
	t.Helper()
	dir := t.TempDir()
	taskTimeout, scenarioTimeout := "30s", "30s"
	command := `sleep 30 &
    child=$!
    printf '%s' "$child" > "$ARTIFACTS/pid"
    touch "$ARTIFACTS/started"
    wait "$child"`
	if mode == "task-timeout" {
		taskTimeout = "200ms"
	}
	if mode == "scenario-timeout" {
		scenarioTimeout = "200ms"
	}
	if mode == "fail-fast" {
		command = "exit 1"
	}
	writeWorkflowFile(t, dir, "first.yaml", fmt.Sprintf(`version: 1
tasks:
- name: active
  cmd: sh
  args:
  - -ec
  - |
    %s
  timeout: %s
- name: dependent
  cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/dependent"']
- name: cleanup
  cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/cleaned"']
  force: true
  timeout: 2s
`, command, taskTimeout))
	writeWorkflowFile(t, dir, "second.yaml", `version: 1
tasks:
- cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/ran"']
`)
	file := writeWorkflowFile(t, dir, "group.yaml", fmt.Sprintf(`version: 1
workflows:
- name: first
  workflow: first.yaml
  timeout: %s
- name: second
  workflow: second.yaml
`, scenarioTimeout))
	return loadTestWorkflow(t, file), file
}

func waitForFile(t *testing.T, file string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(file); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", file)
}

func requireStoppedProcess(t *testing.T, pidFile string) {
	t.Helper()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	// An orphan may briefly be a zombie pending reaping by init on Linux.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "stat=").Output()
		if err != nil || strings.HasPrefix(strings.TrimSpace(string(state)), "Z") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("task descendant %d survived cancellation", pid)
}

func checkLifecycle(t *testing.T, artifacts string, secondSkipped bool) {
	t.Helper()
	requireFile(t, filepath.Join(artifacts, "first", "cleaned"), "")
	requireAbsent(t, filepath.Join(artifacts, "first", "dependent"))
	parent := readReport(t, artifacts)
	if parent.Tests != 2 || parent.Failures != 1 || len(parent.Cases) != 2 || (parent.Cases[1].Skipped != "") != secondSkipped {
		t.Fatalf("parent report: %+v", parent)
	}
	child := readReport(t, filepath.Join(artifacts, "first"))
	if child.Tests != 3 || child.Failures != 1 || len(child.Cases) != 3 || child.Cases[1].Skipped == "" || child.Cases[2].Failure != "" {
		t.Fatalf("child report: %+v", child)
	}
	if secondSkipped {
		requireAbsent(t, filepath.Join(artifacts, "second", "ran"))
	} else {
		requireFile(t, filepath.Join(artifacts, "second", "ran"), "")
	}
}

func TestWorkflowGroupLifecycle(t *testing.T) {
	for _, mode := range []string{"task-timeout", "scenario-timeout", "cancel", "fail-fast"} {
		t.Run(mode, func(t *testing.T) {
			w, _ := lifecycleWorkflow(t, mode)
			artifacts := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- w.runContext(ctx, io.Discard, false, false, mode == "fail-fast", artifacts) }()
			if mode == "cancel" {
				waitForFile(t, filepath.Join(artifacts, "first", "started"))
				cancel()
			}
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("expected failure")
				}
			case <-time.After(10 * time.Second):
				cancel()
				t.Fatal("workflow did not finish")
			}
			checkLifecycle(t, artifacts, mode == "cancel" || mode == "fail-fast")
			if mode != "fail-fast" {
				requireStoppedProcess(t, filepath.Join(artifacts, "first", "pid"))
			}
		})
	}
}

// Exercise actual SIGINT/SIGTERM delivery without signaling the go test process.
func TestWorkflowGroupSignals(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			_, file := lifecycleWorkflow(t, "cancel")
			artifacts := t.TempDir()
			cmd := exec.Command(os.Args[0], "-test.run=^TestWorkflowSignalHelper$")
			cmd.Env = append(os.Environ(), "KINDER_SIGNAL_TEST="+file, "KINDER_SIGNAL_ARTIFACTS="+artifacts)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			waitForFile(t, filepath.Join(artifacts, "first", "started"))
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { result <- cmd.Wait() }()
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("canceled workflow returned success")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("canceled workflow hung")
			}
			checkLifecycle(t, artifacts, true)
			requireStoppedProcess(t, filepath.Join(artifacts, "first", "pid"))
		})
	}
}

func TestWorkflowSignalHelper(t *testing.T) {
	file := os.Getenv("KINDER_SIGNAL_TEST")
	if file == "" {
		return
	}
	w, err := NewWorkflow(file)
	if err == nil {
		err = w.Run(io.Discard, false, false, false, os.Getenv("KINDER_SIGNAL_ARTIFACTS"))
	}
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestWorkflowGroupCanceledBeforeStart(t *testing.T) {
	w, _ := lifecycleWorkflow(t, "cancel")
	artifacts := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.runContext(ctx, io.Discard, false, false, false, artifacts); err != context.Canceled {
		t.Fatalf("expected cancellation, got %v", err)
	}
	report := readReport(t, artifacts)
	if report.Tests != 2 || len(report.Cases) != 2 {
		t.Fatalf("report: %+v", report)
	}
	for _, tc := range report.Cases {
		if tc.Skipped == "" {
			t.Fatalf("scenario was not skipped: %+v", tc)
		}
		requireAbsent(t, filepath.Join(artifacts, tc.Name, "junit_runner.xml"))
	}
	requireAbsent(t, filepath.Join(artifacts, "first", "started"))
	requireAbsent(t, filepath.Join(artifacts, "first", "cleaned"))
	requireAbsent(t, filepath.Join(artifacts, "second", "ran"))
}

func TestWorkflowCleanupTimeout(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowFile(t, dir, "leaf.yaml", `version: 1
tasks:
- cmd: sh
  args: [-ec, 'exit 1']
- name: stuck-cleanup
  cmd: sh
  args: [-ec, 'exec sleep 30']
  timeout: 100ms
  force: true
- name: final-cleanup
  cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/cleaned"']
  force: true
`)
	w := loadTestWorkflow(t, writeWorkflowFile(t, dir, "group.yaml", "version: 1\nworkflows: [{name: first, workflow: leaf.yaml}, {name: second, workflow: leaf.yaml}]\n"))
	artifacts := t.TempDir()
	if err := w.Run(io.Discard, false, false, false, artifacts); err == nil {
		t.Fatal("expected failure")
	}
	for _, name := range []string{"first", "second"} {
		requireFile(t, filepath.Join(artifacts, name, "cleaned"), "")
		report := readReport(t, filepath.Join(artifacts, name))
		if report.Failures != 2 || report.Tests != 3 {
			t.Fatalf("report: %+v", report)
		}
	}
}

func TestWorkflowTaskCompatibility(t *testing.T) {
	dir := t.TempDir()
	file := writeWorkflowFile(t, dir, "tasks.yaml", `version: 1
tasks:
- cmd: sh
  args: [-ec, 'exit 1']
  ignoreError: true
- cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/after-ignored"']
- cmd: kinder-intentionally-nonexistent-command
- cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/dependent"']
- cmd: sh
  args: [-ec, 'touch "$ARTIFACTS/cleaned"']
  force: true
`)
	w := loadTestWorkflow(t, file)
	artifacts := t.TempDir()
	if err := w.Run(io.Discard, false, false, true, artifacts); err == nil {
		t.Fatal("expected start failure")
	}
	requireFile(t, filepath.Join(artifacts, "after-ignored"), "")
	requireFile(t, filepath.Join(artifacts, "cleaned"), "")
	requireAbsent(t, filepath.Join(artifacts, "dependent"))
	report := readReport(t, artifacts)
	if report.Tests != 5 || report.Failures != 1 || len(report.Cases) != 5 || report.Cases[3].Skipped == "" {
		t.Fatalf("report: %+v", report)
	}
}

func TestExistingCIWorkflowConfigurations(t *testing.T) {
	files, err := filepath.Glob("../../../ci/workflows/*.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("workflow discovery: files=%d error=%v", len(files), err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			if _, err := NewWorkflow(file); err != nil {
				t.Fatal(err)
			}
		})
	}
}
