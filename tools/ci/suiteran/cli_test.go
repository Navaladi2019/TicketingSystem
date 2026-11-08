package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/tools/ci/internal/reviewcfg"
)

// helperEnv makes the test binary act as the command suiteran runs (see
// TestHelperProcess), so runCommand's paths need no go toolchain.
const helperEnv = "SUITERAN_HELPER_MODE"

// TestHelperProcess is not a test: run as a child with helperEnv set, it
// writes a go test -json stream (or a broken one) and exits.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv(helperEnv) {
	case "":
		return
	case "exit3":
		fmt.Println(`{"Action":"pass","Package":"example.com/fixture/early","Test":"TestEarlySuite"}`)
		fmt.Println(`{"Action":"pass","Package":"example.com/fixture/early"}`)
		os.Exit(3)
	case "longline":
		// One line past the stream's 16 MiB token limit.
		_, _ = os.Stdout.Write(bytes.Repeat([]byte("x"), 17*1024*1024))
		os.Exit(0)
	}
	os.Exit(99)
}

func helperCommand(t *testing.T, mode string) []string {
	t.Helper()
	t.Setenv(helperEnv, mode)
	return []string{os.Args[0], "-test.run=^TestHelperProcess$"}
}

// earlyStream is a go test -json stream in which the fixture's early suite
// ran, with a subtest, test output and non-event lines.
const earlyStream = `not an event: TestMain wrote this
{"Action":"start","Package":"example.com/fixture/early"}
{"Action":"run","Package":"example.com/fixture/early","Test":"TestEarlySuite"}
{"Action":"output","Package":"example.com/fixture/early","Test":"TestEarlySuite","Output":"=== RUN   TestEarlySuite\n"}
{"Action":"pass","Package":"example.com/fixture/early","Test":"TestEarlySuite/sub"}
{"Action":"pass","Package":"example.com/fixture/early","Test":"TestEarlySuite"}
{"Action":"output","Package":"example.com/fixture/early","Output":"ok  \texample.com/fixture/early\t0.1s\n"}
{"Action":"pass","Package":"example.com/fixture/early"}
`

func writeTemp(t *testing.T, name, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCLI(t *testing.T) {
	stream := writeTemp(t, "early.json", earlyStream)
	missingDir := filepath.Join(t.TempDir(), "no-such-dir")
	fixtureFlags := []string{"-root", fixture, "-paths", "review-paths.yaml"}
	for _, tc := range []struct {
		name       string
		args       func(t *testing.T) []string
		code       int
		wantOut    []string
		notOut     []string
		wantErr    []string
		wantNoErrs bool
	}{
		{
			name:    "no command or files",
			args:    func(*testing.T) []string { return fixtureFlags },
			code:    2,
			wantErr: []string{"suiteran: usage: suiteran [-tags a,b]"},
		},
		{
			name:    "unknown flag",
			args:    func(*testing.T) []string { return []string{"-bogus"} },
			code:    2,
			wantErr: []string{"flag provided but not defined: -bogus"},
		},
		{
			name:    "help",
			args:    func(*testing.T) []string { return []string{"-h"} },
			code:    0,
			wantErr: []string{"-tags string"},
		},
		{
			name: "path list missing",
			args: func(*testing.T) []string {
				return []string{"-root", fixture, "-paths", "no-such.yaml", "-in", stream}
			},
			code:    2,
			wantErr: []string{"suiteran: ", "no-such.yaml"},
		},
		{
			name: "a stream in which the suite ran passes",
			args: func(*testing.T) []string {
				return append(append([]string{}, fixtureFlags...), "-in", stream)
			},
			code: 0,
			wantOut: []string{
				"=== early.json",
				"not an event: TestMain wrote this",
				"ok  \texample.com/fixture/early",
				"ok    early/early_suite_test.go, ran 1 of 1 tests",
				"3 registered files are outside this run",
			},
			// Without -v a passing test's output is not printed.
			notOut:     []string{"=== RUN   TestEarlySuite", "suiteran: FAIL"},
			wantNoErrs: true,
		},
		{
			name: "-v prints a passing test's output",
			args: func(*testing.T) []string {
				return append(append([]string{}, fixtureFlags...), "-v", "-in", stream)
			},
			code:    0,
			wantOut: []string{"=== RUN   TestEarlySuite", "ok    early/early_suite_test.go"},
		},
		{
			name: "-in file missing",
			args: func(*testing.T) []string {
				return append(append([]string{}, fixtureFlags...), "-in", filepath.Join(missingDir, "s.json"))
			},
			code:    2,
			wantErr: []string{"suiteran: ", "s.json"},
		},
		{
			name: "command not found",
			args: func(*testing.T) []string {
				return append(append([]string{}, fixtureFlags...), "--", "suiteran-no-such-command-tick521")
			},
			code:    2,
			wantErr: []string{"suiteran: ", "suiteran-no-such-command-tick521"},
		},
		{
			name: "-tee file cannot be created",
			args: func(t *testing.T) []string {
				return append(append([]string{}, fixtureFlags...), append([]string{"-tee", filepath.Join(missingDir, "tee.json"), "--"}, helperCommand(t, "exit3")...)...)
			},
			code:    2,
			wantErr: []string{"suiteran: ", "tee.json"},
		},
		{
			name: "the command's exit code wins over a passing check",
			args: func(t *testing.T) []string {
				return append(append([]string{}, fixtureFlags...), append([]string{"--"}, helperCommand(t, "exit3")...)...)
			},
			code:    3,
			wantOut: []string{"ok    early/early_suite_test.go, ran 1 of 1 tests"},
		},
		{
			name: "a stream line past the scanner limit",
			args: func(t *testing.T) []string {
				return append(append([]string{}, fixtureFlags...), append([]string{"--"}, helperCommand(t, "longline")...)...)
			},
			code:    2,
			wantErr: []string{"suiteran: bufio.Scanner: token too long"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli(tc.args(t), &stdout, &stderr)
			if code != tc.code {
				t.Errorf("exit %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tc.code, stdout.String(), stderr.String())
			}
			for _, w := range tc.wantOut {
				if !strings.Contains(stdout.String(), w) {
					t.Errorf("stdout lacks %q\n%s", w, stdout.String())
				}
			}
			for _, w := range tc.notOut {
				if strings.Contains(stdout.String(), w) {
					t.Errorf("stdout has %q\n%s", w, stdout.String())
				}
			}
			for _, w := range tc.wantErr {
				if !strings.Contains(stderr.String(), w) {
					t.Errorf("stderr lacks %q\n%s", w, stderr.String())
				}
			}
			if tc.wantNoErrs && stderr.Len() != 0 {
				t.Errorf("stderr %q, want nothing", stderr.String())
			}
		})
	}
}

// A build failure, its output and an event with no package are read as go
// test -json writes them; a failing test prints its buffered output and
// fails the run, a subtest marks its top-level test as run.
func TestStreamEvents(t *testing.T) {
	const pkg = "example.com/p"
	in := strings.Join([]string{
		`{"ImportPath":"example.com/broken","Action":"build-output","Output":"broken.go:1:1: syntax error\n"}`,
		`{"ImportPath":"example.com/broken","Action":"build-fail"}`,
		`{"Action":"output","Output":"no package\n"}`,
		`{"Action":"run","Package":"` + pkg + `","Test":"TestA"}`,
		`{"Action":"output","Package":"` + pkg + `","Test":"TestA","Output":"a says hi\n"}`,
		`{"Action":"output","Package":"` + pkg + `","Test":"TestA/sub","Output":"sub says hi\n"}`,
		`{"Action":"fail","Package":"` + pkg + `","Test":"TestA/sub"}`,
		`{"Action":"fail","Package":"` + pkg + `","Test":"TestA"}`,
		`{"Action":"output","Package":"` + pkg + `","Test":"TestB","Output":"skipped TestB output\n"}`,
		`{"Action":"skip","Package":"` + pkg + `","Test":"TestB"}`,
		`{"Action":"fail","Package":"` + pkg + `"}`,
		`{"Action":""}`,
		`{broken json`,
	}, "\n")
	var out bytes.Buffer
	s := newStream(&out, false)
	if err := s.read(strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, w := range []string{
		"broken.go:1:1: syntax error\n",
		"a says hi\nsub says hi\n",
		"{\"Action\":\"\"}\n",
		"{broken json\n",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("output lacks %q\n%s", w, got)
		}
	}
	for _, n := range []string{"no package", "skipped TestB output"} {
		if strings.Contains(got, n) {
			t.Errorf("output has %q\n%s", n, got)
		}
	}
	if want := "example.com/broken (build), example.com/p TestA, example.com/p"; strings.Join(s.failures, ", ") != want {
		t.Errorf("failures = %q, want %q", s.failures, want)
	}
	if s.events != 11 {
		t.Errorf("events = %d, want 11", s.events)
	}
	if !s.ran[pkg]["TestA"] || s.ran[pkg]["TestB"] || len(s.ran[pkg]) != 1 {
		t.Errorf("ran = %v, want only TestA", s.ran)
	}
	if !s.packages[pkg] || len(s.packages) != 1 {
		t.Errorf("packages = %v, want only %s", s.packages, pkg)
	}
	rep := check(s, nil, nil)
	if !rep.failed() || !strings.Contains(rep.String(nil), "suiteran: FAIL: example.com/broken (build), example.com/p TestA, example.com/p") {
		t.Errorf("report does not fail on the failures:\n%s", rep.String(nil))
	}
}

// pathList writes a path list that registers *_suite_test.go under dir.
func pathList(t *testing.T, dir string) reviewcfg.Config {
	t.Helper()
	p := filepath.Join(dir, "review-paths.yaml")
	if err := os.WriteFile(p, []byte("security: []\ncto_signoff:\n  paths: []\n  isolation_tests:\n    - \"**/*_suite_test.go\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := reviewcfg.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, data := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

const suiteSrc = "package a\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n"

func TestRegistered(t *testing.T) {
	for _, tc := range []struct {
		name    string
		files   map[string]string
		want    string // path=importpath of each file, sorted by walk order
		wantErr string
	}{
		{
			name: "skips what the go command skips",
			files: map[string]string{
				"go.mod":                          "module \"example.com/m\"\n",
				"a/a_suite_test.go":               suiteSrc,
				"a/b/go.mod":                      "// nested\nmodule\texample.com/nested\n",
				"a/b/c/c_suite_test.go":           suiteSrc,
				"testdata/t_suite_test.go":        suiteSrc,
				"vendor/v_suite_test.go":          suiteSrc,
				"node_modules/n_suite_test.go":    suiteSrc,
				".hidden/h_suite_test.go":         suiteSrc,
				"_skip/s_suite_test.go":           suiteSrc,
				"a/not_registered_test.go":        suiteSrc,
				"a/a_suite_test.go.orig_suite.go": suiteSrc,
			},
			want: "a/a_suite_test.go=example.com/m/a a/b/c/c_suite_test.go=example.com/nested/c",
		},
		{
			name:    "no go.mod at or above a registered file",
			files:   map[string]string{"a/a_suite_test.go": suiteSrc},
			wantErr: "a/a_suite_test.go: no go.mod at or above it",
		},
		{
			name:    "go.mod without a module line",
			files:   map[string]string{"go.mod": "go 1.26\nmodulex example.com/m\n", "a/a_suite_test.go": suiteSrc},
			wantErr: "a/a_suite_test.go: ",
		},
		{
			name:    "a registered file that does not parse",
			files:   map[string]string{"go.mod": "module example.com/m\n", "a/a_suite_test.go": "package a\nfunc {"},
			wantErr: "a_suite_test.go:2:",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := pathList(t, t.TempDir())
			writeTree(t, root, tc.files)
			files, err := registered(root, cfg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(filepath.ToSlash(err.Error()), tc.wantErr) {
					t.Fatalf("err %v, want it to contain %q", err, tc.wantErr)
				}
				if tc.name == "go.mod without a module line" && !strings.Contains(err.Error(), "no module line") {
					t.Errorf("err %v, want no module line", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range files {
				got = append(got, f.Path+"="+f.ImportPath)
				if strings.Join(f.Tests, ",") != "TestA" {
					t.Errorf("%s: tests %v, want TestA", f.Path, f.Tests)
				}
			}
			if strings.Join(got, " ") != tc.want {
				t.Errorf("registered = %q, want %q", strings.Join(got, " "), tc.want)
			}
		})
	}
}

func TestRegisteredRootMissing(t *testing.T) {
	cfg := pathList(t, t.TempDir())
	if _, err := registered(filepath.Join(t.TempDir(), "gone"), cfg); err == nil {
		t.Fatal("registered on a missing root: no error")
	}
}

// A registered file whose module cannot be found makes the run uncheckable
// (exit 2), not a pass.
func TestCLIRegisteredError(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{"a/a_suite_test.go": suiteSrc})
	pathList(t, root)
	stream := writeTemp(t, "s.json", earlyStream)
	var stdout, stderr bytes.Buffer
	code := cli([]string{"-root", root, "-paths", "review-paths.yaml", "-in", stream}, &stdout, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "no go.mod at or above it") {
		t.Errorf("exit %d, want 2 with the go.mod error\nstderr: %s", code, stderr.String())
	}
}

func TestModulePath(t *testing.T) {
	for _, tc := range []struct{ gomod, want string }{
		{"module example.com/m\n", "example.com/m"},
		{"// c\n\tmodule\t\"example.com/q\"\ngo 1.26\n", "example.com/q"},
		{"modulex example.com/m\n", ""},
		{"module\n", ""},
		{"", ""},
	} {
		if got := modulePath([]byte(tc.gomod)); got != tc.want {
			t.Errorf("modulePath(%q) = %q, want %q", tc.gomod, got, tc.want)
		}
	}
}

func TestTestFuncsEdgeCases(t *testing.T) {
	p := writeTemp(t, "v_test.go", "package v\n\nimport \"testing\"\n\nfunc TestByValue(t testing.T) {}\nfunc TestIdent(t T) {}\nfunc TestOK(t *testing.T) {}\n\ntype T struct{}\n")
	got, err := testFuncs(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "TestOK" {
		t.Errorf("testFuncs = %v, want [TestOK]", got)
	}
	if _, err := testFuncs(filepath.Join(t.TempDir(), "missing_test.go")); err == nil {
		t.Error("testFuncs on a missing file: no error")
	}
}

// A registered file whose build constraints cannot be read is reported out of
// the run with the reason, and does not count as missing.
func TestCheckUnreadableConstraints(t *testing.T) {
	s := newStream(&bytes.Buffer{}, false)
	s.add(event{Action: "pass", Package: "example.com/m/a"})
	files := []suiteFile{
		{Path: "z/gone_suite_test.go", Dir: filepath.Join(t.TempDir(), "gone"), ImportPath: "example.com/m/a", Tests: []string{"TestA"}},
	}
	rep := check(s, files, nil)
	if len(rep.files) != 1 || !strings.HasPrefix(rep.files[0].NotInRun, "build constraints unreadable: ") {
		t.Fatalf("files = %+v, want the file out of the run as unreadable", rep.files)
	}
	if rep.failed() {
		t.Errorf("report failed:\n%s", rep.String(nil))
	}
	if out := rep.String(nil); !strings.Contains(out, "(none)") || !strings.Contains(out, "1 registered files are outside this run") {
		t.Errorf("report:\n%s", out)
	}
}
