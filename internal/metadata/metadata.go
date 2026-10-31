// Package metadata compiles entity definitions (a JSON Schema 2020-12
// document, the fixed-column mapping, the number prefix and the state model),
// validates records against them, fills defaults, and classifies a new
// version of a definition against the previous one (PLAN-003 D-3-2, D-3-4,
// D-3-5, D-3-18).
//
// The package is pure: it imports no database, network or file-system code
// (import_test.go checks the graph). Every Compile builds its own
// jsonschema.Compiler with a loader that refuses every URL, so a schema can
// never read a file, call the network, or resolve a $ref against another
// tenant's $id.
//
// Paths in ErrInvalidDefinition and in Classify's result are JSON pointers
// into the definition as its JSON form (see Definition's tags), for example
// "/schema/properties/priority/default" or "/states/initial".
package metadata

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Caps on the schema document Compile accepts (TICK-360 (d)). A definition is
// tenant input: the caps bound the work one Compile and every later Validate
// can be made to do.
const (
	// MaxSchemaBytes is the largest Definition.Schema, in bytes.
	MaxSchemaBytes = 64 << 10
	// MaxSchemaDepth is the deepest nesting of JSON objects and arrays in
	// Definition.Schema; the root object is depth 1.
	MaxSchemaDepth = 32
)

// Fixed columns a schema property can map to (Definition.FixedColumns values).
const (
	ColumnState             = "state"
	ColumnRequesterID       = "requester_id"
	ColumnAssigneeID        = "assignee_id"
	ColumnAssignmentGroupID = "assignment_group_id"
)

// UniqueKeyword marks a top-level property whose value must be unique within
// the entity type (P3-11). Its value is a boolean.
const UniqueKeyword = "x-nava-unique"

// Definition is an entity type as a tenant writes it.
type Definition struct {
	// Key names the entity type: ^[a-z][a-z0-9_]{1,40}$.
	Key string `json:"key"`
	// Schema is a JSON Schema 2020-12 document whose root is an object schema.
	Schema json.RawMessage `json:"schema"`
	// NumberPrefix prefixes record numbers: ^[A-Z]{2,6}$.
	NumberPrefix string `json:"number_prefix"`
	// FixedColumns maps a top-level schema property to the fixed column that
	// stores it: one of ColumnState, ColumnRequesterID, ColumnAssigneeID,
	// ColumnAssignmentGroupID. Exactly one property maps to ColumnState.
	FixedColumns map[string]string `json:"fixed_columns"`
	// States is the state model of the property mapped to ColumnState.
	States StateModel `json:"states"`
}

// StateModel is the set of states and the allowed transitions between them.
type StateModel struct {
	Initial     string       `json:"initial"`
	States      []string     `json:"states"`
	Transitions []Transition `json:"transitions"`
}

// Transition allows a record to move from one state to another.
type Transition struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Compat is the compatibility class of a new definition version against the
// previous one (D-3-5). The classes are ordered: a change takes the highest
// class of its parts.
type Compat int

const (
	// Compatible versions read every existing row unchanged.
	Compatible Compat = iota
	// NeedsBackfill versions change how existing rows read, and need a
	// backfill before they become current.
	NeedsBackfill
	// Breaking versions cannot be applied to existing rows. Anything the
	// classifier cannot prove compatible is Breaking.
	Breaking
)

// String returns the class name used in logs and API responses.
func (c Compat) String() string {
	switch c {
	case Compatible:
		return "compatible"
	case NeedsBackfill:
		return "needs_backfill"
	case Breaking:
		return "breaking"
	default:
		return "unknown"
	}
}

// ErrInvalidDefinition is returned by Compile when a definition is refused.
// Paths holds a JSON pointer to every offending location, sorted. It never
// carries library or OS error text.
type ErrInvalidDefinition struct {
	Paths []string
}

func (e *ErrInvalidDefinition) Error() string {
	return "metadata: invalid entity definition at " + strings.Join(e.Paths, ", ")
}

// ErrNilCompiled is returned by Classify when either argument is nil.
var ErrNilCompiled = errors.New("metadata: nil compiled definition")

// ErrKeyMismatch is returned by Classify when prev and next are versions of
// different entity types.
var ErrKeyMismatch = errors.New("metadata: definitions have different keys")

// FieldError is one validation failure: Path is a JSON pointer into the
// record and Code is the JSON Schema keyword that failed ("required",
// "type", "format", ...). For "required" and "additionalProperties" the
// path names the missing or extra property.
type FieldError struct {
	Path string `json:"path"`
	Code string `json:"code"`
}

// Compiled is a definition that passed Compile. It is immutable and safe for
// concurrent use.
type Compiled struct {
	key          string
	numberPrefix string
	fixedColumns map[string]string
	states       StateModel
	schema       *jsonschema.Schema
	// doc is the decoded schema (numbers as json.Number); read only.
	doc map[string]any
	// defaults holds each top-level property's default as JSON, decoded
	// afresh on every ApplyDefaults so callers never share a value.
	defaults map[string]json.RawMessage
	unique   []string
}

// pointer builds a JSON pointer (RFC 6901) from unescaped tokens.
func pointer(tokens ...string) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteByte('/')
		t = strings.ReplaceAll(t, "~", "~0")
		sb.WriteString(strings.ReplaceAll(t, "/", "~1"))
	}
	return sb.String()
}
