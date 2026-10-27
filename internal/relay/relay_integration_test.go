//go:build integration

// T30a against a real PostgreSQL, migrated with the real migrations and
// reached as the real nava_relay role, under RLS and its column grants.
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t testing.TB) *pgx.ConnConfig {
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
	super *pgx.Conn // the platform database as superuser, for seeding and reading back
	app   *pgxpool.Pool
	relay *pgxpool.Pool
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens nava_app and nava_relay pools on the platform
// database.
func migrated(t testing.TB) env {
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

	pool := func(role string) *pgxpool.Pool {
		cfg, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
		if err != nil {
			t.Fatal(err)
		}
		cfg.ConnConfig.User = role
		cfg.ConnConfig.Password, _ = testPassword(role)
		cfg.ConnConfig.Database = m.PlatformDatabase
		p, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	return env{super: conn, app: pool("nava_app"), relay: pool("nava_relay")}
}

func roleConfig(t testing.TB, role, database string) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.User = role
	cfg.Password, _ = testPassword(role)
	cfg.Database = database
	return cfg
}

func mustExec(t testing.TB, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func count(t testing.TB, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// probe is a Relay on e.relay whose metric and log the test reads.
type probe struct {
	*Relay
	reader *sdkmetric.ManualReader
	logs   *bytes.Buffer
}

func newProbe(t *testing.T, q Querier, opts Options) probe {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	var logs bytes.Buffer
	r, err := New(q, sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
		slog.New(slog.NewJSONHandler(&logs, nil)), opts)
	if err != nil {
		t.Fatal(err)
	}
	return probe{r, reader, &logs}
}

func (p probe) histogram(t *testing.T) metricdata.HistogramDataPoint[int64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := p.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	return scannedHistogram(t, rm)
}

// seed inserts, as the superuser, n pending rows for tenant: row i is
// target_seq i+1 on target (i / perTarget), and available i milliseconds
// apart, oldest first. It returns the ids in that order.
func seed(t *testing.T, conn *pgx.Conn, tenant uuid.UUID, n, perTarget int) []uuid.UUID {
	t.Helper()
	mustExec(t, conn, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	ids := make([]uuid.UUID, n)
	rows := make([][]any, n)
	base := time.Now().Add(-time.Hour)
	for i := range n {
		ids[i] = uuid.Must(uuid.NewV7())
		rows[i] = []any{ids[i], tenant, "approve", fmt.Sprintf("seed-%s-%d", tenant, i),
			fmt.Sprintf("ticket.approval:%s-%d", tenant, i/perTarget), int64(i + 1),
			base.Add(time.Duration(i) * time.Millisecond)}
	}
	if _, err := conn.CopyFrom(context.Background(), pgx.Identifier{"outbox"},
		[]string{"id", "tenant_id", "kind", "dedupe_key", "target_key", "target_seq", "available_at"},
		pgx.CopyFromRows(rows)); err != nil {
		t.Fatal(err)
	}
	return ids
}

func claimedIDs(cs []Claimed) map[uuid.UUID]Claimed {
	m := make(map[uuid.UUID]Claimed, len(cs))
	for _, c := range cs {
		m[c.ID] = c
	}
	return m
}

// AC 1: two concurrent claimers never claim the same row and never claim a
// successor before its head.
func TestConcurrentClaimers(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	const tenants, rows, perTarget = 4, 60, 5
	total := 0
	for range tenants {
		total += len(seed(t, e.super, uuid.New(), rows, perTarget))
	}

	var (
		mu         sync.Mutex
		claimed    = map[uuid.UUID]int{}
		dispatched = map[string]int64{} // target_key -> highest dispatched target_seq
		failures   []string
	)
	claimer := func(r probe, done <-chan struct{}) error {
		for {
			select {
			case <-done:
				return nil
			default:
			}
			cs, err := r.Claim(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			for _, c := range cs {
				claimed[c.ID]++
				if claimed[c.ID] > 1 {
					failures = append(failures, fmt.Sprintf("row %s claimed twice", c.ID))
				}
				// Every lower target_seq on the target was dispatched before
				// this claim: seed numbers a target's rows consecutively.
				if head := dispatched[c.TargetKey]; head != c.TargetSeq-1 && c.TargetSeq%perTarget != 1 {
					failures = append(failures, fmt.Sprintf("%s seq %d claimed after seq %d", c.TargetKey, c.TargetSeq, head))
				}
			}
			mu.Unlock()
			// Hold the leases a moment, so a successor claimed too early lands
			// while its head is still undispatched.
			time.Sleep(time.Millisecond)
			for _, c := range cs {
				// Recorded before the commit, so a successor claimed after the
				// commit always sees it.
				mu.Lock()
				dispatched[c.TargetKey] = c.TargetSeq
				mu.Unlock()
				if err := r.MarkDispatched(ctx, c); err != nil {
					return err
				}
			}
		}
	}

	done := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		r := newProbe(t, e.relay, Options{})
		go func() { errs <- claimer(r, done) }()
	}
	deadline := time.Now().Add(60 * time.Second)
	for count(t, e.super, "SELECT count(*) FROM outbox WHERE dispatched_at IS NULL") > 0 {
		if time.Now().After(deadline) {
			t.Fatal("rows still pending after 60 s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	close(done)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range failures {
		t.Error(f)
	}
	if len(claimed) != total {
		t.Errorf("%d rows claimed, want %d", len(claimed), total)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE attempts <> 1"); n != 0 {
		t.Errorf("%d rows claimed more than once", n)
	}
}

// A lease hides a claimed row and its successors from the next poll, and a
// claimer whose lease was taken over changes nothing.
func TestLease(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	ids := seed(t, e.super, tenant, 3, 3) // one target: head, two successors
	r := newProbe(t, e.relay, Options{})

	first, err := r.Claim(ctx)
	if err != nil || len(first) != 1 || first[0].ID != ids[0] || first[0].Attempts != 1 {
		t.Fatalf("first claim %+v, %v; want the head at attempt 1", first, err)
	}
	if again, err := r.Claim(ctx); err != nil || len(again) != 0 {
		t.Fatalf("second claim %+v, %v; want nothing while the head is leased", again, err)
	}

	// The lease ends; another claim takes the head over.
	mustExec(t, e.super, "UPDATE outbox SET available_at = now() - interval '1 second' WHERE id = $1", ids[0])
	second, err := r.Claim(ctx)
	if err != nil || len(second) != 1 || second[0].ID != ids[0] || second[0].Attempts != 2 {
		t.Fatalf("takeover %+v, %v; want the head at attempt 2", second, err)
	}
	if err := r.MarkDispatched(ctx, first[0]); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("stale MarkDispatched: %v, want ErrLeaseLost", err)
	}
	if _, err := r.Backoff(ctx, first[0], "stale"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("stale Backoff: %v, want ErrLeaseLost", err)
	}
	if err := r.MarkDead(ctx, first[0], "stale"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("stale MarkDead: %v, want ErrLeaseLost", err)
	}
	if err := r.MarkDispatched(ctx, second[0]); err != nil {
		t.Fatal(err)
	}
	if err := r.MarkDispatched(ctx, second[0]); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("MarkDispatched twice: %v, want ErrLeaseLost", err)
	}
	next, err := r.Claim(ctx)
	if err != nil || len(next) != 1 || next[0].ID != ids[1] {
		t.Fatalf("after dispatch %+v, %v; want the first successor", next, err)
	}
}

// AC 2: tenant A with many pending rows does not prevent tenant B's row from
// being claimed in the same poll. A's scan stops at the cap, and that is
// logged with A's tenant_id and observed with no tenant label.
func TestFairAcrossTenants(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	a, b := uuid.New(), uuid.New()
	seed(t, e.super, a, 500, 1) // 500 independent targets: every row is a head
	bIDs := seed(t, e.super, b, 1, 1)
	r := newProbe(t, e.relay, Options{})

	cs, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	perTenant := map[uuid.UUID]int{}
	for _, c := range cs {
		perTenant[c.TenantID]++
	}
	if perTenant[a] != PerTenant || perTenant[b] != 1 || len(cs) != PerTenant+1 {
		t.Fatalf("claimed %v, want %d of A and B's one row", perTenant, PerTenant)
	}
	if _, ok := claimedIDs(cs)[bIDs[0]]; !ok {
		t.Fatal("B's row was not claimed")
	}

	h := r.histogram(t)
	if h.Count != 2 || val(h.Max) != ScanCap || val(h.Min) != 1 {
		t.Fatalf("histogram count %d min %v max %v; want one observation per tenant, 1 and %d",
			h.Count, val(h.Min), val(h.Max), ScanCap)
	}
	// TICK-94 item 7: a capped poll is entirely in le="200".
	if le := cumulativeAt(h, ScanCap); le != h.Count {
		t.Errorf(`le="200" is %d, _count is %d`, le, h.Count)
	}
	logs := r.logs.String()
	if !strings.Contains(logs, `"tenant_id":"`+a.String()+`"`) || strings.Contains(logs, b.String()) {
		t.Errorf("cap log names the wrong tenants: %s", logs)
	}
}

// AC 3: a head backing off pushes its successors, so they are not scanned
// while it waits; outbox_relay_rows_scanned stays at or under 200 per tenant.
func TestBackoffPushesSuccessors(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	ids := seed(t, e.super, tenant, 300, 300) // one target: a head and 299 successors
	r := newProbe(t, e.relay, Options{BackoffBase: time.Hour, BackoffMax: time.Hour})

	cs, err := r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[0] {
		t.Fatalf("claim %+v, %v; want only the head", cs, err)
	}
	dead, err := r.Backoff(ctx, cs[0], "temporal unavailable")
	if err != nil || dead {
		t.Fatalf("backoff = %v, %v; want alive, nil", dead, err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox o, outbox h
		WHERE h.id = $1 AND o.tenant_id = h.tenant_id AND o.target_key = h.target_key
		  AND o.id <> h.id AND o.available_at < h.available_at`, ids[0]); n != 0 {
		t.Errorf("%d successors available before their head", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox
		WHERE id = $1 AND available_at > now() + interval '59 minutes' AND attempts = 1
		  AND last_error = 'temporal unavailable' AND dead_at IS NULL`, ids[0]); n != 1 {
		t.Error("head was not backed off by its delay with its error")
	}

	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 0 {
		t.Fatalf("claim while the head backs off: %+v, %v; want nothing", cs, err)
	}
	h := r.histogram(t)
	if h.Count != 2 || val(h.Max) > ScanCap {
		t.Fatalf("histogram count %d max %v; want 2 observations, none over %d", h.Count, val(h.Max), ScanCap)
	}
	if val(h.Min) != 0 {
		t.Errorf("second poll scanned %v rows, want 0: every successor was pushed", val(h.Min))
	}
	if v := cumulativeAt(h, ScanCap); v != h.Count {
		t.Errorf(`le="200" is %d, _count is %d`, v, h.Count)
	}

	// The wait ends: the head is claimed again, alone, at attempt 2.
	mustExec(t, e.super, `UPDATE outbox SET available_at = now() - interval '1 second' WHERE tenant_id = $1`, tenant)
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[0] || cs[0].Attempts != 2 {
		t.Fatalf("claim after the wait %+v, %v; want the head at attempt 2", cs, err)
	}
}

// AC 4: a row 24 h past the start of its redelivery is marked dead at its
// next failure, and its successor becomes the head.
func TestMaxRedeliveryAgeMarksDead(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	ids := seed(t, e.super, tenant, 3, 3)
	r := newProbe(t, e.relay, Options{})

	// Under the age it backs off.
	mustExec(t, e.super, "UPDATE outbox SET created_at = now() - interval '23 hours' WHERE id = $1", ids[0])
	cs, err := r.Claim(ctx)
	if err != nil || len(cs) != 1 {
		t.Fatalf("claim %+v, %v", cs, err)
	}
	if dead, err := r.Backoff(ctx, cs[0], "unavailable"); err != nil || dead {
		t.Fatalf("23 h: dead %v, %v; want a backoff", dead, err)
	}

	// Past it while claimed, the next failure marks it dead and pushes
	// nothing. (Past it before the claim, Claim kills it: see
	// TestExpiredLeaseTakeoverMarksDead.)
	mustExec(t, e.super, "UPDATE outbox SET available_at = now() - interval '1 second' WHERE tenant_id = $1", tenant)
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[0] {
		t.Fatalf("claim %+v, %v; want the head", cs, err)
	}
	mustExec(t, e.super, "UPDATE outbox SET created_at = now() - interval '24 hours 1 minute' WHERE id = $1", ids[0])
	dead, err := r.Backoff(ctx, cs[0], "still unavailable")
	if err != nil || !dead {
		t.Fatalf("24 h: dead %v, %v; want dead", dead, err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND last_error = 'still unavailable'`, ids[0]); n != 1 {
		t.Error("head is not dead with its last error")
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id <> $1 AND available_at > now()`, ids[0]); n != 0 {
		t.Errorf("%d successors pushed by a dead head", n)
	}
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[1] {
		t.Fatalf("claim after death %+v, %v; want the first successor", cs, err)
	}

	// MarkDead is terminal at once, whatever the age.
	if err := r.MarkDead(ctx, cs[0], "signal target not found"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND last_error = 'signal target not found'`, ids[1]); n != 1 {
		t.Error("MarkDead did not mark the row dead")
	}
	if cs, err := r.Claim(ctx); err != nil || len(cs) != 1 || cs[0].ID != ids[2] {
		t.Fatalf("claim after MarkDead %+v, %v; want the last row", cs, err)
	}
}

// Rows written by outbox.Enqueue come back whole, and IsStart tells the start
// from the signal by the TL contract (target_key == kind:id).
func TestClaimedFromProducer(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant, record := uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, "INSERT INTO records (tenant_id, id) VALUES ($1, $2)", tenant, record)
	origin := uuid.New()
	var startID, signalID uuid.UUID
	err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		var err error
		startID, err = outbox.Enqueue(ctx, tx, outbox.Entry{Kind: "ticket.approval", KindVersion: 2, DedupeKey: "s",
			RecordID: record, Target: outbox.TargetStart{Kind: "ticket.approval"},
			Causation: outbox.Causation{OriginID: origin, Depth: 1},
			Payload:   []byte(`{"ticket_id":"` + record.String() + `"}`)})
		if err != nil {
			return err
		}
		signalID, err = outbox.Enqueue(ctx, tx, outbox.Entry{Kind: "approve", KindVersion: 1, DedupeKey: "g",
			RecordID: record, Target: outbox.TargetSignal{Kind: "ticket.approval", SignalName: "approve"}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	r := newProbe(t, e.relay, Options{})

	cs, err := r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != startID {
		t.Fatalf("claim %+v, %v; want only the start (the signal queues behind it)", cs, err)
	}
	s := cs[0]
	if !s.IsStart() || s.TenantID != tenant || s.Kind != "ticket.approval" || s.KindVersion != 2 ||
		s.TargetSeq != 1 || s.CausationID == nil || *s.CausationID != origin || s.CausationDepth != 1 ||
		!bytes.Contains(s.Payload, []byte(record.String())) || s.TraceContext != nil ||
		!s.LeaseUntil.After(time.Now()) {
		t.Fatalf("start row %+v", s)
	}
	if err := r.MarkDispatched(ctx, s); err != nil {
		t.Fatal(err)
	}
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != signalID {
		t.Fatalf("claim %+v, %v; want the signal", cs, err)
	}
	if cs[0].IsStart() || cs[0].TargetKey != "ticket.approval:"+startID.String() || cs[0].TargetSeq != 2 {
		t.Fatalf("signal row %+v", cs[0])
	}
}

// The relay works within nava_relay's grants: it never reads last_error or
// dedupe_key, and it cannot touch a row that is no longer pending.
func TestRelayGrants(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	if _, err := e.relay.Exec(ctx, "SELECT last_error FROM outbox"); err == nil {
		t.Error("nava_relay read last_error")
	}
	if _, err := e.relay.Exec(ctx, "SELECT dedupe_key FROM outbox"); err == nil {
		t.Error("nava_relay read dedupe_key")
	}
	tenant := uuid.New()
	ids := seed(t, e.super, tenant, 1, 1)

	// Writes outside its grant fail at PostgreSQL, whatever the relay's code
	// does (Security review, PR #87). The catalog test also pins the grants.
	for _, sql := range []string{
		"UPDATE outbox SET tenant_id = gen_random_uuid() WHERE id = $1",
		"UPDATE outbox SET payload = '{}' WHERE id = $1",
		"UPDATE outbox SET target_key = 'x', target_seq = 0 WHERE id = $1",
		"DELETE FROM outbox WHERE id = $1",
		"INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq) SELECT $1, tenant_id, 'k', 'd', 't', 1 FROM outbox LIMIT 1",
		"SELECT count(*) FROM records WHERE id = $1",
		"SELECT count(*) FROM workflow_ref WHERE outbox_id = $1",
	} {
		if _, err := e.relay.Exec(ctx, sql, ids[0]); err == nil {
			t.Errorf("nava_relay ran %q", sql)
		}
	}

	// A cause PostgreSQL could not store as text is cleaned, so Backoff
	// still records the failure.
	cs, err := newProbe(t, e.relay, Options{}).Claim(ctx)
	if err != nil || len(cs) != 1 {
		t.Fatalf("claim %+v, %v", cs, err)
	}
	if _, err := newProbe(t, e.relay, Options{}).Backoff(ctx, cs[0], "bad \xff byte and \x00 nul"); err != nil {
		t.Fatalf("Backoff with an unstorable cause: %v", err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND last_error = $2",
		ids[0], "bad � byte and � nul"); n != 1 {
		t.Error("last_error is not the cleaned cause")
	}
	mustExec(t, e.super, "UPDATE outbox SET dispatched_at = now(), attempts = 1 WHERE id = $1", ids[0])
	r := newProbe(t, e.relay, Options{})
	if err := r.MarkDead(ctx, Claimed{ID: ids[0], TenantID: tenant, Attempts: 1}, "late"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("MarkDead on a dispatched row: %v, want ErrLeaseLost", err)
	}
}

// val is an extremum's value; a histogram with no observations has none.
func val(e metricdata.Extrema[int64]) int64 {
	v, _ := e.Value()
	return v
}

// TL must-change (TICK-74): a crash between dispatch and MarkDispatched never
// reaches Backoff. When Claim takes such a row over after its lease and it is
// past MaxRedeliveryAge, it is marked dead, not claimed; a row never
// attempted is claimed whatever its age.
func TestExpiredLeaseTakeoverMarksDead(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	crashed, fresh := uuid.New(), uuid.New()
	ids := seed(t, e.super, crashed, 2, 2) // head and successor on one target
	unattempted := seed(t, e.super, fresh, 1, 1)
	// The head was claimed and dispatched, then the relay crashed; its lease
	// has ended. Both tenants' rows are past the age.
	mustExec(t, e.super, `UPDATE outbox SET attempts = 1, available_at = now() - interval '1 second'
		WHERE id = $1`, ids[0])
	mustExec(t, e.super, `UPDATE outbox SET created_at = now() - interval '24 hours 1 minute'
		WHERE id IN ($1, $2)`, ids[0], unattempted[0])
	r := newProbe(t, e.relay, Options{})

	cs, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].ID != unattempted[0] || cs[0].Attempts != 1 {
		t.Fatalf("claim %+v; want only the never-attempted row, at attempt 1", cs)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND attempts = 1 AND available_at < now() AND last_error = $2`, ids[0], TakeoverExpiredError); n != 1 {
		t.Error("the taken-over head is not dead with its last_error, attempts and available_at")
	}
	logs := r.logs.String()
	if !strings.Contains(logs, "max redelivery age") || !strings.Contains(logs, `"tenant_id":"`+crashed.String()+`"`) ||
		!strings.Contains(logs, ids[0].String()) {
		t.Errorf("no dead-letter log line for the head: %s", logs)
	}

	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[1] {
		t.Fatalf("claim after the death %+v, %v; want the successor", cs, err)
	}
}

// TICK-248: heads whose lease ended unsettled sort behind their successors.
// When those fill the tenant's window, Claim still takes the heads of the
// window's targets, oldest target first, and kills one past MaxRedeliveryAge
// as it would inside the window. Targets with no row in the window wait.
func TestClaimTakesStrandedHeads(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	const depth = ScanCap/2 + 1 // a head and ScanCap/2 successors
	ids := seed(t, e.super, tenant, 4*depth, depth)
	heads := []uuid.UUID{ids[0], ids[depth], ids[2*depth], ids[3*depth]}
	mustExec(t, e.super, `UPDATE outbox SET attempts = 1, available_at = now() - interval '1 second'
		WHERE id = ANY($1)`, heads)
	mustExec(t, e.super, `UPDATE outbox SET created_at = now() - interval '24 hours 1 minute'
		WHERE id = $1`, heads[0])
	r := newProbe(t, e.relay, Options{})

	// The window is the successors of targets 0 and 1. Target 0's head is
	// killed, so target 1's is the only one claimed.
	cs, err := r.Claim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].ID != heads[1] || cs[0].Attempts != 2 {
		t.Fatalf("claim %+v; want target 1's head at attempt 2", cs)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND last_error = $2`, heads[0], TakeoverExpiredError); n != 1 {
		t.Error("target 0's head, past the age, is not dead")
	}

	// Target 0's successor is now its head, inside the window, and claimed.
	// Target 1's head is leased again and targets 2 and 3 have no row in the
	// window, so nothing else is.
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != ids[1] {
		t.Fatalf("claim %+v, %v; want only target 0's successor", cs, err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = ANY($1) AND attempts = 1", heads[2:]); n != 2 {
		t.Errorf("%d of targets 2 and 3's heads untouched, want 2", n)
	}
}

// TICK-248 isolation guard: stranded probes a target's head by target_key,
// and target keys are not unique across tenants. Tenant B holds the lowest
// pending target_seq on the key that fills A's window, available, and B's
// own window and stranded step cannot reach it (B's window is full of
// another target). A probe that lost its tenant predicate would claim B's
// row for A. It must stay untouched while A's own head is claimed once its
// lease ends.
func TestClaimStrandedStaysInTenant(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	a, b := uuid.New(), uuid.New()
	shared := "ticket.approval:shared-" + uuid.NewString()
	deep := func(tenant uuid.UUID, key string) uuid.UUID {
		mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
		head := uuid.Must(uuid.NewV7())
		// The head is leased (another relay holds it); ScanCap successors fill the window.
		mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq,
			attempts, available_at) VALUES ($1, $2, 'approve', $3, $4, 1, 1, now() + interval '1 hour')`,
			head, tenant, "head-"+head.String(), key)
		mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, available_at)
			SELECT gen_random_uuid(), $1, 'approve', $2 || i, $3, i + 1,
			       now() - interval '1 hour' + i * interval '1 millisecond'
			FROM generate_series(1, $4::int) i`, tenant, "succ-"+tenant.String()+"-", key, ScanCap)
		return head
	}
	aHead := deep(a, shared)
	deep(b, "ticket.approval:b-own-"+uuid.NewString())
	victim := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, available_at)
		VALUES ($1, $2, 'approve', 'victim', $3, 0, now() - interval '1 minute')`, victim, b, shared)
	r := newProbe(t, e.relay, Options{})

	cs, err := r.Claim(ctx)
	if err != nil || len(cs) != 0 {
		t.Fatalf("claim %+v, %v; want nothing while both heads are leased", cs, err)
	}

	// A's lease ends: its head is stranded and claimed for A, and B's row on
	// the same key is still not.
	mustExec(t, e.super, "UPDATE outbox SET available_at = now() - interval '1 second' WHERE id = $1", aHead)
	cs, err = r.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != aHead || cs[0].TenantID != a {
		t.Fatalf("claim %+v, %v; want only A's stranded head", cs, err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND attempts = 0", victim); n != 1 {
		t.Error("tenant A's claim touched tenant B's row on the shared target key")
	}
}
