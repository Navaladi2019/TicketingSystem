package main

import (
	"bytes"
	"go/build"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/tools/ci/internal/reviewcfg"
)

// fixture is a module of its own whose path list registers *_suite_test.go:
// early (a TestMain that exits before m.Run under SUITERAN_FIXTURE_EARLY_EXIT),
// tagged (a suite behind //go:build fixturetag beside an untagged test),
// helpers (a registered file with no test) and skipped (a suite that skips
// under SUITERAN_FIXTURE_SKIP).
const fixture = "testdata/fixture"

// goTest is go test -count=1 in the fixture. Its go.mod names the
// repository's toolchain, so no other one is fetched.
func goTest(args ...string) []string {
	return append([]string{"go", "-C", fixture, "test", "-count=1"}, args...)
}

func runFixture(t *testing.T, o options) (int, string) {
	t.Helper()
	t.Setenv("GOFLAGS", "")
	o.root = fixture
	o.pathsFile = "review-paths.yaml"
	var out bytes.Buffer
	code, err := run(o, &out)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out.String())
	}
	_, rep, _ := strings.Cut(out.String(), "suiteran: registered")
	for _, l := range strings.Split(rep, "\n") {
		if goMatcherRe.MatchString(l) {
			t.Errorf("report line %q looks like a Go compiler error to setup-go's problem matcher", l)
		}
	}
	return code, out.String()
}

// goMatcherRe is setup-go's problem matcher (matchers.json): a line it matches
// becomes an error annotation on the pull request.
var goMatcherRe = regexp.MustCompile(`^\s*(.+\.go):(?:(\d+):(\d+):)? (.*)`)

func TestFixture(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
		tags []string
		args []string
		code int
		want []string
	}{
		{
			name: "every suite runs",
			tags: []string{"fixturetag"},
			args: goTest("-json", "-tags", "fixturetag", "./..."),
			want: []string{
				"ok    early/early_suite_test.go, ran 1 of 1 tests",
				"ok    tagged/tagged_suite_test.go, ran 1 of 1 tests",
				"ok    skipped/skipped_suite_test.go, ran 1 of 1 tests",
				"-     helpers/helpers_suite_test.go, defines no test",
				"0 registered files are outside this run",
			},
		},
		{
			name: "TestMain exits before m.Run",
			env:  "SUITERAN_FIXTURE_EARLY_EXIT",
			tags: []string{"fixturetag"},
			args: goTest("-json", "-tags", "fixturetag", "./..."),
			code: 1,
			want: []string{"FAIL  early/early_suite_test.go, ran 0 of 1 tests", "1 registered isolation suites ran zero tests"},
		},
		{
			name: "go test drops the build tag",
			tags: []string{"fixturetag"},
			args: goTest("-json", "./..."),
			code: 1,
			want: []string{"FAIL  tagged/tagged_suite_test.go, ran 0 of 1 tests"},
		},
		{
			name: "every test of a suite skips",
			env:  "SUITERAN_FIXTURE_SKIP",
			tags: []string{"fixturetag"},
			args: goTest("-json", "-tags", "fixturetag", "./..."),
			code: 1,
			want: []string{"FAIL  skipped/skipped_suite_test.go, ran 0 of 1 tests"},
		},
		{
			name: "a -run filter leaves a suite out",
			tags: []string{"fixturetag"},
			args: goTest("-json", "-tags", "fixturetag", "-run", "^TestPlain$", "./tagged"),
			code: 1,
			want: []string{"FAIL  tagged/tagged_suite_test.go, ran 0 of 1 tests", "3 registered files are outside this run"},
		},
		{
			name: "a step without the tag does not expect the tagged suite",
			args: goTest("-json", "./tagged"),
			want: []string{"(none)", "4 registered files are outside this run"},
		},
		{
			name: "no -json",
			tags: []string{"fixturetag"},
			args: goTest("-tags", "fixturetag", "./tagged"),
			code: 1,
			want: []string{"the stream held no go test -json event"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(tc.env, "1")
			}
			code, out := runFixture(t, options{tags: tc.tags, args: tc.args})
			if code != tc.code {
				t.Errorf("exit code %d, want %d\n%s", code, tc.code, out)
			}
			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Errorf("output lacks %q\n%s", w, out)
				}
			}
		})
	}
}

// A failing test fails the run with go test's exit code, and its output is
// printed although the run is not -v.
func TestFixtureCommandFailurePropagates(t *testing.T) {
	t.Setenv("SUITERAN_FIXTURE_FAIL", "1")
	dir := t.TempDir()
	tee := filepath.Join(dir, "stream.json")
	code, out := runFixture(t, options{
		tags: []string{"fixturetag"}, tee: tee,
		args: goTest("-json", "-tags", "fixturetag", "-run", "^TestFails$", "./failing"),
	})
	if code != 1 {
		t.Errorf("exit code %d, want 1\n%s", code, out)
	}
	for _, w := range []string{"fixture failure", "suiteran: FAIL: example.com/fixture/failing TestFails"} {
		if !strings.Contains(out, w) {
			t.Errorf("output lacks %q\n%s", w, out)
		}
	}
	if b, err := os.ReadFile(tee); err != nil || !bytes.Contains(b, []byte(`"Action":"fail"`)) {
		t.Errorf("-tee file: %v\n%s", err, b)
	}
}

// -in checks streams written earlier, together: one stream may run a suite's
// package without its tests (a shard), so long as another runs them.
func TestIn(t *testing.T) {
	ip := "example.com/fixture/early"
	dir := t.TempDir()
	a := filepath.Join(dir, "shard-1")
	b := filepath.Join(dir, "shard-2")
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(a, `{"Action":"start","Package":"`+ip+`"}
{"Action":"output","Package":"`+ip+`","Output":"ok  \t`+ip+`\t0.1s\n"}
{"Action":"pass","Package":"`+ip+`"}
`)
	write(b, `{"Action":"run","Package":"`+ip+`","Test":"TestEarlySuite"}
{"Action":"pass","Package":"`+ip+`","Test":"TestEarlySuite"}
{"Action":"pass","Package":"`+ip+`"}
`)
	code, out := runFixture(t, options{in: true, args: []string{a}})
	if code != 1 || !strings.Contains(out, "FAIL  early/early_suite_test.go") {
		t.Errorf("one shard: exit %d, want 1\n%s", code, out)
	}
	code, out = runFixture(t, options{in: true, args: []string{a, b}})
	if code != 0 || !strings.Contains(out, "ok    early/early_suite_test.go, ran 1 of 1") || !strings.Contains(out, "=== shard-2") {
		t.Errorf("both shards: exit %d, want 0\n%s", code, out)
	}
}

func TestTestFuncs(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x_test.go")
	src := `package x

import (
	"testing"
	tt "testing"
)

func TestMain(m *testing.M) {}
func Test(t *testing.T)     {}
func TestA(t *testing.T)    {}
func Test_b(t *tt.T)        {}
func Testlower(t *testing.T) {}
func TestBench(b *testing.B) {}
func TestTwo(t *testing.T, x int) {}
func helper(t *testing.T)   {}
type s struct{}
func (s) TestMethod(t *testing.T) {}
`
	if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := testFuncs(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Test TestA Test_b"; strings.Join(got, " ") != want {
		t.Errorf("testFuncs = %v, want %s", got, want)
	}
}

const repoRoot = "../../.."

// suiteranTagsRe finds the -tags of each suiteran call in the Makefile and the
// CI scripts.
var suiteranTagsRe = regexp.MustCompile(`tools/ci/suiteran\b(.*)`)

// TestEveryRegisteredSuiteHasACheckedStep fails when a registered isolation
// suite (with a test) has no CI step whose suiteran -tags would hold it to
// running: a suite nobody checks can drop out silently again.
func TestEveryRegisteredSuiteHasACheckedStep(t *testing.T) {
	sources := []string{filepath.Join(repoRoot, "Makefile")}
	scripts, err := filepath.Glob(filepath.Join(repoRoot, "scripts", "ci", "*.sh"))
	if err != nil {
		t.Fatal(err)
	}
	sources = append(sources, scripts...)
	var tagSets [][]string
	for _, s := range sources {
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range suiteranTagsRe.FindAllStringSubmatch(string(b), -1) {
			flags, _, _ := strings.Cut(m[1], " -- ")
			var tags []string
			if f := regexp.MustCompile(`-tags[= ](\S+)`).FindStringSubmatch(flags); f != nil {
				tags = splitTags(f[1])
			}
			tagSets = append(tagSets, tags)
		}
	}
	if len(tagSets) == 0 {
		t.Fatal("no suiteran call in the Makefile or scripts/ci")
	}
	cfg, err := reviewcfg.Load(filepath.Join(repoRoot, "docs", "delivery", "review-paths.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := registered(repoRoot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no registered isolation test file found")
	}
	for _, f := range files {
		if len(f.Tests) == 0 {
			continue
		}
		covered := false
		for _, tags := range tagSets {
			ctxt := build.Default
			ctxt.GOOS, ctxt.GOARCH = "linux", "amd64" // the CI runner
			ctxt.BuildTags = tags
			if ok, err := ctxt.MatchFile(f.Dir, filepath.Base(f.Path)); err == nil && ok {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("%s (%s): no suiteran call has -tags that build it; tag sets %q", f.Path, f.ImportPath, tagSets)
		}
	}
}

// The repository's registered files resolve: each has an import path in its
// own module (temporal-server is a module of its own).
func TestRepoRegisteredFiles(t *testing.T) {
	cfg, err := reviewcfg.Load(filepath.Join(repoRoot, "docs", "delivery", "review-paths.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := registered(repoRoot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"internal/identitystack/adversarial_test.go":       "github.com/Navaladi2019/TicketingSystem/internal/identitystack",
		"temporal-server/authz/adversarial_test.go":        "github.com/Navaladi2019/TicketingSystem/temporal-server/authz",
		"internal/tenancy/isolation/ci2_isolation_test.go": "github.com/Navaladi2019/TicketingSystem/internal/tenancy/isolation",
		// TICK-229: registered by the **/*adversarial*_test.go glob.
		"internal/platform/cursor/cursor_adversarial_test.go": "github.com/Navaladi2019/TicketingSystem/internal/platform/cursor",
		"temporal-server/cmd/server/adversarial_f2_test.go":   "github.com/Navaladi2019/TicketingSystem/temporal-server/cmd/server",
	}
	got := map[string]string{}
	for _, f := range files {
		got[f.Path] = f.ImportPath
		if strings.Contains(f.Path, "testdata/") {
			t.Errorf("%s: testdata is not a package go test runs", f.Path)
		}
	}
	for p, ip := range want {
		if got[p] != ip {
			t.Errorf("%s: import path %q, want %q", p, got[p], ip)
		}
	}
}

func TestStepTags(t *testing.T) {
	for _, tc := range []struct {
		tags, args []string
		in         bool
		want       string
	}{
		{nil, []string{"go", "test", "-json", "./..."}, false, ""},
		{nil, []string{"go", "test", "-json", "-race", "./..."}, false, "race"},
		{[]string{"integration"}, []string{"go", "test", "-race=true"}, false, "integration,race"},
		{nil, []string{"-race"}, true, ""},
	} {
		if got := strings.Join(stepTags(tc.tags, tc.args, tc.in), ","); got != tc.want {
			t.Errorf("stepTags(%q, %q, %v) = %q, want %q", tc.tags, tc.args, tc.in, got, tc.want)
		}
	}
}
