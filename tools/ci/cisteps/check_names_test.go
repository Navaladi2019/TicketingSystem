package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TICK-1145 (TICK-1140 S5, ADR-001-E4 E4-4): branch protection knows a check by its reported
// name and the app, not by its workflow. A name reported by two workflows lets
// the green vulnerability scan stand in for the red Authorizer tests. The rule
// covers every job of every workflow, so no list of required names can drift.
func TestNoTwoWorkflowsReportTheSameCheck(t *testing.T) {
	for name, files := range reportedChecks(t, repo(".github/workflows")) {
		if len(files) > 1 {
			t.Errorf("check %q is reported by %v; branch protection cannot tell them apart", name, files)
		}
	}
}

// TICK-1193: inside one workflow two jobs may report one check only when
// their if: conditions exclude each other, which no test can read. So the one
// such pair is listed: ci.yml's isolation (the suite) and isolation-not-needed
// (a docs-only diff), both under TICK-1146's computed name. Any other pair
// fails here until it is reviewed and added.
func TestOneWorkflowSharesANameOnlyForIsolation(t *testing.T) {
	allowed := map[string][]string{"ci.yml isolation": {"isolation", "isolation-not-needed"}}
	dir := repo(".github/workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); ext != ".yml" && ext != ".yaml" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Jobs map[string]struct {
				Name string `yaml:"name"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(body, &w); err != nil {
			t.Fatal(err)
		}
		jobs := map[string][]string{}
		for id, j := range w.Jobs {
			names := []string{id}
			if j.Name != "" {
				if names, err = reportedNames(j.Name); err != nil {
					t.Fatalf("%s job %s: %v", e.Name(), id, err)
				}
			}
			for _, n := range names {
				jobs[n] = append(jobs[n], id)
			}
		}
		for n, ids := range jobs {
			if len(ids) < 2 {
				continue
			}
			slices.Sort(ids)
			key := e.Name() + " " + n
			if want, ok := allowed[key]; !ok || !slices.Equal(ids, want) {
				t.Errorf("%s: check %q is reported by jobs %v; only %v may share one", e.Name(), n, ids, allowed)
			}
			delete(allowed, key)
		}
	}
	for key, ids := range allowed {
		t.Errorf("%s: jobs %v no longer share the check; update this list", key, ids)
	}
}

func TestReportedChecksReadsNameAndYaml(t *testing.T) {
	dir := t.TempDir()
	for f, body := range map[string]string{
		"a.yml":  "jobs:\n  isolation: {}\n",
		"b.yaml": "jobs:\n  other:\n    name: isolation\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := reportedChecks(t, dir)["isolation"]; !slices.Equal(got, []string{"a.yml", "b.yaml"}) {
		t.Fatalf("isolation reported by %v, want a.yml and b.yaml", got)
	}
}

// ADR-001-E4 decision E4-5 (TICK-1150): a name of the form
// ${{ <cond> && 'x' || 'y' }} reports x or y depending on the event, so both
// are taken. A second workflow with a job named vulncheck-other collides with
// the gate even though it never reports vulncheck.
func TestReportedChecksReadsBothValuesOfAConditionalName(t *testing.T) {
	gate := "jobs:\n  vulncheck:\n    name: ${{ github.base_ref == 'main' && 'vulncheck' || 'vulncheck-other' }}\n"
	for _, c := range []struct {
		name, other string
		want        map[string][]string
	}{
		{"second workflow reports vulncheck-other", "jobs:\n  vulncheck-other: {}\n", map[string][]string{"vulncheck": {"a.yml"}, "vulncheck-other": {"a.yml", "b.yml"}}},
		{"second workflow names a job vulncheck", "jobs:\n  x:\n    name: vulncheck\n", map[string][]string{"vulncheck": {"a.yml", "b.yml"}, "vulncheck-other": {"a.yml"}}},
		{"no collision", "jobs:\n  lint: {}\n", map[string][]string{"vulncheck": {"a.yml"}, "vulncheck-other": {"a.yml"}, "lint": {"b.yml"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			for f, body := range map[string]string{"a.yml": gate, "b.yml": c.other} {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got := reportedChecks(t, dir)
			if len(got) != len(c.want) {
				t.Errorf("reported %v, want %v", got, c.want)
			}
			for name, files := range c.want {
				if !slices.Equal(got[name], files) {
					t.Errorf("%q reported by %v, want %v", name, got[name], files)
				}
			}
		})
	}
}

// A name expression of any other shape cannot be read, so it fails rather
// than being taken as the literal text.
func TestReportedNamesRefusesOtherExpressions(t *testing.T) {
	for _, n := range []string{
		"${{ matrix.name }}",
		"${{ github.base_ref == 'main' && 'vulncheck' }}",
		"${{ a && 'x' || b }}",
		"pre-${{ a && 'x' || 'y' }}",
		"${{ a && 'x' || 'y' }} ${{ b && 'z' || 'w' }}",
	} {
		if got, err := reportedNames(n); err == nil {
			t.Errorf("reportedNames(%q) = %q, want an error", n, got)
		}
	}
	for n, want := range map[string][]string{
		"isolation": {"isolation"},
		"${{ github.base_ref == 'main' && 'vulncheck' || 'vulncheck-other' }}": {"vulncheck", "vulncheck-other"},
		"${{a&&'x'||'y'}}": {"x", "y"},
		"${{ github.event_name && 'isolation' || 'isolation' }}": {"isolation"},
	} {
		if got, err := reportedNames(n); err != nil || !slices.Equal(got, want) {
			t.Errorf("reportedNames(%q) = %q, %v; want %q", n, got, err, want)
		}
	}
}

// conditionalName is contract rule 5's name expression: either literal can be
// the reported name, depending on the event.
var conditionalName = regexp.MustCompile(`^\$\{\{[^'}]*(?:'[^']*'[^'}]*)*?&&\s*'([^']*)'\s*\|\|\s*'([^']*)'\s*\}\}$`)

// reportedNames returns every check name a job's name: can report: the name
// itself, or both literals of ${{ <cond> && 'x' || 'y' }}. Any other
// expression is an error: its reported name cannot be read.
func reportedNames(name string) ([]string, error) {
	if !strings.Contains(name, "${{") {
		return []string{name}, nil
	}
	m := conditionalName.FindStringSubmatch(name)
	if m == nil {
		return nil, fmt.Errorf("name %q: only ${{ <cond> && 'x' || 'y' }} can be read as reported names", name)
	}
	if m[1] == m[2] {
		return []string{m[1]}, nil // ci.yml's isolation job: one name either way
	}
	return []string{m[1], m[2]}, nil
}

// reportedChecks maps each reported check name (a job's name:, else its id) to
// the workflow files, in order, that report it.
func reportedChecks(t *testing.T, dir string) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); ext != ".yml" && ext != ".yaml" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Jobs map[string]struct {
				Name string `yaml:"name"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(body, &w); err != nil {
			t.Fatal(err)
		}
		for id, j := range w.Jobs {
			names := []string{id}
			if j.Name != "" {
				if names, err = reportedNames(j.Name); err != nil {
					t.Fatalf("%s job %s: %v", e.Name(), id, err)
				}
			}
			for _, name := range names {
				// One entry per workflow: two jobs of one workflow may share a
				// name on purpose (TestOneWorkflowSharesANameOnlyForIsolation).
				if !slices.Contains(out[name], e.Name()) {
					out[name] = append(out[name], e.Name())
				}
			}
		}
	}
	return out
}
