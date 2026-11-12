package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var genLine = regexp.MustCompile(`^<!--\s*gen:([a-z-]+)((?:\s+[a-z]+=\S+)*)\s*-->\s*$`)

// expandGen replaces every "<!-- gen:... -->" line of a page body with the
// Markdown it generates. A malformed or unknown directive is an error.
func expandGen(f *Facts, body string) (string, []string) {
	var problems []string
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	fenced := false
	for _, l := range lines {
		if strings.HasPrefix(l, "```") {
			fenced = !fenced
		}
		if fenced || !strings.HasPrefix(strings.TrimSpace(l), "<!-- gen:") {
			out = append(out, l)
			continue
		}
		m := genLine.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			problems = append(problems, fmt.Sprintf("malformed generated block %q", strings.TrimSpace(l)))
			continue
		}
		args := map[string]string{}
		for _, kv := range strings.Fields(m[2]) {
			k, v, _ := strings.Cut(kv, "=")
			args[k] = v
		}
		md, err := f.genBlock(m[1], args)
		if err != nil {
			problems = append(problems, fmt.Sprintf("gen:%s: %v", m[1], err))
			continue
		}
		out = append(out, md)
	}
	return strings.Join(out, "\n"), problems
}

func (f *Facts) genBlock(kind string, args map[string]string) (string, error) {
	allow := map[string][]string{
		"operations": {"tag", "ids"}, "tables": {"area"}, "er": {"area"},
		"migrations": nil, "adr-index": nil, "budgets": nil,
	}
	keys, ok := allow[kind]
	if !ok {
		return "", fmt.Errorf("unknown block kind")
	}
	for k := range args {
		known := false
		for _, a := range keys {
			known = known || a == k
		}
		if !known {
			return "", fmt.Errorf("unknown argument %q", k)
		}
	}
	switch kind {
	case "operations":
		return f.genOperations(args)
	case "tables":
		return f.genTables(args["area"])
	case "er":
		return f.genER(args["area"])
	case "migrations":
		return f.genMigrations()
	case "adr-index":
		return f.genADRIndex(), nil
	default:
		return f.genBudgets()
	}
}

// cell makes s safe inside a Markdown table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}

func (f *Facts) genOperations(args map[string]string) (string, error) {
	var ops []Operation
	switch {
	case args["tag"] != "" && args["ids"] != "":
		return "", fmt.Errorf("give tag or ids, not both")
	case args["tag"] != "":
		for _, o := range f.OpList {
			if o.Tag == args["tag"] {
				ops = append(ops, o)
			}
		}
		if len(ops) == 0 {
			return "", fmt.Errorf("no operation has tag %q", args["tag"])
		}
	case args["ids"] != "":
		for _, id := range strings.Split(args["ids"], ",") {
			o, ok := f.Operations[id]
			if !ok {
				return "", fmt.Errorf("operationId %q is not in api/openapi.bundled.yaml", id)
			}
			ops = append(ops, o)
		}
	default:
		return "", fmt.Errorf("needs tag=<tag> or ids=a,b")
	}
	var b strings.Builder
	b.WriteString("| Operation | Method and path | Summary | Responses |\n|---|---|---|---|\n")
	for _, o := range ops {
		fmt.Fprintf(&b, "| `%s` | `%s %s` | %s | %s |\n", o.ID, o.Method, cell(o.Path), cell(o.Summary), strings.Join(o.Responses, ", "))
	}
	return b.String(), nil
}

func (f *Facts) areaTables(area string) ([]Table, error) {
	if area == "" {
		return nil, fmt.Errorf("needs area=<area>")
	}
	names, ok := f.Areas[area]
	if !ok {
		return nil, fmt.Errorf("area %q is not in docs/handbook/src/database/areas.yaml", area)
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	out := make([]Table, 0, len(sorted))
	for _, n := range sorted {
		out = append(out, f.Tables[n])
	}
	return out, nil
}

func mdList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "`" + s + "`"
	}
	return strings.Join(q, ", ")
}

func (f *Facts) genTables(area string) (string, error) {
	tables, err := f.areaTables(area)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range tables {
		fmt.Fprintf(&b, "#### `%s`\n\n", t.Name)
		kind := t.Kind
		if t.PartitionOf != "" {
			kind += " of " + t.PartitionOf
		}
		fmt.Fprintf(&b, "Kind: %s. Row-level security: enabled %t, forced %t. Primary key: %s.\n\n", cell(kind), t.RLS.Enabled, t.RLS.Forced, mdList(t.PrimaryKey))
		b.WriteString("| Column | Type | Null | Default |\n|---|---|---|---|\n")
		for _, c := range t.Columns {
			null := "no"
			if c.Nullable {
				null = "yes"
			}
			fmt.Fprintf(&b, "| `%s` | `%s` | %s | %s |\n", c.Name, cell(c.Type), null, codeOrDash(c.Default))
		}
		b.WriteString("\n")
		if len(t.Indexes) > 0 {
			b.WriteString("| Index | Method | Columns | Predicate |\n|---|---|---|---|\n")
			for _, i := range t.Indexes {
				name := "`" + i.Name + "`"
				if i.Unique {
					name += " (unique)"
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", name, cell(i.Method), mdList(i.Columns), codeOrDash(i.Predicate))
			}
			b.WriteString("\n")
		}
		if len(t.Policies) > 0 {
			b.WriteString("| Policy | Command | Roles | Using | Check |\n|---|---|---|---|---|\n")
			for _, p := range t.Policies {
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", p.Name, cell(p.Command), mdList(p.Roles), codeOrDash(p.Using), codeOrDash(p.Check))
			}
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func codeOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return "`" + strings.ReplaceAll(cell(s), "`", "'") + "`"
}

func (f *Facts) genER(area string) (string, error) {
	tables, err := f.areaTables(area)
	if err != nil {
		return "", err
	}
	in := map[string]bool{}
	for _, t := range tables {
		in[t.Name] = true
	}
	var b strings.Builder
	b.WriteString("```mermaid\nerDiagram\n")
	for _, t := range tables {
		fmt.Fprintf(&b, "    %s {\n", t.Name)
		pk := map[string]bool{}
		for _, c := range t.PrimaryKey {
			pk[c] = true
		}
		for _, c := range t.Columns {
			key := ""
			if pk[c.Name] {
				key = " PK"
			}
			typ := strings.Map(func(r rune) rune {
				if r == ' ' || r == '(' || r == ')' || r == ',' || r == '"' {
					return '_'
				}
				return r
			}, c.Type)
			fmt.Fprintf(&b, "        %s %s%s\n", typ, c.Name, key)
		}
		b.WriteString("    }\n")
	}
	for _, t := range tables {
		for _, fk := range t.ForeignKeys {
			if in[fk.RefTable] {
				fmt.Fprintf(&b, "    %s }o--|| %s : \"%s\"\n", t.Name, fk.RefTable, strings.Join(fk.Columns, ","))
			}
		}
	}
	b.WriteString("```\n")
	return b.String(), nil
}

func (f *Facts) genMigrations() (string, error) {
	dir := filepath.Join(f.Root, "db", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Migration | Purpose |\n|---|---|\n")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // a migration chosen by directory listing
		if err != nil {
			return "", err
		}
		first := ""
		for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			if strings.HasPrefix(l, "--") && !strings.Contains(l, "+goose") {
				first = strings.TrimSpace(strings.TrimPrefix(l, "--"))
				break
			}
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", e.Name(), cell(first))
	}
	return b.String(), nil
}

func (f *Facts) genADRIndex() string {
	var b strings.Builder
	b.WriteString("| Record | Title | Status | Date |\n|---|---|---|---|\n")
	for _, r := range f.ADRList {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", r.ID, cell(r.Title), cell(r.Status), cell(r.Date))
	}
	return b.String()
}

// genBudgets lists each budget of docs/budgets.md under its section, with its
// owner; the numbers stay in the source, which the page links.
func (f *Facts) genBudgets() (string, error) {
	data, err := os.ReadFile(filepath.Join(f.Root, "docs", "budgets.md"))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("| Section | Budget | Owner |\n|---|---|---|\n")
	section := ""
	for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(l, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(l, "## "))
			continue
		}
		if !strings.HasPrefix(l, "|") || strings.HasPrefix(l, "|---") || strings.HasPrefix(l, "| Budget") {
			continue
		}
		cols := splitRow(l)
		if len(cols) < 2 {
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", cell(section), cell(cols[0]), cell(cols[len(cols)-1]))
	}
	return b.String(), nil
}

// splitRow splits a Markdown table row on unescaped pipes.
func splitRow(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimPrefix(l, "|")
	l = strings.TrimSuffix(l, "|")
	var cols []string
	var cur strings.Builder
	for i := 0; i < len(l); i++ {
		if l[i] == '\\' && i+1 < len(l) && l[i+1] == '|' {
			cur.WriteByte('|')
			i++
			continue
		}
		if l[i] == '|' {
			cols = append(cols, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(l[i])
	}
	return append(cols, strings.TrimSpace(cur.String()))
}
