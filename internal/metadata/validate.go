package metadata

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// Validate checks a record against the schema, with format assertion on. It
// returns one FieldError per failure, sorted by path then code, or nil when
// the record is valid. data must hold JSON values as encoding/json decodes
// them (map[string]any, []any, string, bool, nil, float64 or json.Number);
// any other Go type fails with code "type".
func (c *Compiled) Validate(data map[string]any) []FieldError {
	if data == nil {
		data = map[string]any{}
	}
	err := c.schema.Validate(data)
	if err == nil {
		return nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return []FieldError{{Path: "", Code: "schema"}}
	}
	var out []FieldError
	collectFieldErrors(ve, &out)
	slices.SortFunc(out, func(a, b FieldError) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Code, b.Code))
	})
	return slices.Compact(out)
}

// collectFieldErrors appends one FieldError per leaf of the error tree. The
// inner nodes (allOf, $ref, the root) only group their causes.
func collectFieldErrors(e *jsonschema.ValidationError, out *[]FieldError) {
	if len(e.Causes) > 0 {
		for _, cause := range e.Causes {
			collectFieldErrors(cause, out)
		}
		return
	}
	at := func(extra ...string) string {
		return pointer(append(slices.Clone(e.InstanceLocation), extra...)...)
	}
	switch k := e.ErrorKind.(type) {
	case *kind.Required:
		for _, m := range k.Missing {
			*out = append(*out, FieldError{Path: at(m), Code: "required"})
		}
	case *kind.DependentRequired:
		for _, m := range k.Missing {
			*out = append(*out, FieldError{Path: at(m), Code: "dependentRequired"})
		}
	case *kind.AdditionalProperties:
		for _, p := range k.Properties {
			*out = append(*out, FieldError{Path: at(p), Code: "additionalProperties"})
		}
	case *kind.FalseSchema:
		*out = append(*out, FieldError{Path: at(), Code: "false"})
	case *kind.InvalidJsonValue:
		*out = append(*out, FieldError{Path: at(), Code: "type"})
	default:
		code := "schema"
		if kp := e.ErrorKind.KeywordPath(); len(kp) > 0 {
			code = kp[0]
		}
		*out = append(*out, FieldError{Path: at(), Code: code})
	}
}

// ApplyDefaults returns a copy of data with each missing top-level property
// that has a default set to it. A key that is present, even with a null
// value, is never overwritten, and data itself is never changed. Default
// numbers come back as json.Number. Nested defaults do not exist in v1:
// Compile refuses them.
func (c *Compiled) ApplyDefaults(data map[string]any) map[string]any {
	out := make(map[string]any, len(data)+len(c.defaults))
	maps.Copy(out, data)
	for name, raw := range c.defaults {
		if _, present := out[name]; present {
			continue
		}
		out[name] = decodeValue(raw)
	}
	return out
}

// decodeValue decodes JSON that Compile produced with json.Marshal, so it
// cannot fail; numbers stay exact as json.Number.
func decodeValue(raw json.RawMessage) any {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	_ = dec.Decode(&v) // Compile marshalled raw itself, so it is valid JSON
	return v
}

// UniqueFields returns the top-level properties marked "x-nava-unique": true,
// sorted.
func (c *Compiled) UniqueFields() []string {
	return slices.Clone(c.unique)
}

// NumberPrefix returns the definition's record number prefix.
func (c *Compiled) NumberPrefix() string { return c.numberPrefix }

// FixedColumns returns a copy of the mapping from top-level property to the
// fixed column that stores it (one of the Column constants).
func (c *Compiled) FixedColumns() map[string]string { return maps.Clone(c.fixedColumns) }

// StateProperty returns the top-level property mapped to ColumnState. Compile
// guarantees there is exactly one.
func (c *Compiled) StateProperty() string {
	for name, col := range c.fixedColumns {
		if col == ColumnState {
			return name
		}
	}
	return ""
}

// InitialState returns the state model's initial state.
func (c *Compiled) InitialState() string { return c.states.Initial }

// AllowsTransition reports whether the state model lists a transition from
// one state to another. Staying in the same state is not a transition, and
// this reports it only if the model lists it.
func (c *Compiled) AllowsTransition(from, to string) bool {
	return slices.Contains(c.states.Transitions, Transition{From: from, To: to})
}

// States returns a copy of the state model's states.
func (c *Compiled) States() []string { return slices.Clone(c.states.States) }
