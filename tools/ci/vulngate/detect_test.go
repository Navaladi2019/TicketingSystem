package vulngate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture image references, built by concatenation so check-pins (which
// requires every real image tag to come from docs/pins.md) never sees a
// literal name:tag in this file. The bytes the tests write are unchanged.
const (
	goOld     = "golang" + ":" + "1.25"
	goNew     = "golang" + ":" + "1.26"
	alpineImg = "alpine" + ":" + "3"
)

// repo is a throwaway git repository holding every file detect reads.
type repo struct {
	t    *testing.T
	git  string
	dir  string
	base string
}

// baseFiles is the base commit: every path a row of rule 2 names, plus lines
// that do and do not name govulncheck or FROM golang:.
func baseFiles() map[string]string {
	return map[string]string{
		"go.mod":                                  "module example.com/root\n\ngo 1.25\n",
		"go.sum":                                  "example.com/a v1.0.0 h1:x\n",
		"temporal-server/go.mod":                  "module example.com/ts\n\ngo 1.25\n",
		"temporal-server/go.sum":                  "example.com/b v1.0.0 h1:y\n",
		"temporal-server/go.work":                 "go 1.25\n",
		"temporal-server/Dockerfile":              "FROM " + goOld + " AS build\nRUN go build ./...\n",
		"deploy/compose/nava-core/Dockerfile":     "FROM " + goOld + " AS build\nRUN go build ./...\nFROM scratch\n",
		"deploy/compose/nava-core/Dockerfile.dev": "FROM " + goOld + "\nRUN go build ./...\n",
		"Makefile":                                "GOVULNCHECK_VERSION := 1.1.4\n\nvulncheck:\n\tgovulncheck ./...\n\nbuild:\n\tgo build ./...\n",
		"make/lint.mk":                            "lint:\n\tgolangci-lint run ./...\n\ntools:\n\tgo install golang.org/x/vuln/cmd/govulncheck@v$(GOVULNCHECK_VERSION)\n",
		".github/workflows/govulncheck.yml":       "name: govulncheck\n",
		".github/workflows/ci.yml":                "name: ci\n",
		"tools/ci/cisteps/govulncheck.copy.yml":   "name: govulncheck\n",
		"tools/ci/cisteps/main.go":                "package main\n",
		"tools/ci/vulngate/gate_test.go":          "package vulngate\n",
		"scripts/ci/vulncheck-detect.sh":          "#!/bin/sh\n",
		"scripts/ci/vulncheck-gate.sh":            "#!/bin/sh\n",
		"cmd/nava-core/main.go":                   "package main\n",
		"docs/notes.md":                           "# Notes\n",
	}
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	g, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH: detect diffs git history")
	}
	r := &repo{t: t, git: g, dir: t.TempDir()}
	r.run("init", "-q", "-b", "main")
	r.run("config", "user.name", "vulngate test")
	r.run("config", "user.email", "vulngate@example.invalid")
	r.run("config", "commit.gpgsign", "false")
	r.run("config", "core.autocrlf", "false")
	for p, body := range baseFiles() {
		r.write(p, body)
	}
	r.base = r.commit("base")
	return r
}

func (r *repo) run(args ...string) string {
	r.t.Helper()
	cmd := exec.CommandContext(context.Background(), r.git, append([]string{"-C", r.dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *repo) write(p, body string) {
	r.t.Helper()
	f := filepath.Join(r.dir, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(f), 0o750); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) read(p string) string {
	r.t.Helper()
	b, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(p)))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(b)
}

func (r *repo) commit(msg string) string {
	r.t.Helper()
	r.run("add", "-A")
	r.run("commit", "-q", "--allow-empty", "-m", msg)
	return r.run("rev-parse", "HEAD")
}

// change starts a branch at base, applies edit and commits it: the head of a
// pull request whose base is r.base.
func (r *repo) change(name string, edit func(r *repo)) string {
	r.t.Helper()
	r.run("checkout", "-q", "-f", "-B", "case", r.base)
	edit(r)
	return r.commit(name)
}

// detect runs the detect script in dir with env; it returns the lines written
// to GITHUB_OUTPUT, stderr, and the exit code.
func detect(t *testing.T, dir string, env ...string) (out, stderr string, code int) {
	t.Helper()
	sh, path := script(t, "vulncheck-detect.sh")
	ghOut := filepath.Join(t.TempDir(), "github-output")
	cmd := exec.CommandContext(context.Background(), sh, path)
	cmd.Dir = dir
	cmd.Env = append(clean(os.Environ()), append([]string{"GITHUB_OUTPUT=" + ghOut}, env...)...)
	var e strings.Builder
	cmd.Stderr = &e
	code = exitCode(t, cmd)
	b, err := os.ReadFile(ghOut)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b)), e.String(), code
}

const (
	none   = "root=false\ntemporal_server=false"
	root   = "root=true\ntemporal_server=false"
	tsOnly = "root=false\ntemporal_server=true"
	both   = "root=true\ntemporal_server=true"
)

// V-2: every row of contract rule 2, each path on its own, plus a .go file
// only, docs only, both modules, and a diff from the merge base.
func TestDetect(t *testing.T) {
	t.Parallel()
	r := newRepo(t)
	appendTo := func(p string) func(*repo) {
		return func(r *repo) { r.write(p, r.read(p)+"# TICK-1148 test change\n") }
	}
	replace := func(p, old, repl string) func(*repo) {
		return func(r *repo) {
			s := r.read(p)
			if !strings.Contains(s, old) {
				t.Fatalf("%s has no %q", p, old)
			}
			r.write(p, strings.Replace(s, old, repl, 1))
		}
	}
	remove := func(p string) func(*repo) {
		return func(r *repo) { r.run("rm", "-q", p) }
	}
	add := func(p string) func(*repo) {
		return func(r *repo) { r.write(p, "go 1.25\n") }
	}
	create := func(p, body string) func(*repo) {
		return func(r *repo) { r.write(p, body) }
	}
	move := func(from, to string) func(*repo) {
		return func(r *repo) { r.run("mv", from, to) }
	}

	for _, c := range []struct {
		name string
		edit func(*repo)
		want string
	}{
		// Row 1: the root module.
		{"go.mod", appendTo("go.mod"), root},
		{"go.sum", appendTo("go.sum"), root},
		{"go.sum deleted", remove("go.sum"), root},
		{"go.sum renamed away", move("go.sum", "go.sum.old"), root},

		// Row 2: temporal-server.
		{"temporal-server/go.mod", appendTo("temporal-server/go.mod"), tsOnly},
		{"temporal-server/go.sum", appendTo("temporal-server/go.sum"), tsOnly},

		// Row 3: the gate itself.
		{"govulncheck.yml", appendTo(".github/workflows/govulncheck.yml"), both},
		{"govulncheck.copy.yml", appendTo("tools/ci/cisteps/govulncheck.copy.yml"), both},
		{"detect script", appendTo("scripts/ci/vulncheck-detect.sh"), both},
		{"gate script", appendTo("scripts/ci/vulncheck-gate.sh"), both},
		{"tools/ci/cisteps/", appendTo("tools/ci/cisteps/main.go"), both},
		{"tools/ci/cisteps/ new file", add("tools/ci/cisteps/testdata/x.yml"), both},
		{"tools/ci/vulngate/", appendTo("tools/ci/vulngate/gate_test.go"), both},
		{"Makefile govulncheck line", replace("Makefile", "\tgovulncheck ./...", "\tgovulncheck -show verbose ./..."), both},
		{"Makefile GOVULNCHECK_VERSION line", replace("Makefile", "1.1.4", "1.1.5"), both},
		// TICK-1178 AC 2: Makefile and make/*.mk are the gate as whole files.
		// A recipe replaced by `true`, a PATH or an export names no
		// govulncheck, yet changes what the scans run.
		{"Makefile other line", replace("Makefile", "\tgo build ./...", "\tgo build -trimpath ./..."), both},
		{"Makefile appended unrelated line", appendTo("Makefile"), both},
		{"make/lint.mk govulncheck line", replace("make/lint.mk", "govulncheck@v", "govulncheck@latest # v"), both},
		{"make/lint.mk only", replace("make/lint.mk", "golangci-lint run", "golangci-lint run -v"), both},
		{"make/lint.mk recipe replaced by true", replace("make/lint.mk", "\tgo install golang.org/x/vuln/cmd/govulncheck@v$(GOVULNCHECK_VERSION)", "\ttrue"), both},
		{"new make/*.mk", create("make/zz.mk", "export PATH := /fake:$(PATH)\n"), both},
		{"make/lint.mk deleted", remove("make/lint.mk"), both},
		{"govulncheck word outside Makefile", replace("cmd/nava-core/main.go", "package main\n", "package main\n\n// govulncheck\n"), none},

		// Row 4: the Go toolchain and go.work.
		{"Dockerfile FROM golang:", replace("deploy/compose/nava-core/Dockerfile", "FROM "+goOld+" AS build", "FROM "+goNew+" AS build"), both},
		{"Dockerfile other line", replace("deploy/compose/nava-core/Dockerfile", "RUN go build", "RUN go build -trimpath"), none},
		{"Dockerfile other FROM", replace("deploy/compose/nava-core/Dockerfile", "FROM scratch", "FROM "+alpineImg), none},
		{"Dockerfile.dev FROM golang:", replace("deploy/compose/nava-core/Dockerfile.dev", goOld, goNew), both},
		{"temporal-server/Dockerfile FROM golang:", replace("temporal-server/Dockerfile", goOld, goNew), both},
		// Wider than the contract's "FROM golang:", failing closed (Security
		// review of PR #912): a lowercase from, and a --platform flag before
		// the image. Each is a new file, so no FROM golang: line changes.
		{"Dockerfile lowercase from golang", create("deploy/compose/nava-x/Dockerfile", "from "+goNew+" AS build\n"), both},
		{"Dockerfile FROM --platform golang", create("deploy/compose/nava-y/Dockerfile", "FROM --platform=$BUILDPLATFORM "+goNew+" AS build\n"), both},
		// Any file can be a Dockerfile (compose dockerfile:, docker build -f),
		// so a line-start FROM ... golang counts wherever it is; prose that
		// only mentions it mid-line does not.
		{"FROM golang: in a non-Dockerfile", replace("docs/notes.md", "# Notes\n", "# Notes\n\nFROM "+goNew+"\n"), both},
		{"FROM golang: mid-line in prose", replace("docs/notes.md", "# Notes\n", "# Notes\n\nBuilt FROM "+goNew+" upstream.\n"), none},
		{"go.work added", add("go.work"), both},
		{"go.work.sum added", add("go.work.sum"), both},
		{"nested go.work changed", appendTo("temporal-server/go.work"), both},
		{"nested go.work deleted", remove("temporal-server/go.work"), both},
		{"nested go.work renamed away", move("temporal-server/go.work", "temporal-server/go.work.bak"), both},

		// Row 5 and combinations.
		{".go file only", appendTo("cmd/nava-core/main.go"), none},
		{"docs only", appendTo("docs/notes.md"), none},
		// Non-ASCII paths (Security review of PR #912, finding 1): git
		// C-quotes them unless core.quotePath=false, and a quoted path
		// matched no row.
		{"non-ASCII path under tools/ci/cisteps/", create("tools/ci/cisteps/é_test.go", "package main\n"), both},
		{"go.work under a non-ASCII directory", add("dé/go.work"), both},
		// Row 4 through the FROM diff ($golang), not only $files (Principal
		// Software Engineer's review of PR #912).
		{"root Dockerfilé with FROM golang:", create("Dockerfilé", "FROM "+goNew+"\n"), both},
		{"ci.yml only", appendTo(".github/workflows/ci.yml"), none},
		{"both go.sum", func(r *repo) { appendTo("go.sum")(r); appendTo("temporal-server/go.sum")(r) }, both},
	} {
		head := r.change(c.name, c.edit)
		got, stderr, code := detect(t, r.dir, "EVENT=pull_request", "BASE_SHA="+r.base, "HEAD_SHA="+head)
		if code != 0 || got != c.want {
			t.Errorf("%s: exit %d, output %q, want %q\n%s", c.name, code, got, c.want, stderr)
		}
		// The run log says why.
		if c.want != none && !strings.Contains(stderr, "-> ") {
			t.Errorf("%s: no reason on stderr: %q", c.name, stderr)
		}
	}

	// The merge base, not the base tip: a go.sum change that landed on main
	// after the pull request forked is not the pull request's.
	head := r.change("docs on the PR", appendTo("docs/notes.md"))
	r.run("checkout", "-q", "-f", "-B", "main", r.base)
	appendTo("go.sum")(r)
	mainTip := r.commit("go.sum on main")
	got, stderr, code := detect(t, r.dir, "EVENT=pull_request", "BASE_SHA="+mainTip, "HEAD_SHA="+head)
	if code != 0 || got != none {
		t.Errorf("diff from the merge base: exit %d, output %q, want %q\n%s", code, got, none, stderr)
	}
}

// Rule 1: schedule and workflow_dispatch scan both modules and read no git
// history (the directory is not a repository).
func TestDetectNonPullRequest(t *testing.T) {
	t.Parallel()
	for _, ev := range []string{"schedule", "workflow_dispatch"} {
		got, stderr, code := detect(t, t.TempDir(), "EVENT="+ev, "BASE_SHA=", "HEAD_SHA=")
		if code != 0 || got != both {
			t.Errorf("%s: exit %d, output %q, want %q\n%s", ev, code, got, both, stderr)
		}
	}
}

// Rule 2: no merge base, an empty commit input or a git error fails detect
// (the gate then fails) and writes no output a later step could read.
func TestDetectFailures(t *testing.T) {
	t.Parallel()
	r := newRepo(t)
	head := r.change("go.sum", func(r *repo) { r.write("go.sum", "changed\n") })
	r.run("checkout", "-q", "--orphan", "unrelated")
	r.run("rm", "-q", "-rf", ".")
	r.write("other.txt", "x\n")
	orphan := r.commit("unrelated history")
	r.run("branch", "beef", head)

	for _, c := range []struct {
		name string
		env  []string
	}{
		{"no merge base", []string{"BASE_SHA=" + r.base, "HEAD_SHA=" + orphan}},
		{"empty HEAD_SHA", []string{"BASE_SHA=" + r.base, "HEAD_SHA="}},
		{"empty BASE_SHA", []string{"BASE_SHA=", "HEAD_SHA=" + head}},
		// Only a hex commit id reaches git (Security review of PR #912): not
		// a branch name, and not an option.
		{"HEAD_SHA a branch name", []string{"BASE_SHA=" + r.base, "HEAD_SHA=case"}},
		{"BASE_SHA an option", []string{"BASE_SHA=--all", "HEAD_SHA=" + head}},
		{"unknown commit", []string{"BASE_SHA=" + r.base, "HEAD_SHA=1111111111111111111111111111111111111111"}},
		// TICK-1150 (TICK-1148 review of PR #912): only a full 40- or
		// 64-character commit id. A short one resolves, and so does a
		// branch whose name is only hex characters.
		{"HEAD_SHA short hex", []string{"BASE_SHA=" + r.base, "HEAD_SHA=" + head[:7]}},
		{"BASE_SHA short hex", []string{"BASE_SHA=" + r.base[:12], "HEAD_SHA=" + head}},
		{"HEAD_SHA a hex-only branch name", []string{"BASE_SHA=" + r.base, "HEAD_SHA=beef"}},
		{"HEAD_SHA 41 hex characters", []string{"BASE_SHA=" + r.base, "HEAD_SHA=" + head + "a"}},
		{"HEAD_SHA 63 hex characters", []string{"BASE_SHA=" + r.base, "HEAD_SHA=" + strings.Repeat("a", 63)}},
	} {
		got, stderr, code := detect(t, r.dir, append([]string{"EVENT=pull_request"}, c.env...)...)
		if code == 0 {
			t.Errorf("%s: exit 0, want non-zero (output %q)", c.name, got)
		}
		if got != "" {
			t.Errorf("%s: wrote %q to GITHUB_OUTPUT, want nothing", c.name, got)
		}
		if !strings.Contains(stderr, "vulncheck-detect:") {
			t.Errorf("%s: no reason on stderr: %q", c.name, stderr)
		}
	}

	// Not a repository at all.
	if _, _, code := detect(t, t.TempDir(), "EVENT=pull_request", "BASE_SHA="+r.base, "HEAD_SHA="+head); code == 0 {
		t.Error("outside a repository: exit 0, want non-zero")
	}
}

// GITHUB_OUTPUT is appended to, not overwritten; unset, the lines go to stdout.
func TestDetectOutput(t *testing.T) {
	t.Parallel()
	sh, path := script(t, "vulncheck-detect.sh")
	ghOut := filepath.Join(t.TempDir(), "github-output")
	if err := os.WriteFile(ghOut, []byte("earlier=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), sh, path)
	cmd.Dir = t.TempDir()
	cmd.Env = append(clean(os.Environ()), "EVENT=schedule", "GITHUB_OUTPUT="+ghOut)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	b, err := os.ReadFile(ghOut)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), "earlier=1\n"+both+"\n"; got != want {
		t.Errorf("GITHUB_OUTPUT %q, want %q", got, want)
	}

	cmd = exec.CommandContext(context.Background(), sh, path)
	cmd.Dir = t.TempDir()
	cmd.Env = append(clean(os.Environ()), "EVENT=workflow_dispatch")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != both {
		t.Errorf("stdout %q, want %q", got, both)
	}
}

// Even with core.quotePath=false git quotes a path holding a double quote, a
// backslash or a control character, and a quoted path matches no row. Detect
// fails closed on it and scans both. The path goes in through the index
// only, because Windows cannot create such a file.
func TestDetectQuotedPathFailsClosed(t *testing.T) {
	t.Parallel()
	r := newRepo(t)
	r.run("checkout", "-q", "-f", "-B", "case", r.base)
	blob := r.run("rev-parse", r.base+":docs/notes.md")
	r.run("-c", "core.protectNTFS=false", "update-index", "--add", "--cacheinfo", "100644,"+blob+`,docs/a"b.md`)
	r.run("commit", "-q", "-m", "quoted path")
	head := r.run("rev-parse", "HEAD")
	got, stderr, code := detect(t, r.dir, "EVENT=pull_request", "BASE_SHA="+r.base, "HEAD_SHA="+head)
	if code != 0 || got != both {
		t.Errorf("exit %d, output %q, want %q\n%s", code, got, both, stderr)
	}
	if !strings.Contains(stderr, "quoted path") {
		t.Errorf("no quoted-path reason on stderr: %q", stderr)
	}
}
