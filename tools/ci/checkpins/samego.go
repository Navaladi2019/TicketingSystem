package main

// ADR-001-E4 contract rules 7 and 8 (test V-5, TICK-1151).
//
// Rule 8: govulncheck judges the standard library of the Go named by the
// module's toolchain line, but the image ships the Go of its FROM golang: line
// (the golang images set GOTOOLCHAIN=local, which ignores the toolchain line).
// If the two differ, the scan can be green over a standard library that still
// ships. So every FROM golang: in a tracked Dockerfile must name the toolchain
// of the module that Dockerfile builds, and a Dockerfile that builds Go but
// maps to no module fails, so a new image cannot escape the rule.
//
// Rule 7: a tracked go.work (or go.work.sum) changes the versions the go tool
// resolves without changing go.mod or go.sum, so none may be tracked.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// goImages maps each module (its go.mod, relative to the root) to the glob of
// the Dockerfiles that ship it. A tracked Dockerfile with a FROM golang: line
// outside these globs is unmapped.
func goImages() []struct{ mod, glob string } {
	return []struct{ mod, glob string }{
		{"go.mod", "deploy/compose/nava-*/Dockerfile"},
		{"temporal-server/go.mod", "temporal-server/Dockerfile"},
	}
}

var (
	toolchainRe = regexp.MustCompile(`(?m)^\s*toolchain\s+go(\S+)\s*$`)
	// FROM [--flag...] [registry/path/]golang[:tag][@digest] ...
	fromGolangRe = regexp.MustCompile(`(?i)^\s*FROM\s+(?:--\S+\s+)*(?:\S+/)?golang(?:[:@](\S*))?(?:\s|$)`)
	// The Go version at the start of a golang image tag: 1.27.4 in 1.27.4-alpine.
	goTagRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)*)(?:$|[-@])`)
)

// checkSameGo returns one line per breach of rules 7 and 8 in the git work
// tree at root. Only tracked files count: an untracked go.work on a laptop is
// not the repository's.
func checkSameGo(root string) ([]string, error) {
	tracked, err := trackedFiles(root)
	if err != nil {
		return nil, err
	}
	var problems []string

	for _, f := range tracked {
		if b := path.Base(f); b == "go.work" || b == "go.work.sum" {
			problems = append(problems, fmt.Sprintf("%s: a tracked %s changes the scanned module versions; remove it (ADR-001-E4 rule 7)", f, b))
		}
	}

	want := map[string]string{} // Dockerfile -> toolchain version
	wantMod := map[string]string{}
	for _, gi := range goImages() {
		v, err := toolchainOf(filepath.Join(root, filepath.FromSlash(gi.mod)))
		if err != nil {
			return nil, err
		}
		if v == "" {
			problems = append(problems, fmt.Sprintf("%s: no toolchain line, so the scanned Go is not pinned (ADR-001-E4 rule 8)", gi.mod))
		}
		matched := 0
		for _, f := range tracked {
			if ok, _ := path.Match(gi.glob, f); ok {
				want[f], wantMod[f] = v, gi.mod
				matched++
			}
		}
		if matched == 0 {
			problems = append(problems, fmt.Sprintf("%s: no tracked Dockerfile matches %s (ADR-001-E4 rule 8)", gi.mod, gi.glob))
		}
	}

	for _, f := range tracked {
		if !strings.HasPrefix(path.Base(f), "Dockerfile") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f))) //nolint:gosec // a tracked file
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(string(data), "\n") {
			m := fromGolangRe.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			at := fmt.Sprintf("%s:%d", f, i+1)
			mod, mapped := wantMod[f]
			if !mapped {
				problems = append(problems, fmt.Sprintf("%s: unmapped Dockerfile: it builds from golang but no go.mod is mapped to it in checkpins goImages (ADR-001-E4 rule 8)", at))
				continue
			}
			got := ""
			if t := goTagRe.FindStringSubmatch(m[1]); t != nil {
				got = t[1]
			}
			if got == "" {
				problems = append(problems, fmt.Sprintf("%s: FROM golang without a literal Go version tag; it must name %s's toolchain go%s (ADR-001-E4 rule 8)", at, mod, want[f]))
				continue
			}
			if got != want[f] {
				problems = append(problems, fmt.Sprintf("%s: builds with Go %s, but %s is scanned with toolchain go%s (ADR-001-E4 rule 8)", at, got, mod, want[f]))
			}
		}
	}
	return problems, nil
}

// trackedFiles lists the files git tracks under root, slash-separated and
// relative to root.
func trackedFiles(root string) ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "-C", root, "ls-files", "-z", "--", ".") //nolint:gosec // fixed args
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files in %s: %w: %s", root, err, strings.TrimSpace(stderr.String()))
	}
	var files []string
	for f := range strings.SplitSeq(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	sort.Strings(files)
	return files, nil
}

// toolchainOf returns the version of a go.mod's toolchain line (1.27.4 for
// "toolchain go1.27.4"), or "" when it has none.
func toolchainOf(goMod string) (string, error) {
	data, err := os.ReadFile(goMod) //nolint:gosec // a mapped go.mod
	if err != nil {
		return "", err
	}
	if m := toolchainRe.FindSubmatch(data); m != nil {
		return string(m[1]), nil
	}
	return "", nil
}
