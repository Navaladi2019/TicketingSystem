package metadata_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

func TestCompileAcceptsBaseDefinition(t *testing.T) {
	mustCompile(t, baseDefinition(t, baseSchema()))
}

func TestTemplatesCompile(t *testing.T) {
	keys := metadata.TemplateKeys()
	for _, want := range []string{"change", "incident", "request"} {
		if !contains(keys, want) {
			t.Fatalf("TemplateKeys() = %q, missing %q", keys, want)
		}
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			d, err := metadata.Template(key)
			if err != nil {
				t.Fatal(err)
			}
			if d.Key != key {
				t.Fatalf("template %s has key %q", key, d.Key)
			}
			c := mustCompile(t, d)
			// A record made of defaults plus the required non-default fields is
			// valid, so the template is usable as seeded.
			rec := c.ApplyDefaults(map[string]any{
				"short_description": "printer on fire",
				"requester":         "4b0f5f4e-7c43-4c1b-9d3f-7d7e1b2a9c10",
				"requested_for":     "4b0f5f4e-7c43-4c1b-9d3f-7d7e1b2a9c10",
			})
			if key == "request" {
				delete(rec, "requester")
			} else {
				delete(rec, "requested_for")
			}
			if errs := c.Validate(rec); errs != nil {
				t.Fatalf("Validate(seeded record) = %v", errs)
			}
		})
	}
	if _, err := metadata.Template("nope"); !errors.Is(err, metadata.ErrNoTemplate) {
		t.Fatalf("Template(nope) err = %v, want ErrNoTemplate", err)
	}
	if _, err := metadata.Template("../templates/incident"); !errors.Is(err, metadata.ErrNoTemplate) {
		t.Fatalf("Template(path) err = %v, want ErrNoTemplate", err)
	}
}

func contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

// TestCompileRefusesExternalRefs: no $ref may leave the document (TICK-363).
// Compile refuses these before the compiler runs, and the compiler's loader
// refuses every URL anyway (TestCompilerLoaderDeniesEverything), so this
// test makes no file or network call.
func TestCompileRefusesExternalRefs(t *testing.T) {
	refs := map[string]string{
		"absolute path": "/etc/passwd",
		"relative":      "x.json",
		"file url":      "file:///etc/passwd",
		"http url":      "http://169.254.169.254/latest/meta-data",
		"urn":           "urn:x",
		"nested ref":    "", // set below
	}
	for name, ref := range refs {
		t.Run(name, func(t *testing.T) {
			s := baseSchema()
			want := "/schema/properties/extra/$ref"
			if name == "nested ref" {
				props(s)["extra"] = map[string]any{"type": "object", "properties": map[string]any{
					"inner": map[string]any{"allOf": []any{map[string]any{"$ref": "file:///etc/passwd"}}},
				}}
				want = "/schema/properties/extra/properties/inner/allOf/0/$ref"
			} else {
				props(s)["extra"] = map[string]any{"$ref": ref}
			}
			_, err := metadata.Compile(baseDefinition(t, s))
			wantPath(t, err, want)
			for _, p := range invalidPaths(t, err) {
				if !strings.HasPrefix(p, "/") || strings.ContainsAny(p, " :") {
					t.Fatalf("path %q is not a bare JSON pointer (OS or library text leaked)", p)
				}
			}
			if strings.Contains(err.Error(), "passwd") || strings.Contains(err.Error(), "no such file") {
				t.Fatalf("error text leaks the reference: %v", err)
			}
		})
	}
}

func TestCompileAllowsLocalRefs(t *testing.T) {
	s := baseSchema()
	s["$defs"] = map[string]any{"short": map[string]any{"type": "string", "maxLength": 10}}
	props(s)["tag"] = map[string]any{"$ref": "#/$defs/short", "default": "x"}
	c := mustCompile(t, baseDefinition(t, s))
	errs := c.Validate(map[string]any{"title": "t", "state": "open", "tag": strings.Repeat("y", 11)})
	if len(errs) != 1 || errs[0] != (metadata.FieldError{Path: "/tag", Code: "maxLength"}) {
		t.Fatalf("Validate = %v, want /tag maxLength", errs)
	}
}

// TestCompileCrossTenantRegistry: tenant A's $id must not satisfy tenant B's
// $ref. Compile builds a new compiler per call, so nothing A registered is
// visible to B (TestSharedCompilerWouldLeak shows the hazard).
func TestCompileCrossTenantRegistry(t *testing.T) {
	a := baseSchema()
	a["$id"] = "urn:x"
	mustCompile(t, baseDefinition(t, a))

	b := baseSchema()
	props(b)["leak"] = map[string]any{"$ref": "urn:x"}
	_, err := metadata.Compile(baseDefinition(t, b))
	wantPath(t, err, "/schema/properties/leak/$ref")
}

func TestCompileRefuses(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(d *metadata.Definition, s map[string]any)
		raw    string // used instead of the schema when set
		want   string
	}{
		{
			name:   "(a) default does not satisfy its schema",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "priority")["default"] = 9 },
			want:   "/schema/properties/priority/default",
		},
		{
			name:   "(a) default of the wrong type",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "title")["default"] = 5 },
			want:   "/schema/properties/title/default",
		},
		{
			name:   "(b) fixed id column is not a uuid string",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "requester")["type"] = "integer" },
			want:   "/schema/properties/requester/type",
		},
		{
			name:   "(b) fixed id column without format uuid",
			mutate: func(_ *metadata.Definition, s map[string]any) { delete(prop(s, "requester"), "format") },
			want:   "/schema/properties/requester/format",
		},
		{
			name: "(b) state enum is not the state set",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				prop(s, "state")["enum"] = []any{"open", "closed", "archived"}
			},
			want: "/schema/properties/state/enum",
		},
		{
			name:   "(b) state column is not a string",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "state")["type"] = "integer" },
			want:   "/schema/properties/state/type",
		},
		{
			name: "(d) schema over MaxSchemaBytes",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				prop(s, "title")["description"] = strings.Repeat("x", metadata.MaxSchemaBytes)
			},
			want: "/schema",
		},
		{
			name: "(d) schema deeper than MaxSchemaDepth",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				var deep any = map[string]any{"type": "string"}
				for range metadata.MaxSchemaDepth {
					deep = map[string]any{"not": deep}
				}
				props(s)["deep"] = deep
			},
			want: "", // any path at the cap
		},
		{
			name: "(e) duplicate key",
			raw:  `{"type":"object","properties":{"state":{"type":"string","enum":["open","closed"]},"title":{"type":"string","type":"integer"},"requester":{"type":"string","format":"uuid"}}}`,
			want: "/schema/properties/title/type",
		},
		{
			name: "(e) duplicate key at the root",
			raw:  `{"type":"object","type":"object","properties":{"state":{"type":"string","enum":["open","closed"]},"requester":{"type":"string","format":"uuid"}}}`,
			want: "/schema/type",
		},
		{
			name: "nested default",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["address"] = map[string]any{"type": "object", "properties": map[string]any{
					"city": map[string]any{"type": "string", "default": "Paris"},
				}}
			},
			want: "/schema/properties/address/properties/city/default",
		},
		{
			name:   "root default",
			mutate: func(_ *metadata.Definition, s map[string]any) { s["default"] = map[string]any{} },
			want:   "/schema/default",
		},
		{
			name: "nested x-nava-unique",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["address"] = map[string]any{"type": "object", "properties": map[string]any{
					"city": map[string]any{"type": "string", metadata.UniqueKeyword: true},
				}}
			},
			want: "/schema/properties/address/properties/city/x-nava-unique",
		},
		{
			name:   "non-boolean x-nava-unique",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "code")[metadata.UniqueKeyword] = "yes" },
			want:   "/schema/properties/code/x-nava-unique",
		},
		{
			name: "other draft",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				s["$schema"] = "http://json-schema.org/draft-07/schema#"
			},
			want: "/schema/$schema",
		},
		{
			name:   "root is not an object schema",
			mutate: func(_ *metadata.Definition, s map[string]any) { s["type"] = "array" },
			want:   "/schema/type",
		},
		{
			name:   "metaschema violation",
			mutate: func(_ *metadata.Definition, s map[string]any) { prop(s, "title")["minLength"] = -1 },
			want:   "/schema/properties/title/minLength",
		},
		{
			name:   "open root: no additionalProperties",
			mutate: func(_ *metadata.Definition, s map[string]any) { delete(s, "additionalProperties") },
			want:   "/schema/additionalProperties",
		},
		{
			name: "open root: additionalProperties is a schema",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				s["additionalProperties"] = map[string]any{"type": "string"}
			},
			want: "/schema/additionalProperties",
		},
		{
			name: "root patternProperties",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				s["patternProperties"] = map[string]any{"^x_": map[string]any{"type": "string"}}
			},
			want: "/schema/patternProperties",
		},
		{
			name: "local $ref into an unknown keyword",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				s["definitions"] = map[string]any{"x": map[string]any{"type": "string", "default": "smuggled"}}
				props(s)["extra"] = map[string]any{"$ref": "#/definitions/x"}
			},
			want: "/schema/properties/extra/$ref",
		},
		{
			name: "local $ref to a non-schema location",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["extra"] = map[string]any{"$ref": "#/properties"}
			},
			want: "/schema/properties/extra/$ref",
		},
		{
			name: "local $ref to an undeclared anchor",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["extra"] = map[string]any{"$ref": "#nowhere"}
			},
			want: "/schema/properties/extra/$ref",
		},
		{
			name: "local $ref with a bad percent-escape",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["extra"] = map[string]any{"$ref": "#/%zz"}
			},
			want: "/schema/properties/extra/$ref",
		},
		{
			name: "embedded $id",
			mutate: func(_ *metadata.Definition, s map[string]any) {
				props(s)["extra"] = map[string]any{"$id": "urn:y", "type": "string"}
			},
			want: "/schema/properties/extra/$id",
		},
		{
			name: "two properties on one column",
			mutate: func(d *metadata.Definition, s map[string]any) {
				props(s)["requester2"] = map[string]any{"type": "string", "format": "uuid"}
				d.FixedColumns["requester2"] = metadata.ColumnRequesterID
			},
			want: "/fixed_columns/requester2", // names are checked in sorted order
		},
		{
			name:   "bad key",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.Key = "Ticket" },
			want:   "/key",
		},
		{
			name:   "bad number prefix",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.NumberPrefix = "T" },
			want:   "/number_prefix",
		},
		{
			name:   "unknown fixed column",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.FixedColumns["title"] = "tenant_id" },
			want:   "/fixed_columns/title",
		},
		{
			name:   "fixed column on a missing property",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.FixedColumns["ghost"] = metadata.ColumnAssigneeID },
			want:   "/fixed_columns/ghost",
		},
		{
			name:   "no state column",
			mutate: func(d *metadata.Definition, _ map[string]any) { delete(d.FixedColumns, "state") },
			want:   "/fixed_columns",
		},
		{
			name:   "initial not a state",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.States.Initial = "new" },
			want:   "/states/initial",
		},
		{
			name: "transition to an unknown state",
			mutate: func(d *metadata.Definition, _ map[string]any) {
				d.States.Transitions = append(d.States.Transitions, metadata.Transition{From: "closed", To: "gone"})
			},
			want: "/states/transitions/1/to",
		},
		{
			name:   "unreachable state",
			mutate: func(d *metadata.Definition, _ map[string]any) { d.States.Transitions = nil },
			want:   "/states/states/1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseSchema()
			d := baseDefinition(t, s)
			if tt.mutate != nil {
				tt.mutate(&d, s)
			}
			if tt.raw != "" {
				d.Schema = json.RawMessage(tt.raw)
			} else {
				d.Schema, _ = json.Marshal(s)
			}
			_, err := metadata.Compile(d)
			if tt.want == "" {
				invalidPaths(t, err)
				return
			}
			wantPath(t, err, tt.want)
		})
	}
}

func TestCompileCapsAreInclusive(t *testing.T) {
	// The depth cap is exactly MaxSchemaDepth: one level less compiles.
	s := baseSchema()
	var deep any = map[string]any{"type": "string"}
	// root(1) > properties(2) > deep(3) > not... ; the innermost object sits
	// at MaxSchemaDepth.
	for range metadata.MaxSchemaDepth - 3 {
		deep = map[string]any{"not": deep}
	}
	props(s)["deep"] = deep
	mustCompile(t, baseDefinition(t, s))
}

func TestCompileFormatAssertion(t *testing.T) {
	// (c): format is asserted, so a bad uuid fails Validate.
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	errs := c.Validate(map[string]any{"title": "t", "state": "open", "requester": "not-a-uuid"})
	if len(errs) != 1 || errs[0] != (metadata.FieldError{Path: "/requester", Code: "format"}) {
		t.Fatalf("Validate = %v, want /requester format", errs)
	}
}

func TestErrInvalidDefinitionPathsSorted(t *testing.T) {
	d := baseDefinition(t, baseSchema())
	d.Key, d.NumberPrefix, d.States.Initial = "", "", "zzz"
	paths := invalidPaths(t, func() error { _, err := metadata.Compile(d); return err }())
	want := []string{"/key", "/number_prefix", "/states/initial"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths = %q, want %q", paths, want)
	}
}

func TestCompileAllowsWalkedLocalRefs(t *testing.T) {
	s := baseSchema()
	s["$defs"] = map[string]any{
		"a b":  map[string]any{"type": "string"},
		"name": map[string]any{"$anchor": "short", "type": "string", "maxLength": 3},
	}
	props(s)["spaced"] = map[string]any{"$ref": "#/$defs/a%20b"}
	props(s)["anchored"] = map[string]any{"$ref": "#short"}
	props(s)["self"] = map[string]any{"$ref": "#/properties/title"}
	c := mustCompile(t, baseDefinition(t, s))
	errs := c.Validate(map[string]any{"title": "t", "state": "open", "anchored": "toolong", "spaced": 1})
	want := []metadata.FieldError{{Path: "/anchored", Code: "maxLength"}, {Path: "/spaced", Code: "type"}}
	if len(errs) != 2 || errs[0] != want[0] || errs[1] != want[1] {
		t.Fatalf("Validate = %v, want %v", errs, want)
	}
}

// TestCompileFixedColumnClashIsDeterministic: the flagged name does not
// depend on map iteration order.
func TestCompileFixedColumnClashIsDeterministic(t *testing.T) {
	s := baseSchema()
	for _, n := range []string{"a_req", "z_req", "m_req"} {
		props(s)[n] = map[string]any{"type": "string", "format": "uuid"}
	}
	for range 50 {
		d := baseDefinition(t, s)
		for _, n := range []string{"a_req", "z_req", "m_req"} {
			d.FixedColumns[n] = metadata.ColumnAssigneeID
		}
		_, err := metadata.Compile(d)
		got := strings.Join(invalidPaths(t, err), ",")
		if got != "/fixed_columns/m_req,/fixed_columns/z_req" {
			t.Fatalf("paths = %s, want the clash after a_req: m_req and z_req", got)
		}
	}
}
