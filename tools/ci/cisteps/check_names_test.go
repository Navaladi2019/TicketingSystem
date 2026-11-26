package main

import (
	"os"
	"path/filepath"
	"slices"
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
			name := id
			if j.Name != "" {
				name = j.Name
			}
			out[name] = append(out[name], e.Name())
		}
	}
	return out
}
