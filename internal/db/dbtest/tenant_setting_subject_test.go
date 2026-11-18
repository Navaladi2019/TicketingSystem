//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-854 (for TICK-672, ADR-005 §3.6): the tenant_setting audit subject.

// tenantSettingBefore is the version just before the TICK-854 migrations.
const tenantSettingBefore = 20261114085959

// Acceptance: the audit CHECK accepts a tenant_setting entry of the shape the
// time-zone route writes (domain config, subject_id the tenant id, no
// versions, content_sha256 of the after document) and still refuses an
// unknown subject type; it ends validated.
func TestTick854AuditSubjectTenantSetting(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	const entry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		version_before, version_after, actor_system, source, content_sha256)
		VALUES ($1, gen_random_uuid(), 'config', 'update', $2, $1, NULL, NULL, 'system', 'system',
		        sha256(convert_to('{"time_zone":"Asia/Kolkata"}', 'UTF8')))`
	if _, err := super.Exec(context.Background(), entry, tenantA, "tenant_setting"); err != nil {
		t.Errorf("audit entry with subject_type tenant_setting: %v", err)
	}
	_, err := super.Exec(context.Background(), entry, tenantA, "tenant_settings")
	if code, con := pgErrDetail(err); code != "23514" || con != "audit_log_subject_type_check" {
		t.Errorf("audit entry with subject_type tenant_settings: %v (constraint %q), want 23514 on audit_log_subject_type_check", err, con)
	}
	if got := queryString(t, super, `SELECT convalidated::text FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_subject_type_check'`); got != "true" {
		t.Errorf("audit_log_subject_type_check validated = %s, want true", got)
	}
}

// Acceptance: the migrations go down (the CHECK no longer names
// tenant_setting but keeps number_format) and up again (validated).
func TestTick854Migration(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	super := superNava(t)
	const def = `SELECT pg_get_constraintdef(oid) || ' valid=' || convalidated FROM pg_constraint
		WHERE conrelid = 'audit_log'::regclass AND conname = 'audit_log_subject_type_check'`

	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations(),
		goose.WithAllowOutofOrder(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := p.DownTo(ctx, tenantSettingBefore); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := queryString(t, super, def); strings.Contains(got, "tenant_setting") || !strings.Contains(got, "number_format") {
		t.Errorf("after down: audit_log_subject_type_check = %s, want number_format without tenant_setting", got)
	}
	if _, err := p.Up(ctx); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := queryString(t, super, def); !strings.Contains(got, "'tenant_setting'::text") || !strings.HasSuffix(got, "valid=true") {
		t.Errorf("after up again: audit_log_subject_type_check = %s, want tenant_setting, validated", got)
	}
}
