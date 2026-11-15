//go:build integration

// P7-04 (TICK-744): publishing a definition that flags fields writes, in the
// publish transaction, one registry row and one outbox row per new field,
// within the ADR-006 D12 quota, and nothing when it is refused.

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// etFlaggedProps returns etSchema with one optional string property per name,
// each flagged x-nava-filterable, added to the existing ones.
func etFlaggedProps(t *testing.T, names ...string) string {
	t.Helper()
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, `%q: {"type": "string", "x-nava-filterable": true}, `, n)
	}
	s := strings.Replace(etSchema, `"properties": {`, `"properties": {`+b.String(), 1)
	if s == etSchema {
		t.Fatal("etSchema has no properties")
	}
	return s
}

func etPublishKey(t *testing.T, e *etEnv, key, token, idem, schema string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/entity-types/"+key+"/versions", token, idem, etPublishSchema(schema))
}

type registryRow struct {
	Key, Type, State string
	Progress, Req    int
	ID               uuid.UUID
}

func etRegistry(t *testing.T, e *etEnv, tenant uuid.UUID) []registryRow {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `
SELECT id, field_key, value_type, state, progress, requested_version FROM record_index_fields
WHERE tenant_id = $1 ORDER BY field_key`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []registryRow
	for rows.Next() {
		var r registryRow
		if err := rows.Scan(&r.ID, &r.Key, &r.Type, &r.State, &r.Progress, &r.Req); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

type backfillRow struct {
	Dedupe  string
	Payload map[string]string
	Seq     int64
}

func etBackfillRows(t *testing.T, e *etEnv, tenant uuid.UUID) []backfillRow {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `
SELECT dedupe_key, payload::text, target_seq FROM outbox
WHERE tenant_id = $1 AND kind = 'field_index.backfill' ORDER BY dedupe_key`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []backfillRow
	for rows.Next() {
		var r backfillRow
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

// etWantCounts fails unless tenant A has wantVersions entity-definition
// versions and wantFields registry rows, each with one backfill outbox row.
func etWantCounts(t *testing.T, e *etEnv, wantVersions, wantFields int) {
	t.Helper()
	if n := etVersionCount(t, e); n != wantVersions {
		t.Errorf("%d versions, want %d", n, wantVersions)
	}
	if n := len(etRegistry(t, e, e.a.id)); n != wantFields {
		t.Errorf("%d registry rows, want %d", n, wantFields)
	}
	if n := len(etBackfillRows(t, e, e.a.id)); n != wantFields {
		t.Errorf("%d backfill outbox rows, want %d", n, wantFields)
	}
}

// etFixtureField inserts a registry row for fieldKey of the entity type.
func etFixtureField(t *testing.T, e *etEnv, entityKey, fieldKey, state string) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), `
INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
SELECT tenant_id, $3, id, $4, 'text', $5, 0, 1 FROM entity_definitions WHERE tenant_id = $1 AND key = $2`,
		e.a.id, entityKey, uuid.New(), fieldKey, state); err != nil {
		t.Fatal(err)
	}
}

// covers: publishEntityTypeVersion 201
func TestPublishFlagWritesOneRegistryRowAndOneOutboxRow(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	schema := etFlaggedProps(t, "region")
	first := etPublishKey(t, e, "incident", "admin-a", "k-flag", schema)
	if first.Code != http.StatusCreated {
		t.Fatalf("publish = %d: %s", first.Code, first.Body)
	}
	again := etPublishKey(t, e, "incident", "admin-a", "k-flag", schema)
	if again.Code != http.StatusCreated || again.Body.String() != first.Body.String() {
		t.Errorf("replay = %d, want the same 201 body", again.Code)
	}
	reg := etRegistry(t, e, e.a.id)
	if len(reg) != 1 {
		t.Fatalf("registry = %+v, want one row", reg)
	}
	if r := reg[0]; r.Key != "region" || r.Type != "text" || r.State != "backfilling" || r.Progress != 0 || r.Req != 2 {
		t.Errorf("registry row = %+v, want region text backfilling 0 requested_version 2", r)
	}
	rows := etBackfillRows(t, e, e.a.id)
	if len(rows) != 1 {
		t.Fatalf("outbox = %+v, want one backfill row", rows)
	}
	want := map[string]string{"tenant_id": e.a.id.String(), "field_id": reg[0].ID.String()}
	if len(rows[0].Payload) != 2 || rows[0].Payload["tenant_id"] != want["tenant_id"] || rows[0].Payload["field_id"] != want["field_id"] || rows[0].Seq != 2 {
		t.Errorf("outbox row = %+v, want payload %v target_seq 2", rows[0], want)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2", n)
	}
}

// covers: publishEntityTypeVersion 409 422
func TestPublishFieldIndexRefusalsWriteNothing(t *testing.T) {
	cases := []struct {
		name string
		// bare skips the setup publish of one flagged field "region" (version 2,
		// one registry row), so the type starts at version 1 with no fields.
		bare   bool
		prep   func(t *testing.T, e *etEnv)
		schema func(t *testing.T) string
		status int
		code   problem.Code
		want   []string // pointers that must be among the errors
	}{
		{name: "flag removed", schema: func(t *testing.T) string {
			return strings.Replace(etSchema, `"properties": {`, `"properties": {"region": {"type": "string"}, `, 1)
		}, status: http.StatusUnprocessableEntity, code: problem.FilterableFieldRemovalUnavailable,
			want: []string{"/definition/schema/properties/region"}},
		{name: "flagged property deleted from the schema", schema: func(t *testing.T) string { return etSchema },
			status: http.StatusUnprocessableEntity, code: problem.FilterableFieldRemovalUnavailable,
			want: []string{"/definition/schema/properties/region"}},
		{name: "type change", schema: func(t *testing.T) string {
			return strings.Replace(etFlaggedProps(t, "region"), `"region": {"type": "string"`, `"region": {"type": "number"`, 1)
		}, status: http.StatusUnprocessableEntity, code: problem.FilterableFieldTypeChange,
			want: []string{"/definition/schema/properties/region"}},
		{name: "nested flag", bare: true, schema: func(t *testing.T) string {
			return etFlagged(t, `{"type": "object", "properties": {"x": {"type": "string", "x-nava-filterable": true}}}`)
		}, status: http.StatusUnprocessableEntity, code: problem.FilterableFieldNotTopLevel},
		{name: "searchable", bare: true, schema: func(t *testing.T) string {
			return etFlagged(t, `{"type": "string", "x-nava-searchable": true}`)
		}, status: http.StatusUnprocessableEntity, code: problem.SearchableNotAvailable},
		{name: "sixth flag on the type", schema: func(t *testing.T) string {
			return etFlaggedProps(t, "region", "f2", "f3", "f4", "f5", "f6")
		}, status: http.StatusUnprocessableEntity, code: problem.IndexQuotaExceeded,
			want: []string{"/definition/schema/properties/f6"}},
		{name: "field being removed", bare: true,
			prep: func(t *testing.T, e *etEnv) {
				t.Helper()
				etFixtureField(t, e, "incident", "region", "removing")
			},
			schema: func(t *testing.T) string { return etFlaggedProps(t, "region") },
			status: http.StatusConflict, code: problem.FieldIndexBusy,
			want: []string{"/definition/schema/properties/region"}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newETEnv(t)
			etCreateIncident(t, e)
			versions, fields, outboxRows := 1, 0, 0
			if tc.prep != nil {
				tc.prep(t, e)
				fields = len(etRegistry(t, e, e.a.id))
			}
			if !tc.bare {
				if rec := etPublishKey(t, e, "incident", "admin-a", "k-setup", etFlaggedProps(t, "region")); rec.Code != http.StatusCreated {
					t.Fatalf("setup publish = %d: %s", rec.Code, rec.Body)
				}
				versions, fields, outboxRows = 2, 1, 1
			}
			rec := etPublishKey(t, e, "incident", "admin-a", fmt.Sprintf("k-%d", i), tc.schema(t))
			p := wantProblem(t, rec, tc.status, tc.code)
			for _, w := range tc.want {
				if got := pointers(p); !slices.Contains(got, w) {
					t.Errorf("pointers = %v, want %s among them", got, w)
				}
			}
			if n := etVersionCount(t, e); n != versions {
				t.Errorf("%d versions, want %d", n, versions)
			}
			if n := len(etRegistry(t, e, e.a.id)); n != fields {
				t.Errorf("%d registry rows, want %d", n, fields)
			}
			if n := len(etBackfillRows(t, e, e.a.id)); n != outboxRows {
				t.Errorf("%d backfill outbox rows, want %d", n, outboxRows)
			}
		})
	}
}

// covers: publishEntityTypeVersion 422
func TestPublishFiftyFirstFlagInTheTenantIsRefused(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-bulk", etBody("bulk", "BLK")); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	for i := range 50 {
		etFixtureField(t, e, "bulk", fmt.Sprintf("b%d", i), "ready")
	}
	rec := etPublishKey(t, e, "incident", "admin-a", "k-51", etFlaggedProps(t, "region"))
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.IndexQuotaExceeded)
	if n := len(etRegistry(t, e, e.a.id)); n != 50 {
		t.Errorf("%d registry rows, want 50", n)
	}
	if n := len(etBackfillRows(t, e, e.a.id)); n != 0 {
		t.Errorf("%d backfill outbox rows, want 0", n)
	}
	if n := etVersionCount(t, e); n != 2 {
		t.Errorf("%d versions, want 2 (the two creates)", n)
	}
}

// The quota is counted under a tenant lock: of two concurrent publishes on
// different types that together would pass 50, exactly one commits.
//
// covers: publishEntityTypeVersion 201 422
func TestConcurrentPublishesCannotBothPassTheTenantQuota(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	for key, prefix := range map[string]string{"change": "CHG", "bulk": "BLK"} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-"+key, etBody(key, prefix)); rec.Code != http.StatusCreated {
			t.Fatalf("create %s = %d: %s", key, rec.Code, rec.Body)
		}
	}
	for i := range 48 {
		etFixtureField(t, e, "bulk", fmt.Sprintf("b%d", i), "ready")
	}
	var wg sync.WaitGroup
	recs := make([]*httptest.ResponseRecorder, 2)
	for i, key := range []string{"incident", "change"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs[i] = etPublishKey(t, e, key, "admin-a", "k-conc-"+key, etFlaggedProps(t, "f1", "f2"))
		}()
	}
	wg.Wait()
	var created, refused int
	for _, rec := range recs {
		switch rec.Code {
		case http.StatusCreated:
			created++
		case http.StatusUnprocessableEntity:
			wantProblem(t, rec, http.StatusUnprocessableEntity, problem.IndexQuotaExceeded)
			refused++
		default:
			t.Errorf("publish = %d: %s", rec.Code, rec.Body)
		}
	}
	if created != 1 || refused != 1 {
		t.Errorf("%d created, %d refused, want 1 and 1", created, refused)
	}
	if n := len(etRegistry(t, e, e.a.id)); n != 50 {
		t.Errorf("%d registry rows, want 50", n)
	}
	if n := len(etBackfillRows(t, e, e.a.id)); n != 2 {
		t.Errorf("%d backfill outbox rows, want 2", n)
	}
}

// covers: publishEntityTypeVersion 404
func TestPublishFlagOnAnotherTenantsTypeWritesNothing(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "k-b", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	rec := etPublishKey(t, e, "incident", "admin-a", "k-cross", etFlaggedProps(t, "region"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("publish = %d, want 404: %s", rec.Code, rec.Body)
	}
	for _, tn := range []uuid.UUID{e.a.id, e.b.id} {
		if n := len(etRegistry(t, e, tn)); n != 0 {
			t.Errorf("tenant %s has %d registry rows, want 0", tn, n)
		}
		if n := len(etBackfillRows(t, e, tn)); n != 0 {
			t.Errorf("tenant %s has %d backfill outbox rows, want 0", tn, n)
		}
	}
}

// P7-01 review condition: nava_app writes the registry only inside the audited
// publish transaction. A rollback takes the registry row, the outbox row and
// the publish audit entry together; a commit keeps all three.
func TestRegistryWriteAndPublishAuditEntryShareOneTransaction(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	errAbort := errors.New("abort after publish")
	def := etDefinition(t, "incident", "INC")
	def.Schema = json.RawMessage(etFlaggedProps(t, "region"))
	publish := func(tx pgx.Tx) error {
		v, err := entitytypes.Store{}.Publish(context.Background(), tx, e.a.id, entitytypes.PublishInput{Key: "incident", PublishedBy: e.a.admin.PrincipalID, Definition: def})
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}
		if v.Version != 2 {
			t.Fatalf("version = %d, want 2", v.Version)
		}
		return nil
	}
	err := tenancy.InTenantTx(context.Background(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_ = publish(tx)
		return errAbort // the registry row, the outbox row and the audit entry were written; now roll back
	})
	if !errors.Is(err, errAbort) {
		t.Fatalf("tx = %v, want the abort", err)
	}
	etWantCounts(t, e, 1, 0)
	if rows := etAuditRows(t, e); len(rows) != 1 {
		t.Errorf("%d audit entries after rollback, want 1 (the create)", len(rows))
	}

	if err := tenancy.InTenantTx(context.Background(), e.pool, e.a.id, publish); err != nil {
		t.Fatal(err)
	}
	etWantCounts(t, e, 2, 1)
	rows := etAuditRows(t, e)
	if len(rows) != 2 || rows[1].Action != "publish" {
		t.Errorf("audit entries = %+v, want the create then the publish", rows)
	}
}

// A retry of the same enqueue returns the row it wrote and nothing more; a
// later requested version is a new row.
func TestEnqueueFieldIndexBackfillReplayReturnsTheSameRow(t *testing.T) {
	e := newETEnv(t)
	field := uuid.New()
	var first, again, next uuid.UUID
	err := tenancy.InTenantTx(context.Background(), e.pool, e.a.id, func(tx pgx.Tx) error {
		var err error
		if first, err = outbox.EnqueueFieldIndexBackfill(context.Background(), tx, field, 2); err != nil {
			return err
		}
		if again, err = outbox.EnqueueFieldIndexBackfill(context.Background(), tx, field, 2); err != nil {
			return err
		}
		next, err = outbox.EnqueueFieldIndexBackfill(context.Background(), tx, field, 3)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Errorf("replay returned %s, want the first row %s", again, first)
	}
	if next == first {
		t.Error("a later requested version returned the first row, want a new one")
	}
	if rows := etBackfillRows(t, e, e.a.id); len(rows) != 2 {
		t.Errorf("%d outbox rows, want 2 (version 2 once, version 3)", len(rows))
	}
}
