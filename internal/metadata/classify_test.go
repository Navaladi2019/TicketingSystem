package metadata_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// TestClassify covers every compatibility class (D-3-5). Fail closed: any
// change the classifier cannot prove compatible is Breaking, including
// widening maxLength and dropping a pattern.
func TestClassify(t *testing.T) {
	type mut = func(d *metadata.Definition, s map[string]any)
	tests := []struct {
		name  string
		next  mut
		want  metadata.Compat
		paths []string // checked when set
	}{
		{"no change", func(*metadata.Definition, map[string]any) {}, metadata.Compatible, nil},

		// Compatible
		{"add optional property", func(_ *metadata.Definition, s map[string]any) {
			props(s)["notes"] = map[string]any{"type": "string"}
		}, metadata.Compatible, nil},
		{"add optional property with default", func(_ *metadata.Definition, s map[string]any) {
			props(s)["notes"] = map[string]any{"type": "string", "default": "-"}
		}, metadata.Compatible, nil},
		{"widen enum", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "priority")["enum"] = []any{1, 2, 3, 4}
		}, metadata.Compatible, nil},
		{"add state and transition (widens the state enum)", func(d *metadata.Definition, s map[string]any) {
			d.States.States = append(d.States.States, "reopened")
			d.States.Transitions = append(d.States.Transitions, metadata.Transition{From: "closed", To: "reopened"})
			prop(s, "state")["enum"] = []any{"open", "closed", "reopened"}
		}, metadata.Compatible, nil},
		{"add transition", func(d *metadata.Definition, _ map[string]any) {
			d.States.Transitions = append(d.States.Transitions, metadata.Transition{From: "closed", To: "open"})
		}, metadata.Compatible, nil},
		{"change title and description", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "title")["title"] = "Summary"
			prop(s, "title")["description"] = "One line"
		}, metadata.Compatible, nil},

		// NeedsBackfill
		{"add required property with default", func(_ *metadata.Definition, s map[string]any) {
			props(s)["impact"] = map[string]any{"type": "integer", "default": 3}
			s["required"] = []any{"title", "state", "impact"}
		}, metadata.NeedsBackfill, []string{"/schema/properties/impact"}},
		{"add default to existing property", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "priority")["default"] = 2
		}, metadata.NeedsBackfill, []string{"/schema/properties/priority/default"}},

		// Breaking
		{"change type", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "priority")["type"] = "number"
		}, metadata.Breaking, []string{"/schema/properties/priority/type"}},
		{"narrow enum", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "priority")["enum"] = []any{1, 2}
		}, metadata.Breaking, []string{"/schema/properties/priority/enum"}},
		{"add enum", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "title")["enum"] = []any{"a"}
		}, metadata.Breaking, []string{"/schema/properties/title/enum"}},
		{"add required property without default", func(_ *metadata.Definition, s map[string]any) {
			props(s)["impact"] = map[string]any{"type": "integer"}
			s["required"] = []any{"title", "state", "impact"}
		}, metadata.Breaking, []string{"/schema/properties/impact"}},
		{"remove property", func(_ *metadata.Definition, s map[string]any) {
			delete(props(s), "priority")
		}, metadata.Breaking, []string{"/schema/properties/priority"}},
		{"change fixed columns", func(d *metadata.Definition, s map[string]any) {
			props(s)["owner"] = map[string]any{"type": "string", "format": "uuid"}
			d.FixedColumns["owner"] = metadata.ColumnAssigneeID
		}, metadata.Breaking, []string{"/fixed_columns/owner"}},
		{"change number prefix", func(d *metadata.Definition, _ map[string]any) {
			d.NumberPrefix = "TIX"
		}, metadata.Breaking, []string{"/number_prefix"}},
		{"change existing default", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "state")["default"] = "closed"
		}, metadata.Breaking, []string{"/schema/properties/state/default"}},
		{"remove existing default", func(_ *metadata.Definition, s map[string]any) {
			delete(prop(s, "state"), "default")
		}, metadata.Breaking, []string{"/schema/properties/state/default"}},
		{"add x-nava-unique", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "title")[metadata.UniqueKeyword] = true
		}, metadata.Breaking, []string{"/schema/properties/title/x-nava-unique"}},
		{"remove x-nava-unique", func(_ *metadata.Definition, s map[string]any) {
			delete(prop(s, "code"), metadata.UniqueKeyword)
		}, metadata.Breaking, []string{"/schema/properties/code/x-nava-unique"}},
		{"add new optional property marked unique", func(_ *metadata.Definition, s map[string]any) {
			props(s)["serial"] = map[string]any{"type": "string", metadata.UniqueKeyword: true}
		}, metadata.Breaking, []string{"/schema/properties/serial/x-nava-unique"}},
		{"remove state", func(d *metadata.Definition, s map[string]any) {
			d.States.States = []string{"open"}
			d.States.Transitions = nil
			prop(s, "state")["enum"] = []any{"open"}
		}, metadata.Breaking, nil},
		{"remove transition", func(d *metadata.Definition, _ map[string]any) {
			d.States.Transitions = append(d.States.Transitions, metadata.Transition{From: "closed", To: "open"})
		}, metadata.Compatible, nil}, // the reverse direction is checked below
		{"widen maxLength (fail closed)", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "title")["maxLength"] = 200
		}, metadata.Breaking, []string{"/schema/properties/title/maxLength"}},
		{"drop pattern (fail closed)", func(_ *metadata.Definition, s map[string]any) {
			delete(prop(s, "title"), "pattern")
		}, metadata.Breaking, []string{"/schema/properties/title/pattern"}},
		{"unknown keyword change", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "title")["x-widget"] = "textarea"
		}, metadata.Breaking, []string{"/schema/properties/title/x-widget"}},
		{"root keyword change", func(_ *metadata.Definition, s map[string]any) {
			s["minProperties"] = 1
		}, metadata.Breaking, []string{"/schema/minProperties"}},
		{"make existing property required", func(_ *metadata.Definition, s map[string]any) {
			s["required"] = []any{"title", "state", "priority"}
		}, metadata.Breaking, []string{"/schema/required"}},
		{"make required property optional", func(_ *metadata.Definition, s map[string]any) {
			s["required"] = []any{"state"}
		}, metadata.Breaking, []string{"/schema/required"}},
		{"breaking wins over backfill", func(_ *metadata.Definition, s map[string]any) {
			prop(s, "priority")["default"] = 2
			prop(s, "title")["maxLength"] = 200
		}, metadata.Breaking, []string{"/schema/properties/title/maxLength"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := mustCompile(t, baseDefinition(t, baseSchema()))
			s := baseSchema()
			d := baseDefinition(t, s)
			tt.next(&d, s)
			d = rebuild(t, d, s)
			next := mustCompile(t, d)

			got, paths, err := metadata.Classify(prev, next)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("Classify = %v (%q), want %v", got, paths, tt.want)
			}
			if tt.paths != nil && !slices.Equal(paths, tt.paths) {
				t.Fatalf("paths = %q, want %q", paths, tt.paths)
			}
			if got == metadata.Compatible && paths != nil {
				t.Fatalf("Compatible with paths %q", paths)
			}
			if got == metadata.Breaking && len(paths) == 0 {
				t.Fatal("Breaking with no paths")
			}
		})
	}
}

// TestClassifyRemoveTransition: removing a transition is Breaking.
func TestClassifyRemoveTransition(t *testing.T) {
	s := baseSchema()
	d := baseDefinition(t, s)
	d.States.Transitions = append(d.States.Transitions, metadata.Transition{From: "closed", To: "open"})
	prev := mustCompile(t, d)
	next := mustCompile(t, baseDefinition(t, baseSchema()))
	got, paths, err := metadata.Classify(prev, next)
	if err != nil || got != metadata.Breaking || !slices.Equal(paths, []string{"/states/transitions/1"}) {
		t.Fatalf("Classify = %v %q %v, want Breaking [/states/transitions/1]", got, paths, err)
	}
}

func TestClassifyRemoveStatePaths(t *testing.T) {
	prev := mustCompile(t, baseDefinition(t, baseSchema()))
	s := baseSchema()
	d := baseDefinition(t, s)
	d.States.States, d.States.Transitions = []string{"open"}, nil
	prop(s, "state")["enum"] = []any{"open"}
	next := mustCompile(t, rebuild(t, d, s))
	_, paths, _ := metadata.Classify(prev, next)
	want := []string{"/schema/properties/state/enum", "/states/states/1", "/states/transitions/0"}
	if !slices.Equal(paths, want) {
		t.Fatalf("paths = %q, want %q", paths, want)
	}
}

func TestClassifyErrors(t *testing.T) {
	c := mustCompile(t, baseDefinition(t, baseSchema()))
	if _, _, err := metadata.Classify(nil, c); !errors.Is(err, metadata.ErrNilCompiled) {
		t.Fatalf("err = %v, want ErrNilCompiled", err)
	}
	other := baseDefinition(t, baseSchema())
	other.Key = "other"
	if _, _, err := metadata.Classify(c, mustCompile(t, other)); !errors.Is(err, metadata.ErrKeyMismatch) {
		t.Fatalf("err = %v, want ErrKeyMismatch", err)
	}
}

func TestCompatString(t *testing.T) {
	for c, want := range map[metadata.Compat]string{
		metadata.Compatible: "compatible", metadata.NeedsBackfill: "needs_backfill",
		metadata.Breaking: "breaking", metadata.Compat(9): "unknown",
	} {
		if c.String() != want {
			t.Fatalf("%d.String() = %q, want %q", c, c.String(), want)
		}
	}
}

// rebuild re-marshals s into d.Schema after a test changed s.
func rebuild(t *testing.T, d metadata.Definition, s map[string]any) metadata.Definition {
	t.Helper()
	n := baseDefinition(t, s)
	d.Schema = n.Schema
	return d
}

// TestClassifyAddOptionalPropertyToTemplate is the TICK-341 review case: a
// closed root means no existing row can hold a key that a new optional
// property then types, so Compatible is provable. v1 is the incident
// template; a row with an extra key never validated against it.
func TestClassifyAddOptionalPropertyToTemplate(t *testing.T) {
	d, err := metadata.Template("incident")
	if err != nil {
		t.Fatal(err)
	}
	prev := mustCompile(t, d)
	row := map[string]any{
		"short_description": "x", "state": "new", "priority": 3,
		"requester": "4b0f5f4e-7c43-4c1b-9d3f-7d7e1b2a9c10", "location": 42,
	}
	if errs := prev.Validate(row); !slices.Contains(errs, metadata.FieldError{Path: "/location", Code: "additionalProperties"}) {
		t.Fatalf("v1 Validate(row with extra key) = %v, want /location additionalProperties", errs)
	}

	var s map[string]any
	if err := json.Unmarshal(d.Schema, &s); err != nil {
		t.Fatal(err)
	}
	props(s)["location"] = map[string]any{"type": "string"}
	next := mustCompile(t, rebuild(t, d, s))
	got, paths, err := metadata.Classify(prev, next)
	if err != nil || got != metadata.Compatible || paths != nil {
		t.Fatalf("Classify = %v %q %v, want Compatible", got, paths, err)
	}
}
