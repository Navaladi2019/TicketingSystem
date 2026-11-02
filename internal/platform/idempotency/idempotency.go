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
// tenancy.InTenantTx:
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
	resultError    = "error"
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
// or a retaken one. If storing the response fails after the handler has
// committed its own work, the key stays in_flight and a retry after the lease
// runs the handler again.
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
		out, err := m.acquire(ctx, req, func() *captured { return capture(next, r) })
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
			w.Header().Set("Retry-After", strconv.Itoa(int(Lease/time.Second)))
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
func (m *Middleware) acquire(ctx context.Context, req request, run func() *captured) (*outcome, error) {
	var out *outcome
	err := tenancy.InTenantTx(ctx, m.pool, req.p.TenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING`,
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
		out.resp = run()
		m.complete(ctx, req, out.resp)
	}
	return out, nil
}

// complete stores the response of the request that won the key. The handler's
// side effects already happened, so a failure here is logged and the client
// still gets its response; the key stays in_flight, and a retry after the
// lease runs the handler again.
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
	}
}

func store(ctx context.Context, tx pgx.Tx, req request, resp *captured) error {
	hdrs, err := encodeHeaders(resp.header)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
UPDATE idempotency_keys
   SET state = 'completed', response_status = $4, response_body = $5, response_headers = $6
 WHERE tenant_id = $1 AND principal_id = $2 AND key = $3`,
		req.p.TenantID, req.p.PrincipalID, req.key, int16(resp.status), resp.body, headersParam(hdrs)) //nolint:gosec // an HTTP status is 100-599
	if err != nil {
		return fmt.Errorf("idempotency: store: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("idempotency: store: the row is gone")
	}
	return nil
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
