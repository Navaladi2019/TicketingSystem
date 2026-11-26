//go:build integration

// TICK-949 (ADR-010 D8): the write pipeline sets records.org_unit_id on
// create from the requester's live primary unit, and after that it is a
// stored field.

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// seedOrgUnit inserts a unit of tenant and returns its id.
func seedOrgUnit(t *testing.T, e env, tenant uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, $3, $4)`,
		tenant, id, "kc-"+id.String(), name)
	return id
}

func seedMember(t *testing.T, e env, tenant, unit, principal uuid.UUID, primary bool) {
	t.Helper()
	mustExec(t, e.super, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary) VALUES ($1, $2, $3, $4)`,
		tenant, unit, principal, primary)
}

func recordUnit(t *testing.T, e env, id uuid.UUID) *uuid.UUID {
	t.Helper()
	var u *uuid.UUID
	if err := e.super.QueryRow(context.Background(), `SELECT org_unit_id FROM records WHERE id = $1`, id).Scan(&u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestCreateSetsOrgUnitFromRequestersPrimaryUnit(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	primary, other := seedOrgUnit(t, e, f.id, "Primary"), seedOrgUnit(t, e, f.id, "Other")
	seedMember(t, e, f.id, other, f.principal.PrincipalID, false)
	seedMember(t, e, f.id, primary, f.principal.PrincipalID, true)

	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != primary {
		t.Fatalf("org_unit_id = %v, want %s", got, primary)
	}
	// The audit entry of the create includes org_unit_id.
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1 AND action = 'create' AND diff::text LIKE '%org_unit_id%' AND diff::text LIKE '%' || $2::text || '%'`,
		res.RecordID, primary.String()); n != 1 {
		t.Fatalf("%d create audit entries name org_unit_id, want 1", n)
	}
}

func TestCreateWithoutPrimaryUnitLeavesOrgUnitNull(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	// A non-primary membership is not a primary unit.
	seedMember(t, e, f.id, seedOrgUnit(t, e, f.id, "Other"), f.principal.PrincipalID, false)

	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	if got := recordUnit(t, e, res.RecordID); got != nil {
		t.Fatalf("org_unit_id = %v, want NULL", *got)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1 AND diff::text LIKE '%org_unit_id%'`, res.RecordID); n != 0 {
		t.Fatalf("the audit entry names org_unit_id for a NULL unit")
	}
}

func TestCreateRefusesOrgUnitInBody(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	unit := seedOrgUnit(t, e, f.id, "Primary")
	seedMember(t, e, f.id, unit, f.principal.PrincipalID, true)

	_, err := write(context.Background(), e, p, f.id, createCmd(f, map[string]any{"title": "x", "org_unit_id": uuid.NewString()}))
	var nw *authz.ErrFieldNotWritable
	if !errors.As(err, &nw) || len(nw.Fields) != 1 || nw.Fields[0] != "org_unit_id" {
		t.Fatalf("create error = %v, want ErrFieldNotWritable{org_unit_id}", err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id); n != 0 {
		t.Fatalf("%d records, want 0", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'`, f.id); n != 0 {
		t.Fatalf("%d record audit entries, want 0", n)
	}

	// An update cannot set it through the data either.
	res := mustCreate(t, e, p, f, map[string]any{"title": "ok"})
	_, err = write(context.Background(), e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"org_unit_id": nil}))
	if !errors.As(err, &nw) {
		t.Fatalf("update error = %v, want ErrFieldNotWritable", err)
	}
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != unit {
		t.Fatalf("org_unit_id = %v after a refused update, want %s", got, unit)
	}
}

func TestOrgUnitIdempotentReplayOneRecord(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	unit := seedOrgUnit(t, e, f.id, "Primary")
	seedMember(t, e, f.id, unit, f.principal.PrincipalID, true)

	// An idempotency key is for system writers (a workflow step); the
	// requester is named in the body, so the record still gets its unit.
	cmd := createCmd(f, map[string]any{"title": "once", "requester": f.principal.PrincipalID.String()})
	cmd.Actor, cmd.Source, cmd.IdempotencyKey = f.system(), pipeline.SourceWorkflow, "wf:w949:s1"
	cmd.Causation = workflowCausation()
	first, err := write(context.Background(), e, p, f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	again, err := write(context.Background(), e, p, f.id, cmd)
	if err != nil || again != first {
		t.Fatalf("replay = %+v, %v; want %+v", again, err, first)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id); n != 1 {
		t.Fatalf("%d records, want 1", n)
	}
	if got := recordUnit(t, e, first.RecordID); got == nil || *got != unit {
		t.Fatalf("org_unit_id = %v, want %s", got, unit)
	}
}

func TestOrgUnitStaysWhenRequesterMoves(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	a, b := seedOrgUnit(t, e, f.id, "A"), seedOrgUnit(t, e, f.id, "B")
	seedMember(t, e, f.id, a, f.principal.PrincipalID, true)
	res := mustCreate(t, e, p, f, map[string]any{"title": "stays"})

	mustExec(t, e.super, `DELETE FROM org_unit_members WHERE tenant_id = $1 AND principal_id = $2`, f.id, f.principal.PrincipalID)
	seedMember(t, e, f.id, b, f.principal.PrincipalID, true)

	// An update by the moved requester, and a read, leave the unit alone.
	if _, err := write(context.Background(), e, p, f.id, updateCmd(f, res.RecordID, res.Version, map[string]any{"summary": "later"})); err != nil {
		t.Fatal(err)
	}
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		rec, err := store.Get(context.Background(), tx, f.id, res.RecordID)
		if err != nil {
			return err
		}
		if rec.OrgUnitID == nil || *rec.OrgUnitID != a {
			t.Errorf("read org_unit_id = %v, want %s", rec.OrgUnitID, a)
		}
		return nil
	})
	if got := recordUnit(t, e, res.RecordID); got == nil || *got != a {
		t.Fatalf("org_unit_id = %v, want %s", got, a)
	}
}

// The pipeline reads membership under the tenant's RLS: another tenant's
// membership is never read, and the FK refuses a record of this tenant that
// names another tenant's unit.
func TestOrgUnitNeverReadsAnotherTenantsMembership(t *testing.T) {
	e := migrated(t)
	f, g := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	// Tenant g has a principal with f's requester id, a primary member of
	// g's unit.
	foreign := seedOrgUnit(t, e, g.id, "Foreign")
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		g.id, f.principal.PrincipalID, "kc-twin-"+f.principal.PrincipalID.String())
	seedMember(t, e, g.id, foreign, f.principal.PrincipalID, true)

	res := mustCreate(t, e, p, f, map[string]any{"title": "isolated"})
	if got := recordUnit(t, e, res.RecordID); got != nil {
		t.Fatalf("org_unit_id = %v, want NULL: another tenant's membership was read", *got)
	}

	// The FK: a record of f naming g's unit cannot exist.
	_, err := e.super.Exec(context.Background(), `UPDATE records SET org_unit_id = $1 WHERE tenant_id = $2 AND id = $3`, foreign, f.id, res.RecordID)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23503" {
		t.Fatalf("a record of tenant f naming tenant g's org unit: err = %v, want SQLSTATE 23503", err)
	}
}

// The gapless insert and the at-commit insert are other statements than the
// default one: each stores the unit and names it in the audit entry.
func TestOrgUnitOnEveryInsertPath(t *testing.T) {
	cases := []struct {
		name     string
		gapless  bool
		atCommit bool
	}{
		{"gaps", false, false},
		{"gapless", true, false},
		{"gaps at commit", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := migrated(t)
			d := definition(schemaV1)
			if c.gapless {
				d = gaplessDefinition(d)
			}
			f := seedTenantWith(t, e, d)
			p := newPipeline(e, nil)
			unit := seedOrgUnit(t, e, f.id, "Primary")
			seedMember(t, e, f.id, unit, f.principal.PrincipalID, true)

			cmd := createCmd(f, map[string]any{"title": "path"})
			cmd.InsertAtCommit = c.atCommit
			res, err := write(context.Background(), e, p, f.id, cmd)
			if err != nil {
				t.Fatal(err)
			}
			if got := recordUnit(t, e, res.RecordID); got == nil || *got != unit {
				t.Fatalf("org_unit_id = %v, want %s", got, unit)
			}
			if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1 AND action = 'create' AND diff::text LIKE '%' || $2::text || '%'`,
				res.RecordID, unit.String()); n != 1 {
				t.Fatalf("%d create audit entries name the unit, want 1", n)
			}
		})
	}
}
