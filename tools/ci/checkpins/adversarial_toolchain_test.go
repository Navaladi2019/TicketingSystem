package main

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// TICK-1140 / TICK-1151 (ADR-001-E4 rules 7 and 8, test V-5): govulncheck
// judges the standard library of the Go in the module's toolchain line; the
// image ships the Go of its FROM line, and the golang images set
// GOTOOLCHAIN=local, which ignores the toolchain line. If the two differ, the
// scan can be green over a standard library that still ships. A tracked
// go.work changes the scanned module versions without touching go.mod.

// repoRoot is the repository this package sits in.
const repoRoot = "../../.."

func TestScannedToolchainIsTheShippedToolchain(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		mustSameGo(t, repoRoot, "")
	})
	t.Run("copy is green", func(t *testing.T) {
		mustSameGo(t, goCopy(t), "")
	})
	t.Run("nava-relay lowered", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-relay/Dockerfile", "golang:1.27.3-alpine")
		mustSameGo(t, dir, "deploy/compose/nava-relay/Dockerfile:2: builds with Go 1.27.3, but go.mod is scanned with toolchain go1.27.4")
	})
	t.Run("temporal-server lowered", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "temporal-server/Dockerfile", "golang:1.27.3-bookworm")
		mustSameGo(t, dir, "temporal-server/Dockerfile:3: builds with Go 1.27.3, but temporal-server/go.mod is scanned with toolchain go1.27.4")
	})
	t.Run("toolchain raised without the images", func(t *testing.T) {
		dir := goCopy(t)
		replaceIn(t, dir, "go.mod", "toolchain go1.27.4", "toolchain go1.27.5")
		mustSameGo(t, dir, "builds with Go 1.27.4, but go.mod is scanned with toolchain go1.27.5")
	})
	t.Run("toolchain line removed", func(t *testing.T) {
		dir := goCopy(t)
		replaceIn(t, dir, "temporal-server/go.mod", "toolchain go1.27.4", "")
		mustSameGo(t, dir, "temporal-server/go.mod: no toolchain line")
	})
	t.Run("a later build stage lowered", func(t *testing.T) {
		dir := goCopy(t)
		appendTo(t, dir, "deploy/compose/nava-core/Dockerfile", "\nFROM --platform=$BUILDPLATFORM docker.io/library/golang:1.26.0 AS second\n")
		mustSameGo(t, dir, "nava-core/Dockerfile:")
		mustSameGo(t, dir, "builds with Go 1.26.0")
	})
	t.Run("version hidden in a build ARG", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-worker/Dockerfile", "golang:${GO_VERSION}-alpine")
		mustSameGo(t, dir, "nava-worker/Dockerfile:2: FROM golang without a literal Go version tag")
	})
	t.Run("untagged golang", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-db/Dockerfile", "golang")
		mustSameGo(t, dir, "nava-db/Dockerfile:3: FROM golang without a literal Go version tag")
	})
	// TICK-1164: the image the Go ships in must be one the check can read.
	t.Run("image from a build ARG", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-relay/Dockerfile", "${GO_IMAGE}")
		replaceIn(t, dir, "deploy/compose/nava-relay/Dockerfile", "FROM ${GO_IMAGE}", "ARG GO_IMAGE=golang:1.27.3-alpine\nFROM ${GO_IMAGE}")
		mustSameGo(t, dir, "nava-relay/Dockerfile:3: FROM image ${GO_IMAGE} comes from a variable")
		mustSameGo(t, dir, "deploy/compose/nava-relay/Dockerfile: no FROM golang:<version> stage")
	})
	t.Run("image from a bare $VAR", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-core/Dockerfile", "$GO_IMAGE")
		mustSameGo(t, dir, "nava-core/Dockerfile:2: FROM image $GO_IMAGE comes from a variable")
	})
	t.Run("runtime image from a variable", func(t *testing.T) {
		dir := goCopy(t)
		replaceIn(t, dir, "deploy/compose/nava-db/Dockerfile", "FROM alpine:3.22.1", "FROM ${RUNTIME}")
		mustSameGo(t, dir, "nava-db/Dockerfile:12: FROM image ${RUNTIME} comes from a variable")
	})
	t.Run("build stage from a non-golang Go image", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-relay/Dockerfile", "cgr.dev/chainguard/go")
		mustSameGo(t, dir, "deploy/compose/nava-relay/Dockerfile: no FROM golang:<version> stage")
		mustSameGo(t, dir, "nava-relay/Dockerfile:2: FROM cgr.dev/chainguard/go is neither golang:<version>, an earlier stage nor an allowed runtime image")
	})
	t.Run("non-golang Go stage beside a decoy golang stage", func(t *testing.T) {
		dir := goCopy(t)
		appendTo(t, dir, "deploy/compose/nava-onboard/Dockerfile", "\nFROM cgr.dev/chainguard/go:latest AS real\n")
		mustSameGo(t, dir, "FROM cgr.dev/chainguard/go:latest is neither golang:<version>, an earlier stage nor an allowed runtime image")
	})
	t.Run("image split over a line continuation", func(t *testing.T) {
		dir := goCopy(t)
		lower(t, dir, "deploy/compose/nava-purge/Dockerfile", "\\\n  cgr.dev/chainguard/go")
		mustSameGo(t, dir, "nava-purge/Dockerfile:2: FROM without an image on its line")
	})
	t.Run("runtime stages stay allowed", func(t *testing.T) {
		dir := goCopy(t)
		appendTo(t, dir, "deploy/compose/nava-groupsync/Dockerfile",
			"\nFROM scratch AS a\nFROM gcr.io/distroless/static-debian12:nonroot AS b\nFROM --platform=$BUILDPLATFORM debian:bookworm-slim AS c\nFROM build AS d\nfrom a\n")
		mustSameGo(t, dir, "")
	})
	t.Run("a missing tracked Dockerfile is named", func(t *testing.T) {
		dir := goCopy(t)
		if err := os.Remove(filepath.Join(dir, "deploy", "compose", "nava-relay", "Dockerfile")); err != nil {
			t.Fatal(err)
		}
		_, err := checkSameGo(dir)
		if err == nil || !strings.Contains(err.Error(), "deploy/compose/nava-relay/Dockerfile") {
			t.Fatalf("want an error naming the missing file, got %v", err)
		}
	})
	for _, f := range []string{"deploy/compose/nava-new/Dockerfile.build", "cmd/tool/Dockerfile", "Dockerfile.ci"} {
		t.Run("unmapped "+f, func(t *testing.T) {
			dir := goCopy(t)
			put(t, dir, f, "FROM golang:1.27.4-alpine AS build\n")
			gitAdd(t, dir, f)
			mustSameGo(t, dir, f+":1: unmapped Dockerfile")
		})
	}
	t.Run("untracked unmapped Dockerfile does not count", func(t *testing.T) {
		dir := goCopy(t)
		put(t, dir, "scratch/Dockerfile", "FROM golang:1.20.0\n")
		mustSameGo(t, dir, "")
	})
	t.Run("a mapped image dropped", func(t *testing.T) {
		dir := goCopy(t)
		gitRun(t, dir, "rm", "-q", "--cached", "temporal-server/Dockerfile")
		mustSameGo(t, dir, "temporal-server/go.mod: no tracked Dockerfile matches temporal-server/Dockerfile")
	})
}

func TestNoTrackedGoWork(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		mustSameGo(t, repoRoot, "")
	})
	for _, f := range []string{"go.work", "go.work.sum", "temporal-server/go.work"} {
		t.Run("tracked "+f, func(t *testing.T) {
			dir := goCopy(t)
			put(t, dir, f, "go 1.27.0\n\nuse .\n")
			gitAdd(t, dir, f)
			mustSameGo(t, dir, f+": a tracked "+path.Base(f)+" changes the scanned module versions")
		})
	}
	t.Run("untracked go.work does not count", func(t *testing.T) {
		dir := goCopy(t)
		put(t, dir, "go.work", "go 1.27.0\n\nuse .\n")
		mustSameGo(t, dir, "")
	})
}

// The rule runs from make check-pins by default; it cannot be skipped by a
// root that is not a git work tree.
func TestCLISameGo(t *testing.T) {
	pins, err := filepath.Abs(filepath.Join(repoRoot, "docs", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	dir := goCopy(t)
	lower(t, dir, "deploy/compose/nava-relay/Dockerfile", "golang:1.27.3-alpine")
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"-root", dir, "-pins", pins}, &stdout, &stderr); code != 1 ||
		!strings.Contains(stderr.String(), "builds with Go 1.27.3") {
		t.Fatalf("exit %d, want 1 naming the lowered image\nstderr:\n%s", code, stderr.String())
	}

	notGit := t.TempDir()
	stderr.Reset()
	if code := cli([]string{"-root", notGit, "-pins", pins}, &stdout, &stderr); code != 2 ||
		!strings.Contains(stderr.String(), "git ls-files") {
		t.Fatalf("exit %d, want 2 (cannot check) outside a git work tree\nstderr:\n%s", code, stderr.String())
	}
}

// mustSameGo runs checkSameGo on dir. With want "" it must find nothing;
// otherwise some problem must contain want.
func mustSameGo(t *testing.T, dir, want string) {
	t.Helper()
	problems, err := checkSameGo(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(problems, "\n")
	switch {
	case want == "" && got != "":
		t.Fatalf("want no problems, got:\n%s", got)
	case want != "" && !strings.Contains(got, want):
		t.Fatalf("want a problem containing %q, got:\n%s", want, got)
	}
}

// goCopy copies the repository's go.mod files and every tracked Dockerfile
// into a fresh git repository (files added to the index, no commit) and
// returns it.
func goCopy(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	tracked, err := trackedFiles(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	var files []string
	for _, f := range tracked {
		if f == "go.mod" || f == "temporal-server/go.mod" || strings.HasPrefix(path.Base(f), "Dockerfile") {
			data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(f)))
			if err != nil {
				t.Fatal(err)
			}
			put(t, dir, f, string(data))
			files = append(files, f)
		}
	}
	gitAdd(t, dir, files...)
	return dir
}

// lower replaces the golang image ref of the first FROM golang: line in f.
func lower(t *testing.T, dir, f, ref string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if fromGolangRe.MatchString(l) {
			fields := strings.Fields(l)
			fields[1] = ref
			lines[i] = strings.Join(fields, " ")
			put(t, dir, f, strings.Join(lines, "\n"))
			return
		}
	}
	t.Fatalf("%s has no FROM golang: line", f)
}

func appendTo(t *testing.T, dir, f, text string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	put(t, dir, f, string(data)+text)
}

// put writes dir/f, creating its parent directories.
func put(t *testing.T, dir, f, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, filepath.FromSlash(f))), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, f, data)
}

func gitAdd(t *testing.T, dir string, files ...string) {
	t.Helper()
	gitRun(t, dir, append([]string{"add", "--"}, files...)...)
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
