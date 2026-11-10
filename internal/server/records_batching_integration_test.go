//go:build integration

// TICK-648 (PLAN-003 P3-36b): POST and GET /api/v1/records through the real
// handlers on a pool traced by tenancytest.Guard. Every round trip of a
// request must run under the requesting principal's tenant, set by the
// tenancy BEGIN before the trip was sent, and the round trips and statements
// stay within the budget this task set (before: create 29 statements across
// two transactions, GET 13; the counts are logged for the PR).

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy/tenancytest"
)

// The budgets: round trips and statements of one request, idempotency claim
// included. A change that adds a round trip to either path fails here.
const (
	createTripBudget      = 11 // claim: BEGIN, INSERT, COMMIT; write: BEGIN, prefetch, checks, write batch, COMMIT batch; number (TICK-657): BEGIN, upsert, COMMIT
	createStatementBudget = 20 // was 29, then 17 (TICK-648): claim 4 (BEGIN+GUC, INSERT, COMMIT), write 12, number 4 (BEGIN+GUC, upsert, COMMIT)
	getTripBudget         = 3  // BEGIN, one batch, COMMIT
	getStatementBudget    = 7  // was 13: BEGIN+GUC, definition, record, shares, policy, COMMIT
)

// guardedRecEnv is newRecEnv with the handler on a second pool of the same
// database whose tracer is a Guard, with at most maxConns connections.
func guardedRecEnv(t *testing.T, maxConns int32) (*etEnv, *tenancytest.Guard) {
	t.Helper()
	e := newRecEnv(t)
	cfg := e.pool.Config().Copy()
	g := &tenancytest.Guard{}
	cfg.ConnConfig.Tracer = g
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	h, err := server.NewHandler(slog.New(slog.NewTextHandler(testLogWriter{t, e.logs}, &slog.HandlerOptions{Level: slog.LevelWarn})),
		e.tel, e.tokens, stubPaths{},
		server.WithLimiter(e.lim), server.WithPool(pool), server.WithAllocatorPool(etAllocPool(t, pool)), server.WithAuditReader(etAuditReader(t, pool)), server.WithCursors(kr))
	if err != nil {
		t.Fatal(err)
	}
	e.h, e.pool = h, pool
	return e, g
}

// traced serves one request with a Recorder in its context.
func (e *etEnv) traced(t *testing.T, method, path, token, idemKey, body string) (*httptest.ResponseRecorder, *tenancytest.Recorder) {
	t.Helper()
	ctx, rec := tenancytest.Record(t.Context())
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(w, req)
	return w, rec
}

// underOnly fails the test when a trip ran under any tenant but want.
func underOnly(t *testing.T, what string, rec *tenancytest.Recorder, want uuid.UUID) {
	t.Helper()
	for i, tr := range rec.Trips() {
		if tr.Tenant != want {
			t.Errorf("%s: trip %d ran under tenant %s, want %s: %q", what, i, tr.Tenant, want, tr.SQL)
		}
	}
}

// TestRecordCreateAndGetRoundTrips: a create and a GET each set the tenant
// before their first statement and run every trip under it, within the
// budgets. The create takes its number alone in a transaction of its own
// (BEGIN, upsert, COMMIT; D-3-9 as amended 2026-11-10, TICK-657), then sends
// the write batch and the COMMIT batch, which also completes the idempotency
// key.
func TestRecordCreateAndGetRoundTrips(t *testing.T) {
	e, g := guardedRecEnv(t, 4)
	group := e.a.adminGroup.String()
	w, rec := e.traced(t, http.MethodPost, "/api/v1/records/ticket", "user-a", "k-648",
		`{"data": {"short_description": "batched", "group": "`+group+`"}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", w.Code, w.Body)
	}
	underOnly(t, "create", rec, e.a.id)
	trips := rec.Trips()
	t.Logf("create: %d round trips, %d statements (before: 29 statements, two transactions)", len(trips), rec.Statements())
	if len(trips) > createTripBudget || rec.Statements() > createStatementBudget {
		t.Errorf("create: %d trips, %d statements; budget %d, %d: %s", len(trips), rec.Statements(),
			createTripBudget, createStatementBudget, fmt.Sprint(trips))
	}
	// The number's own transaction, then the write's last two trips: the
	// write batch, then the batch that stores the idempotency response and
	// commits.
	if len(trips) < 5 {
		t.Fatalf("create: %d trips: %s", len(trips), fmt.Sprint(trips))
	}
	begin, upsert, numbered := trips[len(trips)-5], trips[len(trips)-4], trips[len(trips)-3]
	if !strings.HasPrefix(strings.ToLower(begin.SQL[0]), "begin") || len(upsert.SQL) != 1 ||
		!strings.Contains(upsert.SQL[0], "record_counters") || len(numbered.SQL) != 1 || numbered.SQL[0] != "commit" {
		t.Errorf("the three trips before the write batch are %q, %q, %q; want the number's BEGIN, upsert alone, COMMIT",
			begin.SQL, upsert.SQL, numbered.SQL)
	}
	counter, commit := trips[len(trips)-2], trips[len(trips)-1]
	if !counter.Batch || !strings.Contains(counter.SQL[0], "INSERT INTO records") ||
		slices.ContainsFunc(counter.SQL, func(s string) bool { return strings.Contains(s, "record_counters") }) {
		t.Errorf("the trip before the commit is %q, want the write batch, without the counter", counter.SQL)
	}
	if !commit.Batch || len(commit.SQL) != 2 || !strings.Contains(commit.SQL[0], "UPDATE idempotency_keys") || commit.SQL[1] != "COMMIT" {
		t.Errorf("the last trip is %q, want the idempotency completion and COMMIT in one batch", commit.SQL)
	}

	var body recBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	gw, grec := e.traced(t, http.MethodGet, "/api/v1/records/ticket/"+body.ID, "user-a", "", "")
	if gw.Code != http.StatusOK {
		t.Fatalf("get = %d: %s", gw.Code, gw.Body)
	}
	underOnly(t, "get", grec, e.a.id)
	t.Logf("GET: %d round trips, %d statements (before: 13 statements)", len(grec.Trips()), grec.Statements())
	if len(grec.Trips()) > getTripBudget || grec.Statements() > getStatementBudget {
		t.Errorf("GET: %d trips, %d statements; budget %d, %d: %s", len(grec.Trips()), grec.Statements(),
			getTripBudget, getStatementBudget, fmt.Sprint(grec.Trips()))
	}
	// The 201 the create answered with is the record a GET reads back: the
	// create built it from the row as stored, without reading it again.
	var got recBody
	if err := json.Unmarshal(gw.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	created, _ := json.Marshal(body)
	read, _ := json.Marshal(got)
	if !bytes.Equal(created, read) {
		t.Errorf("the 201 body differs from the GET:\n%s\n%s", created, read)
	}
	// And the key's stored response is that same 201.
	var stored []byte
	if err := e.super.QueryRow(t.Context(), "SELECT response_body FROM idempotency_keys WHERE key = 'k-648'").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, w.Body.Bytes()) {
		t.Errorf("stored response %s, want the 201 %s", stored, w.Body)
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}

// TestRecordRequestsOfTwoTenantsOnOneConnection: creates and GETs of
// tenants A and B, interleaved on a pool of one connection, each run every
// trip under their own principal's tenant; B's GET of A's record is the
// missing-record 404, and so is A's of B's.
func TestRecordRequestsOfTwoTenantsOnOneConnection(t *testing.T) {
	e, g := guardedRecEnv(t, 1)
	var ids [2]string
	for i, c := range []struct {
		token, entity string
		tenant        uuid.UUID
	}{{"user-a", "ticket", e.a.id}, {"user-b", "incident", e.b.id}} {
		w, rec := e.traced(t, http.MethodPost, "/api/v1/records/"+c.entity, c.token, "k-two-"+c.token,
			`{"data": {"short_description": "mine"}}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("%s create = %d: %s", c.token, w.Code, w.Body)
		}
		underOnly(t, c.token+" create", rec, c.tenant)
		var b recBody
		if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
			t.Fatal(err)
		}
		ids[i] = b.ID
	}
	for _, c := range []struct {
		token, path string
		tenant      uuid.UUID
		want        int
	}{
		{"user-a", "/api/v1/records/ticket/" + ids[0], e.a.id, http.StatusOK},
		{"user-b", "/api/v1/records/ticket/" + ids[0], e.b.id, http.StatusNotFound},
		{"user-b", "/api/v1/records/incident/" + ids[1], e.b.id, http.StatusOK},
		{"user-a", "/api/v1/records/incident/" + ids[1], e.a.id, http.StatusNotFound},
	} {
		w, rec := e.traced(t, http.MethodGet, c.path, c.token, "", "")
		if w.Code != c.want {
			t.Errorf("%s GET %s = %d, want %d: %s", c.token, c.path, w.Code, c.want, w.Body)
		}
		underOnly(t, c.token+" GET "+c.path, rec, c.tenant)
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}

// TestConcurrentRecordCreatesStayInTheirTenant: 8 clients each of tenants A
// and B create at once on a pool smaller than the clients; every trip of
// every request, its number's transaction on the allocator pool included,
// runs under its own tenant, and as no create fails each tenant's numbers
// have no gap.
func TestConcurrentRecordCreatesStayInTheirTenant(t *testing.T) {
	e, g := guardedRecEnv(t, 4)
	const perClient = 5
	var wg sync.WaitGroup
	for c := range 8 {
		for _, who := range []struct {
			token, entity string
			tenant        uuid.UUID
		}{{"user-a", "ticket", e.a.id}, {"user-b", "incident", e.b.id}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range perClient {
					key := fmt.Sprintf("%s-%d-%d", who.token, c, i)
					ctx, rec := tenancytest.Record(context.Background())
					req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/records/"+who.entity,
						strings.NewReader(`{"data": {"short_description": "load"}}`))
					req.Header.Set("Authorization", "Bearer "+who.token)
					req.Header.Set("Idempotency-Key", key)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					e.h.ServeHTTP(w, req)
					if w.Code != http.StatusCreated {
						t.Errorf("%s create = %d: %s", who.token, w.Code, w.Body)
						return
					}
					underOnly(t, who.token+" create", rec, who.tenant)
				}
			}()
		}
	}
	wg.Wait()
	for _, c := range []struct {
		tenant uuid.UUID
		prefix string
	}{{e.a.id, "TKT"}, {e.b.id, "INC"}} {
		rows, err := e.super.Query(t.Context(), "SELECT number FROM records WHERE tenant_id = $1 ORDER BY number", c.tenant)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatal(err)
			}
			got = append(got, n)
		}
		rows.Close()
		want := make([]string, 0, 8*perClient)
		for i := 1; i <= 8*perClient; i++ {
			want = append(want, fmt.Sprintf("%s%07d", c.prefix, i))
		}
		if !slices.Equal(got, want) {
			t.Errorf("tenant %s numbers %v, want %s0000001..%d with no gap", c.tenant, got, c.prefix, 8*perClient)
		}
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}
