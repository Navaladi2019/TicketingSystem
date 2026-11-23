//go:build integration

package dbtest

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/estimates"
)

// TICK-1071 (S31; ADR-010-A2 A2-9, A2-10): the refresh job, run as
// nava_worker against the migrated platform.

// addRecords inserts n records of type def for tenant, numbered from
// prefix, with their audit entries.
func addRecords(t *testing.T, super *pgx.Conn, tenant, def, prefix string, n int) {
	t.Helper()
	exec(t, super, withRecordEntry(`INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state)
		SELECT $1::uuid, gen_random_uuid(), $2::uuid, 1, $3::text || lpad(g::text, 7, '0'), 'open' FROM generate_series(1, $4::int) g`),
		tenant, def, prefix, n)
}

// secondType gives tenant a second entity type and returns its id.
func secondType(t *testing.T, super *pgx.Conn, tenant string) string {
	t.Helper()
	exec(t, super, insertTypeNewID, tenant, "request", "REQ")
	return queryString(t, super, "SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = 'request'", tenant)
}

// estimatesOf reads tenant's rows as type=count, through nava_app under the
// tenant, so RLS scopes them.
func estimatesOf(t *testing.T, tenant string) map[string]int64 {
	t.Helper()
	got := map[string]int64{}
	err := inTenantTx(asRole(t, "nava_app"), tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(context.Background(), "SELECT entity_definition_id::text, live_records FROM entity_record_estimates")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var def string
			var n int64
			if err := rows.Scan(&def, &n); err != nil {
				return err
			}
			got[def] = n
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func sameCounts(t *testing.T, what string, got, want map[string]int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: rows = %v, want %v", what, got, want)
		return
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: rows = %v, want %v", what, got, want)
			return
		}
	}
}

func refreshedAt(t *testing.T, super *pgx.Conn, tenant string) time.Time {
	t.Helper()
	var at time.Time
	if err := super.QueryRow(context.Background(), "SELECT max(refreshed_at) FROM entity_record_estimates WHERE tenant_id = $1", tenant).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// workerPool is a pool connected as nava_worker, the job's role.
func workerPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig = roleConfig(t, "nava_worker", "nava")
	pool, err := pgxpool.NewWithConfig(context.Background(), pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// failFor makes one tenant's upsert fail.
type failFor struct {
	estimates.Store
	tenant string
}

type failTx struct {
	pgx.Tx
	tenant string
}

func (b failFor) InTenant(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	return b.Store.InTenant(ctx, tenant, func(tx pgx.Tx) error { return fn(failTx{tx, b.tenant}) })
}

func (tx failTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if len(args) == 1 && args[0] == tx.tenant && strings.HasPrefix(sql, "INSERT") {
		return pgconn.CommandTag{}, errors.New("injected")
	}
	return tx.Tx.Exec(ctx, sql, args...)
}

func newJob(t *testing.T, db estimates.Store, now func() time.Time) (*estimates.Refresher, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	r, err := estimates.New(db, metric.NewMeterProvider(metric.WithReader(reader)),
		slog.New(slog.DiscardHandler), estimates.Options{Interval: time.Hour, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return r, reader
}

func metricValue(t *testing.T, reader *metric.ManualReader, name string) float64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				var s int64
				for _, p := range d.DataPoints {
					s += p.Value
				}
				return float64(s)
			case metricdata.Gauge[float64]:
				return d.DataPoints[0].Value
			}
		}
	}
	t.Fatalf("metric %s not collected", name)
	return 0
}

// Acceptance 1 and 2: one cycle writes exactly A = {3}, B = {5, 2}, each
// visible only under its own tenant; a second cycle changes only
// refreshed_at; 4 more inserts show 7. The bare T09 seed record names no
// type and gets no row.
func TestTICK1071RefreshScopesAndIsIdempotent(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA, tenantB)
	b2 := secondType(t, super, tenantB)
	addRecords(t, super, tenantA, tenantA, "TCK", 3)
	addRecords(t, super, tenantB, tenantB, "TCK", 5)
	addRecords(t, super, tenantB, b2, "REQ", 2)

	job, _ := newJob(t, estimates.PoolStore{Pool: workerPool(t)}, time.Now)
	if n := job.Cycle(context.Background()); n != 0 {
		t.Fatalf("failed tenants = %d, want 0", n)
	}
	sameCounts(t, "A", estimatesOf(t, tenantA), map[string]int64{tenantA: 3})
	sameCounts(t, "B", estimatesOf(t, tenantB), map[string]int64{tenantB: 5, b2: 2})
	if n := queryString(t, super, "SELECT count(*)::text FROM entity_record_estimates"); n != "3" {
		t.Errorf("rows = %s, want 3", n)
	}

	first := refreshedAt(t, super, tenantA)
	job.Cycle(context.Background())
	sameCounts(t, "A again", estimatesOf(t, tenantA), map[string]int64{tenantA: 3})
	sameCounts(t, "B again", estimatesOf(t, tenantB), map[string]int64{tenantB: 5, b2: 2})
	if !refreshedAt(t, super, tenantA).After(first) {
		t.Error("refreshed_at did not move")
	}

	addRecords(t, super, tenantA, tenantA, "TCX", 4)
	job.Cycle(context.Background())
	sameCounts(t, "A after inserts", estimatesOf(t, tenantA), map[string]int64{tenantA: 7})
}

// Acceptance 6: a type whose records are all gone is set to 0, not left stale.
func TestTICK1071EmptiedTypeIsZeroed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA)
	addRecords(t, super, tenantA, tenantA, "TCK", 2)
	job, _ := newJob(t, estimates.PoolStore{Pool: workerPool(t)}, time.Now)
	job.Cycle(context.Background())
	sameCounts(t, "before", estimatesOf(t, tenantA), map[string]int64{tenantA: 2})

	// No runtime role deletes records; the superuser stands in for an
	// erasure, with the audit triggers off for this fixture.
	exec(t, super, "SET session_replication_role = replica")
	exec(t, super, "DELETE FROM records WHERE tenant_id = $1", tenantA)
	job.Cycle(context.Background())
	sameCounts(t, "after", estimatesOf(t, tenantA), map[string]int64{tenantA: 0})
}

// Acceptance 3 and 4: a tenant whose transaction fails is counted and keeps
// its previous row, the other tenant is still refreshed, and the age gauge
// keeps counting from the failed tenant's last refresh; one clean cycle
// brings it under 60 s.
func TestTICK1071FailSoftPerTenantAndAge(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedEstimateTypes(t, super, tenantA, tenantB)
	addRecords(t, super, tenantA, tenantA, "TCK", 1)
	addRecords(t, super, tenantB, tenantB, "TCK", 1)
	worker := estimates.PoolStore{Pool: workerPool(t)}

	clock := time.Now()
	now := func() time.Time { return clock }
	clean, cleanReader := newJob(t, worker, now)
	clean.Cycle(context.Background())
	sameCounts(t, "A", estimatesOf(t, tenantA), map[string]int64{tenantA: 1})

	addRecords(t, super, tenantA, tenantA, "TCX", 2)
	addRecords(t, super, tenantB, tenantB, "TCX", 2)
	bad, badReader := newJob(t, failFor{worker, tenantA}, now)
	bad.Cycle(context.Background())
	clock = clock.Add(4 * time.Hour)
	if n := bad.Cycle(context.Background()); n != 1 {
		t.Fatalf("failed tenants = %d, want 1", n)
	}
	if got := metricValue(t, badReader, estimates.FailuresMetric); got != 2 {
		t.Errorf("failures = %v, want 2 (one per cycle)", got)
	}
	sameCounts(t, "A untouched", estimatesOf(t, tenantA), map[string]int64{tenantA: 1})
	sameCounts(t, "B refreshed", estimatesOf(t, tenantB), map[string]int64{tenantB: 3})
	if got := metricValue(t, badReader, estimates.AgeMetric); got <= 10800 {
		t.Errorf("age = %v, want over 10800 while A keeps failing", got)
	}

	clean.Cycle(context.Background())
	if got := metricValue(t, cleanReader, estimates.AgeMetric); got >= 60 {
		t.Errorf("age after a clean cycle = %v, want under 60", got)
	}
	sameCounts(t, "A recovered", estimatesOf(t, tenantA), map[string]int64{tenantA: 3})
}
