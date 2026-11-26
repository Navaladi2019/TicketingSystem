// Package mainred tests scripts/ci/main-red.sh, the job that opens or comments on the P0 issue
// when a workflow is red on main (TICK-289, TICK-1152). The tests run the script with a stub gh
// first on PATH: the stub records every call and answers "issue list" by piping a fixture
// through jq with the script's own --jq expression, so the title-prefix filter really runs.
package mainred

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const script = "../../../scripts/ci/main-red.sh"

// stubGH records each call as the arguments joined by \037, calls separated by \036.
const stubGH = `#!/bin/sh
{ printf 'CALL'; for a in "$@"; do printf '\037%s' "$a"; done; printf '\036'; } >>"$GH_LOG"
case "$1 $2" in
"issue list")
    expr=
    while [ $# -gt 0 ]; do
        if [ "$1" = --jq ]; then expr=$2; break; fi
        shift
    done
    [ -n "$expr" ] || { echo "stub gh: issue list without --jq" >&2; exit 2; }
    exec jq -r "$expr" "$GH_FIXTURE"
    ;;
"issue create" | "issue comment" | "label create") ;;
*) echo "stub gh: unexpected call: $*" >&2; exit 2 ;;
esac
`

const sha = "741623e0123456789abcdef0123456789abcdef0"

// skipOrFail skips on a laptop without the tool but fails in CI (GitHub sets CI=true), so a
// green unit job proves the cases ran rather than skipped.
func skipOrFail(t *testing.T, why string) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatal(why + "; CI must have it")
	}
	t.Skip(why)
}

// run executes main-red.sh with WORKFLOW=workflow ("" leaves it unset) against the open P0
// issues in fixture (a JSON array of {number, title}); it returns the gh calls, each as its
// argument list.
func run(t *testing.T, workflow, fixture string) [][]string {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		skipOrFail(t, "no sh on PATH")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		skipOrFail(t, "no jq on PATH")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	fx := filepath.Join(dir, "issues.json")
	if err := os.WriteFile(fx, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "gh.log")

	cmd := exec.CommandContext(t.Context(), sh, script)
	env := []string{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "WORKFLOW=") && !strings.HasPrefix(kv, "PATH=") && !strings.HasPrefix(kv, "EVENT=") {
			env = append(env, kv)
		}
	}
	env = append(env,
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GH_LOG="+log, "GH_FIXTURE="+fx,
		"GH_REPO=example/repo", "SHA="+sha, "RUN_URL=https://example.invalid/run/1",
		`NEEDS_JSON={"vulncheck":{"result":"failure"}}`,
	)
	if workflow != "" {
		env = append(env, "WORKFLOW="+workflow)
	}
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("main-red.sh: %v\n%s", err, out.String())
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	for rec := range strings.SplitSeq(strings.TrimSuffix(string(raw), "\036"), "\036") {
		calls = append(calls, strings.Split(rec, "\037")[1:])
	}
	return calls
}

// writes returns the issue create and issue comment calls, the ones that change an issue.
func writes(calls [][]string) [][]string {
	var w [][]string
	for _, c := range calls {
		if len(c) >= 2 && c[0] == "issue" && (c[1] == "create" || c[1] == "comment") {
			w = append(w, c)
		}
	}
	return w
}

// flag returns the value after name in args, or "" when it is absent.
func flag(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func TestMainRedTitlePerWorkflow(t *testing.T) {
	const (
		none     = `[]`
		mainOnly = `[{"number":259,"title":"main is red: 741623e"}]`
		both     = `[{"number":259,"title":"main is red: 741623e"},{"number":901,"title":"govulncheck is red: 1234567"}]`
	)
	cases := []struct {
		name, workflow, fixture string
		// wantCreate is the title of the issue created ("" for none); wantComment the issue
		// number commented on ("" for none).
		wantCreate, wantComment string
	}{
		{"a: govulncheck, no open issue", "govulncheck", none, "govulncheck is red: 741623e", ""},
		{"b: govulncheck ignores an open main-is-red issue", "govulncheck", mainOnly, "govulncheck is red: 741623e", ""},
		{"c: govulncheck comments on its own open issue", "govulncheck", both, "", "901"},
		{"d: ci, no open issue", "ci", none, "main is red: 741623e", ""},
		{"d: ci comments on the open main-is-red issue", "ci", both, "", "259"},
		{"d: WORKFLOW unset is ci", "", mainOnly, "", "259"},
		{"d: ci ignores an open govulncheck issue", "ci", `[{"number":901,"title":"govulncheck is red: 1234567"}]`, "main is red: 741623e", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := writes(run(t, tc.workflow, tc.fixture))
			if len(w) != 1 {
				t.Fatalf("want exactly one issue create or comment, got %q", w)
			}
			got := w[0]
			switch {
			case tc.wantCreate != "":
				if got[1] != "create" {
					t.Fatalf("want issue create %q, got %q", tc.wantCreate, got)
				}
				if title := flag(got, "--title"); title != tc.wantCreate {
					t.Errorf("title = %q, want %q", title, tc.wantCreate)
				}
				if label := flag(got, "--label"); label != "P0" {
					t.Errorf("label = %q, want P0", label)
				}
			default:
				if got[1] != "comment" || len(got) < 3 || got[2] != tc.wantComment {
					t.Fatalf("want issue comment on #%s, got %q", tc.wantComment, got)
				}
			}
			wf := tc.workflow
			if wf == "" {
				wf = "ci"
			}
			if body := flag(got, "--body"); !strings.HasPrefix(body, "`"+wf+"` failed on a push to `main` at "+sha) {
				t.Errorf("body = %q, want it to name workflow %s", body, wf)
			}
		})
	}
}
