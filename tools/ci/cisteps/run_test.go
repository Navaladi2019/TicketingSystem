package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The gate: a workflow that drops a required step exits 1 and names the step.
func TestRunExitCodesAndMessages(t *testing.T) {
	ok, err := os.ReadFile("testdata/ok.yml")
	if err != nil {
		t.Fatal(err)
	}
	broken := writeTemp(t, strings.Replace(string(ok), "run: make test\n", "run: make lint\n", 1))
	missing := filepath.Join(t.TempDir(), "nope.yml")
	cases := []struct {
		name      string
		args      []string
		code      int
		wantOut   string
		wantErr   []string
		wantNoOut bool
	}{
		{
			name:    "ok",
			args:    []string{"-workflow", "testdata/ok.yml", "-required", fixtureRequired},
			code:    0,
			wantOut: "check-ci-steps: ok: 3 required steps in testdata/ok.yml\n",
		},
		{
			name: "required step dropped",
			args: []string{"-workflow", broken, "-required", fixtureRequired},
			code: 1,
			wantErr: []string{
				`check-ci-steps: job unit: step "make test" is missing [unit tests]` + "\n",
				"check-ci-steps: " + broken + " must keep every step in " + fixtureRequired + "\n",
			},
			wantNoOut: true,
		},
		{
			name:      "workflow unreadable",
			args:      []string{"-workflow", missing, "-required", fixtureRequired},
			code:      2,
			wantErr:   []string{"check-ci-steps: open " + missing},
			wantNoOut: true,
		},
		{
			name:      "unknown flag",
			args:      []string{"-nope"},
			code:      2,
			wantErr:   []string{"flag provided but not defined: -nope"},
			wantNoOut: true,
		},
		{
			name:      "help",
			args:      []string{"-h"},
			code:      0,
			wantErr:   []string{"-workflow", "-required"},
			wantNoOut: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(c.args, &stdout, &stderr); code != c.code {
				t.Fatalf("exit %d, want %d; stderr %q", code, c.code, stderr.String())
			}
			if c.wantOut != "" && stdout.String() != c.wantOut {
				t.Errorf("stdout %q, want %q", stdout.String(), c.wantOut)
			}
			if c.wantNoOut && stdout.Len() != 0 {
				t.Errorf("stdout %q, want empty", stdout.String())
			}
			for _, w := range c.wantErr {
				if !strings.Contains(stderr.String(), w) {
					t.Errorf("stderr %q lacks %q", stderr.String(), w)
				}
			}
			if c.wantErr == nil && stderr.Len() != 0 {
				t.Errorf("stderr %q, want empty", stderr.String())
			}
		})
	}
}

func TestCheckInputErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.yaml")
	cases := []struct {
		name, workflow, required, want string
	}{
		{"required list missing", "testdata/ok.yml", missing, "nope.yaml"},
		{"required list empty", "testdata/ok.yml", writeTemp(t, "steps: []\n"), ": no steps"},
		{"required step without job", "testdata/ok.yml", writeTemp(t, "steps:\n  - run: make test\n"), "step 1: want job and exactly one of run or artifact"},
		{"required step with both", "testdata/ok.yml", writeTemp(t, "steps:\n  - job: unit\n    run: make test\n  - job: unit\n    run: make test\n    artifact: x\n"), "step 2: want job and exactly one of run or artifact"},
		{"required list not yaml", "testdata/ok.yml", writeTemp(t, "steps: [\n"), "f.yml: yaml:"},
		{"workflow missing", missing, fixtureRequired, "nope.yaml"},
		{"workflow not yaml", writeTemp(t, "jobs: [\n"), fixtureRequired, "f.yml: yaml:"},
		{"workflow jobs not a mapping", writeTemp(t, "jobs: [a, b]\n"), fixtureRequired, "cannot unmarshal"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			problems, n, err := check(c.workflow, c.required)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
			if problems != nil || n != 0 {
				t.Errorf("on error got problems %q and n %d, want none", problems, n)
			}
		})
	}
}

func TestRequiredString(t *testing.T) {
	if got, want := (required{Job: "stack", Artifact: "r"}).String(), `job stack: upload of artifact "r"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := (required{Job: "unit", Run: "make test"}).String(), `job unit: step "make test"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
