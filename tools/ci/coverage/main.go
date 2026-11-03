// Command coverage merges Go coverage profiles and gates the total on a
// minimum (TICK-518).
//
// The profiles are `go test -coverprofile` text files: the unit run (make
// test, coverage.out) and the integration run against a real PostgreSQL (make
// test-integration, coverage-integration.out). Both are written with
// -covermode=atomic -coverpkg=./..., so a block one package's tests run in
// another package counts, and the same block appears once per test binary
// that links it. Merging sums the counts of identical blocks (count and
// atomic modes) or keeps the larger (set mode), the way `go tool cover`
// reads a profile with repeated blocks; the total is then covered statements
// over all statements, the figure `go tool cover -func` prints as "total".
//
// The exclusions file lists files that do not count, one pattern per line
// followed by its reason (`pattern  # reason`); a line without a reason, or a
// pattern that matches no file in the profiles, is an error, so the list
// cannot hide real code silently or keep stale entries. A pattern is matched
// against the file's path inside the module (api/api.gen.go): `dir/...`
// matches every file under dir, `**/` at the start matches any directory
// depth, and the rest is path.Match syntax.
//
// Usage:
//
//	coverage merge -o out.out in.out...
//	coverage check [-min 95] [-exclude file] [-o merged.out] [-worst 20] [-all] [-report-only] in.out...
//
// check exits 1 when the total is below -min, unless -report-only, which
// prints the gap and exits 0. It exits 2 on bad input (a missing profile, a
// mode mismatch, a bad exclusion).
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: coverage merge|check [flags] profile...")
		return 2
	}
	switch args[0] {
	case "merge":
		return runMerge(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "coverage: unknown command %q (want merge or check)\n", args[0])
	return 2
}

func runMerge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("merge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "", "write the merged profile here (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *out == "" || fs.NArg() == 0 {
		fmt.Fprintln(stderr, "coverage merge: want -o and at least one profile")
		return 2
	}
	p, err := readProfiles(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, "coverage merge:", err)
		return 2
	}
	if err := p.write(*out); err != nil {
		fmt.Fprintln(stderr, "coverage merge:", err)
		return 2
	}
	fmt.Fprintf(stdout, "coverage merge: %d profiles, %d blocks -> %s\n", fs.NArg(), len(p.blocks), *out)
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	minPct := fs.Float64("min", 95, "minimum total coverage, in percent")
	exclude := fs.String("exclude", "", "exclusions file (pattern  # reason per line)")
	module := fs.String("module", "", "module path stripped from file names (default: read from go.mod)")
	out := fs.String("o", "", "also write the merged profile, exclusions applied, here")
	worst := fs.Int("worst", 20, "how many of the worst-covered packages to list")
	all := fs.Bool("all", false, "list every package, not only the worst")
	reportOnly := fs.Bool("report-only", false, "print the gap but exit 0 below -min")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "coverage check: want at least one profile")
		return 2
	}
	mod := *module
	if mod == "" {
		var err error
		if mod, err = modulePath("go.mod"); err != nil {
			fmt.Fprintln(stderr, "coverage check:", err)
			return 2
		}
	}
	p, err := readProfiles(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, "coverage check:", err)
		return 2
	}
	if *exclude != "" {
		ex, err := readExclusions(*exclude)
		if err != nil {
			fmt.Fprintln(stderr, "coverage check:", err)
			return 2
		}
		if err := p.exclude(ex, mod); err != nil {
			fmt.Fprintln(stderr, "coverage check:", err)
			return 2
		}
	}
	if *out != "" {
		if err := p.write(*out); err != nil {
			fmt.Fprintln(stderr, "coverage check:", err)
			return 2
		}
	}

	pkgs, total := p.byPackage(mod)
	if total.stmts == 0 {
		fmt.Fprintln(stderr, "coverage check: the profiles hold no statements")
		return 2
	}
	pass := total.atLeast(*minPct)
	n := *worst
	if *all || n > len(pkgs) {
		n = len(pkgs)
	}
	if !pass || *all {
		w := stdout
		fmt.Fprintf(w, "%-60s %8s %12s %8s\n", "package", "cover", "covered/all", "missed")
		for _, s := range pkgs[:n] {
			fmt.Fprintf(w, "%-60s %7.1f%% %12s %8d\n", s.name, s.pct(), fmt.Sprintf("%d/%d", s.covered, s.stmts), s.stmts-s.covered)
		}
	}
	gap := ""
	if !pass {
		// Statements still to cover to reach the minimum.
		need := int64(0)
		for !(stat{covered: total.covered + need, stmts: total.stmts}).atLeast(*minPct) {
			need++
		}
		gap = fmt.Sprintf(", %d more statements to cover", need)
	}
	fmt.Fprintf(stdout, "coverage: total %.1f%% (%d/%d statements), minimum %g%%%s\n",
		total.pct(), total.covered, total.stmts, *minPct, gap)
	switch {
	case pass:
		fmt.Fprintln(stdout, "coverage: ok")
		return 0
	case *reportOnly:
		fmt.Fprintln(stdout, "coverage: BELOW MINIMUM (report-only mode: not failing the build, TICK-518)")
		return 0
	}
	fmt.Fprintln(stderr, "coverage: FAIL: total below the minimum; the worst-covered packages are listed above")
	return 1
}

// block is one profile line's position, file:start.col,end.col.
type block struct {
	file string
	pos  string
}

type counts struct {
	stmts int64
	count int64
}

type profile struct {
	mode   string
	blocks map[block]counts
}

func readProfiles(names []string) (*profile, error) {
	p := &profile{blocks: map[block]counts{}}
	for _, name := range names {
		if err := p.read(name); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *profile) read(name string) error {
	f, err := os.Open(name) //nolint:gosec // a profile named on the command line
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	sawMode := false
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" {
			continue
		}
		if mode, ok := strings.CutPrefix(text, "mode: "); ok {
			// go test writes one mode line per file; a concatenated profile
			// repeats it.
			if p.mode != "" && p.mode != mode {
				return fmt.Errorf("%s:%d: mode %s, earlier profiles are %s: write every profile with the same -covermode", name, line, mode, p.mode)
			}
			p.mode, sawMode = mode, true
			continue
		}
		if !sawMode {
			return fmt.Errorf("%s:%d: no mode line before the first block", name, line)
		}
		b, c, err := parseLine(text)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", name, line, err)
		}
		old, seen := p.blocks[b]
		switch {
		case !seen:
			p.blocks[b] = c
		case old.stmts != c.stmts:
			return fmt.Errorf("%s:%d: %s:%s has %d statements here and %d in an earlier profile: profiles of different sources", name, line, b.file, b.pos, c.stmts, old.stmts)
		case p.mode == "set":
			old.count = max(old.count, c.count)
			p.blocks[b] = old
		default:
			old.count += c.count
			p.blocks[b] = old
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if !sawMode {
		return fmt.Errorf("%s: empty profile (no mode line)", name)
	}
	return nil
}

// parseLine reads "file:12.3,14.5 2 1".
func parseLine(s string) (block, counts, error) {
	fields := strings.Fields(s)
	if len(fields) != 3 {
		return block{}, counts{}, fmt.Errorf("want \"file:pos stmts count\", got %q", s)
	}
	i := strings.LastIndexByte(fields[0], ':')
	if i <= 0 {
		return block{}, counts{}, fmt.Errorf("no file:position in %q", s)
	}
	stmts, err1 := strconv.ParseInt(fields[1], 10, 64)
	count, err2 := strconv.ParseInt(fields[2], 10, 64)
	if err := errors.Join(err1, err2); err != nil || stmts < 0 || count < 0 {
		return block{}, counts{}, fmt.Errorf("bad counts in %q", s)
	}
	return block{file: fields[0][:i], pos: fields[0][i+1:]}, counts{stmts: stmts, count: count}, nil
}

func (p *profile) write(name string) error {
	keys := make([]block, 0, len(p.blocks))
	for b := range p.blocks {
		keys = append(keys, b)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].file != keys[j].file {
			return keys[i].file < keys[j].file
		}
		return keys[i].pos < keys[j].pos
	})
	var sb strings.Builder
	fmt.Fprintf(&sb, "mode: %s\n", p.mode)
	for _, b := range keys {
		c := p.blocks[b]
		fmt.Fprintf(&sb, "%s:%s %d %d\n", b.file, b.pos, c.stmts, c.count)
	}
	return os.WriteFile(name, []byte(sb.String()), 0o600)
}

type exclusion struct {
	pattern string
	line    int
	used    bool
}

func readExclusions(name string) ([]*exclusion, error) {
	data, err := os.ReadFile(name) //nolint:gosec // the exclusions file named on the command line
	if err != nil {
		return nil, err
	}
	var list []*exclusion
	for i, raw := range strings.Split(string(data), "\n") {
		text := strings.TrimSpace(raw)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		pattern, reason, ok := strings.Cut(text, "#")
		pattern = strings.TrimSpace(pattern)
		if !ok || strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%s:%d: %q has no reason: write `pattern  # why it does not count`", name, i+1, pattern)
		}
		if strings.ContainsAny(pattern, " \t") {
			return nil, fmt.Errorf("%s:%d: one pattern per line, got %q", name, i+1, pattern)
		}
		if _, err := path.Match(strings.TrimPrefix(pattern, "**/"), ""); err != nil {
			return nil, fmt.Errorf("%s:%d: bad pattern %q: %w", name, i+1, pattern, err)
		}
		list = append(list, &exclusion{pattern: pattern, line: i + 1})
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s: no exclusions; leave -exclude out instead", name)
	}
	return list, nil
}

// matches reports whether rel, a file path inside the module, is excluded.
func (e *exclusion) matches(rel string) bool {
	if dir, ok := strings.CutSuffix(e.pattern, "/..."); ok {
		return strings.HasPrefix(rel, dir+"/")
	}
	if rest, ok := strings.CutPrefix(e.pattern, "**/"); ok {
		for p := rel; ; {
			if m, _ := path.Match(rest, p); m {
				return true
			}
			i := strings.IndexByte(p, '/')
			if i < 0 {
				return false
			}
			p = p[i+1:]
		}
	}
	m, _ := path.Match(e.pattern, rel)
	return m
}

func (p *profile) exclude(list []*exclusion, mod string) error {
	for b := range p.blocks {
		rel := relPath(b.file, mod)
		// Every pattern that matches counts as used, so two overlapping
		// patterns are not reported stale.
		for _, e := range list {
			if e.matches(rel) {
				e.used = true
				delete(p.blocks, b)
			}
		}
	}
	var stale []string
	for _, e := range list {
		if !e.used {
			stale = append(stale, fmt.Sprintf("line %d: %s", e.line, e.pattern))
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("exclusions that match no file in the profiles (remove them): %s", strings.Join(stale, "; "))
	}
	return nil
}

func relPath(file, mod string) string {
	if rest, ok := strings.CutPrefix(file, mod+"/"); ok {
		return rest
	}
	return file
}

type stat struct {
	name           string
	covered, stmts int64
}

func (s stat) pct() float64 {
	if s.stmts == 0 {
		return 100
	}
	return 100 * float64(s.covered) / float64(s.stmts)
}

// atLeast compares without rounding: 94.99% is below 95.
func (s stat) atLeast(minPct float64) bool {
	return float64(s.covered)*100 >= minPct*float64(s.stmts)
}

// byPackage returns the packages worst covered first (most missed statements
// breaking ties) and the total.
func (p *profile) byPackage(mod string) ([]stat, stat) {
	m := map[string]*stat{}
	var total stat
	for b, c := range p.blocks {
		name := path.Dir(relPath(b.file, mod))
		s := m[name]
		if s == nil {
			s = &stat{name: name}
			m[name] = s
		}
		s.stmts += c.stmts
		total.stmts += c.stmts
		if c.count > 0 {
			s.covered += c.stmts
			total.covered += c.stmts
		}
	}
	list := make([]stat, 0, len(m))
	for _, s := range m {
		list = append(list, *s)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.pct() != b.pct() {
			return a.pct() < b.pct()
		}
		if a.stmts-a.covered != b.stmts-b.covered {
			return a.stmts-a.covered > b.stmts-b.covered
		}
		return a.name < b.name
	})
	return list, total
}

func modulePath(gomod string) (string, error) {
	data, err := os.ReadFile(gomod) //nolint:gosec // go.mod of the working directory
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.Trim(strings.TrimSpace(rest), `"`), nil
		}
	}
	return "", fmt.Errorf("%s: no module line", gomod)
}
