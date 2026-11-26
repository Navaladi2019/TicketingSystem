package entitytypes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

func compileFlagged(t *testing.T, props string) *metadata.Compiled {
	t.Helper()
	c, err := metadata.Compile(metadata.Definition{
		Key: "incident", NumberPrefix: "INC",
		Schema:       json.RawMessage(`{"type":"object","additionalProperties":false,"required":["state"],"properties":{"state":{"type":"string","enum":["new"]},` + props + `}}`),
		FixedColumns: map[string]string{"state": "state"},
		States:       metadata.StateModel{Initial: "new", States: []string{"new"}},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return c
}

func TestRemovedAndAddedFilterable(t *testing.T) {
	t.Parallel()
	const a = `"a": {"type": "string", "x-nava-filterable": true}`
	const b = `"b": {"type": "number", "x-nava-filterable": true}`
	plainA := `"a": {"type": "string"}`
	both, onlyA, onlyB := compileFlagged(t, a+","+b), compileFlagged(t, a), compileFlagged(t, b)
	cases := []struct {
		name       string
		prev, next *metadata.Compiled
		removed    []string
		added      []string
	}{
		{name: "same", prev: both, next: both},
		{name: "added", prev: onlyA, next: both, added: []string{"b"}},
		{name: "flag dropped", prev: both, next: compileFlagged(t, plainA+","+b), removed: []string{"a"}},
		{name: "both dropped", prev: both, next: compileFlagged(t, plainA+`,"b": {"type": "number"}`), removed: []string{"a", "b"}},
		{name: "property deleted", prev: both, next: onlyB, removed: []string{"a"}},
		{name: "swap", prev: onlyA, next: onlyB, removed: []string{"a"}, added: []string{"b"}},
	}
	for _, c := range cases {
		if got := removedFilterable(c.prev, c.next); !slices.Equal(got, c.removed) {
			t.Errorf("%s: removed = %v, want %v", c.name, got, c.removed)
		}
		var got []string
		for _, f := range addedFilterable(c.prev, c.next) {
			got = append(got, f.Key)
		}
		if !slices.Equal(got, c.added) {
			t.Errorf("%s: added = %v, want %v", c.name, got, c.added)
		}
	}
}

func TestFieldPointerEscapes(t *testing.T) {
	t.Parallel()
	if got := fieldPointer("a/b~c"); got != "/schema/properties/a~1b~0c" {
		t.Fatalf("fieldPointer = %q", got)
	}
}

func TestFieldIndexErrorMessages(t *testing.T) {
	t.Parallel()
	q := &IndexQuotaError{Scope: QuotaPerType, Limit: 5}
	if got := q.Error(); got != "entitytypes: index_quota_exceeded: more than 5 filterable fields per type" {
		t.Errorf("IndexQuotaError = %q", got)
	}
	b := &FieldIndexBusyError{Paths: []string{"/schema/properties/a", "/schema/properties/b"}}
	if got := b.Error(); got != "entitytypes: field_index_busy at /schema/properties/a, /schema/properties/b" {
		t.Errorf("FieldIndexBusyError = %q", got)
	}
}

// Seed refuses a flagged definition before any query (tx is nil): onboarding
// must not write the registry (P7-01).
func TestSeedRefusesAFilterableDefinition(t *testing.T) {
	t.Parallel()
	_, err := Store{}.Seed(context.Background(), nil, uuid.New(), SeedInput{
		DefinitionID: uuid.New(), VersionID: uuid.New(), Name: "Incident",
		Definition: metadata.Definition{
			Key: "incident", NumberPrefix: "INC",
			Schema:       json.RawMessage(`{"type":"object","additionalProperties":false,"required":["state"],"properties":{"state":{"type":"string","enum":["new"]},"region":{"type":"string","x-nava-filterable":true}}}`),
			FixedColumns: map[string]string{"state": "state"},
			States:       metadata.StateModel{Initial: "new", States: []string{"new"}},
		},
	})
	if !errors.Is(err, ErrSeedFilterable) {
		t.Fatalf("Seed = %v, want ErrSeedFilterable", err)
	}
}
