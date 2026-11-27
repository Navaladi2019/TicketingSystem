//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-1158 (ADR-010-A4 A4-1, CI-2): the swap of records_list_updated for its
// covering definition finishes from every state a failed run can leave, and a
// re-run over a finished swap builds nothing.

const coveringMigration = "20261126120000_records_list_updated_covering.sql"

const (
	coveringDef = "CREATE INDEX records_list_updated ON public.records USING btree (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id)"
	plainDef    = "CREATE INDEX records_list_updated ON public.records USING btree (tenant_id, entity_definition_id, updated_at DESC, id DESC)"
)

func TestRecordsListUpdatedCoveringRerun(t *testing.T) {
	m := migratedCluster(t)
	super := superNava(t)
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	b, err := fs.ReadFile(db.Migrations(), coveringMigration)
	if err != nil {
		t.Fatal(err)
	}
	up, down, ok := strings.Cut(string(b), "-- +goose Down")
	if !ok {
		t.Fatal("no Down section")
	}
	// run applies one direction of the file as an unversioned Up.
	run := func(label, body string) {
		t.Helper()
		p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fstest.MapFS{coveringMigration: {Data: []byte(body)}},
			goose.WithDisableVersioning(true), goose.WithDisableGlobalRegistry(true))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Up(context.Background()); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
	}
	runUp := func(label string) { t.Helper(); run(label, up) }
	runDown := func(label string) { t.Helper(); run(label, "-- +goose NO TRANSACTION\n-- +goose Up\n"+down) }
	state := func() string {
		return queryString(t, super, `SELECT coalesce(string_agg(c.relname || ' valid=' || i.indisvalid || ' ' || pg_get_indexdef(i.indexrelid), E'\n' ORDER BY c.relname), '')
			FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE c.relname IN ('records_list_updated', 'records_list_updated_v2')`)
	}
	want := func(label, def string) {
		t.Helper()
		if got := state(); got != "records_list_updated valid=true "+def {
			t.Fatalf("%s:\n%s\nwant only:\n%s", label, got, def)
		}
	}
	oid := func() string { return queryString(t, super, `SELECT to_regclass('records_list_updated')::oid::text`) }
	want("migrated", coveringDef)

	// A re-run over the finished swap keeps the index: it builds nothing.
	before := oid()
	runUp("re-run")
	want("re-run", coveringDef)
	if got := oid(); got != before {
		t.Errorf("a re-run over the covering index rebuilt it (oid %s -> %s)", before, got)
	}

	// Stopped after the drop, before the rename: v2 alone, valid.
	exec(t, super, `ALTER INDEX records_list_updated RENAME TO records_list_updated_v2`)
	runUp("stopped before the rename")
	want("stopped before the rename", coveringDef)

	// A failed build: an INVALID v2 beside today's index.
	runDown("down")
	want("down", plainDef)
	exec(t, super, `CREATE INDEX records_list_updated_v2 ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id)`)
	exec(t, super, `UPDATE pg_index SET indisvalid = false WHERE indexrelid = to_regclass('records_list_updated_v2')`)
	invalid := queryString(t, super, `SELECT to_regclass('records_list_updated_v2')::oid::text`)
	runUp("invalid v2")
	want("invalid v2", coveringDef)
	if oid() == invalid {
		t.Error("the re-run kept the INVALID v2")
	}

	// A valid v2 without the INCLUDE, left by a failed Down, is not renamed in.
	runDown("down again")
	exec(t, super, `CREATE INDEX records_list_updated_v2 ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC)`)
	runUp("plain v2")
	want("plain v2", coveringDef)

	// The Down is re-runnable too, and Up after it restores the covering index.
	runDown("down")
	before = oid()
	runDown("down re-run")
	want("down re-run", plainDef)
	if got := oid(); got != before {
		t.Errorf("a Down re-run over today's index rebuilt it (oid %s -> %s)", before, got)
	}
	runUp("up after down")
	want("up after down", coveringDef)
}
