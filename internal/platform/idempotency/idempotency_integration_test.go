//go:build integration

// The Idempotency-Key middleware against a real PostgreSQL, migrated with the
// real migrations and reached as the real nava_app role, under RLS.
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package idempotency_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const thingsPath = "/api/v1/things"

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
	platform *pgx.ConnConfig
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens a nava_app pool on the platform database.
func migrated(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
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
	owner := superConfig(t).Copy()
	owner.User = m.Owner()
	owner.Password, _ = testPassword(m.Owner())
	owner.Database = m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
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
	cfg, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.User = "nava_app"
	cfg.ConnConfig.Password, _ = testPassword("nava_app")
	cfg.ConnConfig.Database = m.PlatformDatabase
	app, err := platformdb.NewPool(ctx, cfg, tel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	return env{super: conn, app: app, platform: platformSuper}
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func countRows(t *testing.T, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// lockedBuffer is a log sink safe for the handler's and the test's goroutines.
type lockedBuffer struct {
	ch chan struct{}
	b  bytes.Buffer
}

func newLockedBuffer() *lockedBuffer {
	l := &lockedBuffer{ch: make(chan struct{}, 1)}
	l.ch <- struct{}{}
	return l
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	<-l.ch
	defer func() { l.ch <- struct{}{} }()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	<-l.ch
	defer func() { l.ch <- struct{}{} }()
	return l.b.String()
}

// handlerRun is what the wrapped handler did on one run.
type handlerRun struct{ runs chan struct{} }

func (h *handlerRun) count() int { return len(h.runs) }

// newMiddleware wraps a handler that answers 201 with an ETag and a JSON body
// naming its run, after calling hook (if any).
func newMiddleware(t *testing.T, e env, logs *lockedBuffer, hook func(*http.Request)) (http.Handler, *handlerRun) {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	if logs == nil {
		logs = newLockedBuffer()
	}
	mw, err := idempotency.New(e.app, reg, slog.New(slog.NewJSONHandler(logs, nil)), noop.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	hr := &handlerRun{runs: make(chan struct{}, 100)}
	return mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hr.runs <- struct{}{}
		if hook != nil {
			hook(r)
		}
		w.Header().Set("ETag", `W/"1"`)
		w.Header().Set("Location", thingsPath+"/1")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"run":` + string(rune('0'+len(hr.runs))) + `}`))
	})), hr
}

func post(h http.Handler, p identity.Principal, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(identity.NewContext(context.Background(), p), http.MethodPost, thingsPath, strings.NewReader(body))
	req.Header.Set(idempotency.Header, key)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func problemCode(t *testing.T, rec *httptest.ResponseRecorder) problem.Code {
	t.Helper()
	var b problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("%d %s: %v", rec.Code, rec.Body, err)
	}
	return b.Code
}

// seedKey inserts a key row as the superuser: in_flight with a lease that
// ends in leaseSeconds (negative: already passed), for the hash of a POST of
// body to thingsPath.
func seedKey(t *testing.T, e env, p identity.Principal, key, body string, leaseSeconds int) {
	t.Helper()
	mustExec(t, e.super, `INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::int * interval '1 second', now() + interval '1 day')`,
		p.TenantID, p.PrincipalID, key, idempotency.RequestHash(http.MethodPost, thingsPath, []byte(body)), leaseSeconds)
}

func principal() identity.Principal {
	return identity.Principal{TenantID: uuid.New(), PrincipalID: uuid.New()}
}

// Replay: the same key and request returns the stored response, byte for
// byte, without running the handler; a different request under the key is
// 409 idempotency_key_reuse. The key is scoped to (tenant, principal): tenant
// B, or another principal of tenant A, using the same string never sees A's
// stored response and runs the handler for itself.
func TestReplayReuseAndScope(t *testing.T) {
	e := migrated(t)
	h, hr := newMiddleware(t, e, nil, nil)
	a := principal()

	first := post(h, a, "key-1", `{"a":1,"b":2}`)
	if first.Code != http.StatusCreated || first.Header().Get(idempotency.ReplayedHeader) != "" || hr.count() != 1 {
		t.Fatalf("first = %d %s replayed=%q, runs %d; want 201 from one run", first.Code, first.Body,
			first.Header().Get(idempotency.ReplayedHeader), hr.count())
	}

	// The same JSON, reformatted and reordered, is the same request.
	replay := post(h, a, "key-1", ` { "b" : 2, "a" : 1 } `)
	if replay.Code != http.StatusCreated || !bytes.Equal(replay.Body.Bytes(), first.Body.Bytes()) ||
		replay.Header().Get(idempotency.ReplayedHeader) != "true" {
		t.Errorf("replay = %d %s replayed=%q; want 201 %s replayed", replay.Code, replay.Body,
			replay.Header().Get(idempotency.ReplayedHeader), first.Body)
	}
	for k, want := range map[string]string{"ETag": `W/"1"`, "Location": thingsPath + "/1", "Content-Type": "application/json"} {
		if got := replay.Header().Get(k); got != want {
			t.Errorf("replay %s = %q, want %q", k, got, want)
		}
	}
	if hr.count() != 1 {
		t.Errorf("the replay ran the handler: %d runs, want 1", hr.count())
	}

	reuse := post(h, a, "key-1", `{"a":2,"b":2}`)
	if reuse.Code != http.StatusConflict || problemCode(t, reuse) != problem.IdempotencyKeyReuse ||
		reuse.Header().Get("Content-Type") != problem.ContentType {
		t.Errorf("reuse = %d %s; want 409 idempotency_key_reuse", reuse.Code, reuse.Body)
	}
	if hr.count() != 1 {
		t.Errorf("the reuse ran the handler: %d runs, want 1", hr.count())
	}

	b := principal()
	other := post(h, b, "key-1", `{"a":1,"b":2}`)
	if other.Code != http.StatusCreated || other.Header().Get(idempotency.ReplayedHeader) != "" ||
		bytes.Equal(other.Body.Bytes(), first.Body.Bytes()) || hr.count() != 2 {
		t.Errorf("tenant B, same key = %d %s replayed=%q, runs %d; want its own run, not A's response",
			other.Code, other.Body, other.Header().Get(idempotency.ReplayedHeader), hr.count())
	}
	// A different body under tenant B is not a reuse of A's key either.
	if rec := post(h, identity.Principal{TenantID: b.TenantID, PrincipalID: uuid.New()}, "key-1", `{"z":1}`); rec.Code != http.StatusCreated || hr.count() != 3 {
		t.Errorf("tenant B, other principal, other body = %d %s, runs %d; want 201 from a run", rec.Code, rec.Body, hr.count())
	}
	samePeer := identity.Principal{TenantID: a.TenantID, PrincipalID: uuid.New()}
	if rec := post(h, samePeer, "key-1", `{"a":1,"b":2}`); rec.Code != http.StatusCreated ||
		rec.Header().Get(idempotency.ReplayedHeader) != "" || hr.count() != 4 {
		t.Errorf("tenant A, other principal = %d replayed=%q, runs %d; want its own run", rec.Code,
			rec.Header().Get(idempotency.ReplayedHeader), hr.count())
	}

	var state string
	var status int16
	if err := e.super.QueryRow(context.Background(),
		"SELECT state, response_status FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = 'key-1'",
		a.TenantID, a.PrincipalID).Scan(&state, &status); err != nil || state != "completed" || status != http.StatusCreated {
		t.Errorf("tenant A's row = %s %d, %v; want completed 201", state, status, err)
	}
	if n := countRows(t, e.super, "SELECT count(*) FROM idempotency_keys WHERE key = 'key-1'"); n != 4 {
		t.Errorf("%d rows for key-1, want 4 (one per tenant and principal)", n)
	}
}

// A duplicate of a request whose lease is still running is 409
// idempotency_in_flight with Retry-After, and never runs the handler.
func TestInFlightDuplicate(t *testing.T) {
	e := migrated(t)
	h, hr := newMiddleware(t, e, nil, nil)
	p := principal()
	seedKey(t, e, p, "busy", `{"a":1}`, 60)
	rec := post(h, p, "busy", `{"a":1}`)
	if rec.Code != http.StatusConflict || problemCode(t, rec) != problem.IdempotencyInFlight ||
		rec.Header().Get("Retry-After") != "60" || hr.count() != 0 {
		t.Errorf("= %d %s Retry-After=%q, runs %d; want 409 idempotency_in_flight, 60, no run",
			rec.Code, rec.Body, rec.Header().Get("Retry-After"), hr.count())
	}
}

// An in_flight row whose lease has passed was abandoned: the next request
// with the same key and request takes it over, runs the handler once and
// stores the response, which a later duplicate replays.
func TestAbandonedKeyIsTakenOver(t *testing.T) {
	e := migrated(t)
	h, hr := newMiddleware(t, e, nil, nil)
	p := principal()
	seedKey(t, e, p, "abandoned", `{"a":1}`, -1)
	rec := post(h, p, "abandoned", `{"a":1}`)
	if rec.Code != http.StatusCreated || rec.Header().Get(idempotency.ReplayedHeader) != "" || hr.count() != 1 {
		t.Fatalf("takeover = %d %s, runs %d; want 201 from one run", rec.Code, rec.Body, hr.count())
	}
	if n := countRows(t, e.super, `SELECT count(*) FROM idempotency_keys
		WHERE tenant_id = $1 AND key = 'abandoned' AND state = 'completed' AND response_status = 201`, p.TenantID); n != 1 {
		t.Errorf("the takeover stored %d completed rows, want 1", n)
	}
	again := post(h, p, "abandoned", `{"a":1}`)
	if again.Header().Get(idempotency.ReplayedHeader) != "true" || !bytes.Equal(again.Body.Bytes(), rec.Body.Bytes()) || hr.count() != 1 {
		t.Errorf("duplicate after takeover = %d %s, runs %d; want the stored response replayed", again.Code, again.Body, hr.count())
	}
	// A different request cannot take an abandoned key over: it is reuse.
	seedKey(t, e, p, "abandoned-2", `{"a":1}`, -1)
	if rec := post(h, p, "abandoned-2", `{"a":2}`); rec.Code != http.StatusConflict || problemCode(t, rec) != problem.IdempotencyKeyReuse || hr.count() != 1 {
		t.Errorf("other request on an abandoned key = %d %s, runs %d; want 409 reuse, no run", rec.Code, rec.Body, hr.count())
	}
}

// Two concurrent takeovers of one abandoned key: the second waits on the
// first's row lock, re-checks the lease after the first commits, finds it
// fresh and gets 409 idempotency_in_flight without running the handler.
func TestConcurrentTakeoverLoses(t *testing.T) {
	e := migrated(t)
	h, hr := newMiddleware(t, e, nil, nil)
	p := principal()
	seedKey(t, e, p, "race", `{"a":1}`, -1)

	// The first retaker, stood in for by a superuser transaction that has
	// moved the lease forward and not yet committed.
	ctx := context.Background()
	first, err := pgx.ConnectConfig(ctx, e.platform)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close(context.Background()) })
	tx, err := first.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE idempotency_keys SET lease_until = now() + interval '60 seconds'
		WHERE tenant_id = $1 AND principal_id = $2 AND key = 'race'`, p.TenantID, p.PrincipalID); err != nil {
		t.Fatal(err)
	}

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- post(h, p, "race", `{"a":1}`) }()
	deadline := time.Now().Add(10 * time.Second)
	for countRows(t, e.super, `SELECT count(*) FROM pg_stat_activity
		WHERE usename = 'nava_app' AND wait_event_type = 'Lock'`) != 1 {
		if time.Now().After(deadline) {
			t.Fatal("the second takeover never waited on the row lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	rec := <-done
	if rec.Code != http.StatusConflict || problemCode(t, rec) != problem.IdempotencyInFlight || hr.count() != 0 {
		t.Errorf("second takeover = %d %s, runs %d; want 409 idempotency_in_flight, no run", rec.Code, rec.Body, hr.count())
	}
}

// A completed row whose stored headers cannot be read back is a 500, not a
// replay with headers missing, and never runs the handler again.
func TestUnreadableStoredHeadersIs500(t *testing.T) {
	e := migrated(t)
	h, hr := newMiddleware(t, e, nil, nil)
	p := principal()
	seedKey(t, e, p, "corrupt", `{"a":1}`, 60)
	mustExec(t, e.super, `UPDATE idempotency_keys SET state = 'completed', response_status = 201,
		response_body = '\x7b7d', response_headers = '{"ETag": 1}' WHERE tenant_id = $1 AND key = 'corrupt'`, p.TenantID)
	rec := post(h, p, "corrupt", `{"a":1}`)
	if rec.Code != http.StatusInternalServerError || problemCode(t, rec) != problem.InternalError || hr.count() != 0 {
		t.Errorf("= %d %s, runs %d; want 500 internal_error, no run", rec.Code, rec.Body, hr.count())
	}
}

// If the key row is gone when the response is stored, the client still gets
// the handler's response and the failure is logged; nothing is stored.
func TestStoreFailureStillResponds(t *testing.T) {
	e := migrated(t)
	logs := newLockedBuffer()
	p := principal()
	h, hr := newMiddleware(t, e, logs, func(*http.Request) {
		mustExec(t, e.super, "DELETE FROM idempotency_keys WHERE tenant_id = $1 AND key = 'vanish'", p.TenantID)
	})
	rec := post(h, p, "vanish", `{"a":1}`)
	if rec.Code != http.StatusCreated || rec.Body.String() != `{"run":1}` || hr.count() != 1 {
		t.Errorf("= %d %s, runs %d; want the handler's 201", rec.Code, rec.Body, hr.count())
	}
	if !strings.Contains(logs.String(), "storing the response failed; the key stays in_flight") ||
		!strings.Contains(logs.String(), "idempotency: store: the row is gone") {
		t.Errorf("no store failure logged: %s", logs)
	}
	if n := countRows(t, e.super, "SELECT count(*) FROM idempotency_keys WHERE tenant_id = $1", p.TenantID); n != 0 {
		t.Errorf("%d rows, want 0", n)
	}
}
