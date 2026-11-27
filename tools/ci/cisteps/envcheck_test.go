package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// The dump prints one record of its own arguments and environment, which
// parseEnvDump reads back. make runs it as SHELL in a subprocess, so the
// make-backed tests never cover it in this process.
func TestEnvDumpRecordsArgsAndEnvironment(t *testing.T) {
	var buf bytes.Buffer
	if code := envDump(&buf); code != 0 {
		t.Fatalf("exit %d", code)
	}
	recs, other, err := parseEnvDump(buf.String())
	if err != nil || len(recs) != 1 || len(other) != 0 {
		t.Fatalf("records %+v, other %q, err %v", recs, other, err)
	}
	if !slices.Equal(recs[0].Args, os.Args[1:]) {
		t.Errorf("args %q, want %q", recs[0].Args, os.Args[1:])
	}
	if len(recs[0].Env) != len(os.Environ()) {
		t.Errorf("%d variables, want %d", len(recs[0].Env), len(os.Environ()))
	}
}

// dumpRunner is a make runner whose environment dump reports recs and extra
// lines of output, or fails with err.
func dumpRunner(extra string, err error, recs ...envRecord) makeRunner {
	return func(...string) ([]byte, error) {
		var out strings.Builder
		for _, r := range recs {
			b, _ := json.Marshal(r)
			out.WriteString(envDumpPrefix + string(b) + "\n")
		}
		out.WriteString(extra)
		return []byte(out.String()), err
	}
}

// Each kind of difference between the recipe's environment and the
// baseline's is named, an allowed export is not, and anything but one shell
// call of the pinned line with no other output fails.
func TestEnvironmentProblemsNamesEachDifference(t *testing.T) {
	r := pinnedRecipes()[0]
	call := []string{"-c", r.dryRun()}
	base := map[string]string{"PATH": "/usr/bin", "HOME": "/home/ci", "GOFLAGS": ""}
	cases := []struct {
		name string
		mk   makeRunner
		want []string
	}{
		{"same", dumpRunner("", nil, envRecord{call, []string{"PATH=/usr/bin", "HOME=/home/ci", "GOFLAGS="}}), nil},
		{"allowed export", dumpRunner("", nil, envRecord{call, []string{"PATH=/usr/bin", "HOME=/home/ci", "GOFLAGS=", "NAVA_IMAGE_TAG=dev"}}), nil},
		{"unexported, changed and added", dumpRunner("", nil, envRecord{call, []string{"PATH=/tmp/fake", "GOFLAGS=", "CGO_ENABLED=0"}}), []string{
			"make vulncheck: recipes get CGO_ENABLED=\"0\", an empty makefile gives none",
			"make vulncheck: recipes get no HOME, an empty makefile gives HOME=\"/home/ci\"",
			"make vulncheck: recipes get PATH=\"/tmp/fake\", an empty makefile gives \"/usr/bin\"",
		}},
		{"two shell calls", dumpRunner("", nil, envRecord{call, nil}, envRecord{[]string{"-c", "stub"}, nil}), []string{"make vulncheck runs "}},
		{"another command", dumpRunner("", nil, envRecord{[]string{"-c", "true"}, nil}), []string{"make vulncheck runs "}},
		{"no call", dumpRunner("", nil), []string{"make vulncheck runs "}},
		{"other output", dumpRunner("make: something\n", nil, envRecord{call, []string{"PATH=/usr/bin", "HOME=/home/ci", "GOFLAGS="}}), []string{"and prints [\"make: something\"]"}},
		{"make fails", dumpRunner("make: *** [vulncheck] Error 2\n", errors.New("exit status 2")), []string{"make vulncheck through the environment dump fails: exit status 2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			problems, err := environmentProblems(r, base, c.mk)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != len(c.want) {
				t.Fatalf("problems %q, want %d starting %q", problems, len(c.want), c.want)
			}
			for i, w := range c.want {
				if !strings.Contains(problems[i], w) {
					t.Errorf("problem %d is %q, want it to contain %q", i, problems[i], w)
				}
			}
		})
	}
	if _, err := environmentProblems(r, base, dumpRunner(envDumpPrefix+"{\n", nil)); err == nil {
		t.Error("a garbled record is not an error")
	}
}

func TestAllowedExportAndFuncName(t *testing.T) {
	for name, want := range map[string]bool{"NAVA_IMAGE_TAG": true, "COMPOSE_FILE": true, "GOFLAGS": false, "HOME": false} {
		if got := allowedExport(name); got != want {
			t.Errorf("allowedExport(%q) = %v, want %v", name, got, want)
		}
	}
	for s, want := range map[string]string{"shell echo x)": "shell", "eval": "eval", "call,x": "call"} {
		if got := funcName(s); got != want {
			t.Errorf("funcName(%q) = %q, want %q", s, got, want)
		}
	}
}
