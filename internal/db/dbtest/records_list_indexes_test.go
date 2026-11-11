//go:build integration

package dbtest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// P3-34b (TICK-640): the records_list_* indexes back every list order the
// P3-34a / P3-34b contract allows, and the platform database runs with JIT
// off. Evidence at 800k: docs/data/records-list-indexes.md.

// listSort is one sortable column, the index that serves it, and its
// tiebreak: a timestamp sort orders (col, id), a state or foreign-key sort
// orders (col, updated_at, id).
type listSort struct {
	col, index string
	tie        bool
}

func listSorts() []listSort {
	return []listSort{
		{"updated_at", "records_list_updated", false},
		{"created_at", "records_list_created", false},
		{"state", "records_list_state", true},
		{"assignment_group_id", "records_list_group", true},
		{"requester_id", "records_list_requester", true},
		{"assignee_id", "records_list_assignee", true},
	}
}

// literal returns a constant of col's type for a keyset or filter value.
func literal(col string) string {
	switch col {
	case "updated_at", "created_at":
		return "'2026-10-01 00:00:00+00'::timestamptz"
	case "state":
		return "'new'"
	}
	return "'0190a000-0000-7000-8000-0000000000ff'::uuid"
}

// planNodes walks an EXPLAIN (FORMAT JSON) plan and returns every node type
// and every index it scans.
func planNodes(raw string) (types, indexes []string, err error) {
	var doc []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, nil, err
	}
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		if s, ok := n["Node Type"].(string); ok {
			types = append(types, s)
		}
		if s, ok := n["Index Name"].(string); ok {
			indexes = append(indexes, s)
		}
		if kids, ok := n["Plans"].([]any); ok {
			for _, k := range kids {
				if m, ok := k.(map[string]any); ok {
					walk(m)
				}
			}
		}
	}
	for _, d := range doc {
		walk(d.Plan)
	}
	return types, indexes, nil
}

// explainAsApp plans q as nava_app in tenant A with every alternative to an
// ordered index scan disabled: a Sort then appears only when no index has
// the requested order, whatever the table's size.
func explainAsApp(t *testing.T, conn *pgx.Conn, q string) (types, indexes []string) {
	t.Helper()
	var raw string
	err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
		for _, s := range []string{"SET LOCAL enable_seqscan = off", "SET LOCAL enable_bitmapscan = off", "SET LOCAL enable_sort = off",
			"SET LOCAL enable_incremental_sort = off"} {
			if _, err := tx.Exec(context.Background(), s); err != nil {
				return err
			}
		}
		return tx.QueryRow(context.Background(), "EXPLAIN (FORMAT JSON) "+q).Scan(&raw)
	})
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	types, indexes, err = planNodes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return types, indexes
}

// Acceptance: every allowed sort, in both directions, with and without a
// keyset cursor, is an index scan on its records_list_* index with no Sort
// node. Dropping an index or changing an order turns this red.
func TestP334bListOrdersNeedNoSort(t *testing.T) {
	migratedPlatform(t)
	conn := asRole(t, "nava_app")
	where := fmt.Sprintf("r.tenant_id = '%s' AND r.entity_definition_id = '%s'", tenantA, tenantA)
	for _, s := range listSorts() {
		for _, desc := range []bool{true, false} {
			dir, nulls, cmp := "ASC", "LAST", ">"
			if desc {
				dir, nulls, cmp = "DESC", "FIRST", "<"
			}
			keyCols, keyVals := "r."+s.col, literal(s.col)
			order := fmt.Sprintf("r.%s %s NULLS %s, ", s.col, dir, nulls)
			if s.tie {
				order += "r.updated_at " + dir + ", "
				keyCols += ", r.updated_at"
				keyVals += ", " + literal("updated_at")
			}
			order += "r.id " + dir
			keyset := fmt.Sprintf(" AND (%s, r.id) %s (%s, %s)", keyCols, cmp, keyVals, literal("id"))
			for _, ks := range []string{"", keyset} {
				q := fmt.Sprintf("SELECT r.id FROM records r WHERE %s%s ORDER BY %s LIMIT 51", where, ks, order)
				types, indexes := explainAsApp(t, conn, q)
				if slices.Contains(types, "Sort") || slices.Contains(types, "Incremental Sort") {
					t.Errorf("sort on %s %s (cursor %t): plan sorts: %v", s.col, dir, ks != "", types)
				}
				if !slices.Contains(indexes, s.index) {
					t.Errorf("sort on %s %s (cursor %t): plan uses %v, want %s", s.col, dir, ks != "", indexes, s.index)
				}
			}
		}
	}
}

// Acceptance: an equality filter on state or a foreign key under the default
// sort (-updated_at) reads that value's range of its access-path index in
// order, with no Sort.
func TestP334bFilterUnderDefaultSortNeedsNoSort(t *testing.T) {
	migratedPlatform(t)
	conn := asRole(t, "nava_app")
	for _, s := range listSorts() {
		if !s.tie {
			continue
		}
		q := fmt.Sprintf(`SELECT r.id FROM records r WHERE r.tenant_id = '%s' AND r.entity_definition_id = '%s' AND r.%s = %s
			ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51`, tenantA, tenantA, s.col, literal(s.col))
		types, indexes := explainAsApp(t, conn, q)
		if slices.Contains(types, "Sort") || !slices.Contains(indexes, s.index) {
			t.Errorf("filter on %s: plan %v over %v, want %s with no Sort", s.col, types, indexes, s.index)
		}
	}
}

// Acceptance: every runtime role's session on the platform database starts
// with jit = off (the database-level setting), and the other databases keep
// the server default.
func TestP334bPlatformDatabaseJITOff(t *testing.T) {
	m := migratedCluster(t)
	for _, role := range []string{"nava_app", "nava_worker", "nava_relay", "nava_purge"} {
		if got := queryString(t, asRole(t, role), "SHOW jit"); got != "off" {
			t.Errorf("%s on %s: jit = %s, want off", role, m.PlatformDatabase, got)
		}
	}
	super := connect(t, superConfig(t))
	if got := queryString(t, super, `SELECT coalesce(string_agg(d.datname || ' ' || array_to_string(s.setconfig, ','), '; ' ORDER BY d.datname), '')
		FROM pg_catalog.pg_db_role_setting s JOIN pg_catalog.pg_database d ON d.oid = s.setdatabase
		WHERE s.setrole = 0 AND d.datname = ANY ($1)`, m.DatabaseNames()); got != m.PlatformDatabase+" jit=off" {
		t.Fatalf("database settings = %q, want %q", got, m.PlatformDatabase+" jit=off")
	}
}

// Acceptance (P3-36d, TICK-656): the group branch of an agent's list
// filtered on state reads records_list_group_state with both
// assignment_group_id and state in its Index Cond, not a Filter, for the
// capped count and for the page. Before, records_list_group matched the group
// only and dropped the other states row by row (22,950 rows removed at 800k,
// docs/data/records-list-indexes.md).
//
// The page as store.List emits it (state = ANY(array)) still sorts: on
// PostgreSQL 16 an = ANY on a non-leading index column drops the index order,
// so the Sort covers only the matching rows. With state = 'new' the same
// index gives the order and the page has no Sort; that case pins it.
//
// An empty table ties every index on cost, so the test plans over 20,000
// rows of 50 groups and 5 states with real statistics. They are written and
// analysed in one superuser transaction that rolls back (FKs and the commit
// trigger off through session_replication_role), and planned as nava_app
// under RLS in that transaction: no other test sees them.
func TestP336dGroupStateFilterIsIndexCond(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	ctx := context.Background()
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range []string{
		"SET LOCAL session_replication_role = replica",
		fmt.Sprintf(`INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, assignment_group_id, created_by, updated_at)
			SELECT '%[1]s', md5('r/' || g)::uuid, '%[1]s', 1, 'TCK' || lpad(g::text, 7, '0'),
			       (ARRAY['new','in_progress','resolved','closed','cancelled'])[1 + g %% 5],
			       md5('group/' || (g / 5) %% 50)::uuid, '%[1]s', now() - g * interval '1 minute'
			FROM generate_series(1, 20000) g`, tenantA),
		"ANALYZE records",
		"SET LOCAL ROLE nava_app",
		fmt.Sprintf("SELECT set_config('app.tenant_id', '%s', true)", tenantA),
	} {
		if _, err := tx.Exec(ctx, s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	pre := fmt.Sprintf(`r.tenant_id = '%s' AND r.entity_definition_id = '%s'
		AND r.assignment_group_id IS NOT NULL AND r.assignment_group_id = md5('group/7')::uuid AND `, tenantA, tenantA)
	order := " ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT 51"
	for _, c := range []struct {
		name, q string
		noSort  bool
	}{
		{"count", "SELECT count(*) FROM (SELECT 1 FROM records r WHERE " + pre + "r.state = ANY('{new}'::text[]) LIMIT 1001) c", true},
		{"page, state = ANY", "SELECT r.id FROM records r WHERE " + pre + "r.state = ANY('{new}'::text[])" + order, false},
		{"page, state =", "SELECT r.id FROM records r WHERE " + pre + "r.state = 'new'" + order, true},
	} {
		var raw string
		if err := tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+c.q).Scan(&raw); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		types, indexes, err := planNodes(raw)
		if err != nil {
			t.Fatal(err)
		}
		if c.noSort && (slices.Contains(types, "Sort") || slices.Contains(types, "Incremental Sort")) {
			t.Errorf("%s: plan sorts: %v", c.name, types)
		}
		if !slices.Contains(indexes, "records_list_group_state") {
			t.Errorf("%s: plan uses %v, want records_list_group_state", c.name, indexes)
			continue
		}
		cond, filter := indexScanConds(t, raw, "records_list_group_state")
		for _, col := range []string{"assignment_group_id", "state"} {
			if !strings.Contains(cond, col) {
				t.Errorf("%s: Index Cond %q lacks %s", c.name, cond, col)
			}
		}
		if strings.Contains(filter, "state") {
			t.Errorf("%s: state is a Filter (%q), want an Index Cond", c.name, filter)
		}
	}
}

// indexScanConds returns the Index Cond and Filter of the plan node that
// scans index.
func indexScanConds(t *testing.T, raw, index string) (cond, filter string) {
	t.Helper()
	var doc []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatal(err)
	}
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		if n["Index Name"] == index {
			cond, _ = n["Index Cond"].(string)
			filter, _ = n["Filter"].(string)
		}
		if kids, ok := n["Plans"].([]any); ok {
			for _, k := range kids {
				if m, ok := k.(map[string]any); ok {
					walk(m)
				}
			}
		}
	}
	for _, d := range doc {
		walk(d.Plan)
	}
	return cond, filter
}
