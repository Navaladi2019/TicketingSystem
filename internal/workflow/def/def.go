// Package def is the workflow definition format v1, "nava.workflow/v1"
// (P3-20, PLAN-003 D-3-11, D-3-12, D-3-15): the JSON Schema in schema.json,
// the Go validator Parse, the duration parser and AssignStepIDs.
//
// It is pure Go with no Temporal import, so the interpreter (P3-22a), the
// publish path (P3-41) and the Epic 12 canvas can all depend on it.
//
// A definition holds literals only: no expressions, and no field reference in
// a duration, a timeout or an edge, so a wait length or a path can never carry
// record data (TICK-363).
package def

import (
	"bytes"
	_ "embed" // schema.json is the shape contract
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// Format is the only accepted value of a definition's "format".
const Format = "nava.workflow/v1"

// MaxSteps bounds the number of steps in one definition.
const MaxSteps = 200

// schemaURL is the $id of schema.json. It is never fetched.
const schemaURL = "https://nava.local/schemas/workflow/v1/schema.json"

//go:embed schema.json
var schemaJSON []byte

// Trigger events.
const (
	OnRecordCreated = "record.created"
	OnRecordUpdated = "record.updated"
)

// StepType is the closed set of step types in v1.
type StepType string

// Step types.
const (
	TypeCondition    StepType = "condition"
	TypeBranch       StepType = "branch"
	TypeTimer        StepType = "timer"
	TypeWaitSignal   StepType = "wait_signal"
	TypeUpdateRecord StepType = "update_record"
	TypeEnd          StepType = "end"
)

// Op is a comparison operator.
type Op string

// Comparison operators.
const (
	OpEq Op = "eq"
	OpNe Op = "ne"
	OpIn Op = "in"
)

// Definition is a parsed, validated workflow definition.
type Definition struct {
	Format  string  `json:"format"`
	Trigger Trigger `json:"trigger"`
	Start   string  `json:"start"`
	Steps   []Step  `json:"steps"`

	index map[string]int
}

// Trigger names the record event that starts the workflow.
type Trigger struct {
	On     string `json:"on"`
	Entity string `json:"entity"`
}

// Condition compares one record field with a literal (see Holds).
type Condition struct {
	Field string          `json:"field,omitempty"` // JSON Pointer into the record
	Op    Op              `json:"op,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// Case is one arm of a branch step.
type Case struct {
	Condition
	Next string `json:"next"`
}

// Step is one step. Which fields are set depends on Type; schema.json allows
// no others.
type Step struct {
	Key  string   `json:"key"`
	Type StepType `json:"type"`

	// condition
	Condition
	Then string `json:"then,omitempty"`
	Else string `json:"else,omitempty"`

	// branch
	Cases   []Case `json:"cases,omitempty"`
	Default string `json:"default,omitempty"`

	// timer
	Duration string `json:"duration,omitempty"`

	// wait_signal
	Name      string `json:"name,omitempty"`
	Timeout   string `json:"timeout,omitempty"`
	OnTimeout string `json:"on_timeout,omitempty"`

	// update_record
	Set map[string]json.RawMessage `json:"set,omitempty"`

	// timer, wait_signal (on signal), update_record
	Next string `json:"next,omitempty"`

	waitSeconds int32
}

// WaitSeconds is the timer duration or the wait_signal timeout in seconds,
// within [MinWaitSeconds, MaxWaitSeconds]; 0 for every other type.
func (s *Step) WaitSeconds() int32 { return s.waitSeconds }

// Edge is one out-edge of a step: the JSON pointer of the field that names it
// (relative to the step) and the target key.
type Edge struct {
	Field string // "/next", "/then", "/else", "/cases/0/next", "/default", "/on_timeout"
	To    string
}

// Edges returns the step's out-edges in a fixed order. An end step has none.
func (s *Step) Edges() []Edge {
	var out []Edge
	switch s.Type {
	case TypeCondition:
		out = append(out, Edge{"/then", s.Then}, Edge{"/else", s.Else})
	case TypeBranch:
		for i, c := range s.Cases {
			out = append(out, Edge{fmt.Sprintf("/cases/%d/next", i), c.Next})
		}
		out = append(out, Edge{"/default", s.Default})
	case TypeTimer, TypeUpdateRecord:
		out = append(out, Edge{"/next", s.Next})
	case TypeWaitSignal:
		out = append(out, Edge{"/next", s.Next}, Edge{"/on_timeout", s.OnTimeout})
	case TypeEnd:
	}
	return out
}

// Step returns the step with the given key.
func (d *Definition) Step(key string) (*Step, bool) {
	i, ok := d.index[key]
	if !ok {
		return nil, false
	}
	return &d.Steps[i], true
}

// SignalNames returns the distinct wait_signal names, sorted.
func (d *Definition) SignalNames() []string {
	seen := map[string]struct{}{}
	for i := range d.Steps {
		if d.Steps[i].Type == TypeWaitSignal {
			seen[d.Steps[i].Name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// ErrInvalidDefinition is returned by Parse. Paths are RFC 6901 JSON pointers
// into the submitted definition, sorted and without duplicates; "" is the
// whole document. It carries no definition or record values.
type ErrInvalidDefinition struct {
	Paths []string
}

func (e *ErrInvalidDefinition) Error() string {
	quoted := make([]string, len(e.Paths))
	for i, p := range e.Paths {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "workflow definition: invalid at " + strings.Join(quoted, ", ")
}

func invalid(paths ...string) *ErrInvalidDefinition {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return &ErrInvalidDefinition{Paths: out}
}

// Parse validates b against schema.json, then checks the step graph: unique
// keys, every edge and start naming a known key, no cycle, every step
// reachable from start, and duration bounds. It returns *ErrInvalidDefinition
// on any failure. It validates keys and literals only: the update_record.set
// check against the entity schema is a publish-time check (P3-41).
func Parse(b []byte) (*Definition, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return nil, invalid("")
	}
	sch, err := compileSchema()
	if err != nil {
		return nil, err
	}
	if err := sch.Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return nil, invalid(leafPaths(ve)...)
		}
		return nil, fmt.Errorf("workflow definition: validate: %w", err)
	}

	var d Definition
	if err := json.Unmarshal(b, &d); err != nil {
		// The schema accepted the document, so this is a bug in the Go types.
		return nil, fmt.Errorf("workflow definition: decode: %w", err)
	}
	if paths := d.check(); len(paths) > 0 {
		return nil, invalid(paths...)
	}
	return &d, nil
}

// compileSchema compiles the embedded schema with a loader that refuses every
// URL (D-3-18), so nothing outside schema.json and the 2020-12 meta-schema is
// ever read. A new Compiler per call keeps Parse free of shared state.
func compileSchema() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("workflow definition: embedded schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.UseLoader(denyAll{})
	if err := c.AddResource(schemaURL, doc); err != nil {
		return nil, fmt.Errorf("workflow definition: embedded schema: %w", err)
	}
	sch, err := c.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("workflow definition: embedded schema: %w", err)
	}
	return sch, nil
}

type denyAll struct{}

func (denyAll) Load(url string) (any, error) {
	return nil, fmt.Errorf("workflow definition: schema loading is disabled (%s)", url)
}

// leafPaths returns the instance pointer of every leaf validation error. A
// missing or extra property is reported at the property itself, so a
// wait_signal without a timeout points at ".../timeout".
func leafPaths(ve *jsonschema.ValidationError) []string {
	if len(ve.Causes) > 0 {
		var out []string
		for _, c := range ve.Causes {
			out = append(out, leafPaths(c)...)
		}
		return out
	}
	base := pointer(ve.InstanceLocation)
	switch k := ve.ErrorKind.(type) {
	case *kind.Required:
		out := make([]string, len(k.Missing))
		for i, m := range k.Missing {
			out[i] = base + "/" + escape(m)
		}
		return out
	case *kind.AdditionalProperties:
		out := make([]string, len(k.Properties))
		for i, p := range k.Properties {
			out[i] = base + "/" + escape(p)
		}
		return out
	}
	return []string{base}
}

func pointer(tokens []string) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteByte('/')
		sb.WriteString(escape(t))
	}
	return sb.String()
}

func escape(token string) string {
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

// check runs the graph and duration rules on a schema-valid definition and
// returns the pointers of every violation.
func (d *Definition) check() []string {
	var bad []string
	if len(d.Steps) > MaxSteps { // also in schema.json; kept so the Go type never trusts it
		return []string{"/steps"}
	}
	d.index = make(map[string]int, len(d.Steps))
	for i := range d.Steps {
		s := &d.Steps[i]
		if !validKey(s.Key) {
			bad = append(bad, fmt.Sprintf("/steps/%d/key", i))
			continue
		}
		if _, dup := d.index[s.Key]; dup {
			bad = append(bad, fmt.Sprintf("/steps/%d/key", i))
			continue
		}
		d.index[s.Key] = i
	}
	if len(bad) > 0 {
		// Edge checks against a partial key index would only add noise.
		return bad
	}

	for i := range d.Steps {
		s := &d.Steps[i]
		switch s.Type {
		case TypeTimer:
			n, err := ParseDuration(s.Duration)
			if err != nil {
				bad = append(bad, fmt.Sprintf("/steps/%d/duration", i))
			}
			s.waitSeconds = n
		case TypeWaitSignal:
			n, err := ParseDuration(s.Timeout)
			if err != nil {
				bad = append(bad, fmt.Sprintf("/steps/%d/timeout", i))
			}
			s.waitSeconds = n
		case TypeCondition, TypeBranch, TypeUpdateRecord, TypeEnd:
		}
		for _, e := range s.Edges() {
			if _, ok := d.index[e.To]; !ok {
				bad = append(bad, fmt.Sprintf("/steps/%d%s", i, e.Field))
			}
		}
	}
	if _, ok := d.index[d.Start]; !ok {
		bad = append(bad, "/start")
	}
	if len(bad) > 0 {
		// Cycle and reachability need a well-formed graph.
		return bad
	}
	if p, ok := d.cycle(); ok {
		return []string{p}
	}
	return d.unreachable()
}

// cycle reports the edge that closes a cycle, if any (iterative DFS over all
// edge fields, so 200 steps never recurse deeply).
func (d *Definition) cycle() (string, bool) {
	const (
		white = iota
		grey
		black
	)
	color := make([]int, len(d.Steps))
	type frame struct {
		step  int
		edges []Edge
		next  int
	}
	for root := range d.Steps {
		if color[root] != white {
			continue
		}
		stack := []frame{{step: root, edges: d.Steps[root].Edges()}}
		color[root] = grey
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			if top.next == len(top.edges) {
				color[top.step] = black
				stack = stack[:len(stack)-1]
				continue
			}
			e := top.edges[top.next]
			top.next++
			to := d.index[e.To]
			switch color[to] {
			case grey:
				return fmt.Sprintf("/steps/%d%s", top.step, e.Field), true
			case white:
				color[to] = grey
				stack = append(stack, frame{step: to, edges: d.Steps[to].Edges()})
			}
		}
	}
	return "", false
}

func (d *Definition) unreachable() []string {
	seen := make([]bool, len(d.Steps))
	queue := []int{d.index[d.Start]}
	seen[queue[0]] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range d.Steps[cur].Edges() {
			if to := d.index[e.To]; !seen[to] {
				seen[to] = true
				queue = append(queue, to)
			}
		}
	}
	var bad []string
	for i, ok := range seen {
		if !ok {
			bad = append(bad, fmt.Sprintf("/steps/%d", i))
		}
	}
	return bad
}

// validKey reports whether k matches ^[a-z][a-z0-9_]{0,39}$ (also in
// schema.json).
func validKey(k string) bool {
	if len(k) == 0 || len(k) > 40 || k[0] < 'a' || k[0] > 'z' {
		return false
	}
	for i := 1; i < len(k); i++ {
		c := k[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
