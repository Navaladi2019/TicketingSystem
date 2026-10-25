//go:build integration

// T29 against a real PostgreSQL, migrated with the real migrations and
// reached as the real nava_app and nava_worker roles, under RLS.
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1). Tests named TestCI10_
// are the ADR-001 CI-10 cases T29 owns (depth limit; the cycle check and audit
// row are TICK-88).

package outbox_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const approval = "ticket.approval"

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

type env struct {
	super    *pgx.Conn // the platform database as superuser, for seeding and reading back
	app      *pgxpool.Pool
	worker   *pgxpool.Pool
	platform string
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens nava_app and nava_worker pools on the platform
// database.
func migrated(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	super, err := pgx.ConnectConfig(ctx, superConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range m.DatabaseNames() {
		mustExec(t, super, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range m.RoleNames() {
		mustExec(t, super, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	_ = super.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, superConfig(t), m, testPassword, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrate.Run(ctx, roleConfig(t, m.Owner(), m.PlatformDatabase), db.Migrations(), m, g, log,
		migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	platformSuper := superConfig(t).Copy()
	platformSuper.Database = m.PlatformDatabase
	conn, err := pgx.ConnectConfig(ctx, platformSuper)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	tel, err := telemetry.New(ctx, config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	pool := func(role string) *pgxpool.Pool {
		cfg, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
		if err != nil {
			t.Fatal(err)
		}
		cfg.ConnConfig.User = role
		cfg.ConnConfig.Password, _ = testPassword(role)
		cfg.ConnConfig.Database = m.PlatformDatabase
		p, err := platformdb.NewPool(ctx, cfg, tel)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	return env{super: conn, app: pool("nava_app"), worker: pool("nava_worker"), platform: m.PlatformDatabase}
}

func roleConfig(t *testing.T, role, database string) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.User = role
	cfg.Password, _ = testPassword(role)
	cfg.Database = database
	return cfg
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seedRecord inserts, as the superuser, the tenant (if new) and a record.
func seedRecord(t *testing.T, conn *pgx.Conn, tenant, record uuid.UUID) {
	t.Helper()
	mustExec(t, conn, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	mustExec(t, conn, "INSERT INTO records (tenant_id, id) VALUES ($1, $2)", tenant, record)
}

func version(t *testing.T, conn *pgx.Conn, tenant, record uuid.UUID) int64 {
	t.Helper()
	var v int64
	if err := conn.QueryRow(context.Background(),
		"SELECT version FROM records WHERE tenant_id = $1 AND id = $2", tenant, record).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func count(t *testing.T, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type row struct {
	tenant          uuid.UUID
	kind, targetKey string
	targetSeq       int64
	causationID     *uuid.UUID
	causationDepth  int16
	traceContext    *string
}

func outboxRow(t *testing.T, conn *pgx.Conn, tenant, id uuid.UUID) row {
	t.Helper()
	var r row
	if err := conn.QueryRow(context.Background(),
		`SELECT tenant_id, kind, target_key, target_seq, causation_id, causation_depth, trace_context
		 FROM outbox WHERE tenant_id = $1 AND id = $2`, tenant, id).
		Scan(&r.tenant, &r.kind, &r.targetKey, &r.targetSeq, &r.causationID, &r.causationDepth, &r.traceContext); err != nil {
		t.Fatal(err)
	}
	return r
}

func start(record uuid.UUID, dedupe string) outbox.Entry {
	return outbox.Entry{Kind: approval, KindVersion: 1, DedupeKey: dedupe, RecordID: record,
		Target: outbox.TargetStart{Kind: approval}}
}

func signal(record uuid.UUID, dedupe string) outbox.Entry {
	return outbox.Entry{Kind: "approve", KindVersion: 1, DedupeKey: dedupe, RecordID: record,
		Target: outbox.TargetSignal{Kind: approval, SignalName: "approve"}}
}

// enqueue runs Enqueue in its own committed tenant transaction.
func enqueue(ctx context.Context, pool *pgxpool.Pool, tenant uuid.UUID, e outbox.Entry) (uuid.UUID, error) {
	var id uuid.UUID
	err := tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		var err error
		id, err = outbox.Enqueue(ctx, tx, e)
		return err
	})
	return id, err
}

// holdOpen runs Enqueue in a tenant transaction that stays open, holding the
// record's row lock, until commit is called; commit returns the transaction's
// error.
func holdOpen(t *testing.T, pool *pgxpool.Pool, tenant uuid.UUID, e outbox.Entry) (id uuid.UUID, commit func() error) {
	t.Helper()
	ids := make(chan uuid.UUID, 1)
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- tenancy.InTenantTx(context.Background(), pool, tenant, func(tx pgx.Tx) error {
			id, err := outbox.Enqueue(context.Background(), tx, e)
			if err != nil {
				return err
			}
			ids <- id
			<-release
			return nil
		})
	}()
	select {
	case id = <-ids:
	case err := <-result:
		t.Fatalf("holdOpen: %v", err)
	}
	var once sync.Once
	commit = func() error {
		once.Do(func() { close(release) })
		return <-result
	}
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	return id, commit
}

func TestEnqueue(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()

	t.Run("start row and its workflow_ref, as each producer role", func(t *testing.T) {
		for _, p := range []struct {
			role string
			pool *pgxpool.Pool
		}{{"nava_app", e.app}, {"nava_worker", e.worker}} {
			tenant, record := uuid.New(), uuid.New()
			seedRecord(t, e.super, tenant, record)
			origin := uuid.New()
			ent := start(record, "start-"+p.role)
			ent.Causation = outbox.Causation{OriginID: origin, Depth: 3}
			id, err := enqueue(ctx, p.pool, tenant, ent)
			if err != nil {
				t.Fatalf("%s: %v", p.role, err)
			}
			if id.Version() != 7 {
				t.Errorf("%s: id %s is not a UUIDv7", p.role, id)
			}
			r := outboxRow(t, e.super, tenant, id)
			if r.tenant != tenant || r.kind != approval || r.targetKey != approval+":"+id.String() ||
				r.targetSeq != 1 || r.causationID == nil || *r.causationID != origin || r.causationDepth != 3 ||
				r.traceContext != nil {
				t.Errorf("%s: row %+v", p.role, r)
			}
			if v := version(t, e.super, tenant, record); v != 1 {
				t.Errorf("%s: version %d, want 1", p.role, v)
			}
			if n := count(t, e.super, `SELECT count(*) FROM workflow_ref
				WHERE tenant_id = $1 AND outbox_id = $2 AND record_id = $3 AND kind = $4 AND closed_at IS NULL`,
				tenant, id, record, approval); n != 1 {
				t.Errorf("%s: %d workflow_ref rows, want 1", p.role, n)
			}
		}
	})

	t.Run("signal queues behind the newest open start on the record", func(t *testing.T) {
		tenant, record := uuid.New(), uuid.New()
		seedRecord(t, e.super, tenant, record)
		first, err := enqueue(ctx, e.app, tenant, start(record, "s1"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := enqueue(ctx, e.app, tenant, start(record, "s2"))
		if err != nil {
			t.Fatal(err)
		}
		// Make the order unambiguous whatever the clock's resolution.
		mustExec(t, e.super, "UPDATE workflow_ref SET created_at = created_at - interval '1 minute' WHERE outbox_id = $1", first)
		sig, err := enqueue(ctx, e.app, tenant, signal(record, "sig"))
		if err != nil {
			t.Fatal(err)
		}
		r := outboxRow(t, e.super, tenant, sig)
		if r.kind != "approve" || r.targetKey != approval+":"+second.String() || r.targetSeq != 3 {
			t.Errorf("signal row %+v, want kind approve, target %s:%s, seq 3", r, approval, second)
		}
		if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 2 {
			t.Errorf("%d workflow_ref rows, want 2 (a signal inserts none)", n)
		}
	})

	t.Run("trace context is the request's traceparent only", func(t *testing.T) {
		tenant, record := uuid.New(), uuid.New()
		seedRecord(t, e.super, tenant, record)
		ts, err := trace.ParseTraceState("vendor=x")
		if err != nil {
			t.Fatal(err)
		}
		sctx := trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{0xaa}, SpanID: trace.SpanID{0xbb}, TraceFlags: trace.FlagsSampled, TraceState: ts,
		}))
		id, err := enqueue(sctx, e.app, tenant, start(record, "traced"))
		if err != nil {
			t.Fatal(err)
		}
		want := "00-aa000000000000000000000000000000-bb00000000000000-01"
		if r := outboxRow(t, e.super, tenant, id); r.traceContext == nil || *r.traceContext != want {
			t.Fatalf("trace_context %v, want %s", r.traceContext, want)
		}
		// The relay reads it with its own grants (db/grants.yaml).
		relay, err := pgx.ConnectConfig(ctx, roleConfig(t, "nava_relay", e.platform))
		if err != nil {
			t.Fatal(err)
		}
		defer relay.Close(ctx)
		var got string
		if err := relay.QueryRow(ctx, "SELECT trace_context FROM outbox WHERE tenant_id = $1 AND id = $2", tenant, id).
			Scan(&got); err != nil || got != want {
			t.Fatalf("relay read %q, %v; want %s", got, err, want)
		}
	})
}

// TestCI10_DepthBeyondLimitWritesNothing: I9's depth limit (D-P7: 8).
func TestCI10_DepthBeyondLimitWritesNothing(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant, record := uuid.New(), uuid.New()
	seedRecord(t, e.super, tenant, record)

	at := start(record, "depth-8")
	at.Causation = outbox.Causation{OriginID: uuid.New(), Depth: config.CascadeDepthLimit}
	if _, err := enqueue(ctx, e.app, tenant, at); err != nil {
		t.Fatalf("depth 8: %v", err)
	}

	over := start(record, "depth-9")
	over.Causation = outbox.Causation{OriginID: uuid.New(), Depth: config.CascadeDepthLimit + 1}
	// The caller's transaction stays usable: nothing was sent to it.
	err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		if _, err := outbox.Enqueue(ctx, tx, over); !errors.Is(err, outbox.ErrCascadeTooDeep) {
			t.Errorf("depth 9: %v, want ErrCascadeTooDeep", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 1 {
		t.Errorf("%d outbox rows, want 1 (depth 9 wrote nothing)", n)
	}
	if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 1 {
		t.Errorf("%d workflow_ref rows, want 1", n)
	}
	if v := version(t, e.super, tenant, record); v != 1 {
		t.Errorf("version %d, want 1 (depth 9 did not bump it)", v)
	}
}

// TestCI10_ConcurrentProducersConsecutiveTargetSeq: the record's row lock
// orders producers, so target_seq follows commit order with no gap.
func TestCI10_ConcurrentProducersConsecutiveTargetSeq(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant, record := uuid.New(), uuid.New()
	seedRecord(t, e.super, tenant, record)
	if _, err := enqueue(ctx, e.app, tenant, start(record, "head")); err != nil {
		t.Fatal(err)
	}

	t.Run("the second producer waits for the first to commit", func(t *testing.T) {
		id1, commit := holdOpen(t, e.app, tenant, signal(record, "p1"))

		type result struct {
			id  uuid.UUID
			err error
		}
		done := make(chan result, 1)
		go func() {
			id, err := enqueue(ctx, e.app, tenant, signal(record, "p2"))
			done <- result{id, err}
		}()
		deadline := time.Now().Add(10 * time.Second)
		for count(t, e.super, `SELECT count(*) FROM pg_stat_activity
			WHERE usename = 'nava_app' AND wait_event_type = 'Lock' AND query LIKE 'UPDATE records%'`) != 1 {
			if time.Now().After(deadline) {
				t.Fatal("the second producer never waited on the record lock")
			}
			time.Sleep(20 * time.Millisecond)
		}
		select {
		case r := <-done:
			t.Fatalf("second producer finished before the first committed: %v", r)
		default:
		}
		if err := commit(); err != nil {
			t.Fatal(err)
		}
		r := <-done
		if r.err != nil {
			t.Fatal(r.err)
		}
		s1 := outboxRow(t, e.super, tenant, id1).targetSeq
		s2 := outboxRow(t, e.super, tenant, r.id).targetSeq
		if s2 != s1+1 {
			t.Fatalf("target_seq %d then %d, want consecutive in commit order", s1, s2)
		}
	})

	t.Run("many producers take every version once", func(t *testing.T) {
		const n = 16
		before := version(t, e.super, tenant, record)
		var wg sync.WaitGroup
		errs := make(chan error, n)
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := enqueue(ctx, e.app, tenant, signal(record, fmt.Sprintf("many-%d", i)))
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		var lo, hi int64
		var rows int
		if err := e.super.QueryRow(ctx, `SELECT min(target_seq), max(target_seq), count(DISTINCT target_seq)
			FROM outbox WHERE tenant_id = $1 AND dedupe_key LIKE 'many-%'`, tenant).Scan(&lo, &hi, &rows); err != nil {
			t.Fatal(err)
		}
		if lo != before+1 || hi != before+n || rows != n {
			t.Fatalf("target_seq %d..%d over %d distinct, want %d..%d over %d", lo, hi, rows, before+1, before+n, n)
		}
	})
}

// TestCI10_DuplicateDedupeKeyWritesOneRow: a redelivered write is a no-op that
// returns the first row's id.
func TestCI10_DuplicateDedupeKeyWritesOneRow(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()

	t.Run("retried after commit", func(t *testing.T) {
		tenant, record := uuid.New(), uuid.New()
		seedRecord(t, e.super, tenant, record)
		first, err := enqueue(ctx, e.app, tenant, start(record, "dup"))
		if err != nil {
			t.Fatal(err)
		}
		again, err := enqueue(ctx, e.app, tenant, start(record, "dup"))
		if err != nil || again != first {
			t.Fatalf("duplicate = %s, %v; want %s, nil", again, err, first)
		}
		if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 1 {
			t.Errorf("%d outbox rows, want 1", n)
		}
		if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 1 {
			t.Errorf("%d workflow_ref rows, want 1", n)
		}
		if v := version(t, e.super, tenant, record); v != 1 {
			t.Errorf("version %d, want 1 (the retry did not bump it)", v)
		}
	})

	t.Run("committed concurrently", func(t *testing.T) {
		tenant, record := uuid.New(), uuid.New()
		seedRecord(t, e.super, tenant, record)
		first, commit := holdOpen(t, e.app, tenant, start(record, "race"))
		type result struct {
			id  uuid.UUID
			err error
		}
		done := make(chan result, 1)
		go func() {
			id, err := enqueue(ctx, e.app, tenant, start(record, "race"))
			done <- result{id, err}
		}()
		deadline := time.Now().Add(10 * time.Second)
		for count(t, e.super, `SELECT count(*) FROM pg_stat_activity
			WHERE usename = 'nava_app' AND wait_event_type = 'Lock'`) != 1 {
			if time.Now().After(deadline) {
				t.Fatal("the duplicate never waited")
			}
			time.Sleep(20 * time.Millisecond)
		}
		if err := commit(); err != nil {
			t.Fatal(err)
		}
		r := <-done
		if r.err != nil || r.id != first {
			t.Fatalf("duplicate = %s, %v; want %s, nil", r.id, r.err, first)
		}
		if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 1 {
			t.Errorf("%d outbox rows, want 1", n)
		}
		if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 1 {
			t.Errorf("%d workflow_ref rows, want 1", n)
		}
	})
}

func TestCI10_SignalWithNoOpenWorkflow(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant, record := uuid.New(), uuid.New()
	seedRecord(t, e.super, tenant, record)

	check := func(name string) {
		t.Helper()
		if _, err := enqueue(ctx, e.app, tenant, signal(record, name)); !errors.Is(err, outbox.ErrNoOpenWorkflow) {
			t.Errorf("%s: %v, want ErrNoOpenWorkflow", name, err)
		}
	}
	check("none at all")

	other := outbox.Entry{Kind: "ticket.escalation", KindVersion: 1, DedupeKey: "other", RecordID: record,
		Target: outbox.TargetStart{Kind: "ticket.escalation"}}
	if _, err := enqueue(ctx, e.app, tenant, other); err != nil {
		t.Fatal(err)
	}
	check("only another kind open")

	closed, err := enqueue(ctx, e.app, tenant, start(record, "closed"))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, e.super, "UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1 AND outbox_id = $2", tenant, closed)
	check("only a closed one")

	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'approve'", tenant); n != 0 {
		t.Errorf("%d signal rows written, want 0", n)
	}
}

// TestEnqueueTenantIsolation: tenant A's transaction cannot reach tenant B's
// record, workflow or dedupe key, even where B uses the same ids. It runs as
// each producer role: nava_worker takes its tenant from a header, so its
// transactions must be held to the same policies as nava_app's.
func TestEnqueueTenantIsolation(t *testing.T) {
	e := migrated(t)
	for _, p := range []struct {
		role string
		pool *pgxpool.Pool
	}{{"nava_app", e.app}, {"nava_worker", e.worker}} {
		t.Run(p.role, func(t *testing.T) { tenantIsolation(t, e, p.pool) })
	}
}

func tenantIsolation(t *testing.T, e env, pool *pgxpool.Pool) {
	ctx := context.Background()
	a, b, shared := uuid.New(), uuid.New(), uuid.New()
	seedRecord(t, e.super, a, shared)
	seedRecord(t, e.super, b, shared)
	bOnly := uuid.New()
	seedRecord(t, e.super, b, bOnly)
	bStart, err := enqueue(ctx, pool, b, start(shared, "shared-key"))
	if err != nil {
		t.Fatal(err)
	}
	bVersion := version(t, e.super, b, shared)

	if _, err := enqueue(ctx, pool, a, start(bOnly, "a1")); !errors.Is(err, outbox.ErrRecordNotFound) {
		t.Errorf("B's record from A: %v, want ErrRecordNotFound", err)
	}
	if _, err := enqueue(ctx, pool, a, signal(shared, "a2")); !errors.Is(err, outbox.ErrNoOpenWorkflow) {
		t.Errorf("signal B's open workflow from A: %v, want ErrNoOpenWorkflow", err)
	}
	aStart, err := enqueue(ctx, pool, a, start(shared, "shared-key"))
	if err != nil {
		t.Fatal(err)
	}
	if aStart == bStart {
		t.Error("A's dedupe key resolved to B's row")
	}
	if r := outboxRow(t, e.super, a, aStart); r.tenant != a {
		t.Errorf("row tenant %s, want %s", r.tenant, a)
	}
	if v := version(t, e.super, b, shared); v != bVersion {
		t.Errorf("B's version moved from %d to %d", bVersion, v)
	}
	if v := version(t, e.super, b, bOnly); v != 0 {
		t.Errorf("B's other record version %d, want 0", v)
	}

	// A transaction with no tenant sees no record at all.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := outbox.Enqueue(ctx, tx, start(shared, "no-tenant")); !errors.Is(err, outbox.ErrRecordNotFound) {
		t.Errorf("no tenant: %v, want ErrRecordNotFound", err)
	}
}
