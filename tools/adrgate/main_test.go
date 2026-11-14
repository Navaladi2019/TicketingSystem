package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const gateADRDoc = `---
id: ADR-008
date: 2026-11-15
accepted: 2026-11-15
---

# ADR-008

### The axes

| Axis | Metric | Method | 10x status |
|---|---|---|---|
| X1a | a | M-X1a: timed run | no |
| X1b | b | M-X1b: pilot | no |
| X2 | c | M-X2: suite | no |
| X3 | d | M-X3: ` + "`docs/budgets.md`" + `; CG-SP-1 | no |
| X4 | e | M-X4: cost model | no |

## Back-apply
`

const registerDoc = `---
id: CP
---

| Grade | Source |
|---|---|
| **A** | vendor |

| ID | Pain point | Hypothesis result | Evidence | Grade |
|---|---|---|---|---|
| **CP-SF-01** | p | Confirmed | [g2](https://g2.com/x) | C (B for the G2 range) |
| **CP-SF-02** | p | Confirmed | [d](https://example.com/a) | A / C |
| **CP-SN-01** | p | Confirmed | [d](https://example.com/b) | A / B / C |
`

const gate = "## Competitive Gate (ADR-008)\n\n**Result:** N/A. Internal tooling. Parity: I1–I9 unchanged.\n"

// base writes a clean tree: ADR-008, the register, the template.
func base(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "docs/adr/ADR-008-competitive-gate.md", gateADRDoc)
	write(t, root, "docs/adr/ADR-008-explained.md", "# explained\n")
	write(t, root, registerPath, registerDoc)
	write(t, root, templatePath, templateDoc)
	return root
}

func templateBody(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(templatePath)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A placeholder-only template, standing in for the real one in base trees.
const templateDoc = "---\nid: ADR-NNN\n---\n## Competitive Gate (ADR-008)\n\n**Result:** PASS | PASS with accepted risk <CP-id> | N/A\n3. **Measured by:** M-X<n> | M-X… | M-X9\n"

func TestRules(t *testing.T) {
	adr := func(date, body string) string {
		return "---\nstatus: PROPOSED\ndate: " + date + "\n---\n\n# Title\n\n## Decision\n\nText.\n\n" + body
	}
	const p9 = "docs/adr/ADR-009-x.md"
	cases := []struct {
		name  string
		files map[string]string
		want  []string // exact findings; nil means clean
	}{
		// 1. ADR numbered above 008.
		{"ADR-009 without gate", map[string]string{p9: adr("2026-11-20", "")},
			[]string{p9 + ": CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"gate without Result", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\nText.\n\n## Next\n\n**Result:** PASS\n")},
			[]string{p9 + ": CI-1: ## Competitive Gate has no **Result:** line before the next ## heading"}},
		{"Result maybe", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\n**Result:** maybe\n")},
			[]string{p9 + `: CI-1: **Result:** "maybe" does not start with PASS, PASS with accepted risk CP-, or N/A`}},
		{"Result PASSED is not PASS", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\n**Result:** PASSED\n")},
			[]string{p9 + `: CI-1: **Result:** "PASSED" does not start with PASS, PASS with accepted risk CP-, or N/A`}},
		{"Result N/A", map[string]string{p9: adr("2026-11-20", gate)}, nil},
		{"Result PASS", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\n### Detail\n\n**Result:** PASS\n")}, nil},
		{"Result PASS with accepted risk", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\n**Result:** PASS with accepted risk CP-SF-02 (owner TL)\n")}, nil},
		{"accepted risk without id", map[string]string{p9: adr("2026-11-20", "## Competitive Gate\n\n**Result:** PASS with accepted risk none\n")},
			[]string{p9 + `: CI-1: **Result:** "PASS with accepted risk none" names no accepted risk CP- id`}},
		{"gate only in a comment", map[string]string{p9: adr("2026-11-20", "<!--\n## Competitive Gate\n\n**Result:** PASS\n-->\n")},
			[]string{p9 + ": CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"gate only in a fence", map[string]string{p9: adr("2026-11-20", "```\n## Competitive Gate\n\n**Result:** PASS\n```\n")},
			[]string{p9 + ": CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"CRLF record", map[string]string{p9: strings.ReplaceAll(adr("2026-11-20", gate), "\n", "\r\n")}, nil},
		{"explainer of ADR-009 is not a record", map[string]string{"docs/adr/ADR-009-explained.md": "# explained\n"}, nil},
		// 2. Amendments and errata, keyed on date.
		{"erratum dated after accepted", map[string]string{"docs/adr/ADR-004-E10-x.md": adr("2026-11-16", "")},
			[]string{"docs/adr/ADR-004-E10-x.md: CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"amendment dated on accepted", map[string]string{"docs/adr/ADR-004-A2-x.md": adr("2026-11-15", "")},
			[]string{"docs/adr/ADR-004-A2-x.md: CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"erratum in flight before accepted", map[string]string{"docs/adr/ADR-004-E10-x.md": adr("2026-11-14", "")}, nil},
		{"erratum with gate", map[string]string{"docs/adr/ADR-004-E10-x.md": adr("2026-11-16", gate)}, nil},
		{"amendment without date", map[string]string{"docs/adr/ADR-004-A2-x.md": "# no front matter\n"},
			[]string{"docs/adr/ADR-004-A2-x.md: CI-1: amendment or erratum has no front-matter date: YYYY-MM-DD"}},
		{"amendment of ADR-009 needs gate whatever its date", map[string]string{"docs/adr/ADR-009-A1-x.md": adr("2026-11-01", "")},
			[]string{"docs/adr/ADR-009-A1-x.md: CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		// 3. Keyed on the number, not the status.
		{"ADR-005 without gate", map[string]string{"docs/adr/ADR-005-x.md": adr("2026-11-20", "")}, nil},
		{"REJECTED ADR-010 without gate", map[string]string{"docs/adr/ADR-010-x.md": strings.Replace(adr("2026-11-20", ""), "PROPOSED", "REJECTED", 1)},
			[]string{"docs/adr/ADR-010-x.md: CI-1: no ## Competitive Gate section (ADR-008 D6)"}},
		{"record in a subdirectory is not CI-1", map[string]string{"docs/adr/ADR-009/ADR-009-notes.md": "# notes\n"}, nil},
		// 4. Citations.
		{"unknown CP id", map[string]string{p9: adr("2026-11-20", gate+"Avoids CP-SF-99.\n")},
			[]string{p9 + ": CI-2: CP-SF-99 is not a row in " + registerPath}},
		{"unknown CP id reported once", map[string]string{"docs/adr/ADR-001/rfc/RFC-9.md": "CP-FW-07 and CP-FW-07, `CP-SN-09`\n"},
			[]string{"docs/adr/ADR-001/rfc/RFC-9.md: CI-2: CP-FW-07 is not a row in " + registerPath,
				"docs/adr/ADR-001/rfc/RFC-9.md: CI-2: CP-SN-09 is not a row in " + registerPath}},
		{"known CP id", map[string]string{p9: adr("2026-11-20", gate+"Avoids CP-SF-01.\n")}, nil},
		{"CP id in a fence", map[string]string{p9: adr("2026-11-20", gate+"~~~text\nCP-SF-99\n~~~\n")}, nil},
		{"CP id in a comment", map[string]string{p9: adr("2026-11-20", gate+"<!-- CP-SF-99 -->\n")}, nil},
		{"CP id not word-bounded", map[string]string{p9: adr("2026-11-20", gate+"CP-SF-011 XCP-SF-99 CP-XX-99\n")}, nil},
		{"CP id in an html explainer is not read", map[string]string{"docs/adr/ADR-009-explained.html": "CP-SF-99 M-X9\n"}, nil},
		// 6. Methods.
		{"unknown method", map[string]string{p9: adr("2026-11-20", gate+"Measured by M-X9.\n")},
			[]string{p9 + `: CI-4: M-X9 is not a method in ADR-008 "The axes"`}},
		{"known method", map[string]string{p9: adr("2026-11-20", gate+"Measured by M-X2 and M-X1a.\n")}, nil},
		{"placeholders are not methods", map[string]string{p9: adr("2026-11-20", gate+"M-X<n>, M-X…, M-Xn\n")}, nil},
		{"M-X1 in an archived revision", map[string]string{"docs/adr/ADR-009/rfc/rev-0/RFC-1.md": "M-X1 and CP-SF-99\n"}, nil},
		{"M-X1 elsewhere", map[string]string{"docs/adr/ADR-009/rfc/RFC-1.md": "M-X1\n"},
			[]string{`docs/adr/ADR-009/rfc/RFC-1.md: CI-4: M-X1 is not a method in ADR-008 "The axes"`}},
		// Sorting across files and rules.
		{"findings sorted by path then rule", map[string]string{
			"docs/adr/ADR-011-b.md": adr("2026-11-20", "M-X7 CP-SF-98\n"),
			"docs/adr/ADR-010-a.md": adr("2026-11-20", gate+"M-X8\n"),
		}, []string{
			`docs/adr/ADR-010-a.md: CI-4: M-X8 is not a method in ADR-008 "The axes"`,
			"docs/adr/ADR-011-b.md: CI-1: no ## Competitive Gate section (ADR-008 D6)",
			"docs/adr/ADR-011-b.md: CI-2: CP-SF-98 is not a row in " + registerPath,
			`docs/adr/ADR-011-b.md: CI-4: M-X7 is not a method in ADR-008 "The axes"`,
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := base(t)
			for p, body := range c.files {
				write(t, root, p, body)
			}
			got, err := run(root)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(c.want, "\n"))
			}
		})
	}
}

// 4. A copy of the real template saved as an ADR passes: S27 AC1 depends on it.
func TestTemplateCopyIsClean(t *testing.T) {
	root := base(t)
	write(t, root, "docs/adr/ADR-099-copy.md", templateBody(t))
	got, err := run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("template copy: %v", got)
	}
}

// 5. Register rows (CI-3).
func TestRegister(t *testing.T) {
	head := "| ID | Pain point | Hypothesis result | Evidence | Grade |\n|---|---|---|---|---|\n"
	cases := []struct {
		name, row, want string
	}{
		{"mixed grades", "| **CP-SF-01** | p | r | [a](https://a.example) | A / C |", ""},
		{"parenthesised grade", "| **CP-SF-01** | p | r | [a](https://a.example) | C (B for the G2 range) |", ""},
		{"no https link", "| **CP-SF-01** | p | r | [a](http://a.example) | A |", registerPath + ": CI-3: CP-SF-01: Evidence has no https:// link"},
		{"https only outside Evidence", "| **CP-SF-01** | https://a.example | r | none | A |", registerPath + ": CI-3: CP-SF-01: Evidence has no https:// link"},
		{"grade D", "| **CP-SF-01** | p | r | [a](https://a.example) | D |", registerPath + `: CI-3: CP-SF-01: Grade "D" is not A, B or C tokens split on /`},
		{"empty grade", "| **CP-SF-01** | p | r | [a](https://a.example) | |", registerPath + `: CI-3: CP-SF-01: Grade "" is not A, B or C tokens split on /`},
		{"comma grade", "| **CP-SF-01** | p | r | [a](https://a.example) | A, B |", registerPath + `: CI-3: CP-SF-01: Grade "A, B" is not A, B or C tokens split on /`},
		{"trailing slash", "| **CP-SF-01** | p | r | [a](https://a.example) | A / |", registerPath + `: CI-3: CP-SF-01: Grade "A /" is not A, B or C tokens split on /`},
		{"short row", "| **CP-SF-01** | p | https://a.example |", registerPath + ": CI-3: CP-SF-01: row has 3 cells, want 5 (ID, Pain point, Hypothesis result, Evidence, Grade)"},
		{"row in a comment is not a row", "<!-- | **CP-SF-01** | p | r | none | D | -->\n| **CP-SF-02** | p | r | [a](https://a.example) | A |", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := base(t)
			write(t, root, registerPath, head+c.row+"\n")
			write(t, root, templatePath, "# template\n")
			got, err := run(root)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "\n") != c.want {
				t.Errorf("findings %q, want %q", got, c.want)
			}
		})
	}
}

// Broken input is exit 2, not a pass or a finding.
func TestInputErrors(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  string
	}{
		{"no root", func(t *testing.T, root string) { os.RemoveAll(root) }, ""},
		{"no docs/adr", func(t *testing.T, root string) { os.RemoveAll(filepath.Join(root, "docs")) }, ""},
		{"no ADR-008", func(t *testing.T, root string) {
			os.Remove(filepath.Join(root, "docs", "adr", "ADR-008-competitive-gate.md"))
		}, "no ADR-008 record in docs/adr"},
		{"two ADR-008", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-other.md", gateADRDoc)
		}, "two ADR-008 records"},
		{"ADR-008 unreadable", func(t *testing.T, root string) {
			p := filepath.Join(root, "docs", "adr", "ADR-008-competitive-gate.md")
			os.Remove(p)
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		}, "ADR-008-competitive-gate.md"},
		{"no accepted", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-competitive-gate.md", strings.Replace(gateADRDoc, "accepted: 2026-11-15\n", "", 1))
		}, "front matter has no accepted: YYYY-MM-DD"},
		{"no axes heading", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-competitive-gate.md", strings.Replace(gateADRDoc, "### The axes", "### Axes", 1))
		}, `no "The axes" heading`},
		{"no Method column", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-competitive-gate.md", strings.Replace(gateADRDoc, "| Method |", "| How |", 1))
		}, "table has no Method column"},
		{"no methods", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-competitive-gate.md", "---\naccepted: 2026-11-15\n---\n## The axes\n\ntext\n\n| Method |\n|---|\n| none |\n\nafter\n| M-X5 |\n")
		}, "names no M-X method"},
		{"axes table ends at heading", func(t *testing.T, root string) {
			write(t, root, "docs/adr/ADR-008-competitive-gate.md", "---\naccepted: 2026-11-15\n---\n## The axes\n| Method |\n|---|\n## Next\n| M-X1a |\n")
		}, "names no M-X method"},
		{"no register", func(t *testing.T, root string) { os.Remove(filepath.Join(root, filepath.FromSlash(registerPath))) }, "pain-points.md"},
		{"empty register", func(t *testing.T, root string) { write(t, root, registerPath, "# empty\n") }, "has no register rows"},
		{"unreadable record", func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, "docs", "adr", "x"), 0o755); err != nil {
				t.Fatal(err)
			}
			// A directory named like a .md file is walked as a dir; a dangling
			// symlink stands in for an unreadable file where the OS allows it.
			if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "docs", "adr", "x", "y.md")); err != nil {
				t.Skip("symlinks unavailable:", err)
			}
		}, "y.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := base(t)
			c.setup(t, root)
			var stdout, stderr bytes.Buffer
			if code := cli([]string{"-dir", root}, &stdout, &stderr); code != 2 {
				t.Fatalf("exit %d, want 2; stdout %q stderr %q", code, stdout.String(), stderr.String())
			}
			if !strings.HasPrefix(stderr.String(), "adrgate: ") || !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr %q lacks %q", stderr.String(), c.want)
			}
		})
	}
}

func TestFrontMatterAndCitable(t *testing.T) {
	if fm := frontMatter("no front matter\n"); len(fm) != 0 {
		t.Errorf("frontMatter without fences: %v", fm)
	}
	fm := frontMatter("---\ndate: 2026-11-16 (rev 1)\nnot a pair\n---\nlater: x\n")
	if fm["date"] != "2026-11-16 (rev 1)" || fm["later"] != "" {
		t.Errorf("frontMatter: %v", fm)
	}
	in := "a\n```go\nCP-SF-99\n```\nb <!-- x\ny --> c\n<!-- unclosed\nz\n"
	out := citable(in)
	if strings.Count(out, "\n") != strings.Count(in, "\n") {
		t.Errorf("line count changed: %q", out)
	}
	if strings.Contains(out, "CP-SF-99") || strings.Contains(out, "x") || strings.Contains(out, "z") || !strings.Contains(out, "b ") || !strings.Contains(out, " c") {
		t.Errorf("citable: %q", out)
	}
}
