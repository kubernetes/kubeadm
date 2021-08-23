/*
Copyright 2021 The Kubernetes Authors.

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

// verify that all the imports have our preferred order.
// https://github.com/kubernetes/kubeadm/issues/2515

package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

func TestReorder(t *testing.T) {
	// determine input files
	match, err := filepath.Glob("testdata/*.input")
	if err != nil {
		t.Fatal(err)
	}

	lp := "k8s.io/kubeadm"
	// add larger examples
	match = append(match, "orderimports.go", "orderimports_test.go")

	for _, in := range match {
		out := in // for files where input and output are identical
		if strings.HasSuffix(in, ".input") {
			out = in[:len(in)-len(".input")] + ".golden"
		}
		runTest(t, lp, in, out)
		if in != out {
			// Check idempotence.
			runTest(t, lp, out, out)
		}
	}
}

func runTest(t *testing.T, lp string, in, out string) {
	var buf bytes.Buffer
	localPrefix = &lp
	err := processFile(in, nil, &buf, false)
	if err != nil {
		t.Error(err)
		return
	}

	expected, err := os.ReadFile(out)
	if err != nil {
		t.Error(err)
		return
	}

	if got := buf.Bytes(); !bytes.Equal(got, expected) {
		if *update {
			if in != out {
				if err := os.WriteFile(out, got, 0666); err != nil {
					t.Error(err)
				}
				return
			}
			// in == out: don't accidentally destroy input
			t.Errorf("WARNING: -update did not rewrite input file %s", in)
		}

		t.Errorf("(gofmt %s) != %s (see %s.gofmt)", in, out, in)
		d, err := diffWithReplaceTempFile(expected, got, in)
		if err == nil {
			t.Errorf("%s", d)
		}
		if err := os.WriteFile(in+".gofmt", got, 0666); err != nil {
			t.Error(err)
		}
	}
}

func TestBackupFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "gofmt_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	name, err := backupFile(filepath.Join(dir, "foo.go"), []byte("  package main"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Created: %s", name)
}

func TestDiff(t *testing.T) {
	if _, err := exec.LookPath("diff"); err != nil {
		t.Skipf("skip test on %s: diff command is required", runtime.GOOS)
	}
	in := []byte("first\nsecond\n")
	out := []byte("first\nthird\n")
	filename := "difftest.txt"
	b, err := diffWithReplaceTempFile(in, out, filename)
	if err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		b = bytes.ReplaceAll(b, []byte{'\r', '\n'}, []byte{'\n'})
	}

	bs := bytes.SplitN(b, []byte{'\n'}, 3)
	line0, line1 := bs[0], bs[1]

	if prefix := "--- difftest.txt.orig"; !bytes.HasPrefix(line0, []byte(prefix)) {
		t.Errorf("diff: first line should start with `%s`\ngot: %s", prefix, line0)
	}

	if prefix := "+++ difftest.txt"; !bytes.HasPrefix(line1, []byte(prefix)) {
		t.Errorf("diff: second line should start with `%s`\ngot: %s", prefix, line1)
	}

	want := `@@ -1,2 +1,2 @@
 first
-second
+third
`

	if got := string(bs[2]); got != want {
		t.Errorf("diff: got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceTempFilename(t *testing.T) {
	diff := []byte(`--- /tmp/tmpfile1	2017-02-08 00:53:26.175105619 +0900
+++ /tmp/tmpfile2	2017-02-08 00:53:38.415151275 +0900
@@ -1,2 +1,2 @@
 first
-second
+third
`)
	want := []byte(`--- path/to/file.go.orig	2017-02-08 00:53:26.175105619 +0900
+++ path/to/file.go	2017-02-08 00:53:38.415151275 +0900
@@ -1,2 +1,2 @@
 first
-second
+third
`)
	// Check path in diff output is always slash regardless of the
	// os.PathSeparator (`/` or `\`).
	sep := string(os.PathSeparator)
	filename := strings.Join([]string{"path", "to", "file.go"}, sep)
	got, err := replaceTempFilename(diff, filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("os.PathSeparator='%s': replacedDiff:\ngot:\n%s\nwant:\n%s", sep, got, want)
	}
}
