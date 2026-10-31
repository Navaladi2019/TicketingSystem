package metadata_test

import (
	"encoding/json"
	"maps"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

func TestValidateOneErrorPerFailure(t *testing.T) {
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	errs := c.Validate(map[string]any{
		// title missing (required)
		"state":     "pending",    // enum
		"priority":  "high",       // type
		"code":      "c",          // valid
		"extra":     true,         // additionalProperties
		"other":     1,            // additionalProperties
		"requester": "not-a-uuid", // format
	})
	want := []metadata.FieldError{
		{Path: "/extra", Code: "additionalProperties"},
		{Path: "/other", Code: "additionalProperties"},
		{Path: "/priority", Code: "type"},
		{Path: "/requester", Code: "format"},
		{Path: "/state", Code: "enum"},
		{Path: "/title", Code: "required"},
	}
	if !reflect.DeepEqual(errs, want) {
		t.Fatalf("Validate =\n %v\nwant\n %v", errs, want)
	}
}

func TestValidateNestedPathIsJSONPointer(t *testing.T) {
	s := baseSchema()
	props(s)["a/b"] = map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}
	c := mustCompile(t, baseDefinition(t, s))
	errs := c.Validate(map[string]any{"title": "t", "state": "open", "a/b": []any{1, "x"}})
	want := []metadata.FieldError{{Path: "/a~1b/1", Code: "type"}}
	if !reflect.DeepEqual(errs, want) {
		t.Fatalf("Validate = %v, want %v", errs, want)
	}
}

func TestValidateValidRecord(t *testing.T) {
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	var rec map[string]any
	if err := json.Unmarshal([]byte(`{"title":"t","state":"open","priority":2}`), &rec); err != nil {
		t.Fatal(err)
	}
	if errs := c.Validate(rec); errs != nil {
		t.Fatalf("Validate = %v, want nil", errs)
	}
}

func TestUniqueFields(t *testing.T) {
	s := baseSchema()
	prop(s, "title")[metadata.UniqueKeyword] = true
	prop(s, "priority")[metadata.UniqueKeyword] = false
	c := mustCompile(t, baseDefinition(t, s))
	if got, want := c.UniqueFields(), []string{"code", "title"}; !slices.Equal(got, want) {
		t.Fatalf("UniqueFields = %q, want %q", got, want)
	}
	c.UniqueFields()[0] = "mutated"
	if c.UniqueFields()[0] != "code" {
		t.Fatal("UniqueFields returned internal state")
	}
}

func TestApplyDefaults(t *testing.T) {
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	in := map[string]any{"title": "t"}
	out := c.ApplyDefaults(in)
	if out["state"] != "open" {
		t.Fatalf("state = %v, want open", out["state"])
	}
	if _, ok := in["state"]; ok {
		t.Fatal("ApplyDefaults changed its input")
	}
	// A present key, even null, is never overwritten.
	out = c.ApplyDefaults(map[string]any{"state": nil})
	if v, ok := out["state"]; !ok || v != nil {
		t.Fatalf("state = %v, want the caller's nil", v)
	}
	// Each call returns a fresh default value.
	s := baseSchema()
	props(s)["tags"] = map[string]any{"type": "array", "default": []any{"a"}}
	c = mustCompile(t, baseDefinition(t, s))
	first := c.ApplyDefaults(nil)
	first["tags"].([]any)[0] = "mutated"
	if got := c.ApplyDefaults(nil)["tags"].([]any)[0]; got != "a" {
		t.Fatalf("default shared between calls: %v", got)
	}
}

// TestApplyDefaultsProperties: for random records, ApplyDefaults never
// changes a key that was present, never changes its input, and is
// idempotent.
func TestApplyDefaultsProperties(t *testing.T) {
	s := baseSchema()
	prop(s, "priority")["default"] = 2
	props(s)["tags"] = map[string]any{"type": "array", "default": []any{"a", map[string]any{"k": 1.5}}}
	c := mustCompile(t, baseDefinition(t, s))

	keys := []string{"title", "state", "priority", "code", "requester", "tags", "unknown"}
	values := []any{nil, "x", 3.0, true, []any{"v"}, map[string]any{"n": "m"}, json.Number("7")}
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 2000 {
		in := map[string]any{}
		for _, k := range keys {
			if rng.IntN(2) == 0 {
				in[k] = values[rng.IntN(len(values))]
			}
		}
		snapshot := maps.Clone(in)

		out := c.ApplyDefaults(in)
		if !reflect.DeepEqual(in, snapshot) {
			t.Fatalf("case %d: input changed: %v -> %v", i, snapshot, in)
		}
		for k, v := range in {
			if !reflect.DeepEqual(out[k], v) {
				t.Fatalf("case %d: present key %q changed: %v -> %v", i, k, v, out[k])
			}
		}
		for _, k := range []string{"state", "priority", "tags"} {
			if _, ok := out[k]; !ok {
				t.Fatalf("case %d: default %q not applied", i, k)
			}
		}
		if again := c.ApplyDefaults(out); !reflect.DeepEqual(again, out) {
			t.Fatalf("case %d: not idempotent: %v -> %v", i, out, again)
		}
	}
}
