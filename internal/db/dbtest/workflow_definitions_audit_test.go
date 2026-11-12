//go:build integration

package dbtest

import (
	"testing"
)

// P3-41b (TICK-737; ADR-004 CI-A3): workflow_definitions and
// workflow_definition_versions carry the audit commit check. A raw write with
// no audit entry for the definition in the same transaction is refused at
// COMMIT with audit_missing, and the same write with its entry commits.

const (
	wfDefAuditedFixture = "0e000000-0000-4000-8000-0000000000f7"
	wfVerAuditedFixture = "0e000000-0000-4000-8000-0000000000f8"
)

func TestP341bWorkflowDefinitionWritesWithoutAnAuditEntryAreRefused(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	app := asRole(t, "nava_app")

	wantAuditMissing := func(what, constraint string, err error) {
		t.Helper()
		if code, c := pgErrDetail(err); code != "23514" || c != constraint {
			t.Errorf("%s with no audit entry: err = %v (constraint %q), want 23514 from %s", what, err, c, constraint)
		}
	}

	// A definition insert, then a version insert, and a current_version move,
	// each bare.
	_, err := execIn(app, tenantA, insertWorkflowDefinitionBare, tenantA, wfDefAuditedFixture, workflowKey(wfDefAuditedFixture))
	wantAuditMissing("creating a definition", "workflow_definitions_audit_commit_check", err)

	if _, err := execIn(app, tenantA, insertWorkflowDefinition, tenantA, wfDefAuditedFixture, workflowKey(wfDefAuditedFixture)); err != nil {
		t.Fatalf("creating a definition with its audit entry: %v", err)
	}
	_, err = execIn(app, tenantA, insertWorkflowVersionBare, tenantA, wfVerAuditedFixture, wfDefAuditedFixture, 1, wfBody, schemaSum(wfBody), tenantA, tenantA)
	wantAuditMissing("publishing a version", "workflow_definition_versions_audit_commit_check", err)
	_, err = execIn(app, tenantA, "UPDATE workflow_definitions SET current_version = 2, updated_at = now() WHERE tenant_id = $1 AND id = $2",
		tenantA, wfDefAuditedFixture)
	wantAuditMissing("moving current_version", "workflow_definitions_audit_commit_check", err)

	if _, err := execIn(app, tenantA, insertWorkflowVersion, tenantA, wfVerAuditedFixture, wfDefAuditedFixture, 1, wfBody, schemaSum(wfBody), tenantA, tenantA); err != nil {
		t.Fatalf("publishing a version with its audit entry: %v", err)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM workflow_definition_versions WHERE definition_id = $1`, wfDefAuditedFixture); got != "1" {
		t.Errorf("versions after the refused and the accepted publish = %s, want 1 (the bare one wrote nothing)", got)
	}
}
