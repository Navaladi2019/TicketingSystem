//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-1025 (S32, ADR-010 D9): the audit action policy_ref_refused, for a
// workflow condition step's refused named-policy reference.

// policyRefRefusedBefore is the version just before the TICK-1025 audit
// migrations (A 20261120160100, B 20261120160200).
const policyRefRefusedBefore = 20261120160099

// policyRefRefusedRow is a config-domain entry with the new action on the
// entity type's access policy, as TICK-957 writes it.
// A config entry that is not a delete carries content_sha256
// (audit_log_config_content_check).
const policyRefRefusedRow = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
	actor_system, source, content_sha256)
	VALUES ($1, gen_random_uuid(), 'config', 'policy_ref_refused', 'acl_policy', gen_random_uuid(),
	        'system', 'workflow', sha256('ref'::bytea))`

// Before A the CHECK refuses the action with 23514; after B it accepts it
// and is validated on the parent and every partition; A's Down refuses while
// such a row exists and goes down cleanly while none does.
func TestTICK1025PolicyRefRefusedAction(t *testing.T) {
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

	if _, err := super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		actor_system, source) VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', gen_random_uuid(),
		'system', 'system')`, tenantA); err != nil {
		t.Fatalf("seed update entry: %v", err)
	}
	if _, err := p.DownTo(ctx, policyRefRefusedBefore); err != nil {
		t.Fatalf("down with no policy_ref_refused row: %v", err)
	}
	_, err = super.Exec(ctx, policyRefRefusedRow, tenantA)
	if code, con := pgErrDetail(err); code != "23514" || con != "audit_log_action_check" {
		t.Fatalf("before A: insert policy_ref_refused: %v (constraint %q), want 23514 on audit_log_action_check", err, con)
	}
	if got := queryString(t, super, `SELECT relforcerowsecurity::text FROM pg_class WHERE oid = 'audit_log'::regclass`); got != "true" {
		t.Errorf("after down: audit_log FORCE ROW LEVEL SECURITY = %s, want true", got)
	}

	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if _, err := super.Exec(ctx, policyRefRefusedRow, tenantA); err != nil {
		t.Fatalf("after B: insert policy_ref_refused: %v", err)
	}
	if got := queryString(t, super, actionCheckNotValidated); got != "" {
		t.Errorf("after B: audit_log_action_check missing or not validated on: %s", got)
	}
	// An unregistered action is still refused.
	_, err = super.Exec(ctx, strings.Replace(policyRefRefusedRow, "'policy_ref_refused'", "'policy_ref_other'", 1), tenantA)
	if code, con := pgErrDetail(err); code != "23514" || con != "audit_log_action_check" {
		t.Errorf("after B: insert unregistered action: %v (constraint %q), want 23514", err, con)
	}

	if _, err := p.DownTo(ctx, policyRefRefusedBefore); err == nil ||
		!strings.Contains(err.Error(), "audit_log has policy_ref_refused rows") {
		t.Fatalf("down with a policy_ref_refused row: %v, want A's exception", err)
	}
	const def = `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_action_check'`
	if got := queryString(t, super, def); !strings.Contains(got, "'policy_ref_refused'::text") {
		t.Errorf("after refused down: audit_log_action_check = %s, want it to keep policy_ref_refused", got)
	}
	if got := queryString(t, super, `SELECT relforcerowsecurity::text FROM pg_class WHERE oid = 'audit_log'::regclass`); got != "true" {
		t.Errorf("after refused down: audit_log FORCE ROW LEVEL SECURITY = %s, want true", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up after refused down: %v", err)
	}
}
