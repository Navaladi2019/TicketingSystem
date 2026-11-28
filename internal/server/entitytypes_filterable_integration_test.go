//go:build integration

// P7-03 (TICK-743): create and publish answer the ADR-006 D8 metadata
// refusals with their own codes and write nothing; adding or removing
// x-nava-filterable publishes as compatible.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// etFlagged returns etSchema with "note" replaced by the given property JSON.
func etFlagged(t *testing.T, note string) string {
	t.Helper()
	s := strings.Replace(etSchema, `"note": {"type": "string", "description": "a <b>&amp; tag"}`, `"note": `+note, 1)
	if s == etSchema {
		t.Fatal("etSchema no longer has the note property")
	}
	return s
}

func etPublishSchema(schema string) string {
	return fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, schema, etDefTail)
}

func TestPublishRefusesFilterableAndSearchableMisuse(t *testing.T) {
	cases := []struct {
		name, note string
		code       problem.Code
		pointer    string
	}{
		{"nested filterable", `{"type": "object", "properties": {"x": {"type": "string", "x-nava-filterable": true}}}`,
			problem.FilterableFieldNotTopLevel, "/definition/schema/properties/note/properties/x/x-nava-filterable"},
		{"filterable inside allOf", `{"allOf": [{"type": "string", "x-nava-filterable": true}]}`,
			problem.FilterableFieldNotTopLevel, "/definition/schema/properties/note/allOf/0/x-nava-filterable"},
		{"searchable", `{"type": "string", "x-nava-searchable": true}`,
			problem.SearchableNotAvailable, "/definition/schema/properties/note/x-nava-searchable"},
		{"unsupported type", `{"type": "object", "x-nava-filterable": true}`,
			problem.InvalidDefinition, "/definition/schema/properties/note/x-nava-filterable"},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newETEnv(t)
			etCreateIncident(t, e)
			rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", fmt.Sprintf("k-fl-%d", i), etPublishSchema(etFlagged(t, tc.note)))
			p := wantProblem(t, rec, http.StatusUnprocessableEntity, tc.code)
			if got := pointers(p); !slices.Contains(got, tc.pointer) {
				t.Errorf("pointers = %v, want %s among them", got, tc.pointer)
			}
			if n := etVersionCount(t, e); n != 1 {
				t.Errorf("%d versions, want 1", n)
			}
			if rows := etAuditRows(t, e); len(rows) != 1 {
				t.Errorf("%d audit entries, want 1", len(rows))
			}
		})
	}
}

func TestCreateRefusesNestedFilterable(t *testing.T) {
	e := newETEnv(t)
	body := fmt.Sprintf(`{"key": "incident", "name": "Incident", "number_prefix": "INC", "definition": {"schema": %s, %s}}`,
		etFlagged(t, `{"type": "array", "items": {"type": "string", "x-nava-filterable": true}}`), etDefTail)
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-fl-create", body)
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.FilterableFieldNotTopLevel)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/properties/note/items/x-nava-filterable"}) {
		t.Errorf("pointers = %v", got)
	}
	if n := etVersionCount(t, e); n != 0 {
		t.Errorf("%d versions, want 0", n)
	}
}

func TestPublishFilterableFlagIsCompatibleAndTypeChangeAndRemovalAreRefused(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	compat := func(t *testing.T, key, schema string) {
		t.Helper()
		rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", key, etPublishSchema(schema))
		if rec.Code != http.StatusCreated {
			t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
		}
		var v struct {
			Compat string `json:"compat"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || v.Compat != "compatible" {
			t.Fatalf("compat = %q (%v), want compatible: %s", v.Compat, err, rec.Body)
		}
	}
	compat(t, "k-fl-add", etFlagged(t, `{"type": "string", "x-nava-filterable": true}`))

	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-fl-type",
		etPublishSchema(etFlagged(t, `{"type": "string", "format": "date-time", "x-nava-filterable": true}`)))
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.FilterableFieldTypeChange)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/properties/note"}) {
		t.Errorf("pointers = %v", got)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}

	// Removing the flag is refused in S21 (D-7-4); TestPublishFieldIndexRefusalsWriteNothing covers it.
	rec = e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-fl-remove", etPublishSchema(etFlagged(t, `{"type": "string"}`)))
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.FilterableFieldRemovalUnavailable)
}

// covers: createEntityType 422
func TestCreateRefusesSearchableAndWritesNothing(t *testing.T) {
	e := newETEnv(t)
	body := fmt.Sprintf(`{"key": "incident", "name": "Incident", "number_prefix": "INC", "definition": {"schema": %s, %s}}`,
		etFlagged(t, `{"type": "string", "x-nava-filterable": true, "x-nava-searchable": true}`), etDefTail)
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-sr-create", body)
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.SearchableNotAvailable)
	if got := pointers(p); !slices.Equal(got, []string{"/definition/schema/properties/note/x-nava-searchable"}) {
		t.Errorf("pointers = %v", got)
	}
	for _, table := range []string{"entity_definitions", "entity_definition_versions", "record_index_fields"} {
		var n int
		if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%d %s rows, want 0", n, table)
		}
	}
	if rows := etBackfillRows(t, e, e.a.id); len(rows) != 0 {
		t.Errorf("%d backfill outbox rows, want 0", len(rows))
	}
}
