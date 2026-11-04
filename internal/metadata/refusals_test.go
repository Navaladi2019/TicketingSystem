package metadata_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

func TestValidateReportsEachKindOfFailure(t *testing.T) {
	t.Parallel()
	s := baseSchema()
	s["dependentRequired"] = map[string]any{"priority": []any{"code"}}
	c := mustCompile(t, baseDefinition(t, s))

	got := c.Validate(map[string]any{"title": "t", "state": "open", "priority": 1})
	want := []metadata.FieldError{
		{Path: "/code", Code: "dependentRequired"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Validate = %v, want %v", got, want)
	}

	// A value encoding/json would never produce is a type error, not a panic.
	got = c.Validate(map[string]any{"title": make(chan int), "state": "open"})
	if len(got) != 1 || got[0].Code != "type" {
		t.Fatalf("Validate(chan) = %v, want one type error", got)
	}

	// nil data is an empty record: both required properties are missing.
	got = c.Validate(nil)
	if len(got) != 2 || got[0].Code != "required" || got[1].Code != "required" {
		t.Fatalf("Validate(nil) = %v", got)
	}
}

func TestCompileRefusesUnusableSchemaDocuments(t *testing.T) {
	t.Parallel()
	big := `{"type":"object","description":"` + strings.Repeat("x", metadata.MaxSchemaBytes) + `"}`
	for name, raw := range map[string]string{
		"empty":     ``,
		"too large": big,
		"null":      `null`,
		"not json":  `{"type":`,
		"array":     `[]`,
	} {
		d := baseDefinition(t, baseSchema())
		d.Schema = json.RawMessage(raw)
		_, err := metadata.Compile(d)
		var ie *metadata.ErrInvalidDefinition
		if !errors.As(err, &ie) {
			t.Errorf("%s: err = %v, want *ErrInvalidDefinition", name, err)
		}
	}
}

func TestTemplateRefusals(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"", "Bad Key", "no_such_template", "../x"} {
		if _, err := metadata.Template(key); !errors.Is(err, metadata.ErrNoTemplate) {
			t.Errorf("Template(%q) err = %v, want ErrNoTemplate", key, err)
		}
	}
}

func TestStatePropertyOfACompiledDefinition(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	if got := c.StateProperty(); got != "state" {
		t.Fatalf("StateProperty = %q, want state", got)
	}
	var zero metadata.Compiled
	if got := zero.StateProperty(); got != "" {
		t.Fatalf("StateProperty of an empty value = %q", got)
	}
}

func TestCompileRefusesBadStateModelsAndShapes(t *testing.T) {
	t.Parallel()
	tests := map[string]func(d *metadata.Definition){
		"no states":        func(d *metadata.Definition) { d.States = metadata.StateModel{} },
		"empty state name": func(d *metadata.Definition) { d.States.States = []string{"open", ""} },
		"duplicate state":  func(d *metadata.Definition) { d.States.States = []string{"open", "open", "closed"} },
		"initial unknown":  func(d *metadata.Definition) { d.States.Initial = "ghost" },
		"transition from ghost": func(d *metadata.Definition) {
			d.States.Transitions = []metadata.Transition{{From: "ghost", To: "closed"}}
		},
		"transition to ghost": func(d *metadata.Definition) {
			d.States.Transitions = []metadata.Transition{{From: "open", To: "ghost"}}
		},
		"duplicate transition": func(d *metadata.Definition) {
			d.States.Transitions = []metadata.Transition{{From: "open", To: "closed"}, {From: "open", To: "closed"}}
		},
		"trailing data":    func(d *metadata.Definition) { d.Schema = append(d.Schema, []byte(` {}`)...) },
		"properties array": func(d *metadata.Definition) { d.Schema = json.RawMessage(`{"type":"object","properties":[]}`) },
		"property not an object": func(d *metadata.Definition) {
			d.Schema = json.RawMessage(`{"type":"object","properties":{"state":true}}`)
		},
	}
	for name, mutate := range tests {
		d := baseDefinition(t, baseSchema())
		mutate(&d)
		_, err := metadata.Compile(d)
		var ie *metadata.ErrInvalidDefinition
		if !errors.As(err, &ie) {
			t.Errorf("%s: err = %v, want *ErrInvalidDefinition", name, err)
		}
	}
}
