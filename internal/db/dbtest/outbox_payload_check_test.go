//go:build integration

package dbtest

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Navaladi2019/TicketingSystem/db"
)

const (
	outboxPayloadMigration = "00006_outbox_payload_references.sql"
	outboxKeysMigration    = "00007_outbox_payload_keys.sql"
)

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
// re-run validates the constraint. It stops before 00007, which replaces the
// constraint.
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

	upTo6 := migrationsBefore(t, outboxKeysMigration)
	_, err := migrateFS(t, m, upTo6)
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
	if _, err := migrateFS(t, m, upTo6); err != nil {
		t.Fatalf("re-run after removing the row: %v", err)
	}
	if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
		WHERE conrelid = 'outbox'::regclass AND conname = 'outbox_payload_references_check'`) {
		t.Fatal("constraint not validated after the re-run")
	}
}

// TICK-240: 00007 fails loudly, naming its constraint, when an existing row
// passed 00006 but breaks the stricter rule (a free-text key, or a UUID in a
// form other than canonical). It records no version, keeps the 00006
// constraint, and leaves the new one NOT VALID but refusing new rows. Once the
// rows are gone, a re-run validates it and drops the 00006 constraint, and a
// further run changes nothing.
func TestT240PayloadKeysMigrationFailsOnExistingRow(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, migrationsBefore(t, outboxKeysMigration)); err != nil {
		t.Fatalf("migrate to 00006: %v", err)
	}
	super := superNava(t)
	insert := `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload)
		VALUES (gen_random_uuid(), $1, 'k', $2, 't', 1, $3::jsonb)`
	exec(t, super, insert, tenantA, "old-refs", `{"record_id":"`+tenantA+`"}`)
	exec(t, super, insert, tenantB, "old-free-text-key", `{"printer on fire":"`+tenantB+`"}`)
	exec(t, super, insert, tenantB, "old-urn", `{"record_id":"urn:uuid:`+tenantB+`"}`)

	constraint := func(name string) string {
		return queryString(t, super, `SELECT coalesce((SELECT convalidated::text FROM pg_constraint
			WHERE conrelid = 'outbox'::regclass AND conname = $1), 'absent')`, name)
	}

	_, err := migrateFS(t, m, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "outbox_payload_check") {
		t.Fatalf("migrate over a free-text key: %v, want a violation of outbox_payload_check", err)
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM goose_db_version WHERE version_id = 7"); n != "0" {
		t.Fatalf("failed 00007 recorded %s version rows", n)
	}
	if got := constraint("outbox_payload_check"); got != "false" {
		t.Fatalf("outbox_payload_check after the failed run: convalidated %s, want false", got)
	}
	if got := constraint("outbox_payload_references_check"); got != "true" {
		t.Fatalf("outbox_payload_references_check after the failed run: %s, want it kept and valid", got)
	}
	if _, err := super.Exec(t.Context(), insert, tenantB, "new-free-text-key", `{"call alice":"`+tenantB+`"}`); sqlState(err) != "23514" {
		t.Fatalf("new free-text key under the NOT VALID constraint: %v, want 23514", err)
	}

	exec(t, super, "DELETE FROM outbox WHERE dedupe_key IN ('old-free-text-key', 'old-urn')")
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("re-run after removing the rows: %v", err)
	}
	if got := constraint("outbox_payload_check"); got != "true" {
		t.Fatalf("outbox_payload_check after the re-run: convalidated %s, want true", got)
	}
	if got := constraint("outbox_payload_references_check"); got != "absent" {
		t.Fatalf("outbox_payload_references_check after the re-run: %s, want it dropped", got)
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM outbox WHERE dedupe_key = 'old-refs'"); n != "1" {
		t.Fatalf("the references-only row: %s rows, want it kept", n)
	}

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("clean re-run: %v", err)
	}
}
