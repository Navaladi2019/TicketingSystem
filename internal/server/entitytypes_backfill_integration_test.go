//go:build integration

// P10-04 (TICK-919, ADR-009 D1, D13): a publish that adds a top-level default
// writes the schema_backfills row and its outbox start row in the publish
// transaction; one that adds none writes neither; default + x-nava-unique is
// refused.

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// etWithDefault is a publish body whose schema adds an optional "severity"
// property with a default to etSchema.
func etWithDefault(t *testing.T, extra string) string {
	t.Helper()
	return etPublishBody(t, `"note": {`, `"severity": {"type": "string", "default": "low"`+extra+`}, "note": {`)
}

type etBackfillState struct {
	Jobs, Outbox int
}

func etBackfills(t *testing.T, e *etEnv, tenant uuid.UUID) etBackfillState {
	t.Helper()
	var s etBackfillState
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM schema_backfills WHERE tenant_id = $1`, tenant).Scan(&s.Jobs); err != nil {
		t.Fatal(err)
	}
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'schema.backfill'`, tenant).Scan(&s.Outbox); err != nil {
		t.Fatal(err)
	}
	return s
}

// covers: publishEntityTypeVersion 201
func TestPublishAddingADefaultWritesOneJobAndOneOutboxRow(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-def", etWithDefault(t, "")); rec.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body)
	}
	var (
		state, causation string
		version          int
		keys             []string
		defID            uuid.UUID
		audits           int
	)
	if err := e.super.QueryRow(t.Context(), `
SELECT state, target_version, default_keys, causation_audit_id::text, entity_definition_id
FROM schema_backfills WHERE tenant_id = $1`, e.a.id).Scan(&state, &version, &keys, &causation, &defID); err != nil {
		t.Fatalf("job row: %v", err)
	}
	if state != "pending" || version != 2 || strings.Join(keys, ",") != "severity,state" {
		t.Errorf("job = %s v%d %v, want pending v2 [severity state]", state, version, keys)
	}
	if err := e.super.QueryRow(t.Context(), `
SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id::text = $2 AND subject_id = $3 AND action = 'publish'`,
		e.a.id, causation, defID).Scan(&audits); err != nil || audits != 1 {
		t.Errorf("causation_audit_id %s is not the publish's audit entry (n=%d, err=%v)", causation, audits, err)
	}
	var dedupe, payload string
	var seq int64
	if err := e.super.QueryRow(t.Context(), `SELECT dedupe_key, payload::text, target_seq FROM outbox WHERE tenant_id = $1 AND kind = 'schema.backfill'`,
		e.a.id).Scan(&dedupe, &payload, &seq); err != nil {
		t.Fatalf("outbox row: %v", err)
	}
	if want := "schema-backfill:" + defID.String() + ":2"; dedupe != want || seq != 2 {
		t.Errorf("outbox dedupe_key %q target_seq %d, want %q 2", dedupe, seq, want)
	}
	var p map[string]string
	if err := json.Unmarshal([]byte(payload), &p); err != nil || len(p) != 1 || p["entity_definition_id"] != defID.String() {
		t.Errorf("payload = %s, want only entity_definition_id (no tenant field)", payload)
	}
}

// covers: publishEntityTypeVersion 201
func TestPublishAddingNoDefaultWritesNoJob(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	bodies := []string{
		etPublishBody(t, `"description": "a <b>&amp; tag"`, `"description": "changed"`),
		etPublishBody(t, `"note": {`, `"extra": {"type": "string"}, "note": {`),
		etPublishBody(t, `"note": {`, `"extra": {"type": "string"}, "note": {`, `"default": "new"`, `"default": "new", "title": "State"`),
	}
	for i, body := range bodies {
		if rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-nodef"+string(rune('a'+i)), body); rec.Code != http.StatusCreated {
			t.Fatalf("publish %d = %d: %s", i, rec.Code, rec.Body)
		}
	}
	if got := etBackfills(t, e, e.a.id); got != (etBackfillState{}) {
		t.Errorf("rows = %+v, want none", got)
	}
}

// covers: publishEntityTypeVersion 422
func TestPublishDefaultWithUniqueIsRefusedAndWritesNothing(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "k-uniq", etWithDefault(t, `, "x-nava-unique": true`))
	got := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidDefinition)
	found := false
	for _, f := range got.Errors {
		found = found || f.Pointer == "/definition/schema/properties/severity/default"
	}
	if !found {
		t.Errorf("errors = %+v, want a pointer at /definition/schema/properties/severity/default", got.Errors)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
	if got := etBackfills(t, e, e.a.id); got != (etBackfillState{}) {
		t.Errorf("rows = %+v, want none", got)
	}
}

// CI-1: a rollback after the job insert leaves no version, no job and no
// outbox row.
func TestRolledBackPublishAfterJobInsertWritesNoJobAndNoOutboxRow(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	errAbort := errors.New("abort after the job insert")
	ctx := context.Background()
	def := etDefinition(t, "incident", "INC")
	def.Schema = json.RawMessage(strings.Replace(etSchema, `"note": {`, `"severity": {"type": "string", "default": "low"}, "note": {`, 1))
	ran := false
	err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Publish(ctx, tx, e.a.id, entitytypes.PublishInput{
			Key: "incident", PublishedBy: e.a.admin.PrincipalID, Definition: def,
			AfterBackfillQueued: func() error { ran = true; return errAbort },
		})
		return err
	})
	if !errors.Is(err, errAbort) || !ran {
		t.Fatalf("tx = %v (hook ran %v), want the abort", err, ran)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions, want 1", n)
	}
	if got := etBackfills(t, e, e.a.id); got != (etBackfillState{}) {
		t.Errorf("rows = %+v, want none", got)
	}
}

// covers: publishEntityTypeVersion 201
func TestPublishReplayWithADefaultKeepsOneVersionOneJobOneOutboxRow(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	body := etWithDefault(t, "")
	first := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "same", body)
	again := e.do(t, http.MethodPost, etVersionsPath, "admin-a", "same", body)
	if first.Code != http.StatusCreated || again.Code != http.StatusCreated || again.Body.String() != first.Body.String() {
		t.Fatalf("publish = %d, replay = %d, want the same 201 body", first.Code, again.Code)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}
	if got := etBackfills(t, e, e.a.id); got != (etBackfillState{Jobs: 1, Outbox: 1}) {
		t.Errorf("rows = %+v, want one job and one outbox row", got)
	}
}

// covers: publishEntityTypeVersion 404
func TestPublishCrossTenantWithADefaultIsNotFoundAndWritesNothingInA(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	foreign := e.do(t, http.MethodPost, etVersionsPath, "admin-b", "k-x", etWithDefault(t, ""))
	missing := e.do(t, http.MethodPost, "/api/v1/entity-types/no_such_type/versions", "admin-b", "k-y", etWithDefault(t, ""))
	if foreign.Code != http.StatusNotFound || foreign.Body.String() != missing.Body.String() {
		t.Errorf("foreign = %d %s, missing = %d %s, want the same 404 bytes", foreign.Code, foreign.Body, missing.Code, missing.Body)
	}
	if n := etVersionCount(t, e); n != 1 {
		t.Errorf("%d versions in A, want 1", n)
	}
	for name, tenant := range map[string]uuid.UUID{"A": e.a.id, "B": e.b.id} {
		if got := etBackfills(t, e, tenant); got != (etBackfillState{}) {
			t.Errorf("tenant %s rows = %+v, want none", name, got)
		}
	}
}
