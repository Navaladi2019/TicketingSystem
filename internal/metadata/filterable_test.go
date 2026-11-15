package metadata_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

const flag = metadata.FilterableKeyword

// refusal asserts err is *RefusalError with code and returns its paths.
func refusal(t *testing.T, err error, code string) []string {
	t.Helper()
	var r *metadata.RefusalError
	if !errors.As(err, &r) {
		t.Fatalf("err = %v, want *RefusalError %s", err, code)
	}
	if r.Code != code {
		t.Fatalf("code = %s, want %s", r.Code, code)
	}
	if len(r.Paths) == 0 {
		t.Fatal("RefusalError with no paths")
	}
	return r.Paths
}

// TestCompileFilterableTopLevelTypes: a top-level property of each supported
// JSON type compiles and is listed with the value type it indexes as
// (ADR-006 D8, P7-03 scope 1-2).
func TestCompileFilterableTopLevelTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema map[string]any
		want   fieldindex.ValueType
	}{
		{"string", map[string]any{"type": "string"}, fieldindex.TypeText},
		{"string with format uuid", map[string]any{"type": "string", "format": "uuid"}, fieldindex.TypeText},
		{"string enum", map[string]any{"type": "string", "enum": []any{"a", "b"}}, fieldindex.TypeText},
		{"enum of strings without type", map[string]any{"enum": []any{"a", "b"}}, fieldindex.TypeText},
		{"number", map[string]any{"type": "number"}, fieldindex.TypeNum},
		{"integer", map[string]any{"type": "integer", "minimum": 0}, fieldindex.TypeNum},
		{"date-time", map[string]any{"type": "string", "format": "date-time"}, fieldindex.TypeTS},
		{"date", map[string]any{"type": "string", "format": "date"}, fieldindex.TypeTS},
		{"boolean", map[string]any{"type": "boolean"}, fieldindex.TypeBool},
		{"array of strings", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, fieldindex.TypeText},
		{"array of integers", map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}, fieldindex.TypeNum},
		{"array of dates", map[string]any{"type": "array", "items": map[string]any{"type": "string", "format": "date"}}, fieldindex.TypeTS},
		{"array of booleans", map[string]any{"type": "array", "items": map[string]any{"type": "boolean"}}, fieldindex.TypeBool},
		{"array of string enum", map[string]any{"type": "array", "items": map[string]any{"enum": []any{"x", "y"}}}, fieldindex.TypeText},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := baseSchema()
			tt.schema[flag] = true
			props(s)["f"] = tt.schema
			c := mustCompile(t, baseDefinition(t, s))
			want := []metadata.FilterableField{{Key: "f", ValueType: tt.want}}
			if got := c.Filterable(); !reflect.DeepEqual(got, want) {
				t.Fatalf("Filterable() = %v, want %v", got, want)
			}
		})
	}
}

func TestCompileFilterableListsSortedAndOnlyTrue(t *testing.T) {
	t.Parallel()
	s := baseSchema()
	prop(s, "title")[flag] = true
	prop(s, "priority")[flag] = true
	prop(s, "code")[flag] = false
	c := mustCompile(t, baseDefinition(t, s))
	want := []metadata.FilterableField{
		{Key: "priority", ValueType: fieldindex.TypeNum},
		{Key: "title", ValueType: fieldindex.TypeText},
	}
	got := c.Filterable()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filterable() = %v, want %v", got, want)
	}
	got[0].Key = "mutated" // the caller's copy
	if c.Filterable()[0].Key != "priority" {
		t.Fatal("Filterable shares its slice with the caller")
	}
	if got := mustCompile(t, baseDefinition(t, baseSchema())).Filterable(); len(got) != 0 {
		t.Fatalf("unflagged Filterable() = %v, want none", got)
	}
}

// TestCompileRefusesFilterableNotTopLevel: the flag anywhere but directly on
// a top-level property is 422 filterable_field_not_top_level, ahead of any
// other problem in the definition.
func TestCompileRefusesFilterableNotTopLevel(t *testing.T) {
	t.Parallel()
	nested := func() map[string]any {
		return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
			"inner": map[string]any{"type": "string", flag: true},
		}}
	}
	tests := []struct {
		name string
		mut  func(s map[string]any)
		want string
	}{
		{"nested property", func(s map[string]any) {
			props(s)["meta"] = nested()
		}, "/schema/properties/meta/properties/inner/" + flag},
		{"$ref-reached nested", func(s map[string]any) {
			s["$defs"] = map[string]any{"leaf": map[string]any{"type": "string", flag: true}}
			props(s)["meta"] = map[string]any{"$ref": "#/$defs/leaf"}
		}, "/schema/$defs/leaf/" + flag},
		{"inside allOf", func(s map[string]any) {
			props(s)["meta"] = map[string]any{"allOf": []any{map[string]any{"type": "string", flag: true}}}
		}, "/schema/properties/meta/allOf/0/" + flag},
		{"inside oneOf", func(s map[string]any) {
			props(s)["meta"] = map[string]any{"oneOf": []any{map[string]any{"type": "string", flag: true}, map[string]any{"type": "integer"}}}
		}, "/schema/properties/meta/oneOf/0/" + flag},
		{"array items", func(s map[string]any) {
			props(s)["tags"] = map[string]any{"type": "array", "items": map[string]any{"type": "string", flag: true}}
		}, "/schema/properties/tags/items/" + flag},
		{"root", func(s map[string]any) { s[flag] = true }, "/schema/" + flag},
		{"false is still misplaced", func(s map[string]any) {
			props(s)["meta"] = map[string]any{"allOf": []any{map[string]any{"type": "string", flag: false}}}
		}, "/schema/properties/meta/allOf/0/" + flag},
		{"ahead of another problem", func(s map[string]any) {
			props(s)["meta"] = nested()
			s["additionalProperties"] = true
		}, "/schema/properties/meta/properties/inner/" + flag},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := baseSchema()
			tt.mut(s)
			_, err := metadata.Compile(baseDefinition(t, s))
			if paths := refusal(t, err, metadata.RefusalFilterableNotTopLevel); !slices.Equal(paths, []string{tt.want}) {
				t.Fatalf("paths = %q, want [%q]", paths, tt.want)
			}
		})
	}
}

// TestCompileRefusesFilterableUnsupportedType: a top-level flag whose value
// type is not one of the four is a plain invalid definition (422
// invalid_definition) at the flag.
func TestCompileRefusesFilterableUnsupportedType(t *testing.T) {
	t.Parallel()
	tests := map[string]map[string]any{
		"object":            {"type": "object"},
		"null":              {"type": "null"},
		"no type":           {"minLength": 1},
		"type list":         {"type": []any{"string", "null"}},
		"mixed enum":        {"enum": []any{"a", 1}},
		"array of objects":  {"type": "array", "items": map[string]any{"type": "object"}},
		"array of arrays":   {"type": "array", "items": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
		"array, no items":   {"type": "array"},
		"array prefixItems": {"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}, "items": map[string]any{"type": "string"}},
		"type through $ref": {"$ref": "#/$defs/s"},
		"string with anyOf": {"type": "string", "anyOf": []any{map[string]any{"minLength": 1}}},
		"not a boolean":     {"type": "string", flag: "yes"},
	}
	for name, p := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := baseSchema()
			s["$defs"] = map[string]any{"s": map[string]any{"type": "string"}}
			if _, set := p[flag]; !set {
				p[flag] = true
			}
			props(s)["f"] = p
			_, err := metadata.Compile(baseDefinition(t, s))
			wantPath(t, err, "/schema/properties/f/"+flag)
		})
	}
}

// TestCompileRefusesSearchable: x-nava-searchable is reserved anywhere, with
// any value, until RFC-016 (422 searchable_not_available), and it wins over
// a misplaced filterable flag.
func TestCompileRefusesSearchable(t *testing.T) {
	t.Parallel()
	const kw = metadata.SearchableKeyword
	tests := []struct {
		name string
		mut  func(s map[string]any)
		want []string
	}{
		{"top-level property", func(s map[string]any) { prop(s, "title")[kw] = true }, []string{"/schema/properties/title/" + kw}},
		{"false", func(s map[string]any) { prop(s, "title")[kw] = false }, []string{"/schema/properties/title/" + kw}},
		{"root", func(s map[string]any) { s[kw] = true }, []string{"/schema/" + kw}},
		{"in $defs", func(s map[string]any) {
			s["$defs"] = map[string]any{"x": map[string]any{"type": "string", kw: true}}
		}, []string{"/schema/$defs/x/" + kw}},
		{"with a nested filterable", func(s map[string]any) {
			prop(s, "title")[kw] = true
			props(s)["meta"] = map[string]any{"allOf": []any{map[string]any{flag: true}}}
		}, []string{"/schema/properties/title/" + kw}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := baseSchema()
			tt.mut(s)
			_, err := metadata.Compile(baseDefinition(t, s))
			if paths := refusal(t, err, metadata.RefusalSearchableNotAvailable); !slices.Equal(paths, tt.want) {
				t.Fatalf("paths = %q, want %q", paths, tt.want)
			}
		})
	}
}

// TestClassifyFilterable: the flag is configuration, so adding or removing
// it is Compatible; a change to the value type of a property the previous
// version flags is refused; the same change unflagged keeps its existing
// class (P7-03 scope 3).
func TestClassifyFilterable(t *testing.T) {
	t.Parallel()
	type mut = func(s map[string]any)
	none := func(map[string]any) {}
	flagPriority := func(s map[string]any) { prop(s, "priority")[flag] = true }
	tests := []struct {
		name       string
		prev, next mut
		want       metadata.Compat
		paths      []string
		refused    bool
	}{
		{"add flag", none, flagPriority, metadata.Compatible, nil, false},
		{"remove flag", flagPriority, none, metadata.Compatible, nil, false},
		{"flag set to false", flagPriority, func(s map[string]any) { prop(s, "priority")[flag] = false }, metadata.Compatible, nil, false},
		{"add flagged optional property", none, func(s map[string]any) {
			props(s)["due_at"] = map[string]any{"type": "string", "format": "date-time", flag: true}
		}, metadata.Compatible, nil, false},
		{"flag with an annotation change", none, func(s map[string]any) {
			flagPriority(s)
			prop(s, "priority")["title"] = "Priority"
		}, metadata.Compatible, nil, false},
		{"flag added alongside a breaking change stays breaking", none, func(s map[string]any) {
			flagPriority(s)
			prop(s, "priority")["enum"] = []any{1, 2}
		}, metadata.Breaking, []string{"/schema/properties/priority/enum"}, false},

		{"type change of flagged field", flagPriority, func(s map[string]any) {
			flagPriority(s)
			prop(s, "priority")["type"] = "string"
			prop(s, "priority")["enum"] = []any{"1", "2", "3"}
		}, metadata.Breaking, []string{"/schema/properties/priority"}, true},
		{"type change while removing the flag", flagPriority, func(s map[string]any) {
			prop(s, "priority")["type"] = "string"
			prop(s, "priority")["enum"] = []any{"1", "2", "3"}
		}, metadata.Breaking, []string{"/schema/properties/priority"}, true},
		{"text to ts through format", func(s map[string]any) { prop(s, "title")[flag] = true }, func(s map[string]any) {
			prop(s, "title")[flag] = true
			prop(s, "title")["format"] = "date-time"
		}, metadata.Breaking, []string{"/schema/properties/title"}, true},
		{"to an unsupported type", flagPriority, func(s map[string]any) {
			prop(s, "priority")["type"] = "object"
			delete(prop(s, "priority"), "enum")
		}, metadata.Breaking, []string{"/schema/properties/priority"}, true},

		// The same changes on an unflagged field keep the existing class.
		{"type change of unflagged field", none, func(s map[string]any) {
			prop(s, "priority")["type"] = "string"
			prop(s, "priority")["enum"] = []any{"1", "2", "3"}
		}, metadata.Breaking, []string{"/schema/properties/priority/enum", "/schema/properties/priority/type"}, false},
		{"integer to number on a flagged field keeps num", flagPriority, func(s map[string]any) {
			flagPriority(s)
			prop(s, "priority")["type"] = "number"
		}, metadata.Breaking, []string{"/schema/properties/priority/type"}, false},
		{"removing a flagged field", flagPriority, func(s map[string]any) {
			delete(props(s), "priority")
		}, metadata.Breaking, []string{"/schema/properties/priority"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ps, ns := baseSchema(), baseSchema()
			tt.prev(ps)
			tt.next(ns)
			prev := mustCompile(t, baseDefinition(t, ps))
			next := mustCompile(t, baseDefinition(t, ns))
			class, paths, err := metadata.Classify(prev, next)
			if tt.refused {
				if got := refusal(t, err, metadata.RefusalFilterableTypeChange); !slices.Equal(got, tt.paths) {
					t.Fatalf("paths = %q, want %q", got, tt.paths)
				}
				return
			}
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			if class != tt.want {
				t.Fatalf("class = %v (%q), want %v", class, paths, tt.want)
			}
			if !slices.Equal(paths, tt.paths) {
				t.Fatalf("paths = %q, want %q", paths, tt.paths)
			}
		})
	}
}
