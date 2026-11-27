package main

// ADR-001-A7 tests V-6 and V-7 (TICK-1188), run through the CLI exactly as
// make check-pins runs it.
//
// V-6 (A7.2): the scan is only as good as the govulncheck it runs. docs/pins.md
// records the version, Makefile installs it for make vulncheck, and the
// workflow runs it in the two scan jobs. If any copy drifts alone, or a job
// gains or loses a govulncheck@ line, the gate can be green on a scanner
// nobody reviewed.
//
// V-7 (contract rule 7, as amended): ./... stops at a nested go.mod, and
// vendored source is not checked against go.sum, so either takes code out of
// the scan without touching the scanned go.mod.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	vulnWorkflow = ".github/workflows/govulncheck.yml"
	vulnLine     = "go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./..."
)

func TestOneGovulncheckVersion(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		mustCLI(t, repoRoot, 0, "")
	})
	t.Run("copy in step", func(t *testing.T) {
		mustCLI(t, vulnCopy(t), 0, "")
	})
	for _, tc := range []struct{ name, file, old, new, want string }{
		{"only in the workflow", vulnWorkflow, "govulncheck@v1.9.0", "govulncheck@v1.9.1", vulnWorkflow + ": job vulncheck-root runs govulncheck@v1.9.1"},
		{"only in Makefile", "Makefile", "GOVULNCHECK_VERSION := v1.9.0", "GOVULNCHECK_VERSION := v1.9.1", "Makefile: GOVULNCHECK_VERSION is v1.9.1"},
		{"only in docs/pins.md", "docs/pins.md", "| v1.9.0 | `make vulncheck`", "| v1.9.1 | `make vulncheck`", "docs/pins.md: the govulncheck row pins v1.9.1"},
		{"workflow on latest", vulnWorkflow, "govulncheck@v1.9.0", "govulncheck@latest", "runs govulncheck@latest"},
		{"version from an expression", vulnWorkflow, "govulncheck@v1.9.0", "govulncheck@${{ env.V }}", "runs govulncheck@${{"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := vulnCopy(t)
			replaceOnce(t, dir, tc.file, tc.old, tc.new)
			mustCLI(t, dir, 1, tc.want, "ADR-001-A7 V-6")
		})
	}

	t.Run("a third job with its own line", func(t *testing.T) {
		dir := vulnCopy(t)
		appendTo(t, dir, vulnWorkflow, `
  vulncheck-extra:
    runs-on: ubuntu-latest
    steps:
      - run: `+vulnLine+`
`)
		mustCLI(t, dir, 1, vulnWorkflow+": job vulncheck-extra runs govulncheck@ but is not a scan job", "ADR-001-A7 V-6")
	})
	t.Run("a third job running it under another key", func(t *testing.T) {
		dir := vulnCopy(t)
		appendTo(t, dir, vulnWorkflow, `
  vulncheck-extra:
    runs-on: ubuntu-latest
    steps:
      - name: sneak
        shell: bash
        run: |
          echo scanning
          `+vulnLine+`
`)
		mustCLI(t, dir, 1, "job vulncheck-extra runs govulncheck@ but is not a scan job")
	})
	t.Run("a scan job with two lines", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, vulnWorkflow, "        run: "+vulnLine+"\n",
			"        run: "+vulnLine+"\n      - run: "+vulnLine+"\n")
		mustCLI(t, dir, 1, vulnWorkflow+": job vulncheck-root has 2 govulncheck@ lines, want exactly 1", "ADR-001-A7 V-6")
	})
	t.Run("two invocations in one run block", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, vulnWorkflow, "        run: "+vulnLine+"\n",
			"        run: |\n          "+vulnLine+"\n          "+vulnLine+"\n")
		mustCLI(t, dir, 1, "job vulncheck-root has 2 govulncheck@ lines, want exactly 1")
	})
	t.Run("a scan job with none", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, vulnWorkflow, "        run: "+vulnLine+"\n", "        run: make vulncheck\n")
		mustCLI(t, dir, 1, vulnWorkflow+": job vulncheck-root has 0 govulncheck@ lines, want exactly 1", "ADR-001-A7 V-6")
	})
	t.Run("a scan job removed", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, vulnWorkflow, "  vulncheck-temporal-server:\n", "  vulncheck-ts:\n")
		mustCLI(t, dir, 1, "job vulncheck-temporal-server has 0 govulncheck@ lines", "job vulncheck-ts runs govulncheck@ but is not a scan job")
	})
	t.Run("Makefile line missing", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, "Makefile", "GOVULNCHECK_VERSION := v1.9.0", "GOVULNCHECK_VERSION = v1.9.0")
		mustCLI(t, dir, 1, "Makefile: 0 GOVULNCHECK_VERSION := lines, want exactly 1", "ADR-001-A7 V-6")
	})
	t.Run("Makefile line twice", func(t *testing.T) {
		dir := vulnCopy(t)
		appendTo(t, dir, "Makefile", "\nGOVULNCHECK_VERSION := v1.9.1\n")
		mustCLI(t, dir, 1, "Makefile: 2 GOVULNCHECK_VERSION := lines, want exactly 1")
	})
	t.Run("pins row missing", func(t *testing.T) {
		dir := vulnCopy(t)
		replaceOnce(t, dir, "docs/pins.md", "| `govulncheck` (", "| `govuln-check` (")
		mustCLI(t, dir, 1, "docs/pins.md: 0 govulncheck rows, want exactly 1", "ADR-001-A7 V-6")
	})
	// A missing copy cannot be compared: the check cannot run (exit 2), it
	// never passes.
	for _, f := range []string{"Makefile", vulnWorkflow} {
		t.Run(f+" missing", func(t *testing.T) {
			dir := vulnCopy(t)
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
				t.Fatal(err)
			}
			mustCLI(t, dir, 2, "check-pins: ")
		})
	}
	t.Run("workflow not YAML", func(t *testing.T) {
		dir := vulnCopy(t)
		put(t, dir, vulnWorkflow, "jobs: [\n")
		mustCLI(t, dir, 1, vulnWorkflow+": not YAML")
	})
}

func TestModuleLayoutKeepsCodeInTheScan(t *testing.T) {
	for _, tc := range []struct{ file, want string }{
		{"internal/foo/go.mod", "internal/foo/go.mod: a tracked go.mod outside the module set takes its directory out of ./... (ADR-001-A7 V-7)"},
		{"temporal-server/internal/x/go.mod", "temporal-server/internal/x/go.mod: a tracked go.mod outside the module set"},
		{"tools/ci/checkpins/go.mod", "tools/ci/checkpins/go.mod: a tracked go.mod outside the module set"},
		{"vendor/modules.txt", "vendor/modules.txt: a tracked vendor/ path is not checked against go.sum (ADR-001-A7 V-7)"},
		{"temporal-server/vendor/modules.txt", "temporal-server/vendor/modules.txt: a tracked vendor/ path"},
		{"internal/pkg/vendor/a.go", "internal/pkg/vendor/a.go: a tracked vendor/ path"},
	} {
		t.Run("tracked "+tc.file, func(t *testing.T) {
			dir := vulnCopy(t)
			put(t, dir, tc.file, "module example.com/x\n\ngo 1.27.0\n")
			gitAdd(t, dir, tc.file)
			mustCLI(t, dir, 1, tc.want)
		})
	}
	for _, f := range []string{
		"tools/ci/checkpins/testdata/x/go.mod",
		"internal/foo/testdata/vendor/modules.txt",
		"docs/go.mod",
		"tools/navalint/go.mod",
		"tools/workflowcheck/go.mod",
		"tools/navalint/vendor/modules.txt",
	} {
		t.Run("allowed "+f, func(t *testing.T) {
			dir := vulnCopy(t)
			put(t, dir, f, "module example.com/x\n\ngo 1.27.0\n")
			gitAdd(t, dir, f)
			mustCLI(t, dir, 0, "")
		})
	}
	t.Run("untracked go.mod does not count", func(t *testing.T) {
		dir := vulnCopy(t)
		put(t, dir, "scratch/go.mod", "module example.com/x\n\ngo 1.27.0\n")
		mustCLI(t, dir, 0, "")
	})
}

// vulnCopy is goCopy plus the three files that carry the govulncheck version,
// all added to the index.
func vulnCopy(t *testing.T) string {
	t.Helper()
	dir := goCopy(t)
	files := []string{"Makefile", "docs/pins.md", vulnWorkflow}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		put(t, dir, f, string(data))
	}
	gitAdd(t, dir, files...)
	return dir
}

// replaceOnce replaces the first occurrence of old in dir/f; old must occur.
func replaceOnce(t *testing.T, dir, f, old, new string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(f))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s lacks %q", f, old)
	}
	put(t, dir, f, strings.Replace(string(data), old, new, 1))
}

// mustCLI runs check-pins on dir with its own docs/pins.md, as make
// check-pins does, and wants exit code and every want in stderr.
func mustCLI(t *testing.T, dir string, code int, want ...string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	got := cli([]string{"-root", dir}, &stdout, &stderr)
	if got != code {
		t.Fatalf("exit %d, want %d\nstderr:\n%s", got, code, stderr.String())
	}
	for _, w := range want {
		if w != "" && !strings.Contains(stderr.String(), w) {
			t.Errorf("stderr lacks %q\n%s", w, stderr.String())
		}
	}
}
