//go:build integration

// P3-07 (TICK-345), PLAN-003 I8: a tenant's entity-type change is rows, never
// DDL. Twenty publishes, one of them NeedsBackfill, must leave every catalogue
// object as it was: same oid, same xmin. Security & Multi-Tenancy review.
// Helpers are in entitytypes_integration_test.go and
// entitytypes_publish_integration_test.go.

package server_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// catalogSnapshotSQL lists every object of the catalogues a schema change
// writes, as (catalogue, identity, xmin). xmin is the transaction that last
// wrote the catalogue row, so an ALTER of an existing object changes it as
// surely as a CREATE adds a row. Statistics (relpages, reltuples) are written
// in place by VACUUM and ANALYZE and keep xmin. Temporary namespaces, and the
// objects in them, are left out: other sessions create and drop them.
const catalogSnapshotSQL = `
WITH ns AS (
  SELECT oid FROM pg_namespace
  WHERE nspname NOT LIKE 'pg\_temp\_%' AND nspname NOT LIKE 'pg\_toast\_temp\_%'
), rel AS (
  SELECT c.oid FROM pg_class c JOIN ns ON ns.oid = c.relnamespace
)
SELECT 'pg_namespace', n.oid::text, n.xmin::text FROM pg_namespace n JOIN ns ON ns.oid = n.oid
UNION ALL SELECT 'pg_class', c.oid::text, c.xmin::text FROM pg_class c JOIN rel ON rel.oid = c.oid
UNION ALL SELECT 'pg_attribute', a.attrelid::text || '.' || a.attnum, a.xmin::text FROM pg_attribute a JOIN rel ON rel.oid = a.attrelid
UNION ALL SELECT 'pg_index', i.indexrelid::text, i.xmin::text FROM pg_index i JOIN rel ON rel.oid = i.indrelid
UNION ALL SELECT 'pg_constraint', k.oid::text, k.xmin::text FROM pg_constraint k JOIN ns ON ns.oid = k.connamespace
UNION ALL SELECT 'pg_policy', p.oid::text, p.xmin::text FROM pg_policy p JOIN rel ON rel.oid = p.polrelid
UNION ALL SELECT 'pg_type', t.oid::text, t.xmin::text FROM pg_type t JOIN ns ON ns.oid = t.typnamespace
UNION ALL SELECT 'pg_proc', f.oid::text, f.xmin::text FROM pg_proc f JOIN ns ON ns.oid = f.pronamespace
UNION ALL SELECT 'pg_trigger', g.oid::text, g.xmin::text FROM pg_trigger g JOIN rel ON rel.oid = g.tgrelid`

// catalogQuerier is a superuser connection or a transaction on one.
type catalogQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// catalogSnapshot returns "catalogue identity xmin" lines, sorted.
func catalogSnapshot(t *testing.T, q catalogQuerier) []string {
	t.Helper()
	rows, err := q.Query(t.Context(), catalogSnapshotSQL)
	if err != nil {
		t.Fatalf("catalogue snapshot: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var cat, id, xmin string
		if err := rows.Scan(&cat, &id, &xmin); err != nil {
			t.Fatal(err)
		}
		out = append(out, cat+" "+id+" "+xmin)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// catalogDiff returns the lines only in before ("-") or only in after ("+").
func catalogDiff(before, after []string) []string {
	var d []string
	for _, l := range before {
		if _, found := slices.BinarySearch(after, l); !found {
			d = append(d, "- "+l)
		}
	}
	for _, l := range after {
		if _, found := slices.BinarySearch(before, l); !found {
			d = append(d, "+ "+l)
		}
	}
	return d
}

// forbiddenDDL is one statement of each kind I8 forbids a tenant change to
// run, aimed at the tables a publish writes.
func forbiddenDDL() []struct{ name, sql string } {
	return []struct{ name, sql string }{
		{"ALTER TABLE", `ALTER TABLE entity_definition_versions ADD COLUMN iso_noddl_col text`},
		{"CREATE FUNCTION", `CREATE FUNCTION iso_noddl_fn() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RETURN NEW; END$$`},
		{"CREATE TRIGGER", `CREATE TRIGGER iso_noddl_tg BEFORE INSERT ON entity_definition_versions FOR EACH ROW EXECUTE FUNCTION iso_noddl_fn()`},
		{"CREATE TYPE", `CREATE TYPE iso_noddl_type AS ENUM ('a', 'b')`},
	}
}

// noDDLPublishBody is publish n (1-based) of etSchema: an annotation change
// every time, which is Compatible, and from publish 10 on a default on
// short_description, which publish 10 adds (NeedsBackfill) and the later
// ones keep (removing it would be Breaking).
func noDDLPublishBody(t *testing.T, n int) string {
	t.Helper()
	repl := []string{`"description": "a <b>&amp; tag"`, fmt.Sprintf(`"description": "revision %d"`, n)}
	if n >= 10 {
		repl = append(repl, `"maxLength": 160`, `"maxLength": 160, "default": "none"`)
	}
	return etPublishBody(t, repl...)
}

// The test is not parallel: another test's DDL on the cluster would change
// the snapshot.
func TestPublishIssuesNoDDL(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)

	// Positive control: the snapshot sees each forbidden statement. Without
	// it a query that matched nothing would pass any publish.
	for _, s := range forbiddenDDL() {
		tx, err := e.super.Begin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		before := catalogSnapshot(t, tx)
		if s.name == "CREATE TRIGGER" {
			if _, err := tx.Exec(t.Context(), forbiddenDDL()[1].sql); err != nil {
				t.Fatal(err)
			}
			before = catalogSnapshot(t, tx)
		}
		if _, err := tx.Exec(t.Context(), s.sql); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if d := catalogDiff(before, catalogSnapshot(t, tx)); len(d) == 0 {
			t.Errorf("control: the catalogue snapshot does not see %s", s.name)
		}
		if err := tx.Rollback(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	before := catalogSnapshot(t, e.super)
	if len(before) == 0 {
		t.Fatal("the catalogue snapshot is empty")
	}
	var compat []string
	for n := 1; n <= 20; n++ {
		rec := e.do(t, http.MethodPost, etVersionsPath, "admin-a", fmt.Sprintf("k-noddl-%d", n), noDDLPublishBody(t, n))
		if rec.Code != http.StatusCreated {
			t.Fatalf("publish %d = %d: %s", n, rec.Code, rec.Body)
		}
		body := rec.Body.String()
		if !strings.Contains(body, fmt.Sprintf(`"version":%d`, n+1)) {
			t.Fatalf("publish %d did not write version %d: %s", n, n+1, body)
		}
		switch {
		case strings.Contains(body, `"compat":"needs_backfill"`):
			compat = append(compat, "needs_backfill")
		case strings.Contains(body, `"compat":"compatible"`):
			compat = append(compat, "compatible")
		default:
			t.Fatalf("publish %d has an unexpected compat: %s", n, body)
		}
	}
	if !slices.Contains(compat, "needs_backfill") {
		t.Fatalf("no publish was NeedsBackfill: %v", compat)
	}
	if d := catalogDiff(before, catalogSnapshot(t, e.super)); len(d) > 0 {
		t.Errorf("20 publishes changed %d catalogue rows (I8: a tenant change is never DDL):\n%s", len(d), strings.Join(d, "\n"))
	}
	if n := etVersionCount(t, e); n != 21 {
		t.Errorf("tenant A has %d versions, want 21", n)
	}
}
