// Package idempotency is the Idempotency-Key middleware for /api/v1 writes
// (ADR-001 D6 and I7, PLAN-001 T22). A POST, PUT, PATCH or DELETE must carry
// an Idempotency-Key header. The first request with a key runs the handler and
// stores its status and body; a replay with the same key and the same request
// returns the stored response without running the handler again; the same key
// with a different request is 409 idempotency_key_reuse.
//
// A key belongs to one (tenant, principal): another tenant, or another user of
// the same tenant, reusing the string gets its own record. The tenant and the
// principal come only from the resolved principal in the request context, so
// the middleware must sit inside the authentication middleware.
//
// State lives in idempotency_keys, reached as nava_app inside
// tenancy.InTenantTx. A key is claimed in one of two ways.
//
// A route registered with ClaimInTx (record create, TICK-688) claims its key
// inside its own write transaction, as the first write:
//
//  1. The handler queues the claim (QueueHold) in the first batch of its
//     transaction, ahead of its reads, and settles it (ResolveClaim) before it
//     validates, authorises or writes. The claim inserts an in_flight row that
//     only this transaction sees; the stored response is written in the
//     COMMIT's round trip (CompleteInTx), so the claim, the write and the
//     response commit together, in one synchronous commit, or not at all.
//  2. A concurrent duplicate blocks on the uncommitted key. If the first
//     request commits, the duplicate replays its stored response and runs
//     nothing; if it rolls back, the duplicate's claim goes ahead and it runs
//     as the first request. The wait is bounded by ClaimWait (lock_timeout on
//     the claim alone, tenancy.QueueUnderLockTimeout); past it the duplicate
//     gets 409 idempotency_in_flight, with Retry-After ClaimRetryAfter (not
//     Lease: see ClaimRetryAfter), and its transaction rolls back.
//  3. While it waits, a duplicate holds one pool connection, and through
//     PgBouncer one server connection, for up to ClaimWait. That is accepted
//     (TICK-688): concurrent duplicates of one key each hold one connection
//     for at most ClaimWait, and give it back when they replay or get the 409.
//  4. A request that fails (4xx, 5xx, rollback) leaves no row: its claim rolls
//     back with it, so the retry runs at once and there is no lease to wait
//     out. The lease and the takeover of a stale in_flight row (below) apply
//     to such a route only for a row an older binary committed in_flight.
//
// Every other write claims its key in a transaction of its own:
//
//  1. The first request inserts an in_flight row and commits, so a concurrent
//     duplicate sees it. It runs the handler, then marks the row completed.
//  2. A concurrent duplicate of a young in_flight row gets 409
//     idempotency_in_flight and never runs the handler.
//  3. An in_flight row whose lease_until has passed (on the database clock)
//     is abandoned: its owner crashed or was cut off. A retaker takes it over
//     with one conditional UPDATE that moves lease_until forward only if it
//     has still passed, commits, and runs the handler after the commit, as a
//     first request does (T22a, migration 00006). Two concurrent retakers
//     serialise on the row: the second re-checks it after the first commits,
//     finds the lease fresh and gets 409 idempotency_in_flight. No
//     transaction or connection is held while the handler runs, so takeovers
//     cannot use up the pool or PgBouncer's server connections.
//  4. A 5xx response is not stored: the row stays in_flight and the client may
//     retry after the lease.
//
// A replay restores Content-Type from the stored status: application/json
// below 400, application/problem+json from 400 (ContentTypeFor). Handlers of
// /api/v1 write only those two kinds (the error contract). Of the handler's
// other headers only ReplayHeaders (ETag, Location) are stored, in
// response_headers, and restored (TICK-412): a client that retries a create
// after a timeout needs the ETag for its next If-Match. A row stored before
// that column existed replays without them.
//
// A row is purged after expires_at (24 h) by nava-purge; until then the key
// stays bound to its first request.
package idempotency

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	// Header is the request header that carries the key.
	Header = "Idempotency-Key"
	// ReplayedHeader is set to "true" on a response served from storage.
	ReplayedHeader = "Idempotent-Replayed"

	// Lease is how long an in_flight row is presumed alive.
	Lease = 60 * time.Second
	// TTL is how long a key is remembered (expires_at).
	TTL = 24 * time.Hour

	maxKeyLen   = 255
	maxBodySize = 1 << 20
	// completeTimeout bounds storing the response after the handler ran, which
	// must survive the client hanging up.
	completeTimeout = 5 * time.Second
	// ClaimWait bounds how long the claim of a ClaimInTx route waits for a
	// concurrent request with the same key to commit or roll back
	// (TICK-688). A healthy record create takes about 50 ms, so this covers
	// a first request 40 times slower than target; past it the duplicate
	// gets 409 idempotency_in_flight. It is also how long each waiting
	// duplicate holds a connection, so it stays short. Not configuration.
	ClaimWait = 2 * time.Second
	// ClaimRetryAfter is the Retry-After of a ClaimInTx route (record create)
	// on 409 idempotency_in_flight and 503 retry_later (TICK-690). Such a
	// route leaves no key row behind when it fails, so its retry need not
	// wait out a Lease: a duplicate past ClaimWait is waiting on a request
	// that is still running, which a retry after about ClaimWait finds
	// committed (and replays) or rolled back (and runs). Every other route
	// advises Lease, for the in_flight row it leaves.
	ClaimRetryAfter = ClaimWait
)

// Results recorded on nava.idempotency.requests. Never a tenant, principal or
// key: those are unbounded.
const (
	resultExecuted = "executed"
	resultRetaken  = "retaken"
	resultReplayed = "replayed"
	resultReuse    = "reuse"
	resultInFlight = "in_flight"
	resultRejected = "rejected"
	// resultStoredAfterCommit: a 2xx response was stored by the fallback, after
	// the handler's transaction committed, because the handler did not call
	// CompleteInTx. A tenant write route should never produce it.
	resultStoredAfterCommit = "stored_after_commit"
	resultError             = "error"
)

// ReplayHeaders are the response headers a replay restores besides
// Content-Type. Anything else a handler sets (Retry-After, caching, tracing)
// belongs to the first run only. Adding one is a code change: the column is a
// JSON object, not one column per header.
func ReplayHeaders() []string { return []string{"ETag", "Location"} }

var errNoPrincipal = errors.New("idempotency: no resolved principal in the request context")

// Middleware builds the handler wrapper.
type Middleware struct {
	pool     *pgxpool.Pool
	reg      *problem.Registry
	log      *slog.Logger
	requests metric.Int64Counter
	// inTx: the route patterns whose handler claims the key in its own
	// transaction (ClaimInTx). Set before the middleware serves.
	inTx map[string]bool
}

// ClaimInTx makes the routes with these patterns (http.Request.Pattern, as
// "POST /api/v1/records/{entity}") claim their key inside the handler's own
// write transaction instead of in a transaction of their own (TICK-688, see
// the package doc). Such a handler must call QueueHold in the first batch of
// its transaction, ResolveClaim right after that batch, and CompleteInTx
// last. Call it before the middleware serves a request.
func (m *Middleware) ClaimInTx(patterns ...string) {
	if m.inTx == nil {
		m.inTx = map[string]bool{}
	}
	for _, p := range patterns {
		m.inTx[p] = true
	}
}

// New returns a Middleware that stores keys through pool, which must log in
// as nava_app. mp must not be nil.
func New(pool *pgxpool.Pool, reg *problem.Registry, log *slog.Logger, mp metric.MeterProvider) (*Middleware, error) {
	if pool == nil || reg == nil || log == nil || mp == nil {
		return nil, errors.New("idempotency: pool, registry, logger and meter provider are required")
	}
	c, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency").Int64Counter(
		"nava.idempotency.requests",
		metric.WithDescription("API write requests by idempotency result."),
		metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	return &Middleware{pool: pool, reg: reg, log: log, requests: c}, nil
}

// Handler wraps next. Reads (GET, HEAD, OPTIONS, TRACE) pass through; every
// other method needs the header. Operator routes are not wrapped (T21 does its
// own idempotency).
//
// The response of the first run is normalised so a replay is byte-identical:
// a body is sent as application/json below 400 and application/problem+json
// from 400 (ContentTypeFor), and a response without a body has no Content-Type.
//
// The handler never runs inside the middleware's transaction, for a fresh key
// or a retaken one. A write handler that commits its own work completes the
// key in that same transaction with CompleteInTx, so the write and the stored
// response commit or roll back together and no window is left in which the
// work is committed and the key is still in_flight (TICK-482). Only a handler
// that does not call CompleteInTx falls back to storing the response after it
// returns, and a failure there leaves the key in_flight.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		p, ok := identity.FromContext(ctx)
		if !ok {
			m.reg.WriteError(w, r, m.log, problem.InternalError, errNoPrincipal)
			return
		}
		key, ok := validKey(r.Header.Values(Header))
		if !ok {
			m.count(ctx, resultRejected)
			m.reg.Write(w, r, problem.ValidationFailed, problem.FieldError{Pointer: "/header/" + Header, Rule: "required"})
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
		if err != nil {
			m.count(ctx, resultRejected)
			rule := "unreadable"
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				rule = "max_size"
			}
			m.reg.Write(w, r, problem.ValidationFailed, problem.FieldError{Pointer: "/body", Rule: rule})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		req := request{p: p, key: key, hash: RequestHash(r.Method, r.URL.RequestURI(), body)}
		run := func(sc *scope) *captured {
			return capture(next, r.WithContext(context.WithValue(ctx, scopeKey{}, sc)))
		}
		var out *outcome
		if m.inTx[r.Pattern] {
			out = m.runInTx(ctx, req, run)
		} else {
			out, err = m.acquire(ctx, req, run)
		}
		if err != nil {
			m.count(ctx, resultError)
			m.reg.WriteError(w, r, m.log, problem.InternalError, err)
			return
		}
		switch out.kind {
		case outcomeReuse:
			m.count(ctx, resultReuse)
			m.reg.Write(w, r, problem.IdempotencyKeyReuse)
		case outcomeInFlight:
			m.count(ctx, resultInFlight)
			retry := Lease
			if m.inTx[r.Pattern] {
				retry = ClaimRetryAfter
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(retry/time.Second)))
			m.reg.Write(w, r, problem.IdempotencyInFlight)
		case outcomeReplay:
			m.count(ctx, resultReplayed)
			writeStored(w, out.status, out.body, out.headers)
		case outcomeRan:
			m.count(ctx, out.result)
			out.resp.flush(w)
		}
	})
}

func (m *Middleware) count(ctx context.Context, result string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

// validKey returns the one header value if it is 1 to 255 printable ASCII
// characters.
func validKey(vals []string) (string, bool) {
	if len(vals) != 1 || vals[0] == "" || len(vals[0]) > maxKeyLen {
		return "", false
	}
	for _, c := range []byte(vals[0]) {
		if c < 0x21 || c > 0x7e {
			return "", false
		}
	}
	return vals[0], true
}

type request struct {
	p    identity.Principal
	key  string
	hash []byte
}

type outcomeKind int

const (
	outcomeRan outcomeKind = iota + 1
	outcomeReplay
	outcomeReuse
	outcomeInFlight
)

type outcome struct {
	kind outcomeKind
	// outcomeRan: the handler's response and how the key was won.
	resp   *captured
	result string
	// outcomeReplay: the stored response.
	status  int
	body    []byte
	headers map[string]string
}

// selectRow reads the key's row. The lease is judged on the database clock, so
// it does not depend on this host's.
const selectRow = `
SELECT request_hash, state, response_status, response_body, response_headers, lease_until < now()
  FROM idempotency_keys
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3`

// takeOver moves an abandoned row's lease forward. It matches only while the
// lease has passed, so of two concurrent takeovers the second, which waits for
// the first's commit and re-checks the row, matches nothing.
const takeOver = `
UPDATE idempotency_keys
   SET lease_until = now() + $5::double precision * interval '1 second'
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3
   AND state = 'in_flight' AND lease_until < now() AND request_hash = $4`

// acquire decides what the request is and, for a request that must run, runs
// it through run after the transaction that won the key has committed: the
// insert of a fresh key, or the takeover of an abandoned one.
func (m *Middleware) acquire(ctx context.Context, req request, run func(*scope) *captured) (*outcome, error) {
	var out *outcome
	err := tenancy.InTenantTx(ctx, m.pool, req.p.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, insertClaim,
			req.p.TenantID, req.p.PrincipalID, req.key, req.hash, Lease.Seconds(), TTL.Seconds())
		if err != nil {
			return fmt.Errorf("idempotency: insert: %w", err)
		}
		if tag.RowsAffected() == 1 {
			out = &outcome{kind: outcomeRan, result: resultExecuted} // the handler runs after the commit
			return nil
		}
		var (
			hash   []byte
			state  string
			status *int16
			body   []byte
			hdrs   []byte
			stale  bool
		)
		if err := tx.QueryRow(ctx, selectRow, req.p.TenantID, req.p.PrincipalID, req.key).
			Scan(&hash, &state, &status, &body, &hdrs, &stale); err != nil {
			return fmt.Errorf("idempotency: select: %w", err)
		}
		switch {
		case !bytes.Equal(hash, req.hash):
			out = &outcome{kind: outcomeReuse}
		case state == "completed" && status != nil:
			headers, err := decodeHeaders(hdrs)
			if err != nil {
				return err
			}
			out = &outcome{kind: outcomeReplay, status: int(*status), body: body, headers: headers}
		case !stale:
			out = &outcome{kind: outcomeInFlight}
		default:
			tag, err := tx.Exec(ctx, takeOver, req.p.TenantID, req.p.PrincipalID, req.key, req.hash, Lease.Seconds())
			if err != nil {
				return fmt.Errorf("idempotency: take over: %w", err)
			}
			if tag.RowsAffected() == 1 {
				out = &outcome{kind: outcomeRan, result: resultRetaken} // the handler runs after the commit
			} else {
				out = &outcome{kind: outcomeInFlight} // another request took it over, or completed it, first
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out.kind == outcomeRan {
		sc := &scope{req: req}
		out.resp = run(sc)
		if !sc.completed.Load() {
			m.complete(ctx, req, out.resp)
		}
	}
	return out, nil
}

// runInTx runs a ClaimInTx route. The handler claims and settles the key in
// its own transaction (QueueHold, ResolveClaim), so the outcome is either
// the handler's response or, when the key was already taken, the outcome
// ResolveClaim settled, which replaces whatever the handler wrote. Nothing is
// stored here: a response not stored in the handler's transaction rolled
// back with its claim.
func (m *Middleware) runInTx(ctx context.Context, req request, run func(*scope) *captured) *outcome {
	sc := &scope{req: req, inTx: true}
	resp := run(sc)
	if o := sc.settled.Load(); o != nil {
		return o
	}
	if !sc.completed.Load() && resp.status >= 200 && resp.status < 300 {
		// A route registered with ClaimInTx answered 2xx without storing
		// the key in its transaction: it did not claim.
		m.log.ErrorContext(ctx, "idempotency: a ClaimInTx route answered 2xx without completing its key in the write",
			slog.Int("status", resp.status))
	}
	result := resultExecuted
	if sc.retaken {
		result = resultRetaken
	}
	return &outcome{kind: outcomeRan, resp: resp, result: result}
}

// complete is the fallback for handlers that do not call CompleteInTx: it
// stores the response of the request that won the key after the handler's
// transaction has committed. Handlers that call CompleteInTx store it in their
// own transaction and never reach this. The side effects already happened, so a
// failure here is logged and the client still gets its response; the key stays
// in_flight, and a retry after the lease runs the handler again.
func (m *Middleware) complete(ctx context.Context, req request, resp *captured) {
	if !resp.storable() {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), completeTimeout)
	defer cancel()
	err := tenancy.InTenantTx(ctx, m.pool, req.p.TenantID, func(tx pgx.Tx) error {
		return store(ctx, tx, req, resp)
	})
	if err != nil {
		m.log.ErrorContext(ctx, "idempotency: storing the response failed; the key stays in_flight",
			slog.String("error", err.Error()))
		return
	}
	if resp.status >= 200 && resp.status < 300 {
		// A 2xx means the handler most likely committed a write. Storing after
		// the commit reopens the TICK-482 window, so make it visible.
		m.count(ctx, resultStoredAfterCommit)
		m.log.WarnContext(ctx, "idempotency: a successful response was stored after commit; the handler should call idempotency.CompleteInTx",
			slog.Int("status", resp.status))
	}
}

func store(ctx context.Context, tx pgx.Tx, req request, resp *captured) error {
	sql, args, err := storeSQL(req, resp)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("idempotency: store: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("idempotency: store: the row is gone or already completed")
	}
	return nil
}

// storeSQL is the UPDATE that marks the key completed with resp.
func storeSQL(req request, resp *captured) (string, []any, error) {
	hdrs, err := encodeHeaders(resp.header)
	if err != nil {
		return "", nil, err
	}
	return `
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'`,
		[]any{req.p.TenantID, req.p.PrincipalID, req.key, int16(resp.status), resp.body, headersParam(hdrs)}, nil //nolint:gosec // an HTTP status is 100-599
}

// mustMatchOne wraps an UPDATE so that, when it matches no row, it fails in
// SQL (division by zero, 22012) and aborts the transaction: a statement
// queued for the COMMIT's round trip cannot have its row count checked
// before the COMMIT runs (tenancy.ExecAtCommit). The hold makes a miss
// impossible; this keeps a miss from committing the write anyway.
func mustMatchOne(update string) string {
	return "WITH done AS (" + update + " RETURNING 1) SELECT 1 / count(*) FROM done"
}

// encodeHeaders is the response_headers value for h: a JSON object of the
// ReplayHeaders h sets, first value each, or nil (SQL NULL) when it sets none.
func encodeHeaders(h http.Header) ([]byte, error) {
	m := map[string]string{}
	for _, k := range ReplayHeaders() {
		if v := h.Get(k); v != "" {
			m[k] = v
		}
	}
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("idempotency: encode headers: %w", err)
	}
	return b, nil
}

// headersParam is the $6 argument for an encoded response_headers value: the
// JSON as a string, or an untyped nil (SQL NULL). Never the []byte itself:
// under QueryExecModeExec, the mode every binary behind PgBouncer uses, pgx
// sends a []byte as bytea text ("\x7b...") and jsonb rejects it with 22P02,
// so the response is never stored (TICK-456). A string is sent as the JSON
// text in every mode.
func headersParam(b []byte) any {
	if b == nil {
		return nil
	}
	return string(b)
}

// decodeHeaders reads response_headers back. NULL (no headers, or a row
// stored before the column existed) is an empty map.
func decodeHeaders(b []byte) (map[string]string, error) {
	if b == nil {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("idempotency: decode headers: %w", err)
	}
	return m, nil
}

// captured is a handler's response, held back until the key is settled.
type captured struct {
	status int
	header http.Header
	body   []byte
}

// storable: a 5xx is not remembered, so a retry can succeed.
func (c *captured) storable() bool { return c.status < http.StatusInternalServerError }

// flush sends the response as writeStored would replay it: Content-Type from
// the status, none without a body.
func (c *captured) flush(w http.ResponseWriter) {
	h := w.Header()
	for k, v := range c.header {
		h[k] = v
	}
	h.Del("Content-Type")
	if len(c.body) > 0 {
		h.Set("Content-Type", ContentTypeFor(c.status))
		h.Set("Content-Length", strconv.Itoa(len(c.body)))
	}
	w.WriteHeader(c.status)
	_, _ = w.Write(c.body)
}

type recorder struct {
	c     *captured
	wrote bool
}

func (r *recorder) Header() http.Header { return r.c.header }

func (r *recorder) WriteHeader(status int) {
	if !r.wrote {
		r.wrote, r.c.status = true, status
	}
}

func (r *recorder) Write(b []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	r.c.body = append(r.c.body, b...)
	return len(b), nil
}

func capture(next http.Handler, req *http.Request) *captured {
	c := &captured{status: http.StatusOK, header: http.Header{}}
	next.ServeHTTP(&recorder{c: c}, req)
	return c
}

// ContentTypeFor is the Content-Type a stored response is replayed with.
func ContentTypeFor(status int) string {
	if status >= http.StatusBadRequest {
		return problem.ContentType
	}
	return "application/json"
}

func writeStored(w http.ResponseWriter, status int, body []byte, headers map[string]string) {
	h := w.Header()
	for _, k := range ReplayHeaders() {
		if v, ok := headers[k]; ok {
			h.Set(k, v)
		}
	}
	h.Set(ReplayedHeader, "true")
	if len(body) > 0 {
		h.Set("Content-Type", ContentTypeFor(status))
		h.Set("Content-Length", strconv.Itoa(len(body)))
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

type scopeKey struct{}

// scope is what the middleware hands a handler through the request context so
// the handler can complete the key in its own transaction.
type scope struct {
	req       request
	completed atomic.Bool
	// inTx: the route claims the key in the handler's transaction
	// (ClaimInTx). The fields below are for that mode: the handler's
	// goroutine sets them, the middleware reads them after it returned.
	inTx bool
	// claimSent, claimed: the claim's INSERT ran, and inserted the row.
	claimSent, claimed bool
	// retaken: the claim took over a stale in_flight row.
	retaken bool
	// settled: the outcome ResolveClaim decided instead of the handler's
	// write (a replay, a reuse, in flight).
	settled atomic.Pointer[outcome]
	// held is the transaction in which QueueHold locked the key's row, once
	// that batch has run.
	held atomic.Pointer[pgx.Tx]
}

// holdSQL locks the request's own key row while it is still in_flight. A
// takeover (takeOver) or another completion of the row waits for this
// transaction to end, and finds the row completed if it commits.
const holdSQL = `
SELECT 1 FROM idempotency_keys
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'
   FOR UPDATE`

// ErrNotInFlight means the request's key is no longer in_flight when its
// handler's transaction tries to hold it: another request took it over and
// completed it. The handler's transaction must roll back, so the write does
// not happen twice.
var ErrNotInFlight = errors.New("idempotency: the key is no longer in flight")

// QueueHold queues, on b, the lock of the request's key row, for a handler
// that sends b in its own InTenantTx transaction tx before it writes
// (TICK-648). Once the batch has run, CompleteInTx stores the response in the
// round trip that commits tx (tenancy.ExecAtCommit), since the held row
// cannot be completed by anyone else first. When the row is not in_flight,
// the batch fails with ErrNotInFlight. Without a middleware scope in ctx it
// queues nothing.
//
// For a ClaimInTx route it queues the claim of the key instead, as the
// transaction's first write, waiting at most ClaimWait
// (tenancy.QueueUnderLockTimeout). The handler must then pass the batch's
// error to ResolveClaim before it does anything else.
func QueueHold(ctx context.Context, tx pgx.Tx, b *pgx.Batch) error {
	sc, ok := ctx.Value(scopeKey{}).(*scope)
	if !ok {
		return nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return fmt.Errorf("idempotency: transaction tenant: %w", err)
	}
	if tenant != sc.req.p.TenantID {
		return errors.New("idempotency: the transaction is scoped to another tenant than the request's")
	}
	if sc.inTx {
		queueClaim(sc, tx, b)
		return nil
	}
	b.Queue(holdSQL, sc.req.p.TenantID, sc.req.p.PrincipalID, sc.req.key).Query(func(rows pgx.Rows) error {
		held := rows.Next()
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("idempotency: hold: %w", err)
		}
		if !held {
			return ErrNotInFlight
		}
		sc.held.Store(&tx)
		return nil
	})
	return nil
}

// CompleteInTx stores the response a handler is about to return, inside tx,
// the handler's own InTenantTx transaction. Call it last in that transaction,
// after the write succeeded: the stored response then commits with the write
// and rolls back with it. visit is the response's Visit method for the
// operation (for example resp.VisitCreateRecordResponse); it runs against a
// recorder, so the stored bytes are the ones the client gets.
//
// Without a middleware scope in ctx (a test calling the handler directly) it
// does nothing. If the key is no longer in_flight, because another request
// took it over and completed it, it returns an error, so the transaction
// rolls back and the write does not happen twice.
func CompleteInTx(ctx context.Context, tx pgx.Tx, visit func(http.ResponseWriter) error) error {
	sc, ok := ctx.Value(scopeKey{}).(*scope)
	if !ok {
		return nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return fmt.Errorf("idempotency: transaction tenant: %w", err)
	}
	if tenant != sc.req.p.TenantID {
		return errors.New("idempotency: the transaction is scoped to another tenant than the request's")
	}
	c := &captured{status: http.StatusOK, header: http.Header{}}
	if err := visit(&recorder{c: c}); err != nil {
		return fmt.Errorf("idempotency: render response: %w", err)
	}
	// A row this transaction holds (QueueHold) is still in_flight and
	// nobody else can complete it, so its UPDATE matches exactly that row
	// and can ride in the COMMIT's round trip.
	if h := sc.held.Load(); h != nil && *h == tx {
		sql, args, err := storeSQL(sc.req, c)
		if err != nil {
			return err
		}
		if tenancy.ExecAtCommit(tx, mustMatchOne(sql), args...) {
			sc.completed.Store(true)
			return nil
		}
	}
	if err := store(ctx, tx, sc.req, c); err != nil {
		return err
	}
	sc.completed.Store(true)
	return nil
}
