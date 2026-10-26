// Command suiteran fails a `go test -json` run in which a registered isolation
// suite (cto_signoff.isolation_tests in docs/delivery/review-paths.yaml) ran
// zero tests (TICK-230). A TestMain that exits before m.Run, or a CI step that
// drops the suite's build tag, leaves `go test` green with the suite never run;
// review-paths only flags the diffs that remove or skip one.
//
// A registered file is in the run when go test reported on its package and its
// build constraints hold for -tags (the tags the step means to test with,
// written next to the go test command, so dropping them from go test alone
// fails here). Such a file must have at least one of its own top-level tests
// end in pass or fail; a skip does not count. A file that defines no test (a
// helper) is listed and not held to that.
//
// It prints the run's output as go test would without -json (with -v, all of
// it), then one line per registered file in the run. It exits non-zero when
// the command failed, a test or package failed, the stream held no go test
// event, or a registered file in the run ran no test.
//
// Usage:
//
//	suiteran [-tags a,b] [-v] [-tee file] -- go test -json [flags] [packages]
//	suiteran [-tags a,b] [-v] -in file...   (streams written by go test -json)
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Navaladi2019/TicketingSystem/tools/ci/internal/reviewcfg"
)

func main() {
	tags := flag.String("tags", "", "comma-separated build tags the step tests with")
	pathsFile := flag.String("paths", "docs/delivery/review-paths.yaml", "review path list")
	root := flag.String("root", ".", "repository root the path list is relative to")
	verbose := flag.Bool("v", false, "print every output line, as go test -v")
	tee := flag.String("tee", "", "also write the raw go test -json stream to this file")
	in := flag.Bool("in", false, "the arguments are go test -json files, not a command")
	flag.Parse()

	code, err := run(options{
		tags: splitTags(*tags), pathsFile: *pathsFile, root: *root,
		verbose: *verbose, tee: *tee, in: *in, args: flag.Args(),
	}, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "suiteran:", err)
		if code == 0 {
			code = 2
		}
	}
	os.Exit(code)
}

type options struct {
	tags      []string
	pathsFile string
	root      string
	verbose   bool
	tee       string
	in        bool
	args      []string
}

func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// run returns the exit code: the command's when it failed, else 1 when the
// check failed. err is for a run that could not be checked at all.
func run(o options, out io.Writer) (int, error) {
	if len(o.args) == 0 {
		return 2, errors.New("usage: suiteran [-tags a,b] [-v] [-tee file] -- go test -json [packages], or suiteran -in files")
	}
	cfg, err := reviewcfg.Load(filepath.Join(o.root, o.pathsFile))
	if err != nil {
		return 2, err
	}
	s := newStream(out, o.verbose)
	cmdCode := 0
	if o.in {
		for _, f := range o.args {
			if err := readFile(s, f); err != nil {
				return 2, err
			}
		}
	} else {
		cmdCode, err = runCommand(s, o.args, o.tee)
		if err != nil {
			return 2, err
		}
	}

	files, err := registered(o.root, cfg)
	if err != nil {
		return 2, err
	}
	rep := check(s, files, o.tags)
	fmt.Fprint(out, rep.String(o.tags))

	switch {
	case cmdCode != 0:
		return cmdCode, nil
	case rep.failed():
		return 1, nil
	}
	return 0, nil
}

func readFile(s *stream, file string) error {
	f, err := os.Open(file) //nolint:gosec // a path the caller chose
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	fmt.Fprintf(s.out, "=== %s\n", filepath.Base(file))
	return s.read(f)
}

// runCommand runs args with its stdout read as the stream and its stderr
// passed through, and returns the command's exit code.
func runCommand(s *stream, args []string, tee string) (int, error) {
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...) //nolint:gosec // the CI step's own command
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	var r io.Reader = stdout
	if tee != "" {
		f, err := os.Create(tee) //nolint:gosec // a path the caller chose
		if err != nil {
			return 0, err
		}
		defer func() { _ = f.Close() }()
		r = io.TeeReader(stdout, f)
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	readErr := s.read(r)
	waitErr := cmd.Wait()
	if readErr != nil {
		return 0, readErr
	}
	var exit *exec.ExitError
	if errors.As(waitErr, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, waitErr
}

// event is one line of `go test -json` (test2json, and go build -json for
// build output).
type event struct {
	Action      string
	Package     string
	ImportPath  string
	Test        string
	Output      string
	FailedBuild string
}

// stream is what a run says: the packages go test reported on, and the
// top-level tests of each that ended in pass or fail.
type stream struct {
	out      io.Writer
	verbose  bool
	events   int
	packages map[string]bool
	ran      map[string]map[string]bool // package -> top-level test
	failures []string
	buffered map[string][]string // package\x00top-level test -> output
}

func newStream(out io.Writer, verbose bool) *stream {
	return &stream{
		out: out, verbose: verbose, packages: map[string]bool{},
		ran: map[string]map[string]bool{}, buffered: map[string][]string{},
	}
}

func (s *stream) read(r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		var e event
		if !strings.HasPrefix(line, "{") || json.Unmarshal([]byte(line), &e) != nil || e.Action == "" {
			// Not an event: go test's own errors, or a TestMain that
			// wrote to stdout before test2json took over.
			fmt.Fprintln(s.out, line)
			continue
		}
		s.add(e)
	}
	return sc.Err()
}

func (s *stream) add(e event) {
	s.events++
	if e.Action == "build-output" || e.Action == "build-fail" {
		fmt.Fprint(s.out, e.Output)
		if e.Action == "build-fail" {
			s.failures = append(s.failures, e.ImportPath+" (build)")
		}
		return
	}
	if e.Package == "" {
		return
	}
	s.packages[e.Package] = true
	top, _, _ := strings.Cut(e.Test, "/")
	key := e.Package + "\x00" + top
	if e.Action == "output" {
		switch {
		case s.verbose || e.Test == "":
			fmt.Fprint(s.out, e.Output)
		default:
			s.buffered[key] = append(s.buffered[key], e.Output)
		}
		return
	}
	if e.Action != "pass" && e.Action != "fail" {
		return
	}
	if e.Test == "" {
		if e.Action == "fail" {
			s.failures = append(s.failures, e.Package)
		}
		return
	}
	if s.ran[e.Package] == nil {
		s.ran[e.Package] = map[string]bool{}
	}
	s.ran[e.Package][top] = true
	if e.Test != top {
		return
	}
	if e.Action == "fail" {
		s.failures = append(s.failures, e.Package+" "+top)
		for _, l := range s.buffered[key] {
			fmt.Fprint(s.out, l)
		}
	}
	delete(s.buffered, key)
}

// suiteFile is one registered isolation test file on disk.
type suiteFile struct {
	Path       string // repo-relative, slash-separated
	Dir        string // on disk
	ImportPath string
	Tests      []string // its top-level Test functions
}

// registered walks root for the test files the path list registers, skipping
// what the go command skips (testdata, vendor, and _ or . directories).
func registered(root string, cfg reviewcfg.Config) ([]suiteFile, error) {
	var files []suiteFile
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			n := d.Name()
			if p != root && (n == "testdata" || n == "vendor" || n == "node_modules" || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasSuffix(rel, "_test.go") || !cfg.IsIsolationTest(rel) {
			return nil
		}
		dir := filepath.Dir(p)
		ip, err := importPath(root, dir)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		tests, err := testFuncs(p)
		if err != nil {
			return err
		}
		files = append(files, suiteFile{Path: rel, Dir: dir, ImportPath: ip, Tests: tests})
		return nil
	})
	return files, err
}

// importPath is dir's import path, from the nearest go.mod at or above it
// (and not above root).
func importPath(root, dir string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	var parts []string
	for {
		data, err := os.ReadFile(filepath.Join(d, "go.mod")) //nolint:gosec // under the repository root
		if err == nil {
			mod := modulePath(data)
			if mod == "" {
				return "", fmt.Errorf("%s: no module line", filepath.Join(d, "go.mod"))
			}
			for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
				parts[i], parts[j] = parts[j], parts[i]
			}
			return path.Join(append([]string{mod}, parts...)...), nil
		}
		if d == absRoot || filepath.Dir(d) == d {
			return "", errors.New("no go.mod at or above it")
		}
		parts = append(parts, filepath.Base(d))
		d = filepath.Dir(d)
	}
}

func modulePath(gomod []byte) string {
	for _, l := range strings.Split(string(gomod), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(l), "module"); ok && rest != "" && (rest[0] == ' ' || rest[0] == '\t') {
			return strings.Trim(strings.TrimSpace(rest), `"`)
		}
	}
	return ""
}

// testFuncs lists the file's top-level tests: func TestXxx(t *testing.T), as
// go test finds them. TestMain is not a test.
func testFuncs(file string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name == "TestMain" || !isTestName(fn.Name.Name) {
			continue
		}
		if p := fn.Type.Params.List; len(p) == 1 && len(p[0].Names) <= 1 && isTestingT(p[0].Type) {
			out = append(out, fn.Name.Name)
		}
	}
	return out, nil
}

// isTestName is go test's rule: Test, then nothing or a rune that is not a
// lower-case letter.
func isTestName(name string) bool {
	rest, ok := strings.CutPrefix(name, "Test")
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(r)
}

func isTestingT(x ast.Expr) bool {
	star, ok := x.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "T"
}

// fileResult is one registered file in the run.
type fileResult struct {
	Path     string
	Defined  int
	Ran      int
	NotInRun string // why it is out of the run; "" when in it
}

type report struct {
	files    []fileResult
	events   int
	failures []string
}

func check(s *stream, files []suiteFile, tags []string) report {
	ctxt := build.Default
	ctxt.BuildTags = tags
	r := report{events: s.events, failures: s.failures}
	for _, f := range files {
		fr := fileResult{Path: f.Path, Defined: len(f.Tests)}
		switch match, err := ctxt.MatchFile(f.Dir, filepath.Base(f.Path)); {
		case err != nil:
			fr.NotInRun = "build constraints unreadable: " + err.Error()
		case !s.packages[f.ImportPath]:
			fr.NotInRun = "package not in the run"
		case !match:
			fr.NotInRun = "build constraints not met by -tags"
		}
		for _, t := range f.Tests {
			if s.ran[f.ImportPath][t] {
				fr.Ran++
			}
		}
		r.files = append(r.files, fr)
	}
	sort.Slice(r.files, func(i, j int) bool { return r.files[i].Path < r.files[j].Path })
	return r
}

func (r report) missing() []fileResult {
	var out []fileResult
	for _, f := range r.files {
		if f.NotInRun == "" && f.Defined > 0 && f.Ran == 0 {
			out = append(out, f)
		}
	}
	return out
}

func (r report) failed() bool {
	return r.events == 0 || len(r.failures) > 0 || len(r.missing()) > 0
}

func (r report) String(tags []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nsuiteran: registered isolation suites in this run (-tags %q):\n", strings.Join(tags, ","))
	in, out := 0, 0
	for _, f := range r.files {
		if f.NotInRun != "" {
			out++
			continue
		}
		in++
		switch {
		case f.Defined == 0:
			fmt.Fprintf(&b, "  -     %s: defines no test\n", f.Path)
		case f.Ran == 0:
			fmt.Fprintf(&b, "  FAIL  %s: ran 0 of %d tests\n", f.Path, f.Defined)
		default:
			fmt.Fprintf(&b, "  ok    %s: ran %d of %d tests\n", f.Path, f.Ran, f.Defined)
		}
	}
	if in == 0 {
		b.WriteString("  (none)\n")
	}
	fmt.Fprintf(&b, "  %d registered files are outside this run (package not tested, or build constraints not met)\n", out)
	if r.events == 0 {
		b.WriteString("suiteran: FAIL: the stream held no go test -json event (is -json missing, or did go test not start?)\n")
	}
	if m := r.missing(); len(m) > 0 {
		fmt.Fprintf(&b, "suiteran: FAIL: %d registered isolation suites ran zero tests. A TestMain that exits before m.Run, a -run filter, or go test missing the build tags above leaves a suite out while go test passes.\n", len(m))
	}
	if len(r.failures) > 0 {
		fmt.Fprintf(&b, "suiteran: FAIL: %s\n", strings.Join(r.failures, ", "))
	}
	return b.String()
}
