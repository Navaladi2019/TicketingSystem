package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const fixtureRequired = "testdata/required.yaml"

func TestFixturePasses(t *testing.T) {
	problems, n, err := check("testdata/ok.yml", fixtureRequired)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 || n != 3 {
		t.Fatalf("want 3 steps and no problems, got %d and %q", n, problems)
	}
}

// Each edit to the fixture workflow must fail with exactly the problem given.
func TestEditsFail(t *testing.T) {
	ok, err := os.ReadFile("testdata/ok.yml")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, old, new, want string
	}{
		{"step removed", "          make   identity-attacks\n", "", `job stack: step "make identity-attacks" is missing [CI-5]`},
		{"target renamed", "run: make test\n", "run: make test-short\n", `job unit: step "make test" is missing [unit tests]`},
		{"step commented out", "          make   identity-attacks", "          # make identity-attacks", `job stack: step "make identity-attacks" is missing [CI-5]`},
		{"artifact renamed", "name: ci5-report", "name: ci5", `job stack: upload of artifact "ci5-report" is missing [the CI-5 report]`},
		{"artifact may be empty", "if-no-files-found: error", "if-no-files-found: warn", `job stack: upload of artifact "ci5-report" may upload nothing: step "CI-5 report" lacks if-no-files-found: error [the CI-5 report]`},
		{"step disabled", "      - name: go test -race\n", "      - name: go test -race\n        if: false\n", `job unit: step "make test" is disabled: step "go test -race" has if: false [unit tests]`},
		{"step disabled by expression", "      - name: go test -race\n", "      - name: go test -race\n        if: ${{ false }}\n", `job unit: step "make test" is disabled: step "go test -race" has if: false [unit tests]`},
		{"job disabled", "    if: needs.changes.outputs.unit == 'true'\n", "    if: false\n", `job unit: step "make test" is disabled: the job has if: false [unit tests]`},
		{"step allowed to fail", "continue-on-error: false", "continue-on-error: true", `job stack: step "make identity-attacks" may fail unnoticed: step "CI-5 adversarial identity suite (T19)" has continue-on-error [CI-5]`},
		{"job allowed to fail", "  unit:\n", "  unit:\n    continue-on-error: ${{ github.event_name == 'push' }}\n", `job unit: step "make test" may fail unnoticed: the job has continue-on-error [unit tests]`},
		{"job removed", "  unit:\n", "  unit-tests:\n", `job unit: step "make test" is missing: no such job [unit tests]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(string(ok), c.old) {
				t.Fatalf("fixture has no %q", c.old)
			}
			wf := writeTemp(t, strings.Replace(string(ok), c.old, c.new, 1))
			problems, _, err := check(wf, fixtureRequired)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != 1 || problems[0] != c.want {
				t.Fatalf("want [%q], got %q", c.want, problems)
			}
		})
	}
}

func TestBadRequiredListIsAnError(t *testing.T) {
	for name, body := range map[string]string{
		"empty":    "steps: []\n",
		"no job":   "steps:\n  - run: make test\n",
		"both":     "steps:\n  - job: unit\n    run: make test\n    artifact: x\n",
		"neither":  "steps:\n  - job: unit\n",
		"not yaml": "steps: [\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := check("testdata/ok.yml", writeTemp(t, body)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func repo(p string) string { return filepath.Join("..", "..", "..", p) }

// The repository's workflow runs every required step.
func TestRepository(t *testing.T) {
	problems, _, err := check(repo(".github/workflows/ci.yml"), repo("tools/ci/cisteps/required.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("%q", problems)
	}
}

// Deleting any one required step from the repository's own ci.yml fails the
// check, naming that step (TICK-297 acceptance).
func TestDroppingAnyRequiredStepFails(t *testing.T) {
	var list struct {
		Steps []required `yaml:"steps"`
	}
	if err := decode(repo("tools/ci/cisteps/required.yaml"), &list); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(repo(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list.Steps {
		t.Run(r.String(), func(t *testing.T) {
			var doc yaml.Node
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			if !dropStep(t, &doc, r) {
				t.Fatalf("ci.yml has no step for %s", r)
			}
			out, err := yaml.Marshal(&doc)
			if err != nil {
				t.Fatal(err)
			}
			problems, _, err := check(writeTemp(t, string(out)), repo("tools/ci/cisteps/required.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != 1 || !strings.HasPrefix(problems[0], r.String()+" is missing") {
				t.Fatalf("want one problem for %s, got %q", r, problems)
			}
		})
	}
}

// dropStep removes the step of job r.Job that satisfies r.
func dropStep(t *testing.T, doc *yaml.Node, r required) bool {
	t.Helper()
	jobs := value(doc.Content[0], "jobs")
	j := value(jobs, r.Job)
	steps := value(j, "steps")
	if steps == nil {
		return false
	}
	for i, n := range steps.Content {
		var s step
		if err := n.Decode(&s); err != nil {
			t.Fatal(err)
		}
		if matches(s, r) {
			steps.Content = append(steps.Content[:i], steps.Content[i+1:]...)
			return true
		}
	}
	return false
}

func value(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
