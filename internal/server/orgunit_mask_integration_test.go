//go:build integration

// TICK-1119 (S31 acceptance fix): a ticket's org unit is no schema property,
// so no mask names it. Record history never shows it, and an audit list entry
// hides it when the caller's policy masks the requester.

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// seedUnit makes a unit in the tenant and the principal its primary member,
// so a record that principal requests is created with org_unit_id set.
func (e *etEnv) seedUnit(t *testing.T, tenant, principal uuid.UUID) uuid.UUID {
	t.Helper()
	unit := uuid.New()
	if _, err := e.super.Exec(t.Context(), `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, $3, $3)`,
		tenant, unit, "u-"+unit.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary) VALUES ($1, $2, $3, true)`,
		tenant, unit, principal); err != nil {
		t.Fatal(err)
	}
	return unit
}

func requesterBody(requester uuid.UUID) string {
	return `{"short_description": "s", "requester": "` + requester.String() + `"}`
}

// orgUnitOp returns the value of the /org_unit_id op of a diff, and whether
// the diff has one.
func orgUnitOp(t *testing.T, diff json.RawMessage) (value string, ok bool) {
	t.Helper()
	var ops []struct {
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(diff, &ops); err != nil {
		t.Fatalf("diff %s: %v", diff, err)
	}
	for _, op := range ops {
		if op.Path == "/org_unit_id" {
			return string(op.Value), true
		}
	}
	return "", false
}

// covers: listAuditEntries 200
func TestAuditListMasksOrgUnitWithTheRequester(t *testing.T) {
	const redacted = `{"redacted":"mask"}`
	for _, tc := range []struct {
		name    string
		masks   func(m *maskEnv) []authz.Mask
		visible bool
	}{
		{"masks the requester", (*maskEnv).deskMasks, false},
		{"masks another property only", func(m *maskEnv) []authz.Mask {
			return []authz.Mask{{Field: "cost", Read: authz.Subject{Groups: []uuid.UUID{m.desk}}}}
		}, true},
		{"masks nothing", func(*maskEnv) []authz.Mask { return nil }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMaskEnv(t)
			unit := m.seedUnit(t, m.a.id, m.a.user.PrincipalID)
			m.publishMasks(t, tc.masks(m))
			r := m.mustCreateRec(t, "lookalike-a", "job", "ou-1", requesterBody(m.a.user.PrincipalID))
			if rec := m.patch(t, "lookalike-a", "job", r.ID, `W/"1"`, `{"data": {"cost": 2}}`); rec.Code != http.StatusOK {
				t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
			}
			rec, page := m.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
			if rec.Code != http.StatusOK || len(page.Items) != 2 {
				t.Fatalf("audit = %d, %d items: %s", rec.Code, len(page.Items), rec.Body)
			}
			seen := false
			for _, it := range page.Items {
				if it.Action != "create" {
					continue
				}
				seen = true
				val, ok := orgUnitOp(t, it.Diff)
				switch {
				case !ok:
					t.Fatalf("the create entry has no org_unit_id op: %s", it.Diff)
				case tc.visible && (val != `"`+unit.String()+`"` || it.DiffSalt == ""):
					t.Errorf("create: org_unit_id = %s salt %q, want the unit and a salt", val, it.DiffSalt)
				case !tc.visible && (val != redacted || it.DiffSalt != "" || !it.DiffRedacted):
					t.Errorf("create: org_unit_id = %s salt %q redacted %v, want %s, no salt, redacted", val, it.DiffSalt, it.DiffRedacted, redacted)
				}
			}
			if !seen {
				t.Fatalf("no create entry: %s", rec.Body)
			}
			if got := strings.Contains(rec.Body.String(), unit.String()); got != tc.visible {
				t.Errorf("unit id in the response = %v, want %v: %s", got, tc.visible, rec.Body)
			}
		})
	}
}

// covers: listAuditEntries 200
func TestAuditListHasNoOrgUnitKeyForARecordWithoutAUnit(t *testing.T) {
	m := newMaskEnv(t)
	r := m.mustCreateRec(t, "lookalike-a", "job", "ou-none", requesterBody(m.a.user.PrincipalID))
	_, page := m.auditGet(t, "admin-a", "?domain=record&subject_type=record&subject_id="+r.ID)
	if len(page.Items) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(page.Items))
	}
	if _, ok := orgUnitOp(t, page.Items[0].Diff); ok || strings.Contains(string(page.Items[0].Diff), "org_unit_id") {
		t.Errorf("a record with no unit has an org_unit_id op: %s", page.Items[0].Diff)
	}
}

// Tenants A and B have a type with the same key; A's policy masks the
// requester and B's does not.
//
// covers: listAuditEntries 200
func TestAuditListOrgUnitMaskIsPerTenant(t *testing.T) {
	m := newMaskEnv(t)
	if rec := m.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "mk-b", maskTypeBody("job", "JOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant B type = %d: %s", rec.Code, rec.Body)
	}
	unitA := m.seedUnit(t, m.a.id, m.a.user.PrincipalID)
	unitB := m.seedUnit(t, m.b.id, m.b.user.PrincipalID)
	ra := m.mustCreateRec(t, "user-a", "job", "ou-a", requesterBody(m.a.user.PrincipalID))
	rb := m.mustCreateRec(t, "user-b", "job", "ou-b", requesterBody(m.b.user.PrincipalID))

	recA, _ := m.auditGet(t, "admin-a", "?domain=record&subject_type=record")
	recB, _ := m.auditGet(t, "admin-b", "?domain=record&subject_type=record")
	if recA.Code != http.StatusOK || recB.Code != http.StatusOK {
		t.Fatalf("audit = %d, %d", recA.Code, recB.Code)
	}
	a, b := recA.Body.String(), recB.Body.String()
	if !strings.Contains(a, ra.ID) || strings.Contains(a, rb.ID) || strings.Contains(a, unitA.String()) || strings.Contains(a, unitB.String()) {
		t.Errorf("tenant A's auditor: want its own entry with the unit redacted and nothing of B: %s", a)
	}
	if !strings.Contains(b, rb.ID) || strings.Contains(b, ra.ID) || !strings.Contains(b, unitB.String()) || strings.Contains(b, unitA.String()) {
		t.Errorf("tenant B's auditor: want its own entry with the unit visible and nothing of A: %s", b)
	}
}

// covers: getRecordHistory 200
func TestRecordHistoryNeverShowsTheOrgUnit(t *testing.T) {
	for _, tc := range []struct{ name, token string }{
		{"a reader with no masks", "lookalike-a"},
		{"a reader whose policy masks the requester", "admin-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMaskEnv(t)
			unit := m.seedUnit(t, m.a.id, m.a.user.PrincipalID)
			r := m.mustCreateRec(t, "lookalike-a", "job", "ou-h", requesterBody(m.a.user.PrincipalID))
			if rec := m.patch(t, "lookalike-a", "job", r.ID, `W/"1"`, `{"data": {"cost": 2}}`); rec.Code != http.StatusOK {
				t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
			}
			rec := m.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID+"/history", tc.token, "", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("history = %d: %s", rec.Code, rec.Body)
			}
			if strings.Contains(rec.Body.String(), unit.String()) || strings.Contains(rec.Body.String(), "org_unit_id") {
				t.Errorf("the unit is in the history: %s", rec.Body)
			}
			var h historyBody
			if err := json.Unmarshal(rec.Body.Bytes(), &h); err != nil || len(h.Items) != 2 {
				t.Fatalf("history = %v, %d items: %s", err, len(h.Items), rec.Body)
			}
			create, update := h.Items[1], h.Items[0]
			if create.After["short_description"] != "s" || len(create.Changed) == 0 {
				t.Errorf("the create lost its other fields: %+v", create)
			}
			if update.Before == nil || (*update.Before)["short_description"] != "s" || update.After["short_description"] != "s" {
				t.Errorf("the update lost its other fields: %+v", update)
			}
		})
	}
}

func inTx(t *testing.T, e *etEnv, tenant uuid.UUID, f func(ctx context.Context, tx pgx.Tx) error) error {
	t.Helper()
	return tenancy.InTenantTx(t.Context(), e.pool, tenant, func(tx pgx.Tx) error { return f(t.Context(), tx) })
}

func TestAddOrgUnitMasks(t *testing.T) {
	m := newMaskEnv(t)
	if rec := m.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "mk-b", maskTypeBody("job", "JOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant B type = %d: %s", rec.Code, rec.Body)
	}
	var defA, defB uuid.UUID
	for tenant, dst := range map[uuid.UUID]*uuid.UUID{m.a.id: &defA, m.b.id: &defB} {
		if err := m.super.QueryRow(t.Context(), `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'job'`, tenant).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	run := func(tenant uuid.UUID, masks map[uuid.UUID][]string) error {
		return inTx(t, m.etEnv, tenant, func(ctx context.Context, tx pgx.Tx) error {
			return server.AddOrgUnitMasks(ctx, tx, tenant, masks)
		})
	}

	t.Run("appends when the requester is masked", func(t *testing.T) {
		masks := map[uuid.UUID][]string{defA: {"requester"}}
		if err := run(m.a.id, masks); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(masks[defA], ","); got != "requester,org_unit_id" {
			t.Errorf("masks = %s", got)
		}
	})
	t.Run("does not append when another property is masked", func(t *testing.T) {
		masks := map[uuid.UUID][]string{defA: {"cost"}}
		if err := run(m.a.id, masks); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(masks[defA], ","); got != "cost" {
			t.Errorf("masks = %s", got)
		}
	})
	t.Run("a definition with no version row is an error", func(t *testing.T) {
		if err := run(m.a.id, map[uuid.UUID][]string{uuid.New(): {"requester"}}); err == nil {
			t.Error("no error for a definition with no version row")
		}
	})
	t.Run("a cancelled context is an error", func(t *testing.T) {
		masks := map[uuid.UUID][]string{defA: {"requester"}}
		err := inTx(t, m.etEnv, m.a.id, func(_ context.Context, tx pgx.Tx) error {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			return server.AddOrgUnitMasks(ctx, tx, m.a.id, masks)
		})
		if err == nil {
			t.Error("no error for a cancelled context")
		}
	})
	t.Run("another tenant's definition is an error and nothing is appended", func(t *testing.T) {
		masks := map[uuid.UUID][]string{defB: {"requester"}}
		if err := run(m.a.id, masks); err == nil {
			t.Error("no error for tenant B's definition in tenant A's transaction")
		}
		if got := strings.Join(masks[defB], ","); got != "requester" {
			t.Errorf("masks = %s, want unchanged", got)
		}
	})
	t.Run("empty masks: no error and no query", func(t *testing.T) {
		// A cancelled context and a nil tx would fail any query.
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := server.AddOrgUnitMasks(ctx, nil, m.a.id, map[uuid.UUID][]string{}); err != nil {
			t.Errorf("err = %v", err)
		}
		if err := server.AddOrgUnitMasks(ctx, nil, m.a.id, nil); err != nil {
			t.Errorf("nil masks: err = %v", err)
		}
	})
}
