//go:build integration

// TICK-750 (S22 P7-10, ADR-006 D8, D11): publishing a definition that drops
// x-nava-filterable flips the field's registry row to removing and writes one
// field_index.remove outbox row, in the publish transaction. A filter on the
// field answers 400 field_not_filterable from that commit on.

package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// etUnflagged returns etSchema with an optional string property "region",
// not flagged.
func etUnflagged(t *testing.T) string {
	t.Helper()
	s := strings.Replace(etSchema, `"properties": {`, `"properties": {"region": {"type": "string"}, `, 1)
	if s == etSchema {
		t.Fatal("etSchema has no properties")
	}
	return s
}

type removeRow struct {
	Dedupe  string
	Payload map[string]string
	Seq     int64
}

func etRemoveRows(t *testing.T, e *etEnv, tenant uuid.UUID) []removeRow {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `
SELECT dedupe_key, payload::text, target_seq FROM outbox
WHERE tenant_id = $1 AND kind = 'field_index.remove' ORDER BY dedupe_key`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []removeRow
	for rows.Next() {
		var r removeRow
		var p string
		if err := rows.Scan(&r.Dedupe, &p, &r.Seq); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(p), &r.Payload); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// etFlagRegionReady publishes region flagged (version 2) as tenant tok's
// admin and marks its registry row ready, as a finished backfill would, so
// a filter on it answers 200.
func etFlagRegionReady(t *testing.T, e *etEnv, tok, idem string, tenant uuid.UUID) registryRow {
	t.Helper()
	if rec := etPublishKey(t, e, "incident", tok, idem, etFlaggedProps(t, "region")); rec.Code != http.StatusCreated {
		t.Fatalf("flag publish = %d: %s", rec.Code, rec.Body)
	}
	if _, err := e.super.Exec(t.Context(), `UPDATE record_index_fields SET state = 'ready', progress = 100
		WHERE tenant_id = $1 AND field_key = 'region'`, tenant); err != nil {
		t.Fatal(err)
	}
	reg := etRegistry(t, e, tenant)
	if len(reg) != 1 || reg[0].State != "ready" {
		t.Fatalf("registry %+v, want region ready", reg)
	}
	return reg[0]
}

// covers: publishEntityTypeVersion 201
//
// Publish removing a flag → 201; the registry row is removing at the new
// version, one outbox row; a filter answers 400 at once. A replay of the same
// publish (same Idempotency-Key) answers the same 201 and writes nothing more.
func TestPublishUnflagFlipsToRemoving(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	flagged := etFlagRegionReady(t, e, "admin-a", "k-flag", e.a.id)
	rec, _ := e.list(t, "admin-a", "incident", "filter[region]=emea")
	if rec.Code != http.StatusOK {
		t.Fatalf("filter while ready = %d: %s", rec.Code, rec.Body)
	}

	first := etPublishKey(t, e, "incident", "admin-a", "k-unflag", etUnflagged(t))
	if first.Code != http.StatusCreated {
		t.Fatalf("unflag publish = %d: %s", first.Code, first.Body)
	}
	reg := etRegistry(t, e, e.a.id)
	if len(reg) != 1 || reg[0].ID != flagged.ID || reg[0].State != "removing" || reg[0].Req != 3 {
		t.Fatalf("registry %+v, want the same row removing at requested_version 3", reg)
	}
	rows := etRemoveRows(t, e, e.a.id)
	if len(rows) != 1 {
		t.Fatalf("removal outbox rows %+v, want one", rows)
	}
	if r := rows[0]; len(r.Payload) != 2 || r.Payload["tenant_id"] != e.a.id.String() ||
		r.Payload["field_id"] != flagged.ID.String() || r.Seq != 3 {
		t.Errorf("removal row %+v, want payload {tenant_id, field_id} and target_seq 3", r)
	}
	rec, _ = e.list(t, "admin-a", "incident", "filter[region]=emea")
	listCode(t, rec, http.StatusBadRequest, "field_not_filterable")

	again := etPublishKey(t, e, "incident", "admin-a", "k-unflag", etUnflagged(t))
	if again.Code != http.StatusCreated || again.Body.String() != first.Body.String() {
		t.Errorf("replay = %d, want the same 201 body", again.Code)
	}
	if n := len(etRemoveRows(t, e, e.a.id)); n != 1 {
		t.Errorf("%d removal outbox rows after the replay, want 1", n)
	}
	if n := etVersionCount(t, e); n != 3 {
		t.Errorf("%d versions, want 3", n)
	}
	// A further compatible publish that still leaves region unflagged writes
	// no second removal: the row is already removing.
	if rec := etPublishKey(t, e, "incident", "admin-a", "k-again", etUnflagged(t)); rec.Code != http.StatusCreated {
		t.Fatalf("second unflagged publish = %d: %s", rec.Code, rec.Body)
	}
	if n := len(etRemoveRows(t, e, e.a.id)); n != 1 {
		t.Errorf("%d removal outbox rows, want 1", n)
	}
}

// covers: publishEntityTypeVersion 409 201
//
// Re-flag while removing → 409 field_index_busy, nothing written; once the
// removal has finished (FieldIndexRemove's end state: no side rows, no
// registry row) the same publish → 201 and a new backfill.
func TestPublishReflagWhileRemovingIsBusyThenBackfills(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	old := etFlagRegionReady(t, e, "admin-a", "k-flag", e.a.id)
	if rec := etPublishKey(t, e, "incident", "admin-a", "k-unflag", etUnflagged(t)); rec.Code != http.StatusCreated {
		t.Fatalf("unflag publish = %d: %s", rec.Code, rec.Body)
	}

	rec := etPublishKey(t, e, "incident", "admin-a", "k-reflag", etFlaggedProps(t, "region"))
	p := wantProblem(t, rec, http.StatusConflict, problem.FieldIndexBusy)
	if got := pointers(p); len(got) != 1 || got[0] != "/definition/schema/properties/region" {
		t.Errorf("pointers = %v", got)
	}
	if n := etVersionCount(t, e); n != 3 {
		t.Errorf("%d versions, want 3", n)
	}
	if reg := etRegistry(t, e, e.a.id); len(reg) != 1 || reg[0].State != "removing" || reg[0].Req != 3 {
		t.Errorf("registry %+v, want it untouched", reg)
	}
	if n, m := len(etBackfillRows(t, e, e.a.id)), len(etRemoveRows(t, e, e.a.id)); n != 1 || m != 1 {
		t.Errorf("%d backfill and %d removal rows, want 1 and 1", n, m)
	}

	// The removal's end state (internal/fieldindex/backfill runs it for real).
	if _, err := e.super.Exec(t.Context(), `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, e.a.id, old.ID); err != nil {
		t.Fatal(err)
	}
	rec = etPublishKey(t, e, "incident", "admin-a", "k-reflag-2", etFlaggedProps(t, "region"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("re-flag after removal = %d: %s", rec.Code, rec.Body)
	}
	reg := etRegistry(t, e, e.a.id)
	if len(reg) != 1 || reg[0].ID == old.ID || reg[0].State != "backfilling" || reg[0].Req != 4 {
		t.Fatalf("registry %+v, want a new region row backfilling at requested_version 4", reg)
	}
	var fresh int
	for _, r := range etBackfillRows(t, e, e.a.id) {
		if r.Payload["field_id"] == reg[0].ID.String() && r.Seq == 4 {
			fresh++
		}
	}
	if fresh != 1 {
		t.Errorf("%d backfill rows for the new field, want 1", fresh)
	}
	rec, _ = e.list(t, "admin-a", "incident", "filter[region]=emea")
	listCode(t, rec, http.StatusUnprocessableEntity, "field_not_ready")
}

// covers: publishEntityTypeVersion 404
//
// Cross-tenant: tenant A cannot unflag tenant B's field. A publish by A's
// admin to a key only B has answers 404 and writes nothing in B; A's own
// unflag of a same-named type flips A's row only.
func TestPublishUnflagCannotReachAnotherTenant(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "k-create-b", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create in B = %d: %s", rec.Code, rec.Body)
	}
	bField := etFlagRegionReady(t, e, "admin-b", "k-flag-b", e.b.id)

	rec := etPublishKey(t, e, "incident", "admin-a", "k-unflag-a", etUnflagged(t))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("A's unflag of B's type = %d: %s, want 404", rec.Code, rec.Body)
	}
	if reg := etRegistry(t, e, e.b.id); len(reg) != 1 || reg[0] != bField {
		t.Errorf("B's registry %+v, want it untouched (%+v)", reg, bField)
	}
	if n := len(etRemoveRows(t, e, e.b.id)) + len(etRemoveRows(t, e, e.a.id)); n != 0 {
		t.Errorf("%d removal rows, want 0", n)
	}

	// A's own type of the same key: A's unflag touches A's row only.
	etCreateIncident(t, e)
	etFlagRegionReady(t, e, "admin-a", "k-flag-a", e.a.id)
	if rec := etPublishKey(t, e, "incident", "admin-a", "k-unflag-a2", etUnflagged(t)); rec.Code != http.StatusCreated {
		t.Fatalf("A's unflag of its own type = %d: %s", rec.Code, rec.Body)
	}
	if reg := etRegistry(t, e, e.a.id); len(reg) != 1 || reg[0].State != "removing" {
		t.Errorf("A's registry %+v, want removing", reg)
	}
	if reg := etRegistry(t, e, e.b.id); len(reg) != 1 || reg[0] != bField {
		t.Errorf("B's registry %+v after A's unflag, want it untouched", reg)
	}
	if n := len(etRemoveRows(t, e, e.b.id)); n != 0 {
		t.Errorf("%d removal rows in B, want 0", n)
	}
	rec, _ = e.list(t, "admin-b", "incident", "filter[region]=emea")
	if rec.Code != http.StatusOK {
		t.Errorf("B's filter after A's unflag = %d: %s, want 200", rec.Code, rec.Body)
	}
}
