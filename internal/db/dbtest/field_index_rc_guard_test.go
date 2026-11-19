//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// P7-05c (TICK-979, ADR-006-E1 E1-5): record_field_values_sync() refuses a
// records write at any isolation level but READ COMMITTED, so a writer can
// never slip past the D9 snapshot barrier. Migration
// 20261119013700_field_index_sync_read_committed.sql.

// rcGuardSQLState is the guard's SQLSTATE: 25000, invalid_transaction_state.
const rcGuardSQLState = "25000"

// writeAt runs sql with args as conn in one tenant transaction opened at
// iso, and commits.
func writeAt(conn *pgx.Conn, iso pgx.TxIsoLevel, tenant, sql string, args ...any) error {
	ctx := context.Background()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenant); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Acceptance: an INSERT or an UPDATE OF data on records, as nava_app and as
// nava_worker, is refused with 25000 at REPEATABLE READ, SERIALIZABLE and
// READ UNCOMMITTED, and through a session default of REPEATABLE READ (the
// operator rule of AR-TICK-745-01). The same writes at READ COMMITTED
// succeed and write their side rows. The refusal holds for a type with no
// flagged field too: the check runs before the registry read.
func TestP705cRecordsWriteRequiresReadCommitted(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	triggerFields(t, super, tenantA) // tenant B's type has no flagged field

	insert := recordWrite(t, super, insertTypedRecord)
	update := recordWrite(t, super, `UPDATE records SET data = jsonb_set(data, '{f_text}', '"b"'), version = version + 1
		WHERE tenant_id = $1 AND id = $2`)
	const data = `{"f_text": "a", "f_num": 1, "f_ts": "2026-11-15", "f_bool": true, "f_tags": ["x", "y"]}`
	sideRows := func(id string) string {
		return queryString(t, super, `SELECT ((SELECT count(*) FROM rfv_text WHERE record_id = $1)
			+ (SELECT count(*) FROM rfv_num WHERE record_id = $1) + (SELECT count(*) FROM rfv_ts WHERE record_id = $1)
			+ (SELECT count(*) FROM rfv_bool WHERE record_id = $1))::text`, id)
	}
	refused := []pgx.TxIsoLevel{pgx.RepeatableRead, pgx.Serializable, pgx.ReadUncommitted}

	n := 0
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, tenant := range []string{tenantA, tenantB} {
			// READ COMMITTED: accepted, with side rows on the flagged type.
			n++
			id := fmt.Sprintf("0190c000-0000-7000-8979-%012d", n)
			if err := writeAt(conn, pgx.ReadCommitted, tenant, insert, tenant, id, tenant, fmt.Sprintf("G-%d", n), data); err != nil {
				t.Fatalf("%s %s: insert at read committed: %v", role, tenant, err)
			}
			want := "0"
			if tenant == tenantA {
				want = "6"
			}
			if got := sideRows(id); got != want {
				t.Errorf("%s %s: side rows after read committed insert = %s, want %s", role, tenant, got, want)
			}
			if err := writeAt(conn, pgx.ReadCommitted, tenant, update, tenant, id); err != nil {
				t.Fatalf("%s %s: update at read committed: %v", role, tenant, err)
			}
			if tenant == tenantA {
				if got := queryString(t, super, `SELECT value FROM rfv_text WHERE record_id = $1 AND ord = 0
					AND field_id = (SELECT id FROM record_index_fields WHERE tenant_id = $2 AND field_key = 'f_text')`, id, tenant); got != "b" {
					t.Errorf("%s: f_text side row after read committed update = %q, want b", role, got)
				}
			}

			// Any other level: refused, nothing written.
			for _, iso := range refused {
				n++
				other := fmt.Sprintf("0190c000-0000-7000-8979-%012d", n)
				err := writeAt(conn, iso, tenant, insert, tenant, other, tenant, fmt.Sprintf("G-%d", n), data)
				wantRCGuard(t, fmt.Sprintf("%s %s insert at %s", role, tenant, iso), err)
				err = writeAt(conn, iso, tenant, update, tenant, id)
				wantRCGuard(t, fmt.Sprintf("%s %s update at %s", role, tenant, iso), err)
				if got := queryString(t, super, "SELECT count(*)::text FROM records WHERE id = $1", other); got != "0" {
					t.Errorf("%s %s at %s: refused insert left a record", role, tenant, iso)
				}
			}
		}

		// A session default above READ COMMITTED is refused the same way.
		ctx := context.Background()
		if _, err := conn.Exec(ctx, "SET default_transaction_isolation = 'repeatable read'"); err != nil {
			t.Fatal(err)
		}
		n++
		_, err := execIn(conn, tenantA, insert, tenantA, fmt.Sprintf("0190c000-0000-7000-8979-%012d", n), tenantA, fmt.Sprintf("G-%d", n), data)
		wantRCGuard(t, role+" insert under default_transaction_isolation repeatable read", err)
		if _, err := conn.Exec(ctx, "RESET default_transaction_isolation"); err != nil {
			t.Fatal(err)
		}
	}
}

// wantRCGuard asserts err is the E1-5 guard: SQLSTATE 25000, with a message
// that names ADR-006-E1 (so it is recognisable in logs).
func wantRCGuard(t *testing.T, what string, err error) {
	t.Helper()
	wantSQLState(t, what, err, rcGuardSQLState)
	if err != nil && !strings.Contains(err.Error(), "ADR-006-E1") {
		t.Errorf("%s: message %q does not name ADR-006-E1", what, err)
	}
}

// rcTriggerViolations reports, from the catalogue, how the records triggers
// break the E1-2 precondition: the D4 sync trigger (the one calling
// record_field_values_sync) must be exactly one AFTER ROW trigger, and no
// other trigger function on records may read record_index_fields. It returns
// "" when they hold, and the names of the triggers it inspected.
//
// pg_trigger.tgtype bits (src/include/catalog/pg_trigger.h): 1 ROW,
// 2 BEFORE, 64 INSTEAD; AFTER is neither 2 nor 64.
const rcTriggerViolations = `
WITH t AS (
    SELECT tg.tgname, tg.tgtype::int AS tgtype, p.proname, p.prosrc
    FROM pg_trigger tg JOIN pg_proc p ON p.oid = tg.tgfoid
    WHERE tg.tgrelid = 'public.records'::regclass
), v AS (
    SELECT 'sync trigger ' || tgname || ' is not AFTER ROW (tgtype ' || tgtype || ')' AS msg
    FROM t WHERE proname = 'record_field_values_sync' AND (tgtype & 1 = 0 OR tgtype & 66 <> 0)
    UNION ALL
    SELECT 'want exactly one sync trigger, found ' || count(*)
    FROM t WHERE proname = 'record_field_values_sync' HAVING count(*) <> 1
    UNION ALL
    SELECT 'trigger ' || tgname || ' (' || proname || ') reads record_index_fields'
    FROM t WHERE proname <> 'record_field_values_sync' AND prosrc ILIKE '%record_index_fields%'
)
SELECT coalesce((SELECT string_agg(msg, '; ' ORDER BY msg) FROM v), ''),
       (SELECT string_agg(tgname || ':' || proname, ',' ORDER BY tgname) FROM t)`

// checkRCTriggers runs rcTriggerViolations in q.
func checkRCTriggers(t *testing.T, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (violations, inspected string) {
	t.Helper()
	if err := q.QueryRow(context.Background(), rcTriggerViolations).Scan(&violations, &inspected); err != nil {
		t.Fatal(err)
	}
	return violations, inspected
}

// Acceptance (ADR-006-E1 rev 2, E1-5): after migrations, the D4 sync trigger
// on records is AFTER ROW, and no other trigger on records reads
// record_index_fields. The check bites: in a rolled-back transaction it
// fails when the sync trigger is redefined BEFORE ROW, and when a second
// records trigger function reads the registry.
func TestP705cRecordsTriggerCatalog(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	violations, inspected := checkRCTriggers(t, super)
	if violations != "" {
		t.Errorf("records triggers break the E1-2 precondition: %s", violations)
	}
	t.Logf("inspected records triggers: %s", inspected)
	for _, want := range []string{"records_field_values_sync:record_field_values_sync", "records_field_values_purge:record_field_values_purge"} {
		if !strings.Contains(","+inspected+",", ","+want+",") {
			t.Errorf("inspected %q, want it to include %s", inspected, want)
		}
	}

	mutations := []struct {
		name, ddl, want string
	}{
		{"sync trigger BEFORE ROW",
			`DROP TRIGGER records_field_values_sync ON records;
			 CREATE TRIGGER records_field_values_sync BEFORE INSERT OR UPDATE OF data ON records
			     FOR EACH ROW EXECUTE FUNCTION record_field_values_sync()`,
			"sync trigger records_field_values_sync is not AFTER ROW"},
		{"sync trigger AFTER STATEMENT",
			`DROP TRIGGER records_field_values_sync ON records;
			 CREATE TRIGGER records_field_values_sync AFTER INSERT OR UPDATE OF data ON records
			     FOR EACH STATEMENT EXECUTE FUNCTION record_field_values_sync()`,
			"sync trigger records_field_values_sync is not AFTER ROW"},
		{"second trigger reads the registry",
			`CREATE FUNCTION e15_mutant() RETURNS trigger LANGUAGE plpgsql AS $f$
			 BEGIN PERFORM 1 FROM record_index_fields WHERE tenant_id = NEW.tenant_id; RETURN NEW; END $f$;
			 CREATE TRIGGER records_e15_mutant BEFORE INSERT ON records FOR EACH ROW EXECUTE FUNCTION e15_mutant()`,
			"trigger records_e15_mutant (e15_mutant) reads record_index_fields"},
	}
	ctx := context.Background()
	for _, m := range mutations {
		tx, err := super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, m.ddl); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("%s: %v", m.name, err)
		}
		got, _ := checkRCTriggers(t, tx)
		_ = tx.Rollback(ctx)
		if !strings.Contains(got, m.want) {
			t.Errorf("%s: violations = %q, want it to contain %q", m.name, got, m.want)
		}
	}
	if violations, _ := checkRCTriggers(t, super); violations != "" {
		t.Errorf("after the rolled-back mutations: %s", violations)
	}
}
