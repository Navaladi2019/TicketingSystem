package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Registry is the hook TICK-727 wires to the workflow registry
// (outbox.Kinds() plus the recording fake). Until then the default knows
// nothing, so a page that names a workflow or activity fails the build.
type Registry interface {
	Workflows() (names []string, wired bool)
	Activities() (names []string, wired bool)
}

type unwiredRegistry struct{}

func (unwiredRegistry) Workflows() ([]string, bool)  { return nil, false }
func (unwiredRegistry) Activities() ([]string, bool) { return nil, false }

// Operation is one operation of the bundled OpenAPI spec.
type Operation struct {
	ID        string
	Method    string
	Path      string
	Tag       string
	Summary   string
	Responses []string
}

// Story is one entry of docs/handbook/src/stories/stories.yaml.
type Story struct {
	ID            string   `yaml:"id"`
	Issue         string   `yaml:"issue"`
	Title         string   `yaml:"title"`
	Status        string   `yaml:"status"`
	VerdictComent string   `yaml:"verdict_comment"`
	Bruno         string   `yaml:"bruno"`
	Evidence      []string `yaml:"evidence"`
}

// Column, Table and friends mirror docs/handbook/src/database/schema.json,
// format 1 (shape agreed with TICK-733). Unknown keys are ignored.
type Column struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"not_null"`
	Default string `json:"default"`
}

// Constraint is one constraint of a table.
type Constraint struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Columns    []string `json:"columns"`
	Definition string   `json:"definition"`
	References struct {
		Table   string   `json:"table"`
		Columns []string `json:"columns"`
	} `json:"references"`
}

// Index is one index of a table.
type Index struct {
	Name      string   `json:"name"`
	Method    string   `json:"method"`
	Unique    bool     `json:"unique"`
	Keys      []string `json:"keys"`
	Predicate string   `json:"predicate"`
}

// Policy is one row-level-security policy of a table.
type Policy struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Roles     []string `json:"roles"`
	Using     string   `json:"using"`
	WithCheck string   `json:"with_check"`
}

// Table is one table of the schema snapshot.
type Table struct {
	Schema         string `json:"schema"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	PartitionOf    string `json:"partition_of"`
	PartitionKey   string `json:"partition_key"`
	PartitionBound string `json:"partition_bound"`
	RLS            struct {
		Enabled bool `json:"enabled"`
		Forced  bool `json:"forced"`
	} `json:"rls"`
	Columns     []Column     `json:"columns"`
	Constraints []Constraint `json:"constraints"`
	Indexes     []Index      `json:"indexes"`
	Policies    []Policy     `json:"policies"`
}

// Key is how a page names the table: the bare name in schema public (the
// db/grants.yaml rule), "schema.name" in any other schema.
func (t Table) Key() string {
	if t.Schema == "" || t.Schema == "public" {
		return t.Name
	}
	return t.Schema + "." + t.Name
}

// PrimaryKey is the primary-key columns, from the constraints.
func (t Table) PrimaryKey() []string {
	for _, c := range t.Constraints {
		if c.Type == "primary_key" {
			return c.Columns
		}
	}
	return nil
}

// ForeignKeys is the foreign-key constraints.
func (t Table) ForeignKeys() []Constraint {
	var out []Constraint
	for _, c := range t.Constraints {
		if c.Type == "foreign_key" {
			out = append(out, c)
		}
	}
	return out
}

// ADRRecord is one ADR, amendment or erratum.
type ADRRecord struct {
	ID     string
	Title  string
	Status string
	Date   string
	Path   string // repository path of the .md; for an inline erratum, its parent's
	Inline bool
}

// Invariant is one row of the ADR-001 section 1 table.
type Invariant struct {
	ID   string
	Name string
}

// Facts is everything on main a page may reference.
type Facts struct {
	Root       string
	ADRs       map[string]ADRRecord
	ADRList    []ADRRecord
	Stories    map[string]Story
	StoryList  []Story
	Operations map[string]Operation
	OpList     []Operation
	Tables     map[string]Table
	TableList  []Table
	Areas      map[string][]string
	Invariants map[string]Invariant
	Tests      map[string]bool
	Registry   Registry
}

// loadFacts reads every source a page can reference. Each failure is a
// problem line; the caller reports them all.
func loadFacts(root string, reg Registry) (*Facts, []string) {
	f := &Facts{Root: root, Registry: reg}
	var problems []string
	add := func(err error) {
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	add(f.loadADRs())
	add(f.loadStories())
	add(f.loadOperations())
	add(f.loadSchema())
	add(f.loadInvariants())
	add(f.loadTests())
	return f, problems
}

var adrFileName = regexp.MustCompile(`^(ADR-\d{3}(?:-[AE]\d+)?)-.+\.md$`)
var inlineHeading = regexp.MustCompile(`^###\s+([AE]\d+)\b\s*(?:\(([^)]*)\))?\s*:?\s*(.*)$`)
var statusLine = regexp.MustCompile(`(?im)^\**\s*status\s*:\**\s*([A-Za-z][A-Za-z -]*[A-Za-z])`)

func (f *Facts) loadADRs() error {
	dir := filepath.Join(f.Root, "docs", "adr")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("docs/adr: %w", err)
	}
	f.ADRs = map[string]ADRRecord{}
	var errs []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.HasSuffix(name, "-explained.md") {
			continue
		}
		m := adrFileName.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		rel := "docs/adr/" + name
		data, err := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // a repository ADR chosen by directory listing
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, err))
			continue
		}
		head, body, err := splitFrontMatter(string(data))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rel, err))
			continue
		}
		meta := adrMeta(head)
		id := meta["id"]
		if id == "" {
			id = m[1]
		}
		if id != m[1] {
			errs = append(errs, fmt.Sprintf("%s: front matter id %q differs from the file name's %q", rel, id, m[1]))
			continue
		}
		if strings.TrimSpace(meta["status"]) == "" {
			errs = append(errs, fmt.Sprintf("%s: ADR record %s has no status", rel, id))
			continue
		}
		rec := ADRRecord{ID: id, Title: meta["title"], Status: shortStatus(meta["status"]), Date: meta["date"], Path: rel}
		f.ADRs[id] = rec
		for _, in := range inlineRecords(id, rel, rec, body) {
			if _, dup := f.ADRs[in.ID]; dup {
				errs = append(errs, fmt.Sprintf("%s: inline record %s duplicates another record", rel, in.ID))
				continue
			}
			f.ADRs[in.ID] = in
		}
	}
	for _, r := range f.ADRs {
		f.ADRList = append(f.ADRList, r)
	}
	sort.Slice(f.ADRList, func(i, j int) bool { return f.ADRList[i].ID < f.ADRList[j].ID })
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

// shortStatus keeps the leading status word(s) before any parenthetical note.
func shortStatus(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, " ("); i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// inlineRecords finds "### E1 (date): title" headings under a parent's
// "## Errata" or "## Amendments". An inline record without its own status
// line inherits the parent's, since an inline erratum is ruled by the same
// acceptance; the status column says so.
func inlineRecords(parent, rel string, parentRec ADRRecord, body string) []ADRRecord {
	var out []ADRRecord
	lines := strings.Split(body, "\n")
	inSection := false
	var cur *ADRRecord
	var curText []string
	flush := func() {
		if cur == nil {
			return
		}
		if m := statusLine.FindStringSubmatch(strings.Join(curText, "\n")); m != nil {
			cur.Status = shortStatus(m[1])
		}
		out = append(out, *cur)
		cur = nil
		curText = nil
	}
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "## "):
			flush()
			h := strings.TrimSpace(strings.TrimPrefix(l, "## "))
			inSection = h == "Errata" || h == "Amendments"
		case inSection && strings.HasPrefix(l, "### "):
			flush()
			if m := inlineHeading.FindStringSubmatch(l); m != nil {
				cur = &ADRRecord{ID: parent + "-" + m[1], Title: strings.TrimSpace(m[3]), Date: m[2], Path: rel, Inline: true, Status: parentRec.Status + " (inherited)"}
			}
		case cur != nil:
			curText = append(curText, l)
		}
	}
	flush()
	return out
}

func (f *Facts) loadStories() error {
	f.Stories = map[string]Story{}
	p := filepath.Join(f.Root, "docs", "handbook", "src", "stories", "stories.yaml")
	data, err := os.ReadFile(p) //nolint:gosec // fixed repository path
	if errors.Is(err, fs.ErrNotExist) {
		return nil // no manifest yet: no page may name a story
	}
	if err != nil {
		return fmt.Errorf("stories.yaml: %w", err)
	}
	var doc struct {
		Stories []Story `yaml:"stories"`
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("stories.yaml: %w", err)
	}
	var errs []string
	for _, s := range doc.Stories {
		switch {
		case s.ID == "":
			errs = append(errs, "stories.yaml: an entry has no id")
		case f.Stories[s.ID].ID != "":
			errs = append(errs, fmt.Sprintf("stories.yaml: duplicate story %s", s.ID))
		case s.Status != "delivered" && s.Status != "in-progress" && s.Status != "planned":
			errs = append(errs, fmt.Sprintf("stories.yaml: story %s has status %q, want delivered, in-progress or planned", s.ID, s.Status))
		default:
			f.Stories[s.ID] = s
			f.StoryList = append(f.StoryList, s)
			for _, ev := range s.Evidence {
				if _, err := os.Stat(filepath.Join(f.Root, filepath.FromSlash(ev))); err != nil {
					errs = append(errs, fmt.Sprintf("stories.yaml: story %s evidence %q does not exist", s.ID, ev))
				}
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (f *Facts) loadOperations() error {
	f.Operations = map[string]Operation{}
	data, err := os.ReadFile(filepath.Join(f.Root, "api", "openapi.bundled.yaml"))
	if err != nil {
		return fmt.Errorf("api/openapi.bundled.yaml: %w", err)
	}
	var doc struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("api/openapi.bundled.yaml: %w", err)
	}
	methods := []string{"get", "put", "post", "delete", "patch", "head", "options"}
	for p, item := range doc.Paths {
		for _, m := range methods {
			node, ok := item[m]
			if !ok {
				continue
			}
			var op struct {
				ID        string               `yaml:"operationId"`
				Summary   string               `yaml:"summary"`
				Responses map[string]yaml.Node `yaml:"responses"`
			}
			if err := node.Decode(&op); err != nil {
				return fmt.Errorf("api/openapi.bundled.yaml: %s %s: %w", m, p, err)
			}
			if op.ID == "" {
				continue
			}
			o := Operation{ID: op.ID, Method: strings.ToUpper(m), Path: p, Tag: pathTag(p), Summary: op.Summary}
			for code := range op.Responses {
				o.Responses = append(o.Responses, code)
			}
			sort.Strings(o.Responses)
			f.Operations[o.ID] = o
			f.OpList = append(f.OpList, o)
		}
	}
	sort.Slice(f.OpList, func(i, j int) bool {
		if f.OpList[i].Path != f.OpList[j].Path {
			return f.OpList[i].Path < f.OpList[j].Path
		}
		return f.OpList[i].Method < f.OpList[j].Method
	})
	return nil
}

// pathTag is the tag of an operation: the first path segment after /api/v1/.
// The spec carries no tags, so the URL group is the grouping.
func pathTag(p string) string {
	p = strings.TrimPrefix(p, "/api/v1/")
	p = strings.TrimPrefix(p, "/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		p = p[:i]
	}
	return p
}

func (f *Facts) loadSchema() error {
	f.Tables = map[string]Table{}
	f.Areas = map[string][]string{}
	dir := filepath.Join(f.Root, "docs", "handbook", "src", "database")
	data, err := os.ReadFile(filepath.Join(dir, "schema.json")) //nolint:gosec // fixed repository path
	if err != nil {
		return fmt.Errorf("docs/handbook/src/database/schema.json: %w (a stub {\"format\":1,\"tables\":[],\"views\":[]} is accepted until TICK-733 lands)", err)
	}
	var doc struct {
		Format int     `json:"format"`
		Tables []Table `json:"tables"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("schema.json: %w", err)
	}
	if doc.Format != 1 {
		return fmt.Errorf("schema.json: format %d, this generator reads format 1", doc.Format)
	}
	for _, t := range doc.Tables {
		f.Tables[t.Key()] = t
		f.TableList = append(f.TableList, t)
	}
	sort.Slice(f.TableList, func(i, j int) bool { return f.TableList[i].Key() < f.TableList[j].Key() })
	areas, err := os.ReadFile(filepath.Join(dir, "areas.yaml")) //nolint:gosec // fixed repository path
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(areas, &f.Areas); err != nil {
		return fmt.Errorf("areas.yaml: %w", err)
	}
	var errs []string
	for a, names := range f.Areas {
		for _, n := range names {
			if _, ok := f.Tables[n]; !ok {
				errs = append(errs, fmt.Sprintf("areas.yaml: area %s names table %q, which is not in schema.json", a, n))
			}
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

var invariantRow = regexp.MustCompile(`^\|\s*\*\*(I\d+)\*\*\s*\|\s*\*\*([^*]+)\*\*`)

func (f *Facts) loadInvariants() error {
	f.Invariants = map[string]Invariant{}
	data, err := os.ReadFile(filepath.Join(f.Root, "docs", "adr", "ADR-001-platform-blueprint.md"))
	if err != nil {
		return fmt.Errorf("ADR-001: %w", err)
	}
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(l, "## ") {
			in = strings.HasPrefix(l, "## 1.")
			continue
		}
		if !in {
			continue
		}
		if m := invariantRow.FindStringSubmatch(l); m != nil {
			f.Invariants[m[1]] = Invariant{ID: m[1], Name: strings.TrimSuffix(strings.TrimSpace(m[2]), ".")}
		}
	}
	if len(f.Invariants) == 0 {
		return errors.New("ADR-001 section 1: no invariant rows (| **I1** | **Name.** ...) found")
	}
	return nil
}

// loadTests records every Test, Benchmark, Fuzz and Example function of every
// _test.go file, by bare name and as "dir.Name". It reads source only (go/ast);
// nothing is compiled or run.
func (f *Facts) loadTests() error {
	f.Tests = map[string]bool{}
	return filepath.WalkDir(f.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "scratch":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
		if perr != nil {
			return nil // a file that does not parse is the compiler's finding, not the handbook's
		}
		rel, _ := filepath.Rel(f.Root, filepath.Dir(p))
		rel = filepath.ToSlash(rel)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			n := fn.Name.Name
			for _, pre := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
				if strings.HasPrefix(n, pre) {
					f.Tests[n] = true
					f.Tests[rel+"."+n] = true
				}
			}
		}
		return nil
	})
}

var metaLine = regexp.MustCompile(`^([a-z_]+):[ \t]*(.*)$`)

// adrMeta reads the top-level "key: value" lines of an ADR header. The
// headers are not strict YAML (values hold unquoted colons), so only the
// first colon splits a line and a quoted value loses its quotes.
func adrMeta(head string) map[string]string {
	out := map[string]string{}
	for _, l := range strings.Split(head, "\n") {
		m := metaLine.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		v := strings.TrimSpace(m[2])
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		out[m[1]] = v
	}
	return out
}
