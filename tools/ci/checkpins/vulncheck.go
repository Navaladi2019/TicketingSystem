package main

// ADR-001-A7 A7.2 (test V-6) and ADR-001-E4 contract rule 7 as amended by A7
// (test V-7), TICK-1188.
//
// A7.2: the govulncheck that scans is the one reviewed. Three copies of its
// version must be the same string: the govulncheck row of docs/pins.md,
// GOVULNCHECK_VERSION := in Makefile (read as text, never by running make),
// and every govulncheck@<v> in a step run: of the govulncheck workflow
// (parsed as YAML). Each scan job has exactly one such line and no other job
// has one.
//
// Rule 7: ./... stops at a nested go.mod, and vendored source is not checked
// against go.sum. So no tracked go.mod outside the module set, and no tracked
// vendor/ path in the root module or temporal-server/.

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

const govulncheckWorkflow = ".github/workflows/govulncheck.yml"

// scanJobs are the jobs that must each run govulncheck exactly once.
func scanJobs() []string { return []string{"vulncheck-root", "vulncheck-temporal-server"} }

// moduleSet is every go.mod that may be tracked outside a testdata/ path.
// Adding a module is a reviewed edit here (AR-TICK-1183-01).
func moduleSet() []string {
	return []string{"go.mod", "temporal-server/go.mod", "docs/go.mod", "tools/navalint/go.mod", "tools/workflowcheck/go.mod"}
}

var (
	makeVulnRe = regexp.MustCompile(`(?m)^GOVULNCHECK_VERSION[ \t]*:=[ \t]*(.*?)[ \t]*$`)
	// govulncheck@<v>: v runs to whitespace or a quote, so an expression such
	// as ${{ env.V }} is read as "${{" and never equals a version.
	runVulnRe = regexp.MustCompile(`govulncheck@([^\s"']*)`)
)

// checkVulnVersion returns one line per breach of A7.2 under root, reading
// the pins table from pinsPath.
func checkVulnVersion(root, pinsPath string) ([]string, error) {
	const id = "(ADR-001-A7 V-6)"
	var problems []string
	pinsRel := relTo(root, pinsPath)

	pinsData, err := os.ReadFile(pinsPath) //nolint:gosec // the pins file the caller chose
	if err != nil {
		return nil, err
	}
	var pinned []string
	for line := range strings.SplitSeq(string(pinsData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "| `govulncheck`") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) >= 2 {
			pinned = append(pinned, strings.TrimSpace(cells[1]))
		} else {
			pinned = append(pinned, "")
		}
	}
	if len(pinned) != 1 {
		problems = append(problems, fmt.Sprintf("%s: %d govulncheck rows, want exactly 1 %s", pinsRel, len(pinned), id))
	}

	makeData, err := os.ReadFile(filepath.Join(root, "Makefile")) //nolint:gosec // root/Makefile
	if err != nil {
		return nil, err
	}
	var made []string
	for _, m := range makeVulnRe.FindAllStringSubmatch(string(makeData), -1) {
		made = append(made, m[1])
	}
	if len(made) != 1 {
		problems = append(problems, fmt.Sprintf("Makefile: %d GOVULNCHECK_VERSION := lines, want exactly 1 %s", len(made), id))
	}

	// The reference is the pins row when there is exactly one, else Makefile.
	want := ""
	switch {
	case len(pinned) == 1:
		want = pinned[0]
	case len(made) == 1:
		want = made[0]
	}
	if len(made) == 1 && len(pinned) == 1 && made[0] != pinned[0] {
		problems = append(problems, fmt.Sprintf("Makefile: GOVULNCHECK_VERSION is %s, but %s: the govulncheck row pins %s %s", made[0], pinsRel, want, id))
	}

	wfData, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(govulncheckWorkflow))) //nolint:gosec // a fixed path
	if err != nil {
		return nil, err
	}
	var wf struct {
		Jobs map[string]struct {
			Steps []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(wfData, &wf); err != nil {
		return append(problems, fmt.Sprintf("%s: not YAML: %v %s", govulncheckWorkflow, err, id)), nil
	}
	seen := map[string][]string{} // job -> versions found
	for job, j := range wf.Jobs {
		for _, s := range j.Steps {
			for _, m := range runVulnRe.FindAllStringSubmatch(s.Run, -1) {
				seen[job] = append(seen[job], m[1])
			}
		}
	}
	isScan := map[string]bool{}
	for _, job := range scanJobs() {
		isScan[job] = true
		vs := seen[job]
		if len(vs) != 1 {
			problems = append(problems, fmt.Sprintf("%s: job %s has %d govulncheck@ lines, want exactly 1 %s", govulncheckWorkflow, job, len(vs), id))
		}
		for _, v := range vs {
			if want != "" && v != want {
				problems = append(problems, fmt.Sprintf("%s: job %s runs govulncheck@%s, but %s: the govulncheck row pins %s %s", govulncheckWorkflow, job, v, pinsRel, want, id))
			}
		}
	}
	for _, job := range sortedKeys(seen) {
		if !isScan[job] {
			problems = append(problems, fmt.Sprintf("%s: job %s runs govulncheck@ but is not a scan job (%s) %s", govulncheckWorkflow, job, strings.Join(scanJobs(), ", "), id))
		}
	}
	return problems, nil
}

// checkLayout returns one line per breach of the amended rule 7 among the
// tracked files.
func checkLayout(tracked []string) []string {
	const id = "(ADR-001-A7 V-7)"
	allowed := map[string]bool{}
	for _, m := range moduleSet() {
		allowed[m] = true
	}
	var problems []string
	for _, f := range tracked {
		segs := strings.Split(f, "/")
		if hasSeg(segs[:len(segs)-1], "testdata") {
			continue
		}
		if segs[len(segs)-1] == "go.mod" && !allowed[f] {
			problems = append(problems, fmt.Sprintf("%s: a tracked go.mod outside the module set takes its directory out of ./... %s; the set is %s", f, id, strings.Join(moduleSet(), ", ")))
		}
		if hasSeg(segs[:len(segs)-1], "vendor") && inScannedModule(f) {
			problems = append(problems, fmt.Sprintf("%s: a tracked vendor/ path is not checked against go.sum %s", f, id))
		}
	}
	return problems
}

// inScannedModule reports whether f belongs to the root module or
// temporal-server/, not to a tooling module of moduleSet.
func inScannedModule(f string) bool {
	for _, m := range moduleSet() {
		dir := strings.TrimSuffix(m, "go.mod")
		if dir != "" && dir != "temporal-server/" && strings.HasPrefix(f, dir) {
			return false
		}
	}
	return true
}

func hasSeg(segs []string, s string) bool { return slices.Contains(segs, s) }

func sortedKeys(m map[string][]string) []string {
	return slices.Sorted(maps.Keys(m))
}

// relTo names p relative to root when it lies under it, else as given.
func relTo(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return filepath.ToSlash(p)
}
