//go:build integration

package dbtest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// T13 (TICK-68): the purge job, connected as nava_purge to a migrated
// platform database.

// purgePool connects as nava_purge. The pool lets two purgers run at once.
func purgePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig = roleConfig(t, "nava_purge", "nava")
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newPurger(t *testing.T, db purge.Beginner, opts purge.Options) *purge.Purger {
	t.Helper()
	log, _ := newLog()
	p, err := purge.New(db, metricnoop.NewMeterProvider(), log, opts)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// seedPurgeable adds, per tenant, rows on both sides of every window:
// 2 expired idempotency keys; outbox rows dispatched 8 days ago and dead 31
// days ago (gone) and dispatched 6 days ago and dead 29 days ago (kept);
// a workflow_ref closed 31 days ago (gone) and one closed 29 days ago (kept).
// seedTenants has already given each tenant one unexpired row per table.
func seedPurgeable(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, id := range tenants {
		for _, k := range []string{"old-1", "old-2"} {
			exec(t, super, `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
				VALUES ($1, $1, $2, '\x00', now() - interval '1 hour')`, id, k)
		}
		for _, o := range []struct{ key, col, age string }{
			{"d8", "dispatched_at", "8 days"}, {"x31", "dead_at", "31 days"},
			{"d6", "dispatched_at", "6 days"}, {"x29", "dead_at", "29 days"},
		} {
			exec(t, super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, `+o.col+`)
				VALUES (gen_random_uuid(), $1::uuid, 'workflow.start', $2, 'rec:x', 1, now() - $3::interval)`, id, o.key, o.age)
		}
		for _, age := range []string{"31 days", "29 days"} {
			exec(t, super, `INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, closed_at)
				VALUES ($1, gen_random_uuid(), $1, 'approval', now() - $2::interval)`, id, age)
		}
	}
}

func count(t *testing.T, super *pgx.Conn, table string) int {
	t.Helper()
	var n int
	if err := super.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func resultFor(t *testing.T, rs []purge.Result, table string) purge.Result {
	t.Helper()
	for _, r := range rs {
		if r.Table == table {
			return r
		}
	}
	t.Fatalf("no result for %s in %+v", table, rs)
	return purge.Result{}
}

// Acceptance: expired rows of two tenants are deleted; unexpired rows survive.
func TestT13PurgeDeletesExpiredRowsOfTwoTenants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB)

	rs := newPurger(t, purgePool(t), purge.Options{}).Once(context.Background())

	for table, want := range map[string]int64{
		purge.TableIdempotencyKeys: 4, purge.TableOutbox: 4, purge.TableWorkflowRef: 2,
	} {
		r := resultFor(t, rs, table)
		if r.Err != nil || r.Skipped || r.Deleted != want {
			t.Errorf("%s: %+v, want %d deleted", table, r, want)
		}
	}
	// Left: per tenant the seed row plus the rows inside the windows.
	for table, want := range map[string]int{"idempotency_keys": 2, "outbox": 6, "workflow_ref": 4} {
		if got := count(t, super, table); got != want {
			t.Errorf("%s holds %d rows after the purge, want %d", table, got, want)
		}
	}
	if got := queryString(t, super, `SELECT string_agg(dedupe_key, ',' ORDER BY dedupe_key) FROM outbox WHERE tenant_id = $1`, tenantA); got != "d6,seed,x29" {
		t.Errorf("tenant A outbox keys = %s, want d6,seed,x29", got)
	}
	// A second run finds nothing.
	for _, r := range newPurger(t, purgePool(t), purge.Options{}).Once(context.Background()) {
		if r.Err != nil || r.Deleted != 0 {
			t.Errorf("second run %+v, want nothing deleted", r)
		}
	}
}

// Statements are batched: five expired rows with a batch of two take three.
func TestT13PurgeDeletesInBatches(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB) // 4 expired keys
	exec(t, super, `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
		VALUES ($1, $1, 'old-3', '\x00', now() - interval '1 hour')`, tenantA)

	rs := newPurger(t, purgePool(t), purge.Options{BatchSize: 2}).Once(context.Background())
	if r := resultFor(t, rs, purge.TableIdempotencyKeys); r.Err != nil || r.Deleted != 5 {
		t.Fatalf("idempotency_keys: %+v, want 5 deleted", r)
	}
	if got := count(t, super, "idempotency_keys"); got != 2 {
		t.Errorf("idempotency_keys holds %d rows, want the 2 unexpired", got)
	}
}

// Longer windows than the policy's keep rows the policy would let go.
func TestT13PurgeWindowsOnlyNarrow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPurgeable(t, super, tenantA)

	day := 24 * time.Hour
	newPurger(t, purgePool(t), purge.Options{Windows: purge.Windows{
		OutboxDispatched: 10 * day, OutboxDead: 40 * day, WorkflowRef: 40 * day,
	}}).Once(context.Background())

	// Only the expired idempotency keys go; outbox and workflow_ref are whole.
	for table, want := range map[string]int{"idempotency_keys": 1, "outbox": 5, "workflow_ref": 3} {
		if got := count(t, super, table); got != want {
			t.Errorf("%s holds %d rows, want %d", table, got, want)
		}
	}
}

// Acceptance: the job's role cannot select payload. The assertion is the
// permission error, not an empty result.
func TestT13PurgeRoleCannotSelectPayload(t *testing.T) {
	migratedPlatform(t)
	seedTenants(t, superNava(t), []string{tenantA}, nil)
	purgeConn := asRole(t, "nava_purge")
	for _, q := range []string{
		"SELECT payload FROM outbox",
		"SELECT payload FROM outbox WHERE false",
		"SELECT * FROM outbox",
		"SELECT count(payload) FROM outbox",
		"SELECT response_body FROM idempotency_keys",
	} {
		_, err := purgeConn.Exec(context.Background(), q)
		wantSQLState(t, q, err, "42501")
	}
}

func lockOutbox(t *testing.T) *pgx.Conn {
	t.Helper()
	holder := superNava(t)
	exec(t, holder, "BEGIN")
	exec(t, holder, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", purge.LockName(purge.TableOutbox))
	return holder
}

// A purge that finds the table's advisory lock taken skips that table, and
// only that table.
func TestT13SecondPurgeOnSameTableSkips(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPurgeable(t, super, tenantA)

	holder := lockOutbox(t) // another replica, mid-batch on outbox

	p := newPurger(t, purgePool(t), purge.Options{})
	rs := p.Once(context.Background())
	if r := resultFor(t, rs, purge.TableOutbox); r.Err != nil || !r.Skipped || r.Deleted != 0 {
		t.Fatalf("outbox while locked: %+v, want skipped with nothing deleted", r)
	}
	if got := count(t, super, "outbox"); got != 5 {
		t.Errorf("outbox holds %d rows while locked, want all 5", got)
	}
	for _, table := range []string{purge.TableIdempotencyKeys, purge.TableWorkflowRef} {
		if r := resultFor(t, rs, table); r.Skipped || r.Err != nil || r.Deleted == 0 {
			t.Errorf("%s: %+v, want purged: the lock is per table", table, r)
		}
	}

	exec(t, holder, "ROLLBACK")
	rs = p.Once(context.Background())
	if r := resultFor(t, rs, purge.TableOutbox); r.Err != nil || r.Skipped || r.Deleted != 2 {
		t.Errorf("outbox after the lock is released: %+v, want 2 deleted", r)
	}
}

// Two purgers racing on the same rows never error and never delete a row twice.
func TestT13ConcurrentPurgesDeleteEachRowOnce(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB)
	for range 20 {
		exec(t, super, `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at)
			VALUES ($1, $1, gen_random_uuid()::text, '\x00', now() - interval '1 hour')`, tenantA)
	}
	pool := purgePool(t)

	var mu sync.Mutex
	var deleted int64
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for _, r := range newPurger(t, pool, purge.Options{BatchSize: 3}).Once(context.Background()) {
				if r.Err != nil {
					t.Errorf("%s: %v", r.Table, r.Err)
				}
				mu.Lock()
				deleted += r.Deleted
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	for _, r := range newPurger(t, pool, purge.Options{}).Once(context.Background()) {
		deleted += r.Deleted
	}
	if want := int64(24 + 4 + 2); deleted != want {
		t.Errorf("deleted %d rows in total, want %d", deleted, want)
	}
	for table, want := range map[string]int{"idempotency_keys": 2, "outbox": 6, "workflow_ref": 4} {
		if got := count(t, super, table); got != want {
			t.Errorf("%s holds %d rows, want %d", table, got, want)
		}
	}
}

// The deleted-rows metric has the table as its only attribute.
func TestT13DeletedRowsMetric(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB)

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	log, _ := newLog()
	p, err := purge.New(purgePool(t), mp, log, purge.Options{})
	if err != nil {
		t.Fatal(err)
	}
	p.Once(context.Background())

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "nava.purge.deleted_rows" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				if dp.Attributes.Len() != 1 {
					t.Errorf("attributes = %v, want only table", dp.Attributes)
				}
				v, _ := dp.Attributes.Value(attribute.Key("table"))
				got[v.AsString()] = dp.Value
			}
		}
	}
	want := map[string]int64{"idempotency_keys": 4, "outbox": 4, "workflow_ref": 2}
	for table, n := range want {
		if got[table] != n {
			t.Errorf("deleted_rows{table=%s} = %d, want %d", table, got[table], n)
		}
	}
}
