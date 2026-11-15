package entitytypes

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

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
		name         string
		prev, next   *metadata.Compiled
		removed      []string
		added        []string
		removedIsNil bool
	}{
		{name: "same", prev: both, next: both, removedIsNil: true},
		{name: "added", prev: onlyA, next: both, removedIsNil: true, added: []string{"b"}},
		{name: "flag dropped", prev: both, next: compileFlagged(t, plainA+","+b), removed: []string{"/schema/properties/a"}},
		{name: "property deleted", prev: both, next: onlyB, removed: []string{"/schema/properties/a"}},
		{name: "swap", prev: onlyA, next: onlyB, removed: []string{"/schema/properties/a"}, added: []string{"b"}},
	}
	for _, c := range cases {
		err := removedFilterable(c.prev, c.next)
		var ref *metadata.RefusalError
		switch {
		case c.removedIsNil && err != nil, !c.removedIsNil && !errors.As(err, &ref):
			t.Errorf("%s: removedFilterable = %v", c.name, err)
		case ref != nil && (ref.Code != metadata.RefusalFilterableRemovalUnavailable || !slices.Equal(ref.Paths, c.removed)):
			t.Errorf("%s: refusal = %+v, want paths %v", c.name, ref, c.removed)
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
