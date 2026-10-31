package metadata_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// baseSchema is a small valid entity schema; tests copy and change it.
func baseSchema() map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"title", "state"},
		"properties": map[string]any{
			"title":    map[string]any{"type": "string", "minLength": 1, "maxLength": 80, "pattern": "^[^<>]*$"},
			"state":    map[string]any{"type": "string", "enum": []any{"open", "closed"}, "default": "open"},
			"priority": map[string]any{"type": "integer", "enum": []any{1, 2, 3}},
			"code":     map[string]any{"type": "string", metadata.UniqueKeyword: true},
			"requester": map[string]any{
				"type": "string", "format": "uuid",
			},
		},
	}
}

func baseDefinition(t *testing.T, schema map[string]any) metadata.Definition {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	return metadata.Definition{
		Key:          "ticket",
		Schema:       raw,
		NumberPrefix: "TKT",
		FixedColumns: map[string]string{"state": metadata.ColumnState, "requester": metadata.ColumnRequesterID},
		States: metadata.StateModel{
			Initial:     "open",
			States:      []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}},
		},
	}
}

// props returns the properties map of a schema built by baseSchema.
func props(s map[string]any) map[string]any { return s["properties"].(map[string]any) }

// prop returns one property schema of a schema built by baseSchema.
func prop(s map[string]any, name string) map[string]any { return props(s)[name].(map[string]any) }

func mustCompile(t *testing.T, d metadata.Definition) *metadata.Compiled {
	t.Helper()
	c, err := metadata.Compile(d)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return c
}

// invalidPaths asserts err is *ErrInvalidDefinition and returns its paths.
func invalidPaths(t *testing.T, err error) []string {
	t.Helper()
	var inv *metadata.ErrInvalidDefinition
	if !errors.As(err, &inv) {
		t.Fatalf("err = %v, want *ErrInvalidDefinition", err)
	}
	if len(inv.Paths) == 0 {
		t.Fatal("ErrInvalidDefinition with no paths")
	}
	return inv.Paths
}

func wantPath(t *testing.T, err error, want string) {
	t.Helper()
	if paths := invalidPaths(t, err); !slices.Contains(paths, want) {
		t.Fatalf("paths = %q, want %q among them", paths, want)
	}
}
