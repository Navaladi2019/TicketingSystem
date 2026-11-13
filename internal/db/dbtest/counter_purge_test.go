//go:build integration

package dbtest

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// S18 P6-10 (TICK-675), ADR-005 §4.3: nava_purge deletes record_counters rows
// more than 2 days past period_end, under the paired record_counters_purge
// policies, and nothing else. The plan is in purge_index_test.go.

// counterPeriod is one seeded counter row: its scope and its period_end as an
// interval from now() (NULL for a field-only or default-scope row).
type counterPeriod struct {
	scope string
	end   *string // nil: period_end NULL
	gone  bool    // the purge deletes it
}

func counterPeriods() []counterPeriod {
	at := func(s string) *string { return &s }
	return []counterPeriod{
		{"e1|d:202609", at("-30 days"), true},
		{"e1|d:202610", at("-3 days"), true},
		// 2 days and 1 minute ago: just past the grace.
		{"e1|d:20261110", at("-2 days -1 minute"), true},
		// Inside the 2-day grace: closed, but a time-zone change could still
		// reopen its local date.
		{"e1|d:20261111", at("-1 day -23 hours"), false},
		{"e1|d:20261112", at("-1 hour"), false},
		// Open periods.
		{"e1|d:20261113", at("1 hour"), false},
		{"e1|d:202611", at("17 days"), false},
		// Field-only and default scope: period_end NULL, never purged.
		{"e1|f:BLR", nil, false},
		{"", nil, false},
	}
}

// seedCounterPeriods gives each tenant one entity type (id = the tenant id)
// and one counter row per counterPeriods entry.
func seedCounterPeriods(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, id := range tenants {
		exec(t, super, insertType, id, "p610", "PER")
		for _, c := range counterPeriods() {
			exec(t, super, `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, period_end, next)
				VALUES ($1, $1, $2, now() + $3::interval, 5)`, id, c.scope, c.end)
		}
	}
}

// scopesOf is the sorted scope keys of tenant's counter rows that conn sees
// (the superuser sees all of them, past RLS).
func scopesOf(t *testing.T, conn *pgx.Conn, tenant string) []string {
	t.Helper()
	rows, err := conn.Query(context.Background(), "SELECT scope_key FROM record_counters WHERE tenant_id = $1", tenant)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// wantScopes is the sorted scope keys of the seeded rows the purge deletes
// (gone) or keeps (!gone); all of them for both.
func wantScopes(gone ...bool) []string {
	var s []string
	for _, c := range counterPeriods() {
		if slices.Contains(gone, c.gone) {
			s = append(s, c.scope)
		}
	}
	slices.Sort(s)
	return s
}

// Acceptance: rows past the grace are deleted in both tenants; rows inside
// it, open rows and NULL rows stay.
func TestP610PurgeDeletesExpiredPeriodCountersOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedCounterPeriods(t, super, tenantA, tenantB)

	rs := newPurger(t, purgePool(t), purge.Options{}).Once(context.Background())

	gone := 0
	for _, c := range counterPeriods() {
		if c.gone {
			gone++
		}
	}
	if r := resultFor(t, rs, purge.TableRecordCounters); r.Err != nil || r.Skipped || r.Deleted != int64(2*gone) {
		t.Errorf("record_counters: %+v, want %d deleted", r, 2*gone)
	}
	want := wantScopes(false)
	for _, tenant := range []string{tenantA, tenantB} {
		if got := scopesOf(t, super, tenant); !slices.Equal(got, want) {
			t.Errorf("tenant %s keeps %q, want %q", tenant, got, want)
		}
	}
}

// Batches of BatchSize, stopping on the first short one: 2 tenants x 3 expired
// rows at batch size 2 is 2, 2, 2 and an empty last batch.
func TestP610PurgeBatchesCounterRows(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedCounterPeriods(t, super, tenantA, tenantB)

	rs := newPurger(t, purgePool(t), purge.Options{BatchSize: 2}).Once(context.Background())
	if r := resultFor(t, rs, purge.TableRecordCounters); r.Err != nil || r.Deleted != 6 {
		t.Errorf("record_counters: %+v, want 6 deleted", r)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM record_counters WHERE period_end < now() - interval '2 days'"); got != "0" {
		t.Errorf("%s expired counter rows left, want 0", got)
	}
}

// ADR-005 §5: nava_purge cannot see or delete a row whose period_end is in
// the future, within the 2-day grace, or NULL, even by naming it; it cannot
// read next, move a period_end into the window or write a counter. nava_app
// and nava_worker have no DELETE, even of their own tenant's expired rows.
func TestP610PurgeRoleReachOnCounters(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedCounterPeriods(t, super, tenantA, tenantB)
	ctx := context.Background()
	purgeConn := asRole(t, "nava_purge")

	// It sees exactly the expired rows of both tenants.
	for _, tenant := range []string{tenantA, tenantB} {
		if got, want := scopesOf(t, purgeConn, tenant), wantScopes(true); !slices.Equal(got, want) {
			t.Errorf("nava_purge sees %q of tenant %s, want %q", got, tenant, want)
		}
	}
	// Naming a kept row deletes nothing.
	for _, c := range counterPeriods() {
		if c.gone {
			continue
		}
		tag, err := purgeConn.Exec(ctx, "DELETE FROM record_counters WHERE tenant_id = $1 AND entity_definition_id = $1 AND scope_key = $2",
			tenantB, c.scope)
		if err != nil || tag.RowsAffected() != 0 {
			t.Errorf("nava_purge deletes kept row %q: %d rows, err %v", c.scope, tag.RowsAffected(), err)
		}
	}
	for _, q := range []string{
		"SELECT next FROM record_counters",
		"SELECT * FROM record_counters",
		"UPDATE record_counters SET period_end = now() - interval '3 days'",
		"INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ('" + tenantB + "', '" + tenantB + "', 'x', 2)",
		"TRUNCATE record_counters",
	} {
		_, err := purgeConn.Exec(ctx, q)
		wantSQLState(t, q, err, "42501")
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		_, err := execIn(asRole(t, role), tenantA, "DELETE FROM record_counters WHERE tenant_id = $1", tenantA)
		wantSQLState(t, role+" DELETE", err, "42501")
	}
	all := wantScopes(true, false)
	for _, tenant := range []string{tenantA, tenantB} {
		if got := scopesOf(t, super, tenant); !slices.Equal(got, all) {
			t.Errorf("tenant %s holds %q, want every seeded row %q", tenant, got, all)
		}
	}
}
