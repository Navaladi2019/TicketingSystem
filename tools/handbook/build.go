package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

//go:embed assets/handbook.css assets/handbook.js layout.html.tmpl
var embedded embed.FS

// Config is the input of one build. The stamp (Sha, Date) is an input, so
// tests inject it and need no git.
type Config struct {
	Root     string // repository root
	Sha      string // main commit the content was generated against
	Date     string // its date, YYYY-MM-DD
	Registry Registry
	Scanner  Scanner
}

// Output maps a path relative to docs/handbook (slash-separated) to its bytes.
type Output map[string][]byte

var shaRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

const handbookDir = "docs/handbook"

type navSection struct {
	Name  string
	Index string
	Pages []navPage
}

type navPage struct {
	Title, Href, Slug string
	Current           bool
}

type backlinkView struct {
	Label string
	Pages []navPage
}

type pageView struct {
	Title       string
	Section     string
	Crumbs      []navPage
	Body        template.HTML
	Nav         []navSection
	Prefix      string
	Stamp       string
	Banner      bool
	NormPath    string
	NormHref    string
	InProgress  bool
	Backlinks   []backlinkView
	HasMermaid  bool
	IsHome      bool
	SectionHref string
}

// Build renders the whole handbook in memory. Every problem is collected and
// returned together, each naming its page and the reference.
func Build(cfg Config) (Output, error) {
	var problems []string
	if !shaRe.MatchString(cfg.Sha) {
		problems = append(problems, fmt.Sprintf("stamp sha %q is not a 7-40 digit lower-case hex commit", cfg.Sha))
	}
	if !dateRe.MatchString(cfg.Date) {
		problems = append(problems, fmt.Sprintf("stamp date %q is not YYYY-MM-DD", cfg.Date))
	}
	reg := cfg.Registry
	if reg == nil {
		reg = unwiredRegistry{}
	}
	facts, fp := loadFacts(cfg.Root, reg)
	problems = append(problems, fp...)
	pages, pp := loadPages(cfg.Root)
	problems = append(problems, pp...)

	byRel := map[string]*Page{}
	for _, p := range pages {
		byRel[p.srcRel()] = p
	}
	if _, ok := byRel["index.md"]; !ok {
		problems = append(problems, "docs/handbook/src/index.md: the home page is missing")
	}

	layout, err := embedded.ReadFile("layout.html.tmpl")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("layout").Parse(string(lf(layout)))
	if err != nil {
		return nil, err
	}
	out := Output{}
	diagrams := map[string][]byte{}
	var searchEntries []searchEntry
	nav := buildNav(pages)
	stamp := fmt.Sprintf("Content generated against main @ %s (%s)", shortSha(cfg.Sha), cfg.Date)

	for _, p := range pages {
		problems = append(problems, facts.checkRefs(p)...)
		prefix := "../"
		if p.Section == homeSection {
			prefix = ""
		}
		expanded, gp := expandGen(facts, p.Body)
		for _, g := range gp {
			problems = append(problems, p.Src+": "+g)
		}
		r := &pageRenderer{facts: facts, page: p, byRel: byRel, prefix: prefix, diagrams: diagrams}
		body, plain := r.render(expanded)
		problems = append(problems, r.problems...)

		view := pageView{
			Title: p.Front.Title, Section: p.Section, Body: template.HTML(body), //nolint:gosec // body is goldmark output with raw HTML off; diagrams are built by diagramRenderer with escaped fields
			Nav: markCurrent(nav, p), Prefix: prefix, Stamp: stamp, HasMermaid: r.mermaid,
			InProgress: p.Front.Status == "in-progress", IsHome: p.Section == homeSection,
		}
		if p.Section != homeSection {
			view.Crumbs = []navPage{{Title: "Handbook", Href: prefix + "index.html"}, {Title: p.Section, Href: prefix + p.Section + "/index.html"}}
		}
		if p.Front.Normative != "" {
			view.Banner = true
			view.NormPath = p.Front.Normative
			view.NormHref = prefix + path.Join("../..", p.Front.Normative)
		}
		for _, bl := range backlinks(p, pages) {
			v := backlinkView{Label: bl.Label}
			for _, o := range bl.Pages {
				v.Pages = append(v.Pages, navPage{Title: o.Front.Title, Href: prefix + o.Out()})
			}
			view.Backlinks = append(view.Backlinks, v)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, view); err != nil {
			return nil, fmt.Errorf("%s: %w", p.Src, err)
		}
		out[p.Out()] = buf.Bytes()
		searchEntries = append(searchEntries, searchEntry{URL: p.Out(), Title: p.Front.Title, Section: p.Section, Text: plain})
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, &BuildError{Problems: problems}
	}

	for name, data := range diagrams {
		out["diagrams/"+name] = data
	}
	idx, err := searchIndexJS(searchEntries)
	if err != nil {
		return nil, err
	}
	out["assets/search-index.js"] = idx
	for _, a := range []string{"handbook.css", "handbook.js"} {
		data, err := embedded.ReadFile("assets/" + a)
		if err != nil {
			return nil, err
		}
		out["assets/"+a] = lf(data) // LF whatever the checkout's line endings
	}
	{
		m, err := os.ReadFile(filepath.Join(cfg.Root, handbookDir, "assets", "mermaid.min.js"))
		if err != nil {
			return nil, fmt.Errorf("pinned mermaid.min.js: %w", err)
		}
		out["assets/mermaid.min.js"] = m
	}
	return out, nil
}

// BuildError lists every problem of a build.
type BuildError struct{ Problems []string }

func (e *BuildError) Error() string {
	return "handbook build failed:\n  " + strings.Join(e.Problems, "\n  ")
}

func buildNav(pages []*Page) []navSection {
	var out []navSection
	for _, s := range sections() {
		ns := navSection{Name: s}
		for _, p := range pages {
			if p.Section != s {
				continue
			}
			if p.Slug == "index" {
				ns.Index = p.Out()
				continue
			}
			ns.Pages = append(ns.Pages, navPage{Title: p.Front.Title, Href: p.Out(), Slug: p.Slug})
		}
		sort.Slice(ns.Pages, func(i, j int) bool { return ns.Pages[i].Slug < ns.Pages[j].Slug })
		if ns.Index != "" || len(ns.Pages) > 0 {
			out = append(out, ns)
		}
	}
	return out
}

// markCurrent returns the nav with hrefs made relative to page p.
func markCurrent(nav []navSection, p *Page) []navSection {
	prefix := "../"
	if p.Section == homeSection {
		prefix = ""
	}
	out := make([]navSection, len(nav))
	for i, s := range nav {
		ns := navSection{Name: s.Name}
		if s.Index != "" {
			ns.Index = prefix + s.Index
		}
		for _, np := range s.Pages {
			ns.Pages = append(ns.Pages, navPage{Title: np.Title, Slug: np.Slug, Href: prefix + np.Href, Current: np.Href == p.Out()})
		}
		out[i] = ns
	}
	return out
}

type searchEntry struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Section string `json:"section"`
	Text    string `json:"text"`
}

// searchIndexJS writes the index as a script, not a JSON file, because
// fetch() is blocked on file://. encoding/json escapes <, > and & as <,
// > and & and U+2028/2029 as  / , so the text can never
// close the script element, open "<!--" or break a JS string.
func searchIndexJS(entries []searchEntry) ([]byte, error) {
	data, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	if bytes.ContainsAny(data, "<>&") || bytes.Contains(data, []byte(" ")) || bytes.Contains(data, []byte(" ")) {
		return nil, errors.New("search index: JSON encoder left an unescaped character")
	}
	return append(append([]byte("window.HB_SEARCH = "), data...), []byte(";\n")...), nil
}

// Write replaces the generated files under docs/handbook with out, leaving
// src/, README.md and the pinned mermaid file alone, and removing generated
// files that no longer exist.
func Write(root string, out Output) error {
	base := filepath.Join(root, handbookDir)
	existing, err := generatedFiles(base)
	if err != nil {
		return err
	}
	for rel := range existing {
		if _, keep := out[rel]; !keep && rel != "assets/mermaid.min.js" {
			if err := os.Remove(filepath.Join(base, filepath.FromSlash(rel))); err != nil {
				return err
			}
		}
	}
	for rel, data := range out {
		p := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0o644); err != nil { //nolint:gosec // generated, committed, world-readable pages
			return err
		}
	}
	return nil
}

// shortSha is the 12-character form shown in the footer.
func shortSha(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// lf returns data with CRLF line ends turned into LF, so the output does not
// depend on the checkout's line-ending conversion.
func lf(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte{'\r', '\n'}, []byte{'\n'})
}
