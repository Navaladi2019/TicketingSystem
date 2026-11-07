//go:build integration

package dbtest

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

// P5-40a (TICK-612): the reads the R19/R20 export (P5-40, TICK-444) needs.
//
//   - nava_audit_reader reads audit_log.prev_hash and entry_hash, so the
//     export pages (A1-5: reader pool, masker in the same transaction) can
//     emit the R20 anchor and stubs. Row access is still audit_log_select
//     under the canonical predicate.
//   - nava_app takes the R19 lease and reads head_seq and head_hash in one
//     statement (A1-7 keeps the lease on nava_app). Its table-level SELECT on
//     audit_chain_head already covers the WHERE on export_lease_until and the
//     RETURNING; this test pins that, so a later narrowing to columns breaks
//     here and not in the export.
//   - nava_audit_reader stays off audit_chain_head, and nava_app off the
//     audit_log hashes: neither is needed.

// exportLeaseTake is the R19 lease take with the R20 head read: tenant $1.
const exportLeaseTake = `UPDATE audit_chain_head SET export_lease_until = now() + interval '15 minutes'
	WHERE tenant_id = $1 AND (export_lease_until IS NULL OR export_lease_until < now())
	RETURNING seq, entry_hash`

func TestP540aReaderReadsChainHashes(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")
	worker := asRole(t, "nava_worker")
	reader := asRole(t, audited.ReaderRole)

	a := appendEntry(t, app, tenantA, 1)
	b := appendEntry(t, app, tenantB, 1)
	for tenant, id := range map[string]string{tenantA: a, tenantB: b} {
		if n, err := execIn(worker, tenant, sealEntry, tenant, id); err != nil || n != 1 {
			t.Fatalf("seal %s: %d rows, %v", tenant, n, err)
		}
	}

	const stub = `SELECT seq, prev_hash, entry_hash FROM audit_log WHERE id = $1`
	read := func(tenant, id string) (rows int, prev, entry []byte, err error) {
		err = inTenantTx(reader, tenant, func(tx pgx.Tx) error {
			r, err := tx.Query(context.Background(), stub, id)
			if err != nil {
				return err
			}
			defer r.Close()
			for r.Next() {
				var seq int64
				if err := r.Scan(&seq, &prev, &entry); err != nil {
					return err
				}
				rows++
			}
			return r.Err()
		})
		return rows, prev, entry, err
	}
	n, prev, entry, err := read(tenantA, a)
	if err != nil || n != 1 {
		t.Fatalf("the reader in tenant A reads its own stub: %d rows, %v; want 1", n, err)
	}
	if want := queryString(t, super, `SELECT encode(sha256('p'), 'hex') || encode(sha256('e'), 'hex')`); hex.EncodeToString(prev)+hex.EncodeToString(entry) != want {
		t.Errorf("stub hashes = %x %x, want sha256('p') sha256('e')", prev, entry)
	}
	if n, _, _, err := read(tenantA, b); err != nil || n != 0 {
		t.Errorf("the reader in tenant A reads tenant B's stub: %d rows, %v; want 0", n, err)
	}
	if n, _, _, err := read("", a); err != nil || n != 0 {
		t.Errorf("the reader with no tenant reads a stub: %d rows, %v; want 0", n, err)
	}

	// The widening is the two hashes only: the reader still cannot read txid
	// or erasure_ref, and nava_app still cannot read the hashes.
	for _, col := range []string{"txid", "erasure_ref"} {
		_, err := execIn(reader, tenantA, "SELECT "+col+" FROM audit_log WHERE id = $1", a)
		wantSQLState(t, "nava_audit_reader reads "+col, err, "42501")
	}
	for _, col := range []string{"prev_hash", "entry_hash"} {
		_, err := execIn(app, tenantA, "SELECT "+col+" FROM audit_log WHERE id = $1", a)
		wantSQLState(t, "nava_app reads audit_log."+col, err, "42501")
	}
}

func TestP540aExportLeaseAndHeadRead(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	for _, tenant := range []string{tenantA, tenantB} {
		exec(t, super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", tenant)
	}
	app := asRole(t, "nava_app")

	take := func(tenant, head string) (rows int, seq int64, hash []byte, err error) {
		err = inTenantTx(app, tenant, func(tx pgx.Tx) error {
			r, err := tx.Query(context.Background(), exportLeaseTake, head)
			if err != nil {
				return err
			}
			defer r.Close()
			for r.Next() {
				if err := r.Scan(&seq, &hash); err != nil {
					return err
				}
				rows++
			}
			return r.Err()
		})
		return rows, seq, hash, err
	}

	n, seq, hash, err := take(tenantA, tenantA)
	if err != nil || n != 1 {
		t.Fatalf("nava_app takes tenant A's free lease: %d rows, %v; want 1", n, err)
	}
	if seq != 0 || hex.EncodeToString(hash) != strings.Repeat("0", 64) {
		t.Errorf("head read = %d %x, want seq 0 and the genesis hash", seq, hash)
	}
	if n, _, _, err := take(tenantA, tenantA); err != nil || n != 0 {
		t.Errorf("a second take while the lease is held: %d rows, %v; want 0 (R19 409)", n, err)
	}
	if n, _, _, err := take(tenantA, tenantB); err != nil || n != 0 {
		t.Errorf("nava_app in tenant A takes tenant B's lease: %d rows, %v; want 0", n, err)
	}
	if got := queryString(t, super, "SELECT (export_lease_until IS NULL)::text FROM audit_chain_head WHERE tenant_id = $1", tenantB); got != "true" {
		t.Errorf("tenant B's lease after tenant A's attempt: null = %s, want true", got)
	}
	exec(t, super, "UPDATE audit_chain_head SET export_lease_until = now() - interval '1 second' WHERE tenant_id = $1", tenantA)
	if n, _, _, err := take(tenantA, tenantA); err != nil || n != 1 {
		t.Errorf("nava_app takes tenant A's expired lease: %d rows, %v; want 1", n, err)
	}

	// The reader never touches the head (R14 as amended by A1).
	for _, q := range []string{"SELECT seq FROM audit_chain_head", "SELECT entry_hash FROM audit_chain_head"} {
		_, err := execIn(asRole(t, audited.ReaderRole), tenantA, q)
		wantSQLState(t, "nava_audit_reader: "+q, err, "42501")
	}
}
