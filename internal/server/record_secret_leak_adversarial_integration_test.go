//go:build integration

// P5-E5-7 (TICK-540), ADR-004-E5 D8, CI-E5-7 (leak half): a value at a
// secret-marked record property (writeOnly or x-nava-secret) is refused with
// audit_secret_in_diff, and the value reaches none of the surfaces a write
// touches: the response body, any table of the platform database (records,
// record_write_keys, audit_log, idempotency_keys and every other one), the
// request logs at debug level, and the exported trace spans. A replay with
// the same Idempotency-Key is refused again and leaks nothing, and the
// attacker's tenant A write leaves tenant B's rows unchanged.
//
// The handler runs as in production: its pool is the traced one
// (internal/platform/db.NewPool, otelpgx), every span is sampled and exported
// in memory, and the logger is logging.New at debug level, the most verbose
// a deployment can set.

package server_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// leakSchema marks pin writeOnly and api_key x-nava-secret (ADR-004 R6).
const leakSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string", "minLength": 1, "maxLength": 160},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "pin": {"type": "string", "writeOnly": true},
    "api_key": {"type": "string", "x-nava-secret": true}
  }
}`

func leakTypeBody() string {
	return fmt.Sprintf(`{"key": "account", "name": "Account", "number_prefix": "ACC", "definition": {"schema": %s,
  "fixed_columns": {"state": "state"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`, leakSchema)
}

// leakEnv is etEnv's database and tenants behind a handler built the way
// nava-core builds it, with every span and log line captured.
type leakEnv struct {
	*etEnv
	spans *tracetest.InMemoryExporter
	logs  *etLogBuf
}

func newLeakEnv(t *testing.T) *leakEnv {
	t.Helper()
	base := newETEnv(t)
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{},
		telemetry.WithSpanExporter(spans), telemetry.WithSampler(sdktrace.AlwaysSample()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })

	// The production pool: traced through tel (otelpgx), so DB spans are
	// captured too. Same login as etCluster's nava_app pool.
	pc, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig.User, pc.ConnConfig.Password = "nava_app", etPassword("nava_app")
	pc.ConnConfig.Database = base.pool.Config().ConnConfig.Database
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = 8
	pool, err := platformdb.NewPool(t.Context(), pc, tel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	lim := &spyLimiter{}
	lim.reset(true)
	logs := &etLogBuf{}
	log := logging.New(testLogWriter{t, logs}, config.Log{Level: slog.LevelDebug, Format: config.FormatJSON})
	h, err := server.NewHandler(log, tel, base.tokens, stubPaths{},
		server.WithLimiter(lim), server.WithPool(pool), server.WithCursors(kr))
	if err != nil {
		t.Fatal(err)
	}
	e := &leakEnv{etEnv: &etEnv{h: h, tokens: base.tokens, pool: pool, a: base.a, b: base.b, super: base.super, logs: logs}, spans: spans, logs: logs}
	for _, admin := range []string{"admin-a", "admin-b"} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", admin, "leak-type-"+admin, leakTypeBody()); rec.Code != http.StatusCreated {
			t.Fatalf("seed type as %s = %d: %s", admin, rec.Code, rec.Body)
		}
	}
	return e
}

// patchWithKey sends a merge patch with a fixed Idempotency-Key, so the same
// request can be replayed.
func (e *leakEnv) patchWithKey(t *testing.T, token, id, ifMatch, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/records/account/"+id, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/merge-patch+json")
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("If-Match", ifMatch)
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// wantSecretRefused asserts a 422 record_invalid whose only error is
// audit_secret_in_diff at /data/<prop>, and that the body never quotes the
// canary.
func wantSecretRefused(t *testing.T, rec *httptest.ResponseRecorder, prop, canary string) {
	t.Helper()
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.RecordInvalid)
	want := problem.FieldError{Pointer: "/data/" + prop, Rule: "audit_secret_in_diff"}
	if len(p.Errors) != 1 || p.Errors[0] != want {
		t.Errorf("errors = %+v, want exactly %+v", p.Errors, want)
	}
	if strings.Contains(rec.Body.String(), canary) {
		t.Errorf("LEAK response body: the canary is in %s", rec.Body)
	}
	for k, vs := range rec.Header() {
		for _, v := range vs {
			if strings.Contains(v, canary) {
				t.Errorf("LEAK response header %s: %q", k, v)
			}
		}
	}
}

// tablesHolding returns every table of the platform database whose rows,
// cast to text, contain the canary as text or as hex (a bytea column such
// as idempotency_keys.response_body renders as \x<hex>). The superuser
// connection bypasses RLS, so every tenant's rows are searched.
func (e *leakEnv) tablesHolding(t *testing.T, canary string) []string {
	t.Helper()
	ctx := t.Context()
	rows, err := e.super.Query(ctx, `
SELECT format('%I.%I', n.nspname, c.relname)
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p', 'm')
  AND NOT c.relispartition
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	tables, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) < 5 {
		t.Fatalf("found only %d tables to search: %v", len(tables), tables)
	}
	hexCanary := hex.EncodeToString([]byte(canary))
	var hits []string
	for _, tbl := range tables {
		var n int
		q := "SELECT count(*) FROM " + tbl + " AS x WHERE strpos(x::text, $1) > 0 OR strpos(x::text, $2) > 0"
		if err := e.super.QueryRow(ctx, q, canary, hexCanary).Scan(&n); err != nil {
			t.Fatalf("search %s: %v", tbl, err)
		}
		if n > 0 {
			hits = append(hits, fmt.Sprintf("%s (%d rows)", tbl, n))
		}
	}
	return hits
}

// spanLeaks returns every place in the exported spans that carries the
// canary: names, attributes, events and their attributes, status, links.
func (e *leakEnv) spanLeaks(canary string) []string {
	var out []string
	check := func(where, s string) {
		if strings.Contains(s, canary) {
			out = append(out, where+": "+s)
		}
	}
	for _, s := range e.spans.GetSpans() {
		check("span name", s.Name)
		check("span "+s.Name+" status", s.Status.Description)
		for _, kv := range s.Attributes {
			check("span "+s.Name+" attribute "+string(kv.Key), kv.Value.String())
		}
		for _, ev := range s.Events {
			check("span "+s.Name+" event name", ev.Name)
			for _, kv := range ev.Attributes {
				check("span "+s.Name+" event "+ev.Name+" attribute "+string(kv.Key), kv.Value.String())
			}
		}
		for _, l := range s.Links {
			for _, kv := range l.Attributes {
				check("span "+s.Name+" link attribute "+string(kv.Key), kv.Value.String())
			}
		}
		if s.Resource != nil {
			for _, kv := range s.Resource.Attributes() {
				check("resource attribute "+string(kv.Key), kv.Value.String())
			}
		}
	}
	return out
}

// wantServerSpan asserts that a server span with http.route route was
// exported after the first before spans.
func (e *leakEnv) wantServerSpan(t *testing.T, before int, route string) {
	t.Helper()
	spans := e.spans.GetSpans()
	if len(spans) <= before {
		t.Fatalf("no span was exported for the request (%d spans before and after)", before)
	}
	for _, s := range spans[before:] {
		if s.SpanKind != trace.SpanKindServer {
			continue
		}
		for _, kv := range s.Attributes {
			if kv.Key == semconv.HTTPRouteKey && kv.Value.AsString() == route {
				return
			}
		}
	}
	t.Fatalf("no server span with http.route %q among the %d spans the request exported", route, len(spans)-before)
}

// assertNoLeak checks every surface but the response body, which the caller
// checks per response.
func (e *leakEnv) assertNoLeak(t *testing.T, canary string) {
	t.Helper()
	if hits := e.tablesHolding(t, canary); len(hits) > 0 {
		t.Errorf("LEAK database: the canary is stored in %v", hits)
	}
	if strings.Contains(e.logs.String(), canary) {
		t.Errorf("LEAK request logs: the canary is logged:\n%s", e.logs.String())
	}
	if len(e.spans.GetSpans()) == 0 {
		t.Error("no spans were exported: the trace surface was not exercised")
	}
	if leaks := e.spanLeaks(canary); len(leaks) > 0 {
		t.Errorf("LEAK trace spans: %v", leaks)
	}
}

// tenantBSnapshot is tenant B's rows of every record-domain table, as text.
func (e *leakEnv) tenantBSnapshot(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	for _, tbl := range []string{"records", "record_write_keys", "audit_log", "idempotency_keys"} {
		var s string
		q := "SELECT coalesce(string_agg(x::text, E'\\n' ORDER BY x::text), '') FROM " + tbl + " AS x WHERE tenant_id = $1"
		if err := e.super.QueryRow(t.Context(), q, e.b.id).Scan(&s); err != nil {
			t.Fatalf("snapshot %s: %v", tbl, err)
		}
		sb.WriteString(tbl + ":\n" + s + "\n")
	}
	return sb.String()
}

func TestRecordSecretLeaksToNoSurface(t *testing.T) {
	e := newLeakEnv(t)

	// Tenant B holds a record before tenant A's attack.
	bRec := e.do(t, http.MethodPost, "/api/v1/records/account", "user-b", "leak-b-seed", `{"data": {"short_description": "tenant B's account"}}`)
	if bRec.Code != http.StatusCreated {
		t.Fatalf("seed tenant B record = %d: %s", bRec.Code, bRec.Body)
	}
	bBefore := e.tenantBSnapshot(t)

	// Tenant A's own record, the PATCH target.
	target := e.do(t, http.MethodPost, "/api/v1/records/account", "user-a", "leak-a-seed", `{"data": {"short_description": "tenant A's account"}}`)
	if target.Code != http.StatusCreated {
		t.Fatalf("seed tenant A record = %d: %s", target.Code, target.Body)
	}
	var tb recBody
	if err := json.Unmarshal(target.Body.Bytes(), &tb); err != nil {
		t.Fatal(err)
	}
	recordsBefore := e.recordCount(t, "a")

	for _, prop := range []string{"pin", "api_key"} {
		for _, method := range []string{http.MethodPost, http.MethodPatch} {
			t.Run(method+" "+prop, func(t *testing.T) {
				canary := "CANARY-" + strings.ToUpper(prop) + "-" + uuid.NewString()
				key := "leak-" + method + "-" + prop
				route := "/api/v1/records/{entity}"
				if method == http.MethodPatch {
					route = "/api/v1/records/{entity}/{id}"
				}
				// send sends the attack and proves it was traced: a server span
				// for its route is exported after the send, so the span check
				// below looks at the attack's own spans.
				send := func() *httptest.ResponseRecorder {
					t.Helper()
					before := len(e.spans.GetSpans())
					var rec *httptest.ResponseRecorder
					if method == http.MethodPost {
						body := fmt.Sprintf(`{"data": {"short_description": "attack", %q: %q}}`, prop, canary)
						rec = e.do(t, http.MethodPost, "/api/v1/records/account", "user-a", key, body)
					} else {
						rec = e.patchWithKey(t, "user-a", tb.ID, `W/"1"`, key, fmt.Sprintf(`{"data": {%q: %q}}`, prop, canary))
					}
					e.wantServerSpan(t, before, route)
					return rec
				}

				// AC1, AC2: refused, naming the pointer only, and no surface holds it.
				wantSecretRefused(t, send(), prop, canary)
				e.assertNoLeak(t, canary)

				// AC3: the replay with the same Idempotency-Key is refused again.
				wantSecretRefused(t, send(), prop, canary)
				e.assertNoLeak(t, canary)
			})
		}
	}

	if n := e.recordCount(t, "a"); n != recordsBefore {
		t.Errorf("tenant A has %d records after the refused writes, want %d", n, recordsBefore)
	}
	got := e.do(t, http.MethodGet, "/api/v1/records/account/"+tb.ID, "user-a", "", "")
	if got.Code != http.StatusOK || got.Header().Get("ETag") != `W/"1"` {
		t.Errorf("tenant A's target after the refused patches = %d ETag %q, want 200 W/\"1\": %s", got.Code, got.Header().Get("ETag"), got.Body)
	}

	// AC4: tenant B's rows are byte-for-byte unchanged.
	if bAfter := e.tenantBSnapshot(t); bAfter != bBefore {
		t.Errorf("tenant B's rows changed during tenant A's writes:\nbefore:\n%s\nafter:\n%s", bBefore, bAfter)
	}

	// Positive control: an unmarked value is stored, so the database search
	// above finds what is there and its passes are not vacuous.
	control := "CONTROL-" + uuid.NewString()
	if rec := e.do(t, http.MethodPost, "/api/v1/records/account", "user-a", "leak-control", fmt.Sprintf(`{"data": {"short_description": %q}}`, control)); rec.Code != http.StatusCreated {
		t.Fatalf("control create = %d: %s", rec.Code, rec.Body)
	}
	// The log capture sees the handler's records, so the log checks above
	// are not vacuous either.
	if !strings.Contains(e.logs.String(), `"entity type created"`) {
		t.Errorf("control: the captured logs lack \"entity type created\"; the log capture is blind:\n%s", e.logs.String())
	}
	hits := strings.Join(e.tablesHolding(t, control), " ")
	for _, tbl := range []string{"public.records ", "public.audit_log ", "public.idempotency_keys "} {
		if !strings.Contains(hits, tbl) {
			t.Errorf("control: the unmarked value was not found in %s (hits: %s); the search is blind", tbl, hits)
		}
	}
}
