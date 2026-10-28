// Command cisteps fails when the GitHub workflow stops running a required CI
// step (TICK-297). The list is tools/ci/cisteps/required.yaml; a step counts
// when its job runs it as a line of run: (or, for a report, uploads the named
// artifact with if-no-files-found: error), without `if: false` on the job or
// step and without continue-on-error.
//
// Usage: cisteps [-workflow .github/workflows/ci.yml] [-required tools/ci/cisteps/required.yaml]
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	workflow := flag.String("workflow", ".github/workflows/ci.yml", "the GitHub workflow to check")
	required := flag.String("required", "tools/ci/cisteps/required.yaml", "the list of required steps")
	flag.Parse()

	problems, n, err := check(*workflow, *required)
	if err != nil {
		fmt.Fprintln(os.Stderr, "check-ci-steps:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "check-ci-steps:", p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "check-ci-steps: %s must keep every step in %s\n", *workflow, *required)
		os.Exit(1)
	}
	fmt.Printf("check-ci-steps: ok: %d required steps in %s\n", n, *workflow)
}

// required is one entry of required.yaml: Run or Artifact, not both.
type required struct {
	Job      string `yaml:"job"`
	Run      string `yaml:"run"`
	Artifact string `yaml:"artifact"`
	Why      string `yaml:"why"`
}

func (r required) String() string {
	if r.Artifact != "" {
		return fmt.Sprintf("job %s: upload of artifact %q", r.Job, r.Artifact)
	}
	return fmt.Sprintf("job %s: step %q", r.Job, r.Run)
}

type step struct {
	Name            string            `yaml:"name"`
	If              yaml.Node         `yaml:"if"`
	Run             string            `yaml:"run"`
	Uses            string            `yaml:"uses"`
	With            map[string]string `yaml:"with"`
	ContinueOnError yaml.Node         `yaml:"continue-on-error"`
}

type job struct {
	If              yaml.Node `yaml:"if"`
	ContinueOnError yaml.Node `yaml:"continue-on-error"`
	Steps           []step    `yaml:"steps"`
}

// check returns one problem per required step the workflow does not run, and
// the number of required steps.
func check(workflowPath, requiredPath string) ([]string, int, error) {
	var list struct {
		Steps []required `yaml:"steps"`
	}
	if err := decode(requiredPath, &list); err != nil {
		return nil, 0, err
	}
	if len(list.Steps) == 0 {
		return nil, 0, fmt.Errorf("%s: no steps", requiredPath)
	}
	for i, r := range list.Steps {
		if r.Job == "" || (r.Run == "") == (r.Artifact == "") {
			return nil, 0, fmt.Errorf("%s: step %d: want job and exactly one of run or artifact", requiredPath, i+1)
		}
	}
	var wf struct {
		Jobs map[string]job `yaml:"jobs"`
	}
	if err := decode(workflowPath, &wf); err != nil {
		return nil, 0, err
	}

	var problems []string
	for _, r := range list.Steps {
		if p := find(wf.Jobs, r); p != "" {
			problems = append(problems, fmt.Sprintf("%s %s [%s]", r, p, r.Why))
		}
	}
	return problems, len(list.Steps), nil
}

// find returns why r is not run, or "" when it is.
func find(jobs map[string]job, r required) string {
	j, ok := jobs[r.Job]
	if !ok {
		return "is missing: no such job"
	}
	if disabled(j.If) {
		return "is disabled: the job has if: false"
	}
	if truthy(j.ContinueOnError) {
		return "may fail unnoticed: the job has continue-on-error"
	}
	for _, s := range j.Steps {
		if !matches(s, r) {
			continue
		}
		switch {
		case disabled(s.If):
			return fmt.Sprintf("is disabled: step %q has if: false", s.Name)
		case truthy(s.ContinueOnError):
			return fmt.Sprintf("may fail unnoticed: step %q has continue-on-error", s.Name)
		case r.Artifact != "" && s.With["if-no-files-found"] != "error":
			return fmt.Sprintf("may upload nothing: step %q lacks if-no-files-found: error", s.Name)
		}
		return ""
	}
	return "is missing"
}

func matches(s step, r required) bool {
	if r.Artifact != "" {
		return strings.HasPrefix(s.Uses, "actions/upload-artifact@") && s.With["name"] == r.Artifact
	}
	for line := range strings.SplitSeq(s.Run, "\n") {
		if strings.Join(strings.Fields(line), " ") == r.Run {
			return true
		}
	}
	return false
}

// disabled is an if: that can never hold: false, or ${{ false }}.
func disabled(n yaml.Node) bool {
	v := strings.TrimSpace(n.Value)
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(v, "${{"), "}}"))
	return n.Kind == yaml.ScalarNode && (v == "false" || v == "0" || v == "''")
}

// truthy is a continue-on-error that is not plainly false (an expression may
// be true at run time).
func truthy(n yaml.Node) bool {
	return n.Kind == yaml.ScalarNode && strings.TrimSpace(n.Value) != "false"
}

func decode(path string, v any) error {
	data, err := os.ReadFile(path) //nolint:gosec // a path the caller chose
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
