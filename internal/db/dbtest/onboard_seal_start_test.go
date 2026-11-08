//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

// P5-33 (TICK-441, ADR-001-A6.2, CI-A16 onboarding): nava_onboard seeds the
// chain head and the first audit.seal start row for its own provisioning
// tenant only, with INSERT and nothing else.

const insertSealRow = `INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
	causation_id, causation_depth, payload, trace_context)
	VALUES ($1::uuid, $2, $3, 1, 'audit.seal:1', 'audit.seal:' || ($1::uuid)::text, 1, $1::uuid, 0, '{}', NULL)`

func TestP533OnboardInsertsSealRowForOwnProvisioningTenantOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	on := asRole(t, audited.OnboardRole)

	if _, err := execIn(on, tenantX, insertSealRow, uuid.NewString(), tenantX, "audit.seal"); err != nil {
		t.Fatalf("own provisioning tenant: %v", err)
	}
	for _, tc := range []struct{ what, ctx, target, kind string }{
		{"another kind", tenantX, tenantX, "workflow.start"},
		{"another provisioning tenant", tenantX, tenantY, "audit.seal"},
		{"an active tenant", tenantX, tenantA, "audit.seal"},
		{"an active tenant's own context", tenantA, tenantA, "audit.seal"},
		{"no tenant set", "", tenantX, "audit.seal"},
	} {
		_, err := execIn(on, tc.ctx, insertSealRow, uuid.NewString(), tc.target, tc.kind)
		wantSQLState(t, "nava_onboard inserts "+tc.what, err, "42501")
	}
	// No SELECT on outbox, and no UPDATE or DELETE.
	for _, q := range []string{
		"SELECT count(*) FROM outbox",
		"SELECT id FROM outbox",
		"UPDATE outbox SET attempts = 1",
		"DELETE FROM outbox",
	} {
		_, err := execIn(on, tenantX, q)
		wantSQLState(t, "nava_onboard "+q, err, "42501")
	}
	if n := countRows(t, "SELECT count(*) FROM outbox WHERE kind = 'audit.seal'"); n != 1 {
		t.Errorf("seal rows = %d, want 1", n)
	}
}

func TestP533OnboardInsertsHeadForOwnProvisioningTenantOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	on := asRole(t, audited.OnboardRole)
	const ins = "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())"

	for _, tc := range []struct{ what, ctx, target string }{
		{"another provisioning tenant", tenantX, tenantY},
		{"an active tenant", tenantX, tenantA},
		{"an active tenant's own context", tenantA, tenantA},
	} {
		_, err := execIn(on, tc.ctx, ins, tc.target)
		wantSQLState(t, "nava_onboard inserts a head for "+tc.what, err, "42501")
	}
	if _, err := execIn(on, tenantX, ins, tenantX); err != nil {
		t.Fatalf("own provisioning tenant: %v", err)
	}
	// The head's version, seq and entry_hash are the defaults only.
	_, err := execIn(on, tenantY, "INSERT INTO audit_chain_head (tenant_id, id, version) VALUES ($1, gen_random_uuid(), 9)", tenantY)
	wantSQLState(t, "nava_onboard names head.version", err, "42501")
	_, err = execIn(on, tenantX, "SELECT * FROM audit_chain_head")
	wantSQLState(t, "nava_onboard reads the head", err, "42501")
	// TICK-580: no UPDATE or DELETE on the head, not even its own provisioning
	// tenant's. No WHERE clause and no column read on the right-hand side, so
	// only the UPDATE/DELETE privilege is tested: a WHERE would fail on the
	// missing SELECT and mask a regressed grant.
	for _, q := range []string{
		"UPDATE audit_chain_head SET seq = 1",
		"UPDATE audit_chain_head SET next_seal_outbox_id = NULL",
		"DELETE FROM audit_chain_head",
	} {
		_, err := execIn(on, tenantX, q)
		wantSQLState(t, "nava_onboard "+q, err, "42501")
	}
	if n := countRows(t, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1 AND seq = 0", tenantX); n != 1 {
		t.Errorf("own head rows with seq 0 = %d, want 1 (head changed or deleted)", n)
	}
}

func TestP533SeedAuditChainIsIdempotent(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if err := s.SeedAuditChain(t.Context(), id); err != nil {
			t.Fatalf("SeedAuditChain #%d: %v", i+1, err)
		}
	}
	if n := countRows(t, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", id); n != 1 {
		t.Errorf("heads = %d, want 1", n)
	}
	if n := countRows(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'audit.seal'", id); n != 1 {
		t.Errorf("seal rows = %d, want 1 (one live lineage)", n)
	}
	// The row has the C5 shape and the head points at it.
	if n := countRows(t, `SELECT count(*) FROM outbox o JOIN audit_chain_head h
		ON h.tenant_id = o.tenant_id AND h.next_seal_outbox_id = o.id
		WHERE o.tenant_id = $1 AND o.kind_version = 1 AND o.dedupe_key = 'audit.seal:1'
		  AND o.target_key = 'audit.seal:' || o.id::text AND o.target_seq = h.version AND h.version = 1
		  AND o.causation_id = o.id AND o.causation_depth = 0 AND o.payload = '{}'
		  AND h.next_seal_at IS NOT NULL AND h.seq = 0
		  AND NOT EXISTS (SELECT 1 FROM workflow_ref w WHERE w.tenant_id = o.tenant_id)`, id); n != 1 {
		t.Errorf("seal row does not have the C5 shape pointed at by the head")
	}
}

func TestP533SeedAuditChainRefusesActiveTenant(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA}, nil)
	if err := onboardStore(t, 4).SeedAuditChain(t.Context(), uuid.MustParse(tenantA)); sqlState(err) != "42501" {
		t.Errorf("SeedAuditChain on an active tenant: %v, want SQLSTATE 42501", err)
	}
}

// Activation order (Technical Lead, carry-over from TICK-428): the commit
// check reads under the pinned provisioning policy, so activating in the
// transaction of an audited write fails at COMMIT, closed.
func TestP533ActivatingInTheAuditedWritesTransactionFailsClosed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantX})
	createTestTable(t, super, []string{audited.OnboardRole})
	on := asRole(t, audited.OnboardRole)
	ctx := context.Background()
	row := uuid.NewString()

	err := inTenantTx(on, tenantX, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO "+testTable+" (tenant_id, id, val) VALUES ($1, $2, 'a')", tenantX, row); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, insertEntryAs, tenantX, uuid.NewString(), row, 1, "record"); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantX)
		return err
	})
	if code, _ := pgErrDetail(err); code != "23514" {
		t.Fatalf("commit err = %v, want SQLSTATE 23514", err)
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantX); got != "provisioning" {
		t.Errorf("tenant state = %q, want provisioning", got)
	}
	if n := countRows(t, "SELECT count(*) FROM "+testTable+" WHERE tenant_id = $1", tenantX); n != 0 {
		t.Errorf("%d partial rows, want 0", n)
	}
	if n := countRows(t, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenantX); n != 0 {
		t.Errorf("%d partial audit entries, want 0", n)
	}
}

// The two-transaction order succeeds, and a retry after a crash between the
// audited writes and the activation is idempotent.
func TestP533AuditedWritesThenActivateSucceedsAndRetryIsIdempotent(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	recordAdmin(t, s, id)
	createTestTable(t, super, []string{audited.OnboardRole})
	on := asRole(t, audited.OnboardRole)
	ctx := context.Background()
	tenant := id.String()
	row := uuid.NewString()

	writeAudited := func() error {
		return inTenantTx(on, tenant, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "INSERT INTO "+testTable+" (tenant_id, id, val) VALUES ($1, $2, 'a') ON CONFLICT DO NOTHING", tenant, row); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, insertEntryAs, tenant, uuid.NewString(), row, 1, "record")
			return err
		})
	}
	// Transaction 1: the audited writes, the tenant still provisioning.
	if err := s.SeedAuditChain(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := writeAudited(); err != nil {
		t.Fatalf("audited write while provisioning: %v", err)
	}
	// Crash before Activate; the retry runs the same steps again.
	if err := s.SeedAuditChain(ctx, id); err != nil {
		t.Fatalf("retried SeedAuditChain: %v", err)
	}
	if err := writeAudited(); err != nil {
		t.Fatalf("retried audited write: %v", err)
	}
	// Transaction 2: activate.
	if err := s.Activate(ctx, id); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", id); got != "active" {
		t.Errorf("tenant state = %q, want active", got)
	}
	if n := countRows(t, "SELECT count(*) FROM audit_chain_head WHERE tenant_id = $1", id); n != 1 {
		t.Errorf("heads = %d, want 1", n)
	}
	if n := countRows(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'audit.seal'", id); n != 1 {
		t.Errorf("seal rows = %d, want 1", n)
	}
}
