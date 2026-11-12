//go:build integration

package dbtest

import (
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// P3-40 (TICK-709): workflow_definitions, workflow_definition_versions,
// workflow_cause, cascade_start's definition foreign key and workflow_ref's
// waiting state (S11b contract, D-3-29).

const (
	cascadeDefinitionMigration = "20261111123330_cascade_start_definition_fkey.sql"
	cascadeValidateMigration   = "20261111123340_cascade_start_definition_fkey_validate.sql"

	wfDefA   = "0e000000-0000-4000-8000-0000000000f1" // tenant A's definition
	wfDefB   = "0e000000-0000-4000-8000-0000000000f2" // tenant B's definition
	wfVerA   = "0e000000-0000-4000-8000-0000000000f3"
	wfVerB   = "0e000000-0000-4000-8000-0000000000f4"
	wfCause  = "0e000000-0000-4000-8000-0000000000f5" // a delivery id in both tenants
	wfSignal = "0e000000-0000-4000-8000-0000000000f6"
	wfBody   = `{"format": "v1", "trigger": {"on": "record.created", "entity": "ticket"}}`

	// insertWorkflowDefinitionBare: tenant $1, id $2, key $3; the entity type
	// and the creator are seedEntityTypes' (id = the tenant id). It writes no
	// audit entry, so it commits only where the commit check is absent or
	// refused (P3-41b); insertWorkflowDefinition is the one that commits.
	insertWorkflowDefinitionBare = `INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
		VALUES ($1, $2, $3, $1, 1, $1)`
	insertWorkflowDefinition = `WITH r AS (` + insertWorkflowDefinitionBare + ` RETURNING tenant_id, id) ` + workflowDefinitionEntry
	// insertWorkflowVersionBare: tenant $1, id $2, definition $3, version $4,
	// body $5, body_sha256 $6, trigger entity $7, publisher $8.
	insertWorkflowVersionBare = `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256,
		step_ids, trigger_on, trigger_entity_definition_id, published_by)
		VALUES ($1, $2, $3, $4, $5::json, $6, '{"start": "0e000000-0000-4000-8000-0000000000aa"}', 'record.created', $7, $8)`
	insertWorkflowVersion = `WITH r AS (` + insertWorkflowVersionBare + ` RETURNING tenant_id, definition_id, version) ` + workflowVersionEntry
	// moveWorkflowCurrentVersion: tenant $1, definition $2, to version 1, with
	// the audit entry the commit check wants (P3-41b).
	moveWorkflowCurrentVersion = `WITH r AS (UPDATE workflow_definitions SET current_version = 1, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 RETURNING tenant_id, id AS definition_id, current_version AS version) ` + workflowVersionEntry
	insertWorkflowCause = `INSERT INTO workflow_cause (tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth)
		VALUES ($1, $2, $3, $4, $5)`
)

// workflowDefinitionEntry and workflowVersionEntry write the audit entry of
// each row of the CTE r, so the fixture's COMMIT passes the commit check the
// two tables have since P3-41b (the shape of aclEntry). A definition's entry
// has no version_after and a version's carries its number, so a transaction
// that writes both keeps R2's (subject, version_after) key unique. Fixtures
// that can go through workflowdefs.Store.Publish do; these are for the tests
// that need a raw row.
const (
	workflowDefinitionEntry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
			actor_system, source, content_sha256)
		SELECT tenant_id, gen_random_uuid(), 'config', 'create', 'workflow_definition', id,
			'system', 'system', sha256('workflow_definition') FROM r`
	workflowVersionEntry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
			version_after, actor_system, source, content_sha256)
		SELECT tenant_id, gen_random_uuid(), 'config', 'publish', 'workflow_definition', definition_id,
			version, 'system', 'system', sha256('workflow_definition_version') FROM r`
)

const (
	workflowTablesMigration = "20261111123300_workflow_definitions.sql"
	waitingColumnMigration  = "20261111123350_workflow_ref_record_changed_signal.sql"
	// definitionAuditMigration attaches the commit check to the two definition
	// tables (P3-41b).
	definitionAuditMigration = "20261112180000_workflow_definitions_audit_commit_check.sql"
)

// workflowTables are the tables workflowTablesMigration creates.
var workflowTables = []string{"workflow_definitions", "workflow_definition_versions", "workflow_cause"}

// beforeWorkflowTables is m and a as a run of fsys sees them when fsys stops
// before P3-40: without the three tables when fsys lacks
// workflowTablesMigration, and without nava_worker's UPDATE of
// workflow_ref.record_changed_signal_id when it lacks waitingColumnMigration.
// beforeNumberingMode calls it, so every older-schema run gets it. A run that
// has the tables but stops before definitionAuditMigration sees the two
// definition tables as not attached, as the manifest said before P3-41b.
func beforeWorkflowTables(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, definitionAuditMigration); err != nil {
		am := *a
		am.Audited = maps.Clone(a.Audited)
		for _, tbl := range workflowTables {
			if e, ok := am.Audited[tbl]; ok {
				e.Attached = false
				am.Audited[tbl] = e
			}
		}
		a = &am
	}
	_, errTables := fs.Stat(fsys, workflowTablesMigration)
	_, errColumn := fs.Stat(fsys, waitingColumnMigration)
	if errTables == nil && errColumn == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := make(map[string]map[string][]string, len(r.Tables))
		for tbl, privs := range r.Tables {
			if errTables != nil && slices.Contains(workflowTables, tbl) {
				continue
			}
			if tbl == "workflow_ref" {
				p := make(map[string][]string, len(privs))
				for priv, cols := range privs {
					p[priv] = slices.DeleteFunc(slices.Clone(cols), func(c string) bool { return c == "record_changed_signal_id" })
				}
				privs = p
			}
			tables[tbl] = privs
		}
		r.Tables = tables
		gm.Roles[name] = r
	}
	if errTables == nil {
		return &gm, a
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	am.Exempt = maps.Clone(a.Exempt)
	for _, tbl := range workflowTables {
		delete(am.Audited, tbl)
		delete(am.Exempt, tbl)
	}
	return &gm, &am
}

// seedWorkflowDefinition writes, as the superuser, a definition with id for
// each tenant (which needs seedEntityTypes' rows), keyed by its id.
func seedWorkflowDefinition(t *testing.T, super *pgx.Conn, id string, tenants ...string) {
	t.Helper()
	for _, tenant := range tenants {
		exec(t, super, insertWorkflowDefinition, tenant, id, workflowKey(id))
	}
}

// workflowKey is a workflow_definitions_key_check-valid key unique to id.
func workflowKey(id string) string { return "wf_" + strings.ReplaceAll(id, "-", "") }

func TestP340WorkflowDefinitionsTenantIsolation(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	// Each tenant publishes its own definition and version through its own
	// transaction, as nava_app.
	for _, s := range []struct{ tenant, def, ver string }{{tenantA, wfDefA, wfVerA}, {tenantB, wfDefB, wfVerB}} {
		if _, err := execIn(app, s.tenant, insertWorkflowDefinition, s.tenant, s.def, "triage"); err != nil {
			t.Fatalf("tenant %s creates its definition: %v", s.tenant, err)
		}
		if _, err := execIn(app, s.tenant, insertWorkflowVersion, s.tenant, s.ver, s.def, 1, wfBody, schemaSum(wfBody), s.tenant, s.tenant); err != nil {
			t.Fatalf("tenant %s publishes version 1: %v", s.tenant, err)
		}
		if _, err := execIn(app, s.tenant, moveWorkflowCurrentVersion, s.tenant, s.def); err != nil {
			t.Fatalf("tenant %s moves current_version: %v", s.tenant, err)
		}
		if _, err := execIn(app, s.tenant, insertWorkflowCause, s.tenant, wfCause, s.def, originX, 1); err != nil {
			t.Fatalf("tenant %s records a cause: %v", s.tenant, err)
		}
	}

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, q := range []string{
			"SELECT count(*) FROM workflow_definitions WHERE tenant_id <> $1",
			"SELECT count(*) FROM workflow_definition_versions WHERE tenant_id <> $1",
			"SELECT count(*) FROM workflow_cause WHERE tenant_id <> $1",
		} {
			var n int
			if err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
				return tx.QueryRow(t.Context(), q, tenantA).Scan(&n)
			}); err != nil || n != 0 {
				t.Errorf("%s in tenant A: %s = %d, %v; want 0", role, q, n, err)
			}
			// The control: tenant A's own row is there.
			if err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
				return tx.QueryRow(t.Context(), strings.Replace(q, "<>", "=", 1), tenantA).Scan(&n)
			}); err != nil || n != 1 {
				t.Errorf("control, %s in tenant A reads its own row: %d, %v; want 1", role, n, err)
			}
			if err := inTenantTx(conn, "", func(tx pgx.Tx) error {
				return tx.QueryRow(t.Context(), q, tenantA).Scan(&n)
			}); err != nil || n != 0 {
				t.Errorf("%s with no tenant: %s = %d, %v; want 0", role, q, n, err)
			}
		}

		// Inserts for tenant B from tenant A's transaction: the policy holds the tenant.
		_, err := execIn(conn, tenantA, insertWorkflowCause, tenantB, newUUID(t, super), wfDefB, originX, 1)
		wantSQLState(t, role+" in tenant A records a cause for tenant B", err, "42501")
	}

	_, err := execIn(app, tenantA, insertWorkflowDefinition, tenantB, newUUID(t, super), "iso_new")
	wantSQLState(t, "nava_app in tenant A creates a definition for tenant B", err, "42501")
	_, err = execIn(app, tenantA, insertWorkflowVersion, tenantB, newUUID(t, super), wfDefB, 2, wfBody, schemaSum(wfBody), tenantB, tenantB)
	wantSQLState(t, "nava_app in tenant A publishes for tenant B", err, "42501")
	if n, err := execIn(app, tenantA, "UPDATE workflow_definitions SET current_version = 9 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_app in tenant A moves tenant B's current_version: %d rows, %v; want 0", n, err)
	}

	// Reference tenant B's rows from tenant A's own rows: the composite
	// foreign keys hold the tenant, though the policy passes each row.
	for name, args := range map[string][]any{
		"a version of tenant B's definition":            {tenantA, newUUID(t, super), wfDefB, 2, wfBody, schemaSum(wfBody), tenantA, tenantA},
		"a version triggered by tenant B's entity type": {tenantA, newUUID(t, super), wfDefA, 2, wfBody, schemaSum(wfBody), tenantB, tenantA},
		"a version published by tenant B's principal":   {tenantA, newUUID(t, super), wfDefA, 2, wfBody, schemaSum(wfBody), tenantA, tenantB},
	} {
		_, err := execIn(app, tenantA, insertWorkflowVersion, args...)
		wantSQLState(t, "tenant A publishes "+name, err, "23503")
	}
	_, err = execIn(app, tenantA, `INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
		VALUES ($1, $2, 'iso_b_entity', $3, 1, $1)`, tenantA, newUUID(t, super), tenantB)
	wantSQLState(t, "tenant A creates a definition on tenant B's entity type", err, "23503")
	_, err = execIn(app, tenantA, `INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by)
		VALUES ($1, $2, 'iso_b_creator', $1, 1, $3)`, tenantA, newUUID(t, super), tenantB)
	wantSQLState(t, "tenant A creates a definition naming tenant B's principal", err, "23503")

	// The key is unique per tenant, not across tenants: both own triage.
	if got := queryString(t, super, "SELECT count(*)::text FROM workflow_definitions WHERE key = 'triage'"); got != "2" {
		t.Errorf("definitions keyed triage = %s, want one per tenant", got)
	}
	_, err = execIn(app, tenantA, insertWorkflowDefinition, tenantA, newUUID(t, super), "triage")
	wantSQLState(t, "a second triage in tenant A", err, "23505")

	for _, table := range []string{"workflow_definitions", "workflow_definition_versions", "workflow_cause"} {
		if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class
			WHERE oid = $1::regclass`, table); got != "truetrue" {
			t.Errorf("%s RLS enabled and forced = %s, want truetrue", table, got)
		}
	}
	for _, role := range []string{"nava_onboard", "nava_groupsync", "nava_relay", "nava_purge", "nava_audit_reader"} {
		for _, table := range []string{"workflow_definitions", "workflow_definition_versions", "workflow_cause"} {
			_, err := execIn(asRole(t, role), tenantA, "SELECT count(*) FROM "+table)
			wantSQLState(t, role+" reads "+table, err, "42501")
		}
	}
}

// Acceptance: a published version is immutable for every runtime role, a
// definition changes only current_version and updated_at, and a cause is
// insert-only. Nobody deletes any of them.
func TestP340WorkflowVersionsImmutable(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	seedWorkflowDefinition(t, super, wfDefA, tenantA)
	exec(t, super, insertWorkflowVersion, tenantA, wfVerA, wfDefA, 1, wfBody, schemaSum(wfBody), tenantA, tenantA)
	exec(t, super, insertWorkflowCause, tenantA, wfCause, wfDefA, originX, 1)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, q := range []string{
			"UPDATE workflow_definition_versions SET step_ids = '{}' WHERE tenant_id = $1",
			"UPDATE workflow_definition_versions SET body = '{}'::json, body_sha256 = sha256('{}') WHERE tenant_id = $1",
			"UPDATE workflow_definition_versions SET trigger_on = 'record.updated' WHERE tenant_id = $1",
			"DELETE FROM workflow_definition_versions WHERE tenant_id = $1",
			"DELETE FROM workflow_definitions WHERE tenant_id = $1",
			"UPDATE workflow_definitions SET key = 'renamed' WHERE tenant_id = $1",
			"UPDATE workflow_definitions SET entity_definition_id = tenant_id WHERE tenant_id = $1",
			"UPDATE workflow_cause SET causation_depth = 0 WHERE tenant_id = $1",
			"UPDATE workflow_cause SET ref_outbox_id = outbox_id WHERE tenant_id = $1",
			"DELETE FROM workflow_cause WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, q, tenantA)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	_, err := execIn(asRole(t, "nava_worker"), tenantA, "UPDATE workflow_definitions SET current_version = 2 WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "nava_worker moves current_version", err, "42501")
	_, err = execIn(asRole(t, "nava_worker"), tenantA, insertWorkflowVersion, tenantA, newUUID(t, super), wfDefA, 2, wfBody, schemaSum(wfBody), tenantA, tenantA)
	wantSQLState(t, "nava_worker publishes a version", err, "42501")

	if got := queryString(t, super, "SELECT step_ids::text || ' ' || body::text FROM workflow_definition_versions WHERE id = $1", wfVerA); got !=
		`{"start": "0e000000-0000-4000-8000-0000000000aa"} `+wfBody {
		t.Errorf("version after the refused writes = %s", got)
	}
}

// Acceptance: the body_sha256 CHECK refuses a digest that is not of body's
// exact bytes, including a body that differs only in whitespace or key
// order (json keeps both), and the other CHECKs refuse what they name.
func TestP340WorkflowVersionChecks(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	seedWorkflowDefinition(t, super, wfDefA, tenantA)
	app := asRole(t, "nava_app")

	reordered := `{"trigger": {"on": "record.created", "entity": "ticket"}, "format": "v1"}`
	for name, sum := range map[string][]byte{
		"another body's digest":         schemaSum(`{"format": "v1"}`),
		"the digest of reordered keys":  schemaSum(reordered),
		"the digest with no whitespace": schemaSum(strings.ReplaceAll(wfBody, " ", "")),
		"an empty digest":               {},
	} {
		_, err := execIn(app, tenantA, insertWorkflowVersion, tenantA, newUUID(t, super), wfDefA, 1, wfBody, sum, tenantA, tenantA)
		wantSQLState(t, "a version with "+name, err, "23514")
	}
	// The control: the digest of the exact bytes passes, and the bytes are kept.
	if _, err := execIn(app, tenantA, insertWorkflowVersion, tenantA, wfVerA, wfDefA, 1, wfBody, schemaSum(wfBody), tenantA, tenantA); err != nil {
		t.Fatalf("control: a version with its own digest: %v", err)
	}
	if got := queryString(t, super, "SELECT body::text FROM workflow_definition_versions WHERE id = $1", wfVerA); got != wfBody {
		t.Errorf("stored body = %s, want the published bytes %s", got, wfBody)
	}

	for name, q := range map[string]string{
		"version 0": `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256, step_ids,
			trigger_on, trigger_entity_definition_id, published_by) VALUES ($1, gen_random_uuid(), $2, 0, '{}', sha256('{}'), '{}', 'record.created', $1, $1)`,
		"a step_ids array": `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256, step_ids,
			trigger_on, trigger_entity_definition_id, published_by) VALUES ($1, gen_random_uuid(), $2, 5, '{}', sha256('{}'), '[]', 'record.created', $1, $1)`,
		"an unknown trigger": `INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256, step_ids,
			trigger_on, trigger_entity_definition_id, published_by) VALUES ($1, gen_random_uuid(), $2, 5, '{}', sha256('{}'), '{}', 'record.deleted', $1, $1)`,
		"current_version 0": "UPDATE workflow_definitions SET current_version = 0 WHERE tenant_id = $1 AND id = $2",
		"a bad key":         "INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version, created_by) VALUES ($1, gen_random_uuid(), 'Bad Key' || $2::text, $1, 1, $1)",
		"a negative depth":  "INSERT INTO workflow_cause (tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth) VALUES ($1, gen_random_uuid(), $2, $2, -1)",
	} {
		_, err := execIn(app, tenantA, q, tenantA, wfDefA)
		wantSQLState(t, name, err, "23514")
	}
}

// Acceptance: nava_worker updates workflow_ref.record_changed_signal_id (and
// closed_at, CloseWorkflowRef's) and no other workflow_ref column; nava_app
// updates none. The waiting index is partial and serves the sink's lookup.
func TestP340WorkflowRefWaitingState(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	ref := newUUID(t, super)
	for _, tenant := range []string{tenantA, tenantB} {
		exec(t, super, "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ($1, $2, $1, 'nava.interpreter')", tenant, ref)
	}
	worker := asRole(t, "nava_worker")

	if n, err := execIn(worker, tenantA, "UPDATE workflow_ref SET record_changed_signal_id = $2 WHERE tenant_id = $1 AND outbox_id = $3",
		tenantA, wfSignal, ref); err != nil || n != 1 {
		t.Fatalf("nava_worker sets the waiting state: %d rows, %v; want 1", n, err)
	}
	if got := queryString(t, super, "SELECT tenant_id::text FROM workflow_ref WHERE record_changed_signal_id IS NOT NULL"); got != tenantA {
		t.Errorf("waiting refs = tenant %s, want only tenant A's", got)
	}
	if n, err := execIn(worker, tenantA, "UPDATE workflow_ref SET record_changed_signal_id = $2 WHERE tenant_id = $1", tenantB, wfSignal); err != nil || n != 0 {
		t.Errorf("nava_worker in tenant A sets tenant B's waiting state: %d rows, %v; want 0", n, err)
	}
	if n, err := execIn(worker, tenantA, "UPDATE workflow_ref SET record_changed_signal_id = NULL WHERE tenant_id = $1 AND outbox_id = $2",
		tenantA, ref); err != nil || n != 1 {
		t.Errorf("nava_worker clears the waiting state: %d rows, %v; want 1", n, err)
	}
	for _, col := range []string{"tenant_id = tenant_id", "outbox_id = outbox_id", "record_id = record_id", "kind = 'k'", "created_at = now()"} {
		_, err := execIn(worker, tenantA, "UPDATE workflow_ref SET "+col+" WHERE tenant_id = $1", tenantA)
		wantSQLState(t, "nava_worker updates workflow_ref."+col, err, "42501")
	}
	_, err := execIn(asRole(t, "nava_app"), tenantA, "UPDATE workflow_ref SET record_changed_signal_id = $2 WHERE tenant_id = $1", tenantA, wfSignal)
	wantSQLState(t, "nava_app sets the waiting state", err, "42501")

	if got := queryString(t, super, `SELECT pg_get_indexdef('workflow_ref_waiting'::regclass)`); !strings.Contains(got,
		"(tenant_id, record_id) WHERE ((closed_at IS NULL) AND (record_changed_signal_id IS NOT NULL))") {
		t.Errorf("workflow_ref_waiting = %s", got)
	}
	if !queryBool(t, super, "SELECT indisvalid FROM pg_index WHERE indexrelid = 'workflow_ref_waiting'::regclass") {
		t.Error("workflow_ref_waiting is not valid")
	}
}

// Acceptance: the cascade_start foreign-key pair applies on a database that
// already holds claims. The claims were written before any definition table
// existed, so their definitions are seeded once the table exists (the
// recovery 20261111123330's header names) and VALIDATE succeeds. A claim on a
// definition nobody holds stops the validate migration, which records no
// version; afterwards a claim naming a missing definition, or another
// tenant's, is refused.
func TestP340CascadeStartDefinitionKeyOverExistingClaims(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, "20261111123300_workflow_definitions.sql")); err != nil {
		t.Fatalf("migrate to before P3-40: %v", err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	rec := pipelineRecord(tenantA) // the same id in both tenants
	claim := `INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id) VALUES ($1, $2, $3, $4, gen_random_uuid())`
	exec(t, super, claim, tenantA, originX, rec, wfDefA)
	exec(t, super, claim, tenantB, originX, rec, wfDefB)
	exec(t, super, claim, tenantB, outboxX, rec, wfDefA) // tenant B's claim on an id only tenant A will hold

	// Through the NOT VALID step: the old claims stay, and every new claim
	// is checked from here on.
	if err := migrateOnlyFS(t, m, migrationsBefore(t, cascadeValidateMigration)); err != nil {
		t.Fatalf("migrate through %s: %v", cascadeDefinitionMigration, err)
	}
	if queryBool(t, super, `SELECT convalidated FROM pg_constraint WHERE conname = 'cascade_start_definition_fkey'`) {
		t.Fatal("cascade_start_definition_fkey is validated by the expand migration")
	}
	_, err := super.Exec(t.Context(), claim, tenantA, outboxY, rec, wfDefA)
	if sqlState(err) != "23503" {
		t.Fatalf("a new claim on a missing definition under the NOT VALID key: %v, want 23503", err)
	}

	seedWorkflowDefinition(t, super, wfDefA, tenantA)
	seedWorkflowDefinition(t, super, wfDefB, tenantB)
	// Tenant B's claim on tenant A's definition id is the orphan: VALIDATE
	// fails on it and records no version.
	_, err = migrateFS(t, m, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "cascade_start_definition_fkey") {
		t.Fatalf("validate over a claim on another tenant's definition: %v, want a violation of cascade_start_definition_fkey", err)
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM goose_db_version WHERE version_id = 20261111123340"); n != "0" {
		t.Fatalf("failed validate recorded %s version rows", n)
	}
	noMigratorPolicies(t, super, "after the failed validate")

	seedWorkflowDefinition(t, super, wfDefA, tenantB)
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("validate once every claim's definition exists: %v", err)
	}
	if !queryBool(t, super, `SELECT convalidated FROM pg_constraint WHERE conname = 'cascade_start_definition_fkey'`) {
		t.Fatal("cascade_start_definition_fkey not validated")
	}
	noMigratorPolicies(t, super, "after the validate")
	if got := queryString(t, super, "SELECT count(*)::text FROM cascade_start"); got != "3" {
		t.Errorf("claims after the migration = %s, want the 3 seeded", got)
	}

	app := asRole(t, "nava_app")
	_, err = claimRows(app, tenantA, tenantA, outboxY, rec, newUUID(t, super), outboxX)
	wantSQLState(t, "a claim naming a missing definition", err, "23503")
	_, err = claimRows(app, tenantA, tenantA, outboxY, rec, wfDefB, outboxX)
	wantSQLState(t, "tenant A claims a start of tenant B's definition", err, "23503")
	if n, err := claimRows(app, tenantA, tenantA, outboxY, rec, wfDefA, outboxX); err != nil || n != 1 {
		t.Errorf("control: tenant A claims a start of its own definition: %d rows, %v; want 1", n, err)
	}
}

// noMigratorPolicies is the Security reviewer's guard on 20261111123340
// (TICK-709): the SELECT USING (true) policies it gives nava_migrator to
// validate cascade_start_definition_fkey must not outlive its transaction,
// whether VALIDATE fails (rollback) or succeeds (dropped before commit). A
// survivor would be an unscoped read path on two tenant tables.
func noMigratorPolicies(t *testing.T, conn *pgx.Conn, when string) {
	t.Helper()
	if got := queryString(t, conn, `SELECT coalesce(string_agg(tablename || '.' || policyname, ', '), '') FROM pg_policies
		WHERE 'nava_migrator' = ANY (roles) OR policyname LIKE '%migrator_validate'`); got != "" {
		t.Errorf("%s: nava_migrator policies left behind: %s", when, got)
	}
}

// newUUID is a fresh random uuid from the server.
func newUUID(t *testing.T, conn *pgx.Conn) string {
	t.Helper()
	return queryString(t, conn, "SELECT gen_random_uuid()::text")
}
