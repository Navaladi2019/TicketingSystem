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
// A mapped Dockerfile must also be readable (TICK-1164): at least one literal
// FROM golang:<version> stage, no FROM image from a variable, and every other
// FROM an earlier stage or an allowed runtime image (runtimeImages), so Go
// cannot ship from an image the rule never compared.
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
	// FROM [--flag...] image [AS name], image possibly empty or a lone \; FROM
	// must end the line or be followed by whitespace, so FROMX is not FROM.
	fromAnyRe = regexp.MustCompile(`(?i)^\s*FROM(?:\s+--\S+)*(?:\s+(\S+))?(?:\s+AS\s+(\S+))?(?:\s|$)`)
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
			return nil, fmt.Errorf("%s: tracked but not readable in the work tree: %w", f, err)
		}
		_, mappedFile := wantMod[f]
		stages := map[string]bool{} // AS names seen so far
		golangStages := 0
		lines, starts, unclosed := dockerLines(string(data))
		for _, i := range unclosed {
			problems = append(problems, fmt.Sprintf("%s:%d: heredoc %s is never closed; its lines are read as instructions (ADR-001-E4 rule 8)", f, i+1, heredocWord(lines[i])))
		}
		for i, line := range lines {
			if !starts[i] {
				continue // a heredoc body, comment or continuation (TICK-1165)
			}
			at := fmt.Sprintf("%s:%d", f, i+1)
			m := fromGolangRe.FindStringSubmatch(line)
			if mappedFile {
				// TICK-1164: in a mapped Dockerfile the check must be able to
				// read every image, or Go can ship from one it never compared.
				if fm := fromAnyRe.FindStringSubmatch(line); fm != nil {
					problems = append(problems, readableImage(at, fm[1], m != nil, stages)...)
					if fm[2] != "" {
						stages[strings.ToLower(fm[2])] = true
					}
				}
			}
			if m == nil {
				continue
			}
			golangStages++
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
		if mappedFile && golangStages == 0 {
			problems = append(problems, fmt.Sprintf("%s: no FROM golang:<version> stage, so the Go it ships is not the Go %s is scanned with (ADR-001-E4 rule 8)", f, wantMod[f]))
		}
	}
	return problems, nil
}

// readableImage checks one FROM image of a mapped Dockerfile (TICK-1164). The
// image must be spelled literally, and must be golang (checked by the caller
// against the toolchain), an earlier stage, or a runtime image from
// runtimeImages, which ship no Go toolchain. Any other base could build the
// shipped Go unseen, so it fails until it is added here in review.
func readableImage(at, image string, golang bool, stages map[string]bool) []string {
	switch {
	case image == "" || image == `\`:
		return []string{fmt.Sprintf("%s: FROM without an image on its line; the check cannot read it (ADR-001-E4 rule 8)", at)}
	case strings.Contains(image, "$"):
		return []string{fmt.Sprintf("%s: FROM image %s comes from a variable; spell the image literally so the check can read it (ADR-001-E4 rule 8)", at, image)}
	case golang || stages[strings.ToLower(image)]:
		return nil
	}
	name, _ := splitImage(image)
	name = strings.TrimPrefix(strings.TrimPrefix(name, "docker.io/"), "library/")
	for _, r := range runtimeImages() {
		if name == r || (strings.HasSuffix(r, "/") && strings.HasPrefix(name, r)) {
			return nil
		}
	}
	return []string{fmt.Sprintf("%s: FROM %s is neither golang:<version>, an earlier stage nor an allowed runtime image (checkpins runtimeImages); Go built there is not the scanned Go (ADR-001-E4 rule 8)", at, image)}
}

// runtimeImages are the non-Go bases a mapped Dockerfile may use after its
// golang stage. An entry ending in / allows every image under that path.
func runtimeImages() []string {
	return []string{"scratch", "alpine", "debian", "gcr.io/distroless/"}
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
