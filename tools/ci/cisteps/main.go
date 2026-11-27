// Command cisteps fails when the GitHub workflow stops running a required CI
// step (TICK-297). The list is tools/ci/cisteps/required.yaml; a step counts
// when its job runs it as a line of run: (or, for a report, uploads the named
// artifact with if-no-files-found: error), without `if: false` on the job or
// step and without continue-on-error. The list's outputs pin a job output to
// one expression (TICK-1031): the changes job's full must be the stages
// step's, not a second copy of the full-run rule that can drift from it. A job
// that restores ~/go/pkg/mod must use setup-go v6 or later (TICK-1142).
// The list's checks pin a required status check to a job that ran
// (TICK-1146): GitHub names a skipped job's check run after its unevaluated
// name: expression when it reads a context, so a job named
// ${{ github.event_name && 'isolation' || 'isolation' }} reports "isolation"
// only when it ran, and a skipped run on the same commit (a label-only run, a
// fast run, an unlabelled run) can no longer replace a red result. A constant
// ${{ 'isolation' }} does not work: GitHub folds it to the literal name even
// on a skipped run (verification PR 904).
//
// With -copy (TICK-1149) the check is a different one: the whole workflow,
// comment lines aside, must equal the checked-in copy, and the vulncheck make
// recipes must be the pinned lines (copy.go).
//
// Usage: cisteps [-workflow .github/workflows/ci.yml] [-required tools/ci/cisteps/required.yaml]
//
//	cisteps -workflow .github/workflows/govulncheck.yml -copy tools/ci/cisteps/govulncheck.copy.yml
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run returns the exit code: 0 every required step runs, 1 one does not,
// 2 bad usage or an unreadable input.
func run(args []string, stdout, stderr io.Writer) int {
	return runWith(args, stdout, stderr, makeIn(""))
}

// runWith is run with the make runner of the -copy mode's recipe check.
func runWith(args []string, stdout, stderr io.Writer, mk makeRunner) int {
	fs := flag.NewFlagSet("cisteps", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workflow := fs.String("workflow", ".github/workflows/ci.yml", "the GitHub workflow to check")
	required := fs.String("required", "tools/ci/cisteps/required.yaml", "the list of required steps")
	copyPath := fs.String("copy", "", "a checked-in copy the whole workflow must equal, comment lines aside (needs -workflow, excludes -required)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if set["copy"] {
		if set["required"] || !set["workflow"] || *copyPath == "" {
			fmt.Fprintln(stderr, "check-ci-steps: -copy needs -workflow and excludes -required")
			fs.Usage()
			return 2
		}
		return runCopy(*workflow, *copyPath, mk, stdout, stderr)
	}

	problems, n, err := check(*workflow, *required)
	if err != nil {
		fmt.Fprintln(stderr, "check-ci-steps:", err)
		return 2
	}
	for _, p := range problems {
		fmt.Fprintln(stderr, "check-ci-steps:", p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(stderr, "check-ci-steps: %s must keep every step and output in %s\n", *workflow, *required)
		return 1
	}
	fmt.Fprintf(stdout, "check-ci-steps: ok: %d required steps and outputs in %s\n", n, *workflow)
	return 0
}

// runCopy is run's -copy mode.
func runCopy(workflow, copyPath string, mk makeRunner, stdout, stderr io.Writer) int {
	problems, err := checkCopy(workflow, copyPath, mk)
	if err != nil {
		fmt.Fprintln(stderr, "check-ci-steps:", err)
		return 2
	}
	for _, p := range problems {
		fmt.Fprintln(stderr, "check-ci-steps:", p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(stderr, "check-ci-steps: edit %s and %s together, or not at all (ADR-001-E4 contract rule 9)\n", workflow, copyPath)
		return 1
	}
	fmt.Fprintf(stdout, "check-ci-steps: ok: %s equals %s and the vulncheck recipes are pinned\n", workflow, copyPath)
	return 0
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

// output is one entry of required.yaml's outputs: the job's output must be
// Value, whitespace aside.
type output struct {
	Job    string `yaml:"job"`
	Output string `yaml:"output"`
	Value  string `yaml:"value"`
	Why    string `yaml:"why"`
}

// requiredCheck is one entry of required.yaml's checks: job Job must be
// named ${{ github.event_name && 'Check' || 'Check' }}.
type requiredCheck struct {
	Job   string `yaml:"job"`
	Check string `yaml:"check"`
	Why   string `yaml:"why"`
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
	Name            string            `yaml:"name"`
	If              yaml.Node         `yaml:"if"`
	ContinueOnError yaml.Node         `yaml:"continue-on-error"`
	Steps           []step            `yaml:"steps"`
	Outputs         map[string]string `yaml:"outputs"`
}

// check returns one problem per required step the workflow does not run and
// per pinned output it does not set as listed, and the number of both.
func check(workflowPath, requiredPath string) ([]string, int, error) {
	var list struct {
		Steps   []required      `yaml:"steps"`
		Outputs []output        `yaml:"outputs"`
		Checks  []requiredCheck `yaml:"checks"`
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
	for i, o := range list.Outputs {
		if o.Job == "" || o.Output == "" || o.Value == "" {
			return nil, 0, fmt.Errorf("%s: output %d: want job, output and value", requiredPath, i+1)
		}
	}
	for i, c := range list.Checks {
		if c.Job == "" || c.Check == "" {
			return nil, 0, fmt.Errorf("%s: check %d: want job and check", requiredPath, i+1)
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
	for _, o := range list.Outputs {
		if p := findOutput(wf.Jobs, o); p != "" {
			problems = append(problems, fmt.Sprintf("job %s: output %q %s [%s]", o.Job, o.Output, p, o.Why))
		}
	}
	if len(list.Checks) > 0 {
		others, err := fixedNames(workflowPath)
		if err != nil {
			return nil, 0, err
		}
		for _, c := range list.Checks {
			if p := findCheck(wf.Jobs, c); p != "" {
				problems = append(problems, fmt.Sprintf("job %s: check %q %s [%s]", c.Job, c.Check, p, c.Why))
			}
			problems = append(problems, otherReporters(others, workflowPath, c)...)
		}
	}
	problems = append(problems, oldSetupGo(wf.Jobs)...)
	return problems, len(list.Steps) + len(list.Outputs) + len(list.Checks), nil
}

// reporter is a job of some workflow and the name its check run gets even
// when the job is skipped: a literal name, a constant ${{ 'x' }}, or its id
// when it has no name. A name that reads a context has none.
type reporter struct {
	file, job, name string
}

// fixedNames lists the fixed check names of every job in every workflow of
// workflowPath's directory, workflowPath included, sorted by file and job.
func fixedNames(workflowPath string) ([]reporter, error) {
	dir := filepath.Dir(workflowPath)
	var files []string
	for _, pat := range []string{"*.yml", "*.yaml"} {
		m, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			return nil, err
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	var out []reporter
	for _, f := range files {
		var wf struct {
			Jobs map[string]job `yaml:"jobs"`
		}
		if err := decode(f, &wf); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(wf.Jobs))
		for id := range wf.Jobs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if n, ok := fixedName(id, wf.Jobs[id].Name); ok {
				out = append(out, reporter{file: f, job: id, name: n})
			}
		}
	}
	return out, nil
}

// fixedName is the check name a skipped run of the job gets, when that name
// does not depend on the run.
func fixedName(id, name string) (string, bool) {
	n := squash(name)
	switch {
	case n == "":
		return id, true
	case !strings.Contains(n, "${{"):
		return n, true
	}
	inner, ok := strings.CutPrefix(n, "${{")
	if !ok {
		return "", false
	}
	inner, ok = strings.CutSuffix(inner, "}}")
	if !ok {
		return "", false
	}
	inner = strings.TrimSpace(inner)
	if len(inner) >= 2 && inner[0] == '\'' && inner[len(inner)-1] == '\'' && !strings.Contains(inner[1:len(inner)-1], "'") {
		return inner[1 : len(inner)-1], true
	}
	return "", false
}

// otherReporters returns one problem per job other than c's pinned one that
// reports c.Check under a fixed name.
func otherReporters(all []reporter, workflowPath string, c requiredCheck) []string {
	var problems []string
	for _, r := range all {
		if r.name != c.Check || (filepath.Clean(r.file) == filepath.Clean(workflowPath) && r.job == c.Job) {
			continue
		}
		problems = append(problems, fmt.Sprintf("job %s in %s: reports check %q under a fixed name, so a skipped run of it passes the required check; only job %s in %s may report it [%s]",
			r.job, filepath.Base(r.file), c.Check, c.Job, filepath.Base(workflowPath), c.Why))
	}
	return problems
}

// findCheck returns why c's job may report c.Check without running, or
// never reports it, or "".
func findCheck(jobs map[string]job, c requiredCheck) string {
	j, ok := jobs[c.Job]
	if !ok {
		return "is missing: no such job"
	}
	want := "${{ github.event_name && '" + c.Check + "' || '" + c.Check + "' }}"
	got := squash(j.Name)
	if inner, ok := strings.CutPrefix(got, "${{"); ok {
		if inner, ok = strings.CutSuffix(inner, "}}"); ok {
			got = "${{ " + strings.TrimSpace(inner) + " }}"
		}
	}
	switch got {
	case want:
		return ""
	case "", c.Check, "${{ '" + c.Check + "' }}":
		return fmt.Sprintf("is reported by a skipped run: name is %q, want %q", got, want)
	}
	return fmt.Sprintf("is never reported: name is %q, want %q", got, want)
}

const minSetupGo = 6

// oldSetupGo returns one problem per setup-go step older than v6 in a job that
// restores ~/go/pkg/mod (TICK-1142).
func oldSetupGo(jobs map[string]job) []string {
	var problems []string
	for _, name := range slices.Sorted(maps.Keys(jobs)) {
		steps := jobs[name].Steps
		if !slices.ContainsFunc(steps, restoresModCache) {
			continue
		}
		for _, s := range steps {
			ref, ok := strings.CutPrefix(s.Uses, "actions/setup-go@")
			if !ok {
				continue
			}
			major, _, _ := strings.Cut(strings.TrimPrefix(ref, "v"), ".")
			if n, err := strconv.Atoi(major); err != nil || n < minSetupGo {
				problems = append(problems, fmt.Sprintf("job %s: %s must be v%d or later: older ones ignore go.mod's toolchain line, so go downloads it into ~/go/pkg/mod and the Go cache restore fails (TICK-1142)", name, s.Uses, minSetupGo))
			}
		}
	}
	return problems
}

func restoresModCache(s step) bool {
	cache := strings.HasPrefix(s.Uses, "actions/cache@") || strings.HasPrefix(s.Uses, "actions/cache/restore@")
	return cache && strings.Contains(s.With["path"], "~/go/pkg/mod")
}

// findOutput returns why o's job does not set o.Output to o.Value, or "".
func findOutput(jobs map[string]job, o output) string {
	j, ok := jobs[o.Job]
	if !ok {
		return "is missing: no such job"
	}
	got, ok := j.Outputs[o.Output]
	if !ok {
		return "is missing"
	}
	if squash(got) != squash(o.Value) {
		return fmt.Sprintf("is %q, want %q", squash(got), squash(o.Value))
	}
	return ""
}

// squash collapses every run of whitespace to one space.
func squash(s string) string {
	return strings.Join(strings.Fields(s), " ")
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
