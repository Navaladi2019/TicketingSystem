//go:build integration

package dbtest

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Navaladi2019/TicketingSystem/db"
)

const outboxPayloadMigration = "00006_outbox_payload_references.sql"

// migrationsBefore is the real migrations up to, not including, name.
func migrationsBefore(t *testing.T, name string) fs.FS {
	t.Helper()
	out := fstest.MapFS{}
	entries, err := fs.ReadDir(db.Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() >= name {
			continue
		}
		b, err := fs.ReadFile(db.Migrations(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = &fstest.MapFile{Data: b}
	}
	return out
}

// TICK-214: 00006 fails loudly, naming its constraint, when an existing
// outbox row holds record data; it records no version, and the NOT VALID
// constraint it leaves still refuses new rows. Once the row is gone, a
// re-run validates the constraint.
func TestT214PayloadCheckMigrationFailsOnExistingRecordData(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, migrationsBefore(t, outboxPayloadMigration)); err != nil {
		t.Fatalf("migrate to 00005: %v", err)
	}
	super := superNava(t)
	insert := `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
		VALUES ($1, $1, 'k', $2, 't', 1, $3::jsonb)`
	exec(t, super, insert, tenantA, "old-refs", `{"record_id":"`+tenantA+`"}`)
	exec(t, super, insert, tenantB, "old-record-data", `{"title":"printer on fire"}`)

	_, err := migrateFS(t, m, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "outbox_payload_references_check") {
		t.Fatalf("migrate over a record-data row: %v, want a violation of outbox_payload_references_check", err)
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM goose_db_version WHERE version_id = 6"); n != "0" {
		t.Fatalf("failed 00006 recorded %s version rows", n)
	}
	if queryBool(t, super, `SELECT convalidated FROM pg_constraint
		WHERE conrelid = 'outbox'::regclass AND conname = 'outbox_payload_references_check'`) {
		t.Fatal("constraint validated over a record-data row")
	}
	if _, err := super.Exec(t.Context(), insert, tenantB, "new-record-data", `{"amount":42}`); sqlState(err) != "23514" {
		t.Fatalf("new record-data row under the NOT VALID constraint: %v, want 23514", err)
	}

	exec(t, super, "DELETE FROM outbox WHERE dedupe_key = 'old-record-data'")
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("re-run after removing the row: %v", err)
	}
	if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
		WHERE conrelid = 'outbox'::regclass AND conname = 'outbox_payload_references_check'`) {
		t.Fatal("constraint not validated after the re-run")
	}
}
