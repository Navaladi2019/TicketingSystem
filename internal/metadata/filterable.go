package metadata

import (
	"slices"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

// FilterableKeyword flags a top-level property whose values the platform
// copies into a typed side table so a list can filter on it (ADR-006 D8). Its
// value is a boolean.
const FilterableKeyword = "x-nava-filterable"

// SearchableKeyword is reserved for free-text search (ADR-006, RFC-016).
// Compile refuses it anywhere until RFC-016 is accepted.
const SearchableKeyword = "x-nava-searchable"

// Refusal codes. Each is a registered API error code (api/errors.yaml) that
// a caller returns as is, with 422.
const (
	// RefusalFilterableNotTopLevel: x-nava-filterable appears somewhere other
	// than directly on a top-level property.
	RefusalFilterableNotTopLevel = "filterable_field_not_top_level"
	// RefusalFilterableTypeChange: a property flagged in the previous version
	// changes the value type it indexes as (Classify).
	RefusalFilterableTypeChange = "filterable_field_type_change"
	// RefusalSearchableNotAvailable: x-nava-searchable appears anywhere.
	RefusalSearchableNotAvailable = "searchable_not_available"
)

// RefusalError is returned by Compile and Classify for a definition the
// platform refuses under a specific code, rather than as a generic invalid
// definition. Code is one of the Refusal constants; Paths are the offending
// JSON pointers, sorted, as in ErrInvalidDefinition.
type RefusalError struct {
	Code  string
	Paths []string
}

func (e *RefusalError) Error() string {
	return "metadata: " + e.Code + " at " + strings.Join(e.Paths, ", ")
}

// FilterableField is one top-level property flagged x-nava-filterable.
// ValueType is the side table its values are stored in; an array property
// stores each element, so it has its element's type.
type FilterableField struct {
	Key       string
	ValueType fieldindex.ValueType
}

// Filterable returns the flagged properties, sorted by Key. The slice is the
// caller's.
func (c *Compiled) Filterable() []FilterableField {
	return slices.Clone(c.filterable)
}

// typeApplicators are keywords that could change, at validation time, which
// JSON type a property's value has. A flagged property may carry none of them,
// so its value type is read from its own "type", "format" and "enum".
func typeApplicators() []string {
	return []string{"$ref", "$dynamicRef", "allOf", "anyOf", "oneOf", "not", "if", "then", "else"}
}

// filterableValueType returns the value type a top-level property indexes
// as: string → text (with format date-time or date → ts), number or integer
// → num, boolean → bool, an array of one of those → that type, and a
// property with no "type" whose enum holds only strings → text. Anything else
// gives false: a property is filterable only when its value type is certain.
func filterableValueType(p map[string]any) (fieldindex.ValueType, bool) {
	if v, ok := scalarValueType(p); ok {
		return v, true
	}
	if p["type"] != "array" || hasAny(p, typeApplicators()) || hasAny(p, []string{"prefixItems", "contains"}) {
		return "", false
	}
	items, ok := p["items"].(map[string]any)
	if !ok {
		return "", false
	}
	return scalarValueType(items)
}

// scalarValueType is filterableValueType for a non-array schema.
func scalarValueType(p map[string]any) (fieldindex.ValueType, bool) {
	if hasAny(p, typeApplicators()) {
		return "", false
	}
	t, hasType := p["type"]
	if !hasType {
		if allStrings(p["enum"]) {
			return fieldindex.TypeText, true
		}
		return "", false
	}
	switch t {
	case "string":
		if f := p["format"]; f == "date-time" || f == "date" {
			return fieldindex.TypeTS, true
		}
		return fieldindex.TypeText, true
	case "number", "integer":
		return fieldindex.TypeNum, true
	case "boolean":
		return fieldindex.TypeBool, true
	}
	return "", false
}

func hasAny(p map[string]any, kws []string) bool {
	for _, kw := range kws {
		if _, ok := p[kw]; ok {
			return true
		}
	}
	return false
}

// allStrings reports whether v is a non-empty array of strings.
func allStrings(v any) bool {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return false
	}
	for _, e := range arr {
		if _, ok := e.(string); !ok {
			return false
		}
	}
	return true
}

func isFilterable(p map[string]any) bool {
	v, _ := p[FilterableKeyword].(bool)
	return v
}

// collectFilterable returns the flagged top-level properties, sorted, and
// adds a path for each flagged one whose value type is not one of the four.
// The walker has already refused a non-boolean flag.
func collectFilterable(props map[string]map[string]any, bad *problems) []FilterableField {
	var out []FilterableField
	for name, p := range props {
		if !isFilterable(p) {
			continue
		}
		v, ok := filterableValueType(p)
		if !ok {
			bad.add("schema", "properties", name, FilterableKeyword)
			continue
		}
		out = append(out, FilterableField{Key: name, ValueType: v})
	}
	slices.SortFunc(out, func(a, b FilterableField) int { return strings.Compare(a.Key, b.Key) })
	return out
}

// refusal returns the refusal the walker found, searchable first, or nil.
func (w *walker) refusal() error {
	for _, r := range []struct {
		code  string
		paths [][]string
	}{
		{RefusalSearchableNotAvailable, w.searchable},
		{RefusalFilterableNotTopLevel, w.nestedFilterable},
	} {
		if len(r.paths) == 0 {
			continue
		}
		paths := make([]string, len(r.paths))
		for i, p := range r.paths {
			paths[i] = pointer(p...)
		}
		slices.Sort(paths)
		return &RefusalError{Code: r.code, Paths: slices.Compact(paths)}
	}
	return nil
}

// classifyFilterable refuses a change to the value type of a property that
// prev flags: the side table already holds its values under the old type
// (ADR-006 D8). Removing the property is left to the Breaking rule.
func classifyFilterable(prev, next *Compiled) error {
	nextProps, _ := next.doc["properties"].(map[string]any)
	var paths []string
	for _, f := range prev.filterable {
		n, ok := nextProps[f.Key].(map[string]any)
		if !ok {
			continue
		}
		if v, ok := filterableValueType(n); !ok || v != f.ValueType {
			paths = append(paths, pointer("schema", "properties", f.Key))
		}
	}
	if len(paths) == 0 {
		return nil
	}
	return &RefusalError{Code: RefusalFilterableTypeChange, Paths: paths} // prev.filterable is sorted
}
