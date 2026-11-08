package entitytypes

import (
	"encoding/json"
	"slices"
	"strconv"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// SecretRequiredError is returned by Create and Publish when a definition
// requires a property marked x-nava-secret or writeOnly (ADR-004-E5 D9, code
// definition_secret_required). A record may never hold a value at a marked
// path, so such a definition could not be satisfied by any record. Paths are
// JSON pointers into the schema, one per offending required entry. Nothing is
// written once it is returned.
type SecretRequiredError struct {
	Paths []string
}

func (e *SecretRequiredError) Error() string {
	return "entitytypes: definition_secret_required at " + strconv.Quote(joinPaths(e.Paths))
}

func joinPaths(p []string) string {
	var out string
	for i, s := range p {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

// CheckSchemaSecretRequired refuses a schema that requires a marked top-level
// property, through `required` or `dependentRequired`, at the root or in any
// subschema that applies at the root instance location. The marker itself
// stays legal. A schema of an unexpected shape is not judged here:
// metadata.Compile reports it.
func CheckSchemaSecretRequired(schema json.RawMessage) error {
	var root map[string]any
	if json.Unmarshal(schema, &root) != nil || root == nil {
		return nil
	}
	marked := map[string]bool{}
	props, _ := root["properties"].(map[string]any)
	for name, raw := range props {
		p, _ := raw.(map[string]any)
		secret, _ := p["x-nava-secret"].(bool)
		wo, _ := p["writeOnly"].(bool)
		if secret || wo {
			marked[name] = true
		}
	}
	if len(marked) == 0 {
		return nil
	}
	var bad []patchwalk.Pointer
	walkRequired(root, nil, marked, &bad)
	if len(bad) == 0 {
		return nil
	}
	slices.Sort(bad)
	paths := make([]string, len(bad))
	for i, p := range bad {
		paths[i] = string(p)
	}
	return &SecretRequiredError{Paths: paths}
}

// walkRequired visits the root and the subschemas that apply at the root
// instance location; properties, items and the like apply to children and are
// not walked.
func walkRequired(node any, path []string, marked map[string]bool, bad *[]patchwalk.Pointer) {
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	at := func(segs ...string) []string { return append(slices.Clone(path), segs...) }
	if req, ok := obj["required"].([]any); ok {
		for i, v := range req {
			if s, ok := v.(string); ok && marked[s] {
				*bad = append(*bad, patchwalk.FromSegments(at("required", strconv.Itoa(i))))
			}
		}
	}
	if dr, ok := obj["dependentRequired"].(map[string]any); ok {
		for trigger, list := range dr {
			arr, _ := list.([]any)
			for i, v := range arr {
				if s, ok := v.(string); ok && marked[s] {
					*bad = append(*bad, patchwalk.FromSegments(at("dependentRequired", trigger, strconv.Itoa(i))))
				}
			}
		}
	}
	for _, kw := range []string{"not", "if", "then", "else"} {
		if v, ok := obj[kw]; ok {
			walkRequired(v, at(kw), marked, bad)
		}
	}
	for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
		arr, _ := obj[kw].([]any)
		for i, s := range arr {
			walkRequired(s, at(kw, strconv.Itoa(i)), marked, bad)
		}
	}
	if ds, ok := obj["dependentSchemas"].(map[string]any); ok {
		for name, s := range ds {
			walkRequired(s, at("dependentSchemas", name), marked, bad)
		}
	}
}
