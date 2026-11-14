// Command adrgate checks the design records under docs/adr against the
// Competitive Gate rules CI-1..CI-4 (ADR-008 D6, D7; RFC-020 §8; PLAN-009
// D-9-4, TICK-880).
//
//   - CI-1: every ADR numbered above 008, and every amendment or erratum dated
//     on or after ADR-008's `accepted:`, in any status, has a
//     `## Competitive Gate` section with a `**Result:**` line whose value
//     starts with PASS, "PASS with accepted risk CP-" or N/A.
//   - CI-2: every CP-(SF|SN|FW)-nn cited under docs/adr is a register row in
//     docs/adr/competitive/pain-points.md.
//   - CI-3: every register row has an https:// link in its Evidence cell and a
//     Grade of A, B or C tokens split on "/" (parenthesised text dropped).
//   - CI-4: every M-X<digit>[a-z] cited under docs/adr is a method named in
//     ADR-008's "The axes" table.
//
// CI-2 and CI-4 skip docs/adr/_template-adr.md and the archived RFC revisions
// docs/adr/*/rfc/rev-*/. Fenced code blocks and HTML comments are not
// citations and do not count as a gate. Only .md files are read.
//
// Usage: adrgate [-dir repo-root]. Exit 0 clean, 1 with one line per finding
// (`path: CI-n: message`), 2 bad usage or unreadable input. `task lint` and
// `make lint` run it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	adrDir       = "docs/adr"
	registerPath = "docs/adr/competitive/pain-points.md"
	templatePath = "docs/adr/_template-adr.md"
	gateADR      = 8 // ADR-008 introduced the gate; records above it need one.
)

var (
	// ADR-NNN-<slug>.md at the top of docs/adr; amendments and errata are
	// ADR-NNN-A<n>-<slug>.md and ADR-NNN-E<n>-<slug>.md.
	recordName    = regexp.MustCompile(`^ADR-(\d{3})-.+\.md$`)
	amendmentName = regexp.MustCompile(`^ADR-\d{3}-[AE]\d+-.+\.md$`)
	// docs/adr/<any>/rfc/rev-<n>/... are archived RFC revisions.
	archivedRev = regexp.MustCompile(`^docs/adr/[^/]+/rfc/rev-[^/]+/`)

	cpCitation     = regexp.MustCompile(`\bCP-(?:SF|SN|FW)-\d{2}\b`)
	methodCitation = regexp.MustCompile(`\bM-X\d[a-z]?\b`)
	registerID     = regexp.MustCompile(`^\*\*(CP-[^*]+)\*\*$`)
	isoDate        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	parenthesised  = regexp.MustCompile(`\([^)]*\)`)
	comment        = regexp.MustCompile(`(?s)<!--.*?(?:-->|$)`)
	resultValue    = regexp.MustCompile(`^(?:PASS|N/A)(?:$|[^A-Za-z0-9])`)
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr))
}

// cli returns the exit code: 0 clean, 1 findings (printed to stdout), 2 bad
// usage or input.
func cli(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("adrgate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "adrgate: unexpected arguments:", strings.Join(flags.Args(), " "))
		return 2
	}

	findings, err := run(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "adrgate:", err)
		return 2
	}
	for _, f := range findings {
		fmt.Fprintln(stdout, f)
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}

type finding struct{ path, rule, msg string }

func (f finding) String() string { return f.path + ": " + f.rule + ": " + f.msg }

// run checks the tree under root and returns the findings sorted by path,
// then rule, then message.
func run(root string) ([]string, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	fsys := r.FS()

	accepted, methods, err := readGateADR(fsys)
	if err != nil {
		return nil, err
	}
	raw, err := fs.ReadFile(fsys, registerPath)
	if err != nil {
		return nil, err
	}
	ids, findings := checkRegister(citable(normalise(raw)))
	if len(ids) == 0 {
		return nil, errors.New(registerPath + " has no register rows")
	}

	err = fs.WalkDir(fsys, adrDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path.Ext(p) != ".md" || p == templatePath || archivedRev.MatchString(p) {
			return nil
		}
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		doc := normalise(raw)
		text := citable(doc)
		findings = append(findings, checkCitations(p, text, ids, methods)...)
		if path.Dir(p) == adrDir {
			if f, ok := checkGate(p, doc, text, accepted); !ok {
				findings = append(findings, f)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.path != b.path {
			return a.path < b.path
		}
		if a.rule != b.rule {
			return a.rule < b.rule
		}
		return a.msg < b.msg
	})
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = f.String()
	}
	return out, nil
}

// readGateADR reads ADR-008's `accepted:` date and the methods of its axes
// table.
func readGateADR(fsys fs.FS) (accepted string, methods map[string]bool, err error) {
	entries, err := fs.ReadDir(fsys, adrDir)
	if err != nil {
		return "", nil, err
	}
	var name string
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, fmt.Sprintf("ADR-%03d-", gateADR)) && isRecord(n) {
			if name != "" {
				return "", nil, fmt.Errorf("two ADR-%03d records: %s and %s", gateADR, name, n)
			}
			name = n
		}
	}
	if name == "" {
		return "", nil, fmt.Errorf("no ADR-%03d record in %s", gateADR, adrDir)
	}
	p := adrDir + "/" + name
	raw, err := fs.ReadFile(fsys, p)
	if err != nil {
		return "", nil, err
	}
	doc := normalise(raw)
	accepted = frontMatter(doc)["accepted"]
	if !isoDate.MatchString(accepted) {
		return "", nil, fmt.Errorf("%s: front matter has no accepted: YYYY-MM-DD", p)
	}
	methods, err = axesMethods(citable(doc))
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", p, err)
	}
	return accepted[:10], methods, nil
}

// isRecord reports whether a file name in docs/adr is an ADR, amendment or
// erratum. Explainers (ADR-NNN-explained.md) are reading guides, not records.
func isRecord(name string) bool {
	return recordName.MatchString(name) && !strings.HasSuffix(name, "-explained.md")
}

// axesMethods parses the Method column of the table under "The axes".
func axesMethods(text string) (map[string]bool, error) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "#") && strings.TrimSpace(strings.TrimLeft(l, "#")) == "The axes" {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil, errors.New(`no "The axes" heading`)
	}
	col := -1
	methods := map[string]bool{}
	for _, l := range lines[start:] {
		if strings.HasPrefix(l, "#") {
			break
		}
		if !strings.HasPrefix(strings.TrimSpace(l), "|") {
			if col >= 0 {
				break // end of the table
			}
			continue
		}
		cells := tableCells(l)
		if col < 0 {
			for i, c := range cells {
				if c == "Method" {
					col = i
				}
			}
			if col < 0 {
				return nil, errors.New(`"The axes" table has no Method column`)
			}
			continue
		}
		if col < len(cells) {
			for _, m := range methodCitation.FindAllString(cells[col], -1) {
				methods[m] = true
			}
		}
	}
	if len(methods) == 0 {
		return nil, errors.New(`"The axes" table names no M-X method`)
	}
	return methods, nil
}

// checkRegister returns the register's ids and its CI-3 findings.
func checkRegister(text string) (map[string]bool, []finding) {
	ids := map[string]bool{}
	var findings []finding
	for _, l := range strings.Split(text, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "|") {
			continue
		}
		cells := tableCells(l)
		m := registerID.FindStringSubmatch(cells[0])
		if m == nil {
			continue
		}
		id := m[1]
		ids[id] = true
		bad := func(msg string) { findings = append(findings, finding{registerPath, "CI-3", id + ": " + msg}) }
		if len(cells) < 5 {
			bad(fmt.Sprintf("row has %d cells, want 5 (ID, Pain point, Hypothesis result, Evidence, Grade)", len(cells)))
			continue
		}
		if !strings.Contains(cells[3], "https://") {
			bad("Evidence has no https:// link")
		}
		if grade := cells[len(cells)-1]; !validGrade(grade) {
			bad(fmt.Sprintf("Grade %q is not A, B or C tokens split on /", grade))
		}
	}
	return ids, findings
}

// validGrade: drop parenthesised text, split on "/", trim; at least one token
// and every token is A, B or C (PLAN-009 D-9-4).
func validGrade(cell string) bool {
	for _, tok := range strings.Split(parenthesised.ReplaceAllString(cell, ""), "/") {
		switch strings.TrimSpace(tok) {
		case "A", "B", "C":
		default:
			return false
		}
	}
	return true
}

// checkCitations returns the CI-2 and CI-4 findings of one file, one per
// unknown id.
func checkCitations(p, text string, ids, methods map[string]bool) []finding {
	var findings []finding
	seen := map[string]bool{}
	for _, id := range cpCitation.FindAllString(text, -1) {
		if !ids[id] && !seen[id] {
			seen[id] = true
			findings = append(findings, finding{p, "CI-2", id + " is not a row in " + registerPath})
		}
	}
	for _, m := range methodCitation.FindAllString(text, -1) {
		if !methods[m] && !seen[m] {
			seen[m] = true
			findings = append(findings, finding{p, "CI-4", m + ` is not a method in ADR-008 "The axes"`})
		}
	}
	return findings
}

// checkGate applies CI-1 to a file at the top of docs/adr. ok is true when the
// file needs no gate or has a valid one.
func checkGate(p, doc, text, accepted string) (finding, bool) {
	name := path.Base(p)
	if !isRecord(name) {
		return finding{}, true
	}
	if num, _ := strconv.Atoi(name[4:7]); num <= gateADR { // isRecord: three digits
		if !amendmentName.MatchString(name) {
			return finding{}, true
		}
		date := frontMatter(doc)["date"]
		if !isoDate.MatchString(date) {
			return finding{p, "CI-1", "amendment or erratum has no front-matter date: YYYY-MM-DD"}, false
		}
		if date[:10] < accepted {
			return finding{}, true // in flight before ADR-008 was accepted (D6 guard 1)
		}
	}
	if msg := gateProblem(text); msg != "" {
		return finding{p, "CI-1", msg}, false
	}
	return finding{}, true
}

// gateProblem returns "" when text has a valid Competitive Gate section.
func gateProblem(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if !strings.HasPrefix(l, "## Competitive Gate") {
			continue
		}
		for _, s := range lines[i+1:] {
			if strings.HasPrefix(s, "## ") || strings.HasPrefix(s, "# ") {
				break
			}
			v, ok := strings.CutPrefix(strings.TrimSpace(s), "**Result:**")
			if !ok {
				continue
			}
			v = strings.TrimSpace(v)
			if risk, ok := strings.CutPrefix(v, "PASS with accepted risk"); ok {
				if strings.HasPrefix(strings.TrimSpace(risk), "CP-") {
					return ""
				}
				return fmt.Sprintf("**Result:** %q names no accepted risk CP- id", v)
			}
			if resultValue.MatchString(v) {
				return ""
			}
			return fmt.Sprintf("**Result:** %q does not start with PASS, PASS with accepted risk CP-, or N/A", v)
		}
		return "## Competitive Gate has no **Result:** line before the next ## heading"
	}
	return "no ## Competitive Gate section (ADR-008 D6)"
}

// frontMatter parses the key: value lines between the leading --- fences.
func frontMatter(doc string) map[string]string {
	fm := map[string]string{}
	rest, ok := strings.CutPrefix(doc, "---\n")
	if !ok {
		return fm
	}
	for _, l := range strings.Split(rest, "\n") {
		if strings.TrimSpace(l) == "---" {
			break
		}
		if k, v, ok := strings.Cut(l, ":"); ok {
			fm[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return fm
}

// tableCells splits a Markdown table row into trimmed cells.
func tableCells(line string) []string {
	l := strings.TrimSpace(line)
	l = strings.TrimPrefix(l, "|")
	l = strings.TrimSuffix(l, "|")
	cells := strings.Split(l, "|")
	for i, c := range cells {
		cells[i] = strings.TrimSpace(c)
	}
	return cells
}

func normalise(raw []byte) string {
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

// citable blanks fenced code blocks and HTML comments, keeping line count.
func citable(doc string) string {
	lines := strings.Split(doc, "\n")
	fence := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case fence == "" && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")):
			fence = t[:3]
			lines[i] = ""
		case fence != "":
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			lines[i] = ""
		}
	}
	return comment.ReplaceAllStringFunc(strings.Join(lines, "\n"), func(c string) string {
		return strings.Repeat("\n", strings.Count(c, "\n"))
	})
}
