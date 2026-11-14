//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-871 (ADR-004-E9, PLAN-005-E9 T-E9-1): the audit action
// number_space_exhausted.

// numberSpaceExhaustedBefore is the version just before the TICK-871
// migrations (A 20261115090000, B 20261115090100).
const numberSpaceExhaustedBefore = 20261115085959

// numberSpaceExhaustedRow is an audit entry with the new action; every other
// column is a valid record-domain entry (TestP510bAuditLogChecks' base).
const numberSpaceExhaustedRow = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
	actor_system, source)
	VALUES ($1, gen_random_uuid(), 'record', 'number_space_exhausted', 'record', gen_random_uuid(),
	        'system', 'system')`

// actionCheckNotValidated lists every audit_log table (the parent and each
// partition) whose audit_log_action_check is not validated, or lacks it.
const actionCheckNotValidated = `SELECT coalesce(string_agg(c.relname, ', ' ORDER BY c.relname), '')
	FROM pg_class c
	WHERE (c.oid = 'audit_log'::regclass
	       OR c.oid IN (SELECT inhrelid FROM pg_inherits WHERE inhparent = 'audit_log'::regclass))
	  AND NOT EXISTS (SELECT 1 FROM pg_constraint k
	                  WHERE k.conrelid = c.oid AND k.conname = 'audit_log_action_check' AND k.convalidated)`

// Acceptance (CI-A18 (e), [PG 2]): before migration A the CHECK refuses the
// action with 23514; after B it accepts it and is validated on the parent
// and every partition; A's Down then refuses with its exception while such a
// row exists, and goes down cleanly while none does.
func TestTick871NumberSpaceExhaustedAction(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	ctx := context.Background()

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}

	// Down with no number_space_exhausted row succeeds (the RLS-forced
	// table holds other rows the guard must not trip on).
	if _, err := super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		actor_system, source) VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', gen_random_uuid(),
		'system', 'system')`, tenantA); err != nil {
		t.Fatalf("seed update entry: %v", err)
	}
	if _, err := p.DownTo(ctx, numberSpaceExhaustedBefore); err != nil {
		t.Fatalf("down with no number_space_exhausted row: %v", err)
	}
	_, err = super.Exec(ctx, numberSpaceExhaustedRow, tenantA)
	if code, con := pgErrDetail(err); code != "23514" || con != "audit_log_action_check" {
		t.Fatalf("before A: insert number_space_exhausted: %v (constraint %q), want 23514 on audit_log_action_check", err, con)
	}
	if got := queryString(t, super, `SELECT relforcerowsecurity::text FROM pg_class WHERE oid = 'audit_log'::regclass`); got != "true" {
		t.Errorf("after down: audit_log FORCE ROW LEVEL SECURITY = %s, want true", got)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if _, err := super.Exec(ctx, numberSpaceExhaustedRow, tenantA); err != nil {
		t.Fatalf("after B: insert number_space_exhausted: %v", err)
	}
	if got := queryString(t, super, actionCheckNotValidated); got != "" {
		t.Errorf("after B: audit_log_action_check missing or not validated on: %s", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_inherits WHERE inhparent = 'audit_log'::regclass`); got == "0" {
		t.Errorf("audit_log has no partitions; the per-partition check proves nothing")
	}

	// Down with the row present: B's no-op Down passes, A's raises.
	if _, err := p.DownTo(ctx, numberSpaceExhaustedBefore); err == nil ||
		!strings.Contains(err.Error(), "audit_log has number_space_exhausted rows") {
		t.Fatalf("down with a number_space_exhausted row: %v, want A's exception", err)
	}
	const def = `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_action_check'`
	if got := queryString(t, super, def); !strings.Contains(got, "'number_space_exhausted'::text") {
		t.Errorf("after refused down: audit_log_action_check = %s, want it to keep number_space_exhausted", got)
	}
	if got := queryString(t, super, `SELECT relforcerowsecurity::text FROM pg_class WHERE oid = 'audit_log'::regclass`); got != "true" {
		t.Errorf("after refused down: audit_log FORCE ROW LEVEL SECURITY = %s, want true", got)
	}
	// Leave the cluster at the latest version for the next test.
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up after refused down: %v", err)
	}
}
