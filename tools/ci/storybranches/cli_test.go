package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The test binary doubles as a fake gh: cli runs `gh pr list`, and the tests
// put a copy of this binary named gh first on PATH. As gh it records its
// arguments, then prints the JSON file it is given or fails.
const (
	fakeGHEnv  = "STORYBRANCHES_FAKE_GH"      // set: this process is the fake gh
	ghOutEnv   = "STORYBRANCHES_GH_OUT"       // file whose content gh prints
	ghFailEnv  = "STORYBRANCHES_GH_FAIL"      // set: gh fails with this message
	ghArgsFile = "STORYBRANCHES_GH_ARGS_FILE" // file gh writes its arguments to
)

func TestMain(m *testing.M) {
	if os.Getenv(fakeGHEnv) != "" {
		os.Exit(fakeGH())
	}
	os.Exit(m.Run())
}

func fakeGH() int {
	if f := os.Getenv(ghArgsFile); f != "" {
		_ = os.WriteFile(f, []byte(strings.Join(os.Args[1:], " ")), 0o600)
	}
	if msg := os.Getenv(ghFailEnv); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
		return 1
	}
	data, err := os.ReadFile(os.Getenv(ghOutEnv))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = os.Stdout.Write(data)
	return 0
}

// installFakeGH puts a gh that prints prs first on PATH and returns the file
// it records its arguments in.
func installFakeGH(t *testing.T, prs []PR) string {
	t.Helper()
	dir := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	name := "gh"
	if runtime.GOOS == "windows" {
		name = "gh.exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), bin, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(fakeGHEnv, "1")
	t.Setenv(ghFailEnv, "")
	setPRs(t, prs)
	args := filepath.Join(dir, "args")
	t.Setenv(ghArgsFile, args)
	return args
}

func setPRs(t *testing.T, prs []PR) {
	t.Helper()
	data, err := json.Marshal(prs)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "prs.json")
	if err := os.WriteFile(out, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ghOutEnv, out)
}

// testRepo is a work clone of a bare origin; the tests run inside it.
type testRepo struct {
	t   *testing.T
	dir string
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	root := t.TempDir()
	r := &testRepo{t: t, dir: filepath.Join(root, "work")}
	r.git(root, "init", "-q", "--bare", "-b", "main", "origin.git")
	r.git(root, "init", "-q", "-b", "main", "work")
	r.run("remote", "add", "origin", filepath.Join(root, "origin.git"))
	t.Chdir(r.dir)
	return r
}

func (r *testRepo) git(dir string, args ...string) string {
	r.t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func (r *testRepo) run(args ...string) string { r.t.Helper(); return r.git(r.dir, args...) }

func (r *testRepo) commit(file, body string) string {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, file), []byte(body), 0o600); err != nil {
		r.t.Fatal(err)
	}
	r.run("add", "-A")
	r.run("-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", file)
	return r.run("rev-parse", "HEAD")
}

func runCLI(args ...string) (code int, stdout, stderr string) {
	var o, e bytes.Buffer
	code = cli(args, &o, &e)
	return code, o.String(), e.String()
}

func prAt(n int, state, head, base, oid string, created, mergedAt time.Time) PR {
	p := PR{Number: n, State: state, Head: head, Base: base, HeadOid: oid, CreatedAt: created}
	if state == "MERGED" {
		p.MergedAt, p.ClosedAt = mergedAt, mergedAt
	}
	return p
}

const story = "TICK-1/story"

// storyRepo: TICK-1/story squash-merged into main by story PR #2, plus two
// in-flight story branches (TICK-5 with two task PRs and no story PR, TICK-6
// with one task PR and open story PR #7).
func storyRepo(t *testing.T) (r *testRepo, storyHead string, prs []PR, merged time.Time) {
	t.Helper()
	r = newTestRepo(t)
	r.commit("a", "base\n")
	r.run("checkout", "-q", "-b", story)
	storyHead = r.commit("b", "task 1\n")
	r.run("checkout", "-q", "main")
	r.commit("b", "task 1\n")
	for _, b := range []string{"TICK-5/wip", "TICK-6/review"} {
		r.run("checkout", "-q", "-b", b, "main")
		r.commit(strings.ReplaceAll(b, "/", "-"), b+"\n")
	}
	r.run("checkout", "-q", "main")
	r.run("push", "-q", "origin", "main", story, "TICK-5/wip", "TICK-6/review")

	now := time.Now().UTC().Truncate(time.Second)
	merged = now.Add(-10 * time.Minute)
	prs = []PR{
		prAt(1, "MERGED", "TICK-2/task", story, "t2", now.Add(-time.Hour), now.Add(-30*time.Minute)),
		prAt(2, "MERGED", story, "main", storyHead, now.Add(-20*time.Minute), merged),
		prAt(3, "MERGED", "TICK-3/a", "TICK-5/wip", "t3", now.Add(-time.Hour), time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)),
		prAt(4, "MERGED", "TICK-4/b", "TICK-5/wip", "t4", now.Add(-time.Hour), time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)),
		prAt(6, "MERGED", "TICK-8/c", "TICK-6/review", "t8", now.Add(-time.Hour), time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)),
		prAt(7, "OPEN", "TICK-6/review", "main", "", now.Add(-time.Hour), time.Time{}),
	}
	return r, storyHead, prs, merged
}

const (
	wip      = "TICK-5/wip: last task PR #4 merged on 2026-10-02 09:00Z, 2 task PRs merged into it, no story PR yet"
	review   = "TICK-6/review: last task PR #6 merged on 2026-10-03 09:00Z, 1 task PR merged into it, story PR #7 open"
	okOrigin = "check-story-branches: ok: every merged story branch is on origin/main (now)\n"
)

// A story branch on main is green, lists the in-flight story branches, and
// asks gh for every pull request.
func TestCLIGreen(t *testing.T) {
	_, _, prs, _ := storyRepo(t)
	args := installFakeGH(t, prs)
	report := filepath.Join(t.TempDir(), "report.md")

	code, out, errOut := runCLI("-resolved", "none.yaml", "-closed", "none.yaml", "-report", report)
	if code != 0 || errOut != "" {
		t.Fatalf("want exit 0 and no stderr, got %d, %q", code, errOut)
	}
	want := "check-story-branches: in flight: " + wip + "\ncheck-story-branches: in flight: " + review + "\n" + okOrigin
	if out != want {
		t.Fatalf("stdout:\n got %q\nwant %q", out, want)
	}
	md, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(md), "`make check-story-branches` against `origin/main`, now: 0 findings.\n") ||
		!strings.Contains(string(md), "- `TICK-6/review`: last task PR #6") {
		t.Fatalf("report:\n%s", md)
	}
	gotArgs, err := os.ReadFile(args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(gotArgs), "pr list --state all --limit 5000 --json number,state,headRefName") {
		t.Fatalf("gh called with %q", gotArgs)
	}
}

// The gate: a task PR merged into the story branch after its story PR, and
// not on main, fails with exit 1, the finding and the count, in the report
// too. resolved.yaml accepts it at that head only; -as-of before the late PR
// replays it green.
func TestCLILateTaskFails(t *testing.T) {
	r, storyHead, prs, merged := storyRepo(t)
	r.run("checkout", "-q", story)
	late := r.commit("d", "late task\n")
	r.run("push", "-q", "origin", story)
	prs = append(prs, prAt(9, "MERGED", "TICK-9/late", story, "t9", merged.Add(time.Minute), merged.Add(5*time.Minute)))
	installFakeGH(t, prs)
	dir := t.TempDir()
	report := filepath.Join(dir, "report.md")

	code, out, errOut := runCLI("-resolved", "none.yaml", "-closed", "none.yaml", "-report", report)
	finding := fmt.Sprintf("%s: story PR #2 merged on %s, but changes since then are not on main: PR #9 merged into it afterwards (head %s, story PR head %s)",
		story, day(merged), late[:7], storyHead[:7])
	wantErr := "check-story-branches: " + finding + "\ncheck-story-branches: 1 findings (now); see \"Story branches\" in docs/quality/ci-pipeline.md\n"
	if code != 1 || errOut != wantErr {
		t.Fatalf("want exit 1 and the finding, got %d\nstderr %q\nwant   %q", code, errOut, wantErr)
	}
	if strings.Contains(out, "ok:") {
		t.Fatalf("a failing run printed ok: %q", out)
	}
	md, _ := os.ReadFile(report)
	if !strings.Contains(string(md), ": 1 findings.\n") || !strings.Contains(string(md), "- `"+story+"`: story PR #2 merged on") {
		t.Fatalf("report lacks the finding:\n%s", md)
	}

	// Accepted at this head: green, and the acceptance is reported.
	resolved := filepath.Join(dir, "resolved.yaml")
	if err := os.WriteFile(resolved, []byte("resolved:\n  - branch: "+story+"\n    head: "+late+"\n    why: kept off main, TICK-99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runCLI("-resolved", resolved, "-closed", "none.yaml", "-report", report)
	if code != 0 || errOut != "" || !strings.HasPrefix(out, "check-story-branches: accepted: "+story+" at "+late[:7]+": kept off main, TICK-99\n") {
		t.Fatalf("want accepted and green, got %d\nstdout %q\nstderr %q", code, out, errOut)
	}
	md, _ = os.ReadFile(report)
	if !strings.Contains(string(md), "Accepted in `tools/ci/storybranches/resolved.yaml`:\n\n- "+story+" at "+late[:7]) {
		t.Fatalf("report lacks the acceptance:\n%s", md)
	}

	// As of before PR #9 merged it is open into a story branch whose story
	// PR merged (rule 2), and the branch head is current, so its commit is
	// still caught as pushed afterwards.
	asOf := merged.Add(2 * time.Minute)
	code, _, errOut = runCLI("-as-of", asOf.Format(time.RFC3339), "-resolved", "none.yaml", "-closed", "none.yaml")
	wantErr = "check-story-branches: " + story + ": open PR #9 targets it, but its story PR #2 merged on " + day(merged) + "; retarget it to main\n" +
		fmt.Sprintf("check-story-branches: %s: story PR #2 merged on %s, but changes since then are not on main: commits pushed to it afterwards (head %s, story PR head %s)\n",
			story, day(merged), late[:7], storyHead[:7]) +
		"check-story-branches: 2 findings (as of " + day(asOf) + "); see \"Story branches\" in docs/quality/ci-pipeline.md\n"
	if code != 1 || errOut != wantErr {
		t.Fatalf("as of: want exit 1, got %d\nstderr %q\nwant   %q", code, errOut, wantErr)
	}
}

// Every failure that is not a findings count exits 2 with its reason.
func TestCLIErrorsExitTwo(t *testing.T) {
	_, _, prs, _ := storyRepo(t)
	installFakeGH(t, prs)
	dir := t.TempDir()
	file := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	notYAML := file("notyaml.yaml", "resolved: [\n")
	short := file("short.yaml", "resolved:\n  - branch: "+story+"\n    head: abc\n    why: x\n")
	noWhy := file("nowhy.yaml", "resolved:\n  - branch: "+story+"\n    head: "+strings.Repeat("a", 40)+"\n")
	none := filepath.Join(dir, "none.yaml")

	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"-bogus"}, "flag provided but not defined: -bogus"},
		{"bad -as-of", []string{"-as-of", "yesterday"}, "check-story-branches: -as-of: parsing time"},
		{"resolved not yaml", []string{"-resolved", notYAML}, notYAML + ": yaml:"},
		{"resolved short head", []string{"-resolved", short}, `"` + story + `" needs branch, a full 40-character head sha and why`},
		{"resolved no why", []string{"-resolved", noWhy}, "needs branch, a full 40-character head sha and why"},
		{"resolved unreadable", []string{"-resolved", dir}, "check-story-branches: read " + dir},
		{"closed unreadable", []string{"-resolved", none, "-closed", dir}, "check-story-branches: read " + dir},
		{"no such remote", []string{"-resolved", none, "-closed", none, "-remote", "no-such-remote"}, "check-story-branches: git fetch --quiet --prune no-such-remote"},
		{"report unwritable", []string{"-resolved", none, "-closed", none, "-report", dir}, "check-story-branches: open " + dir},
	} {
		code, _, errOut := runCLI(c.args...)
		if code != 2 || !strings.Contains(errOut, c.want) {
			t.Errorf("%s: want exit 2 and %q, got %d %q", c.name, c.want, code, errOut)
		}
	}

	if code, _, errOut := runCLI("-h"); code != 0 || !strings.Contains(errOut, "-as-of") {
		t.Errorf("-h: want exit 0 and the usage, got %d %q", code, errOut)
	}

	t.Run("gh fails", func(t *testing.T) {
		t.Setenv(ghFailEnv, "HTTP 401: Bad credentials")
		code, _, errOut := runCLI("-resolved", none, "-closed", none)
		if code != 2 || !strings.Contains(errOut, "check-story-branches: gh pr list: exit status 1: HTTP 401: Bad credentials") {
			t.Fatalf("want exit 2 and gh's message, got %d %q", code, errOut)
		}
	})
	t.Run("gh prints no JSON", func(t *testing.T) {
		t.Setenv(ghOutEnv, file("garbage.json", "not json"))
		code, _, errOut := runCLI("-resolved", none, "-closed", none)
		if code != 2 || !strings.Contains(errOut, "check-story-branches: gh pr list: invalid character") {
			t.Fatalf("want exit 2 and the JSON error, got %d %q", code, errOut)
		}
	})
	t.Run("story PR head unknown", func(t *testing.T) {
		gone := strings.Repeat("0", 39) + "1"
		bad := append([]PR(nil), prs...)
		bad[1].HeadOid = gone
		setPRs(t, bad)
		code, _, errOut := runCLI("-resolved", none, "-closed", none)
		if code != 2 || !strings.Contains(errOut, "check-story-branches: "+story+": git fetch --quiet origin "+gone) {
			t.Fatalf("want exit 2 naming the branch and the failed fetch, got %d %q", code, errOut)
		}
	})
}

// The git helpers report which git command failed.
func TestGitErrors(t *testing.T) {
	r := newTestRepo(t)
	head := r.commit("a", "base\n")

	if _, err := branches("no-such-remote"); err == nil || !strings.Contains(err.Error(), "git ls-remote --heads no-such-remote") {
		t.Errorf("branches: want the ls-remote error, got %v", err)
	}
	if _, err := landed("origin", "main", "", "no-such-ref", time.Time{}); err == nil || !strings.Contains(err.Error(), "git log -1 --format=%cI no-such-ref") {
		t.Errorf("landed, unknown head: want the git log error, got %v", err)
	}
	if _, err := landed("origin", "no-such-main", "", head, time.Time{}); err == nil || !strings.Contains(err.Error(), "git rev-list --first-parent") {
		t.Errorf("landed, unknown main: want the rev-list error, got %v", err)
	}
	if _, err := mergesClean(strings.Repeat("d", 40), head, head); err == nil || !strings.Contains(err.Error(), "git merge-tree --write-tree --no-messages --merge-base=dddd") {
		t.Errorf("mergesClean, unknown merge base: want the merge-tree error, got %v", err)
	}

	// No main commit since: landed falls back to main's tip.
	ok, err := landed("origin", "main", "", head, time.Now().Add(24*time.Hour))
	if err != nil || !ok {
		t.Errorf("landed, since in the future: want main's tip tried and on main, got %v, %v", ok, err)
	}
	r.run("checkout", "-q", "-b", "side")
	side := r.commit("b", "side\n")
	if ok, err := landed("origin", "main", "", side, time.Now().Add(24*time.Hour)); err != nil || ok {
		t.Errorf("landed, change not on main's tip: want false, got %v, %v", ok, err)
	}

	got, err := branches(".")
	if err != nil || got["main"] != head || got["side"] != side {
		t.Errorf("branches: want main and side heads, got %v, %v", got, err)
	}
}

func TestSmallHelpers(t *testing.T) {
	if got := failed(3).Error(); got != "3 findings" {
		t.Errorf("failed(3): got %q", got)
	}
	for _, c := range []struct {
		prs  []PR
		want string
	}{
		{nil, "no PR"},
		{[]PR{{Number: 4}}, "PR #4"},
		{[]PR{{Number: 4}, {Number: 5}}, "PRs #4, #5"},
	} {
		if got := list(c.prs); got != c.want {
			t.Errorf("list(%v): got %q, want %q", c.prs, got, c.want)
		}
	}
	for n, want := range map[int]string{1: "1 task PR", 2: "2 task PRs", 0: "0 task PRs"} {
		if got := countPRs(n); got != want {
			t.Errorf("countPRs(%d): got %q, want %q", n, got, want)
		}
	}
	if _, err := readResolved(filepath.Join(t.TempDir(), "missing.yaml")); err != nil {
		t.Errorf("a missing resolved.yaml lists nothing, got %v", err)
	}
	if got, err := readResolved("resolved.yaml"); err != nil || len(got) != 0 {
		t.Errorf("checked-in resolved.yaml: got %v, %v", got, err)
	}
}

// A git error comparing an orphan with main stops Evaluate, naming the branch.
func TestOrphanLandedErrorStops(t *testing.T) {
	_, err := Evaluate(milestone1(), func(base, _ string, _ time.Time) (bool, error) {
		if base == "" {
			return false, fmt.Errorf("git broke")
		}
		return true, nil
	})
	if err == nil || err.Error() != s6+": git broke" {
		t.Fatalf("want %q, got %v", s6+": git broke", err)
	}
}
