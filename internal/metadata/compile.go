package metadata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	keyPattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{1,40}$`)
	prefixPattern = regexp.MustCompile(`^[A-Z]{2,6}$`)
)

// schemaURL is the URL each definition's schema is registered under in its
// own compiler. It is absolute, so the library never consults the working
// directory to resolve it.
const schemaURL = "urn:nava:entity-schema"

// draft2020URL is the only $schema a definition may name.
const draft2020URL = "https://json-schema.org/draft/2020-12/schema"

// errLoadDenied is what denyAllLoader returns. It never reaches a caller:
// Compile maps every compiler error to a path.
var errLoadDenied = errors.New("metadata: schema loading is disabled")

// denyAllLoader refuses every URL. v6's NewCompiler installs FileLoader{} by
// default (roots.go:23); Compile replaces it with this so no schema can read
// a file or reach the network (TICK-363).
type denyAllLoader struct{}

func (denyAllLoader) Load(string) (any, error) { return nil, errLoadDenied }

// newSchemaCompiler returns a fresh compiler. Compile calls it once per call
// and never shares the result: a compiler keeps a registry of every $id it
// has seen, so a shared one would let tenant B's "$ref":"urn:x" resolve
// against tenant A's "$id":"urn:x".
func newSchemaCompiler() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.UseLoader(denyAllLoader{})
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat() // TICK-360 (c): "format" is an assertion, not an annotation
	return c
}

// Kinds of keyword value walkSchema descends through. Data keywords (enum,
// const, default, examples) and unknown keywords are never walked, so data is
// never mistaken for a schema.
const (
	notSubschema    = iota
	oneSubschema    // the value is a subschema
	subschemaArray  // the value is an array of subschemas
	subschemaByName // the value is an object whose values are subschemas
)

func subschemaKind(kw string) int {
	switch kw {
	case "additionalProperties", "contains", "else", "if", "items", "not", "propertyNames",
		"then", "unevaluatedItems", "unevaluatedProperties", "contentSchema":
		return oneSubschema
	case "allOf", "anyOf", "oneOf", "prefixItems":
		return subschemaArray
	case "$defs", "dependentSchemas", "patternProperties", "properties":
		return subschemaByName
	default:
		return notSubschema
	}
}

// problems collects the JSON pointers Compile refuses.
type problems []string

func (p *problems) add(tokens ...string) { *p = append(*p, pointer(tokens...)) }

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}
	paths := slices.Clone(p)
	slices.Sort(paths)
	return &ErrInvalidDefinition{Paths: slices.Compact(paths)}
}

// Compile checks a definition and compiles its schema. A refused definition
// gives *ErrInvalidDefinition with every offending path.
func Compile(d Definition) (*Compiled, error) {
	var bad problems
	if !keyPattern.MatchString(d.Key) {
		bad.add("key")
	}
	if !prefixPattern.MatchString(d.NumberPrefix) {
		bad.add("number_prefix")
	}
	checkStates(d.States, &bad)

	doc, ok := decodeSchema(d.Schema, &bad)
	if !ok {
		return nil, bad.err()
	}
	props := checkShape(doc, &bad)
	if len(bad) > 0 {
		return nil, bad.err()
	}

	c := newSchemaCompiler()
	if err := c.AddResource(schemaURL, doc); err != nil {
		bad.add("schema")
		return nil, bad.err()
	}
	sch, err := c.Compile(schemaURL)
	if err != nil {
		schemaErrorPaths(err, &bad)
		return nil, bad.err()
	}

	defaults := checkDefaults(c, props, &bad)
	checkFixedColumns(d, props, &bad)
	if len(bad) > 0 {
		return nil, bad.err()
	}

	var unique []string
	for name, p := range props {
		if v, _ := p[UniqueKeyword].(bool); v {
			unique = append(unique, name)
		}
	}
	slices.Sort(unique)

	fixed := make(map[string]string, len(d.FixedColumns))
	for k, v := range d.FixedColumns {
		fixed[k] = v
	}
	return &Compiled{
		key:          d.Key,
		numberPrefix: d.NumberPrefix,
		fixedColumns: fixed,
		states: StateModel{
			Initial:     d.States.Initial,
			States:      slices.Clone(d.States.States),
			Transitions: slices.Clone(d.States.Transitions),
		},
		schema:   sch,
		doc:      doc,
		defaults: defaults,
		unique:   unique,
	}, nil
}

// decodeSchema enforces the size cap, the depth cap and the no-duplicate-keys
// rule (TICK-360 (d), (e)) in one token pass, then decodes the document with
// numbers kept exact. Duplicate keys are refused because Go keeps the last
// value and PostgreSQL's json type keeps every one, so two readers could see
// two different schemas.
func decodeSchema(raw json.RawMessage, bad *problems) (map[string]any, bool) {
	if len(raw) == 0 {
		bad.add("schema")
		return nil, false
	}
	if len(raw) > MaxSchemaBytes {
		bad.add("schema")
		return nil, false
	}
	if !scanSchema(raw, bad) {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil || doc == nil {
		bad.add("schema")
		return nil, false
	}
	return doc, true
}

// scanFrame is one open object or array during scanSchema.
type scanFrame struct {
	path   []string
	object bool
	keys   map[string]bool
	next   int    // next array index
	key    string // pending object key
	hasKey bool
}

// scanSchema walks the token stream once, refusing malformed JSON, trailing
// data, nesting deeper than MaxSchemaDepth and duplicate object keys.
func scanSchema(raw []byte, bad *problems) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var stack []*scanFrame
	ok := true
	// childPath is the path of the value that the next token starts.
	childPath := func() []string {
		if len(stack) == 0 {
			return []string{"schema"}
		}
		top := stack[len(stack)-1]
		var tok string
		if top.object {
			tok = top.key
			top.hasKey = false
		} else {
			tok = strconv.Itoa(top.next)
			top.next++
		}
		return append(slices.Clone(top.path), tok)
	}
	for {
		t, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			bad.add("schema")
			return false
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && !top.hasKey {
				if d, isDelim := t.(json.Delim); isDelim && d == '}' {
					stack = stack[:len(stack)-1]
					continue
				}
				k, _ := t.(string)
				if top.keys[k] {
					bad.add(append(slices.Clone(top.path), k)...)
					ok = false
				}
				top.keys[k] = true
				top.key, top.hasKey = k, true
				continue
			}
		}
		d, isDelim := t.(json.Delim)
		switch {
		case isDelim && (d == '}' || d == ']'):
			stack = stack[:len(stack)-1]
		case isDelim:
			p := childPath()
			if len(stack)+1 > MaxSchemaDepth {
				bad.add(p...)
				return false
			}
			stack = append(stack, &scanFrame{path: p, object: d == '{', keys: map[string]bool{}})
		default:
			childPath()
		}
		if len(stack) == 0 && dec.More() {
			bad.add("schema") // trailing data after the document
			return false
		}
	}
	return ok
}

// checkShape refuses what Compile does not accept in the schema's structure
// and returns the top-level property schemas. The root is a closed object
// schema: "type":"object", a "properties" object whose values are object
// schemas, "additionalProperties": false and no "patternProperties". A
// closed root is what makes adding an optional property provably
// Compatible: no existing row can already hold a key the new version types.
func checkShape(doc map[string]any, bad *problems) map[string]map[string]any {
	if s, ok := doc["$schema"]; ok && s != draft2020URL {
		bad.add("schema", "$schema")
	}
	if doc["type"] != "object" {
		bad.add("schema", "type")
	}
	if doc["additionalProperties"] != false {
		bad.add("schema", "additionalProperties")
	}
	if _, ok := doc["patternProperties"]; ok {
		bad.add("schema", "patternProperties") // a pattern could match a name added later
	}
	w := walker{bad: bad, schemas: map[string]bool{}, anchors: map[string]bool{}}
	w.walk(doc, []string{"schema"})
	w.checkRefs()

	rawProps, ok := doc["properties"].(map[string]any)
	if !ok {
		bad.add("schema", "properties")
		return nil
	}
	props := make(map[string]map[string]any, len(rawProps))
	for name, v := range rawProps {
		p, ok := v.(map[string]any)
		if !ok {
			bad.add("schema", "properties", name)
			continue
		}
		props[name] = p
	}
	return props
}

// isTopLevelProperty reports whether path is /schema/properties/<name>.
func isTopLevelProperty(path []string) bool {
	return len(path) == 3 && path[1] == "properties"
}

// refSite is one $ref or $dynamicRef found by the walker.
type refSite struct {
	path  []string // where the keyword is, from /schema
	value string
}

// walker visits every subschema reachable through the known subschema
// keywords. It records where each one is and which anchors they declare, so
// checkRefs can refuse a local $ref whose target was never walked (one that
// points into an unknown keyword, where a nested default or x-nava-unique
// would go unchecked).
type walker struct {
	bad     *problems
	schemas map[string]bool // JSON pointers, relative to the schema root
	anchors map[string]bool // $anchor and $dynamicAnchor names
	refs    []refSite
}

// walk checks one subschema: $ref and $dynamicRef are local, "default" and
// x-nava-unique appear only on top-level properties, x-nava-unique is a
// boolean, and $schema and $id appear only at the root (an embedded $id
// would move the base the local refs resolve against).
func (w *walker) walk(node any, path []string) {
	w.schemas[pointer(path[1:]...)] = true
	obj, ok := node.(map[string]any)
	if !ok {
		return // boolean schema
	}
	root := len(path) == 1
	at := func(kw string) []string { return append(slices.Clone(path), kw) }
	for kw, v := range obj {
		switch {
		case kw == "$ref" || kw == "$dynamicRef":
			s, ok := v.(string)
			if !ok || len(s) == 0 || s[0] != '#' {
				w.bad.add(at(kw)...)
				continue
			}
			w.refs = append(w.refs, refSite{path: at(kw), value: s[1:]})
		case kw == "$anchor" || kw == "$dynamicAnchor":
			if s, ok := v.(string); ok {
				w.anchors[s] = true
			}
		case kw == "$schema" || kw == "$id":
			if !root {
				w.bad.add(at(kw)...)
			}
		case kw == "default":
			if !isTopLevelProperty(path) {
				w.bad.add(at(kw)...)
			}
		case kw == UniqueKeyword:
			if _, isBool := v.(bool); !isBool || !isTopLevelProperty(path) {
				w.bad.add(at(kw)...)
			}
		case subschemaKind(kw) == oneSubschema:
			w.walk(v, at(kw))
		case subschemaKind(kw) == subschemaArray:
			arr, _ := v.([]any)
			for i, s := range arr {
				w.walk(s, append(at(kw), strconv.Itoa(i)))
			}
		case subschemaKind(kw) == subschemaByName:
			m, _ := v.(map[string]any)
			for name, s := range m {
				w.walk(s, append(at(kw), name))
			}
		}
	}
}

// checkRefs refuses a local reference whose target is not a walked
// subschema: a JSON-pointer fragment must name one exactly, and a plain-name
// fragment must name an anchor a walked subschema declares.
func (w *walker) checkRefs() {
	for _, r := range w.refs {
		target, ok := percentDecode(r.value)
		switch {
		case !ok:
			w.bad.add(r.path...)
		case target == "" || target[0] == '/':
			if !w.schemas[target] {
				w.bad.add(r.path...)
			}
		case !w.anchors[target]:
			w.bad.add(r.path...)
		}
	}
}

// percentDecode undoes URL percent-encoding in a fragment, the step the
// library applies before reading it as a JSON pointer.
func percentDecode(s string) (string, bool) {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			sb.WriteByte(s[i])
			continue
		}
		if i+2 >= len(s) {
			return "", false
		}
		b, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
		if err != nil {
			return "", false
		}
		sb.WriteByte(byte(b))
		i += 2
	}
	return sb.String(), true
}

// schemaErrorPaths turns a compiler error into paths, never into text. A
// metaschema failure points at each offending location in the schema; any
// other failure (a regex that does not compile, a load the loader refused)
// points at the schema as a whole.
func schemaErrorPaths(err error, bad *problems) {
	var sve *jsonschema.SchemaValidationError
	var ve *jsonschema.ValidationError
	if errors.As(err, &sve) && errors.As(sve.Err, &ve) {
		before := len(*bad)
		var leaves func(e *jsonschema.ValidationError)
		leaves = func(e *jsonschema.ValidationError) {
			if len(e.Causes) == 0 {
				bad.add(append([]string{"schema"}, e.InstanceLocation...)...)
				return
			}
			for _, c := range e.Causes {
				leaves(c)
			}
		}
		leaves(ve)
		if len(*bad) > before {
			return
		}
	}
	bad.add("schema")
}

// checkDefaults refuses a default that does not satisfy its own property
// schema (TICK-360 (a)) and returns each default as JSON. The property
// subschema is compiled by the same per-call compiler, so local $refs
// resolve against this document only.
func checkDefaults(c *jsonschema.Compiler, props map[string]map[string]any, bad *problems) map[string]json.RawMessage {
	defaults := map[string]json.RawMessage{}
	for name, p := range props {
		v, ok := p["default"]
		if !ok {
			continue
		}
		sub, err := c.Compile(schemaURL + "#" + fragment("properties", name))
		if err != nil || sub.Validate(v) != nil {
			bad.add("schema", "properties", name, "default")
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			bad.add("schema", "properties", name, "default")
			continue
		}
		defaults[name] = raw
	}
	return defaults
}

// fragment builds a URL fragment holding a JSON pointer, escaped the way the
// library decodes it (percent-decoding, then RFC 6901).
func fragment(tokens ...string) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteByte('/')
		for _, b := range []byte(pointer(t)[1:]) {
			if isUnreserved(b) {
				sb.WriteByte(b)
			} else {
				fmt.Fprintf(&sb, "%%%02X", b)
			}
		}
	}
	return sb.String()
}

// isUnreserved reports whether b needs no percent-encoding (RFC 3986).
func isUnreserved(b byte) bool {
	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9' ||
		b == '-' || b == '.' || b == '_' || b == '~'
}

// checkFixedColumns refuses a mapping to an unknown column, a column mapped
// twice, a property the schema lacks, a property whose type does not match
// its column, and a state property whose enum is not the state set
// (TICK-360 (b)). Exactly one property maps to ColumnState.
func checkFixedColumns(d Definition, props map[string]map[string]any, bad *problems) {
	used := map[string]bool{}
	names := slices.Sorted(maps.Keys(d.FixedColumns)) // sorted, so a clash flags the same name every time
	for _, name := range names {
		col := d.FixedColumns[name]
		at := []string{"fixed_columns", name}
		p, ok := props[name]
		if !ok || used[col] {
			bad.add(at...)
			continue
		}
		used[col] = true
		switch col {
		case ColumnState:
			if p["type"] != "string" {
				bad.add("schema", "properties", name, "type")
			}
			if !sameStringSet(p["enum"], d.States.States) {
				bad.add("schema", "properties", name, "enum")
			}
		case ColumnRequesterID, ColumnAssigneeID, ColumnAssignmentGroupID:
			if p["type"] != "string" {
				bad.add("schema", "properties", name, "type")
			}
			if p["format"] != "uuid" {
				bad.add("schema", "properties", name, "format")
			}
		default:
			bad.add(at...)
		}
	}
	if !used[ColumnState] {
		bad.add("fixed_columns")
	}
}

// sameStringSet reports whether enum is an array of strings equal, as a set,
// to states.
func sameStringSet(enum any, states []string) bool {
	arr, ok := enum.([]any)
	if !ok || len(arr) != len(states) {
		return false
	}
	got := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return false
		}
		got = append(got, s)
	}
	want := slices.Clone(states)
	slices.Sort(got)
	slices.Sort(want)
	return reflect.DeepEqual(got, want)
}

// checkStates checks the state model: states are non-empty and distinct,
// Initial is one of them, every transition names known states and is listed
// once, and every state is reachable from Initial.
func checkStates(m StateModel, bad *problems) {
	known := map[string]bool{}
	if len(m.States) == 0 {
		bad.add("states", "states")
	}
	for i, s := range m.States {
		if s == "" || known[s] {
			bad.add("states", "states", strconv.Itoa(i))
		}
		known[s] = true
	}
	if !known[m.Initial] {
		bad.add("states", "initial")
	}
	next := map[string][]string{}
	seen := map[Transition]bool{}
	for i, t := range m.Transitions {
		at := []string{"states", "transitions", strconv.Itoa(i)}
		if !known[t.From] {
			bad.add(append(at, "from")...)
		}
		if !known[t.To] {
			bad.add(append(at, "to")...)
		}
		if seen[t] {
			bad.add(at...)
		}
		seen[t] = true
		next[t.From] = append(next[t.From], t.To)
	}
	if !known[m.Initial] {
		return
	}
	reached := map[string]bool{m.Initial: true}
	queue := []string{m.Initial}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, to := range next[s] {
			if !reached[to] {
				reached[to] = true
				queue = append(queue, to)
			}
		}
	}
	for i, s := range m.States {
		if !reached[s] {
			bad.add("states", "states", strconv.Itoa(i))
		}
	}
}
