package configsurface

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
)

// Sources of truth, relative to this package. The suite reads them and never
// edits them.
const (
	inventoryPath   = "testdata/inventory.json"
	fixturesDir     = "testdata/fixtures"
	workflowSchema  = "../workflow/def/schema.json"
	metadataSrcDir  = "../metadata"
	scriptStepType  = "script"
	workflowSection = "workflow"
	typedefSection  = "type_definition"
)

// inventory is testdata/inventory.json: the configuration surface tenants can
// use (ADR-008 X2). Lists are item names; step_fields maps a step type to the
// property names its step accepts.
type inventory struct {
	Major    int `json:"major"`
	Workflow struct {
		Formats      []string            `json:"formats"`
		TriggerOn    []string            `json:"trigger_on"`
		StepTypes    []string            `json:"step_types"`
		ConditionOps []string            `json:"condition_ops"`
		StepFields   map[string][]string `json:"step_fields"`
	} `json:"workflow"`
	TypeDefinition struct {
		Keywords   []string `json:"keywords"`
		Extensions []string `json:"extensions"`
		Fields     []string `json:"fields"`
	} `json:"type_definition"`
	Script     []string     `json:"script"`
	Deprecated []deprecated `json:"deprecated"`
	Reserved   []reserved   `json:"reserved"`
}

// reserved is an item the source of truth names but Compile refuses, so no
// tenant configuration can use it yet (x-nava-searchable, ADR-006 D8). It is
// not surface: it has no accepted fixture and is left out of items(). Check 2
// still sees it as listed, and TestReservedStillRefused fails the day Compile
// accepts it, so it must then move into the inventory proper.
type reserved struct {
	Item   string `json:"item"`
	Reason string `json:"reason"`
}

// deprecated is one item kept for compatibility: it stays listed and its
// fixture keeps parsing until the next major (ADR-008 R6).
type deprecated struct {
	Item  string `json:"item"`
	Since string `json:"since"`
}

func decodeInventory(b []byte) (inventory, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var inv inventory
	if err := dec.Decode(&inv); err != nil {
		return inventory{}, fmt.Errorf("inventory: %w", err)
	}
	return inv, nil
}

// item names one surface entry, "<section>.<list>/<name>", for example
// "workflow.step_types/timer" or "workflow.step_fields/timer.duration".
func item(list, name string) string { return list + "/" + name }

// itemSet is a set of item names.
type itemSet map[string]bool

func (s itemSet) add(list string, names ...string) {
	for _, n := range names {
		s[item(list, n)] = true
	}
}

func (s itemSet) sorted() []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// listed is items() plus the reserved items: what check 2 (unlisted
// addition) treats as known.
func (inv inventory) listed() itemSet {
	s := inv.items()
	for _, r := range inv.Reserved {
		s[r.Item] = true
	}
	return s
}

// items flattens the inventory into item names.
func (inv inventory) items() itemSet {
	s := itemSet{}
	w := inv.Workflow
	s.add("workflow.formats", w.Formats...)
	s.add("workflow.trigger_on", w.TriggerOn...)
	s.add("workflow.step_types", w.StepTypes...)
	s.add("workflow.condition_ops", w.ConditionOps...)
	for typ, fields := range w.StepFields {
		for _, f := range fields {
			s.add("workflow.step_fields", typ+"."+f)
		}
	}
	t := inv.TypeDefinition
	s.add("type_definition.keywords", t.Keywords...)
	s.add("type_definition.extensions", t.Extensions...)
	s.add("type_definition.fields", t.Fields...)
	s.add("script", inv.Script...)
	return s
}

// enumerable reports whether a source of truth can list the item, so checks 1
// and 2 apply to it. type_definition.keywords is hand-curated: check 3 alone
// covers it.
func enumerable(it string) bool {
	return strings.HasPrefix(it, workflowSection+".") ||
		strings.HasPrefix(it, "type_definition.extensions/") ||
		strings.HasPrefix(it, "type_definition.fields/") ||
		strings.HasPrefix(it, "script/")
}

// --- Sources of truth ----------------------------------------------------

// schemaNode is the part of schema.json the surface is read from.
// A boolean subschema ("key": true) reads as the zero node.
type schemaNode struct {
	Const      string                     `json:"const"`
	Enum       []string                   `json:"enum"`
	Properties map[string]json.RawMessage `json:"properties"`
	Defs       map[string]json.RawMessage `json:"$defs"`
}

// node decodes a subschema; anything but an object is the zero node.
func node(raw json.RawMessage) schemaNode {
	var n schemaNode
	_ = json.Unmarshal(raw, &n)
	return n
}

// workflowSurface lists the items schema.json offers: the format const, the
// trigger.on enum, the step type enum, the condition op enum and each
// <type>_step definition's property names. A step type with no <type>_step
// definition is an error: the schema would accept a type it cannot shape.
func workflowSurface(schema []byte) (itemSet, error) {
	var root schemaNode
	if err := json.Unmarshal(schema, &root); err != nil {
		return nil, fmt.Errorf("workflow schema: %w", err)
	}
	s := itemSet{}
	if f := node(root.Properties["format"]).Const; f != "" {
		s.add("workflow.formats", f)
	}
	s.add("workflow.trigger_on", node(node(root.Properties["trigger"]).Properties["on"]).Enum...)
	types := node(node(root.Defs["step"]).Properties["type"]).Enum
	s.add("workflow.step_types", types...)
	s.add("workflow.condition_ops", node(node(root.Defs["condition"]).Properties["op"]).Enum...)
	for _, typ := range types {
		raw, ok := root.Defs[typ+"_step"]
		if !ok {
			return nil, fmt.Errorf("workflow schema: step type %q has no $defs/%s_step", typ, typ)
		}
		for f := range node(raw).Properties {
			s.add("workflow.step_fields", typ+"."+f)
		}
	}
	if slices.Contains(types, scriptStepType) {
		s.add("script", scriptStepType)
	}
	return s, nil
}

var extensionLiteral = regexp.MustCompile(`^x-nava-[a-z0-9-]+$`)

// typedefSurface lists what internal/metadata can enumerate: every "x-nava-…"
// string literal in its non-test Go files, and every JSON field name of the
// definition document Compile decodes (metadata.Definition, nested with ".").
func typedefSurface(srcDir string) (itemSet, error) {
	s := itemSet{}
	files, err := filepath.Glob(filepath.Join(srcDir, "*.go"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("type definitions: no Go files in %s", srcDir)
	}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("type definitions: %w", err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if v, err := strconv.Unquote(lit.Value); err == nil && extensionLiteral.MatchString(v) {
				s.add("type_definition.extensions", v)
			}
			return true
		})
	}
	s.add("type_definition.fields", jsonFields(reflect.TypeFor[metadata.Definition](), "")...)
	return s, nil
}

// jsonFields returns the JSON names of t's fields, descending into struct and
// slice-of-struct fields with a "." prefix. A json.RawMessage is a leaf.
func jsonFields(t reflect.Type, prefix string) []string {
	var out []string
	for f := range t.Fields() {
		f := f
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if !f.IsExported() || name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		out = append(out, prefix+name)
		ft := f.Type
		if ft == reflect.TypeFor[json.RawMessage]() {
			continue
		}
		if ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			out = append(out, jsonFields(ft, prefix+name+".")...)
		}
	}
	return out
}

// --- Checks 1 and 2 --------------------------------------------------------

// removed is check 1: an enumerable inventory item the source no longer has.
func removed(inv, src itemSet) []string {
	var out []string
	for _, it := range inv.sorted() {
		if enumerable(it) && !src[it] {
			out = append(out, "removed: "+it)
		}
	}
	return out
}

// unlisted is check 2: a source item the inventory does not list.
func unlisted(inv, src itemSet) []string {
	var out []string
	for _, it := range src.sorted() {
		if !inv[it] {
			out = append(out, "unlisted: "+it+"; add it to inventory.json")
		}
	}
	return out
}

// --- Check 3 ----------------------------------------------------------------

// fixtureResult is one fixture: the items it uses and why it no longer
// parses, if it does not.
type fixtureResult struct {
	name    string // "workflow/timer.json"
	uses    itemSet
	problem string // "" when accepted
}

// problemText is err.Error() plus the paths of an ErrInvalidDefinition.
func problemText(err error) string {
	if we, ok := errors.AsType[*def.ErrInvalidDefinition](err); ok {
		return fmt.Sprintf("%s (paths %q)", err.Error(), we.Paths)
	}
	if me, ok := errors.AsType[*metadata.ErrInvalidDefinition](err); ok {
		return fmt.Sprintf("%s (paths %q)", err.Error(), me.Paths)
	}
	return err.Error()
}

// workflowFixture parses one workflow definition with def.Parse and lists the
// items it uses.
func workflowFixture(name string, b []byte) fixtureResult {
	r := fixtureResult{name: name, uses: itemSet{}}
	if _, err := def.Parse(b); err != nil {
		r.problem = problemText(err)
	}
	var doc struct {
		Format  string `json:"format"`
		Trigger struct {
			On string `json:"on"`
		} `json:"trigger"`
		Steps []map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		if r.problem == "" {
			r.problem = err.Error()
		}
		return r
	}
	r.uses.add("workflow.formats", doc.Format)
	r.uses.add("workflow.trigger_on", doc.Trigger.On)
	for _, step := range doc.Steps {
		var typ string
		_ = json.Unmarshal(step["type"], &typ)
		r.uses.add("workflow.step_types", typ)
		for f := range step {
			r.uses.add("workflow.step_fields", typ+"."+f)
		}
		if _, policy := step["policy"]; !policy { // a policy test's op is read or write
			addOp(r.uses, step["op"])
		}
		var cases []map[string]json.RawMessage
		_ = json.Unmarshal(step["cases"], &cases)
		for _, c := range cases {
			addOp(r.uses, c["op"])
		}
	}
	return r
}

func addOp(s itemSet, raw json.RawMessage) {
	var op string
	if json.Unmarshal(raw, &op) == nil && op != "" {
		s.add("workflow.condition_ops", op)
	}
}

// typedefFixture decodes one type definition strictly (as metadata.Template
// does), compiles it and lists the fields, keywords and extensions it uses.
func typedefFixture(name string, b []byte) fixtureResult {
	r := fixtureResult{name: name, uses: itemSet{}}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var d metadata.Definition
	if err := dec.Decode(&d); err != nil {
		r.problem = err.Error()
		return r
	}
	if _, err := metadata.Compile(d); err != nil {
		r.problem = problemText(err)
	}
	var doc map[string]any
	_ = json.Unmarshal(b, &doc)
	documentFields(r.uses, doc, "")
	var schema any
	_ = json.Unmarshal(d.Schema, &schema)
	schemaKeywords(r.uses, schema)
	return r
}

// documentFields records the definition-document field paths in v. The
// schema and fixed_columns values are data, not fields.
func documentFields(s itemSet, v any, prefix string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			s.add("type_definition.fields", prefix+k)
			if prefix == "" && (k == "schema" || k == "fixed_columns") {
				continue
			}
			documentFields(s, child, prefix+k+".")
		}
	case []any:
		for _, child := range t {
			documentFields(s, child, prefix)
		}
	}
}

// schemaKeywords records every keyword of a JSON Schema, descending only
// through applicators, so property names and enum values are not keywords.
func schemaKeywords(s itemSet, node any) {
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	for kw, v := range obj {
		if strings.HasPrefix(kw, "x-nava-") {
			s.add("type_definition.extensions", kw)
		} else {
			s.add("type_definition.keywords", kw)
		}
		switch kw {
		case "additionalProperties", "contains", "else", "if", "items", "not", "propertyNames",
			"then", "unevaluatedItems", "unevaluatedProperties", "contentSchema":
			schemaKeywords(s, v)
		case "allOf", "anyOf", "oneOf", "prefixItems":
			arr, _ := v.([]any)
			for _, sub := range arr {
				schemaKeywords(s, sub)
			}
		case "$defs", "dependentSchemas", "patternProperties", "properties":
			m, _ := v.(map[string]any)
			for _, sub := range m {
				schemaKeywords(s, sub)
			}
		}
	}
}

// loadFixtures reads every fixture under dir/workflow and dir/typedef.
func loadFixtures(dir string) ([]fixtureResult, error) {
	var out []fixtureResult
	for kind, parse := range map[string]func(string, []byte) fixtureResult{
		"workflow": workflowFixture,
		"typedef":  typedefFixture,
	} {
		files, err := filepath.Glob(filepath.Join(dir, kind, "*.json"))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			out = append(out, parse(kind+"/"+filepath.Base(f), b))
		}
	}
	slices.SortFunc(out, func(a, b fixtureResult) int { return strings.Compare(a.name, b.name) })
	return out, nil
}

// stillAccepted is check 3: every fixture is accepted, every step type,
// keyword and extension has its own fixture using it, and every inventory
// item is used by at least one accepted fixture.
func stillAccepted(inv inventory, fixtures []fixtureResult) []string {
	var out []string
	byName := map[string]fixtureResult{}
	used := itemSet{}
	for _, f := range fixtures {
		byName[f.name] = f
		if f.problem != "" {
			out = append(out, "not accepted: "+f.name+": "+f.problem)
			continue
		}
		for it := range f.uses {
			used[it] = true
		}
	}
	own := func(list, kind, name string) {
		f, ok := byName[kind+"/"+name+".json"]
		if !ok {
			out = append(out, "no fixture: "+item(list, name)+"; add testdata/fixtures/"+kind+"/"+name+".json")
			return
		}
		if !f.uses[item(list, name)] {
			out = append(out, "fixture "+f.name+" does not use "+item(list, name))
		}
	}
	for _, t := range inv.Workflow.StepTypes {
		own("workflow.step_types", "workflow", t)
	}
	for _, k := range inv.TypeDefinition.Keywords {
		own("type_definition.keywords", "typedef", k)
	}
	for _, e := range inv.TypeDefinition.Extensions {
		own("type_definition.extensions", "typedef", e)
	}
	for _, it := range inv.items().sorted() {
		if !used[it] {
			out = append(out, "unused: "+it+" is used by no accepted fixture")
		}
	}
	return out
}

// --- Check 4 ----------------------------------------------------------------

// deprecatedStillAccepted is check 4: a deprecated item is still listed and
// at least one accepted fixture uses it.
func deprecatedStillAccepted(inv inventory, fixtures []fixtureResult) []string {
	var out []string
	listed := inv.items()
	for _, d := range inv.Deprecated {
		if !listed[d.Item] {
			out = append(out, "deprecated: "+d.Item+" (since "+d.Since+") is no longer in the inventory")
			continue
		}
		ok := false
		for _, f := range fixtures {
			if f.problem == "" && f.uses[d.Item] {
				ok = true
				break
			}
		}
		if !ok {
			out = append(out, "deprecated: "+d.Item+" (since "+d.Since+") has no accepted fixture")
		}
	}
	return out
}
