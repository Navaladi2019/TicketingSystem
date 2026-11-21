//go:build integration

// ADR-006-A2 A2-8 (5) (TICK-1047, T-A2-3): the count budget adds no isolation
// or refusal-path regression. Ported from the assertions of PR #812
// (records_list_probe_integration_test.go), without the pre-count, broadCap,
// CountWhere or the forced probe that A2 rev 2 rejected. Every test asserts
// total_state.
//
// Map to the ADR:
//
//   - TestCountBudgetNoCountOnRefusedRequest: A2-8 (5) AC 1, A2-6 (a). A
//     request refused before the SQL (masked filter, backfilling field,
//     invalid value, invalid cursor, cursor on a masked sort key) sends no
//     count statement and opens no SAVEPOINT list_count, so the budget gives
//     no timing oracle on a refused request.
//   - TestCountBudgetTenantWall: A2-8 (5) AC 2, A2-6 (b). Tenant B's 100k side
//     rows on the value, plus rows forged under tenant A's field id that
//     reference tenant A's own record ids (the EXISTS's v.tenant_id predicate
//     and RLS are all that keep them out), change nothing in tenant A's items,
//     total or total_state, with the count completing (exact) and with it out
//     of budget (unknown).
//   - TestCountBudgetACLDeniedBroadValueLooksAbsent: A2-8 (5) AC 3, A2-6 (c).
//     A value on 20k records the agent cannot read answers exactly like a
//     value that exists nowhere: exact, total 0, and no count statement for
//     either (A2-4), so neither the total nor the time tells them apart.
//   - TestCountBudgetMetricLabels: A2-8 (5) AC 4, A2-7. The indexed-list
//     histogram carries acl, selectivity_band and total_state only, and no
//     tenant id, field key, field id, entity key or filter value.
//
// Statements are counted by a pgx QueryTracer on a test pool, the pattern of
// records/store/list_budget_integration_test.go; the delay is the test-only
// WithCountDelay. No production seam.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// countTrace records every statement a pool sends.
type countTrace struct {
	mu   sync.Mutex
	stmt []string
}

func (l *countTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	l.mu.Lock()
	l.stmt = append(l.stmt, d.SQL)
	l.mu.Unlock()
	return ctx
}
func (*countTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (l *countTrace) reset() { l.mu.Lock(); l.stmt = nil; l.mu.Unlock() }

// countStmt is store.CountStatement, plain or branched, wrapped or not by the
// count delay.
var countStmt = regexp.MustCompile(`count\(\*\) FROM \(SELECT 1 FROM (records r|\()`)

// counts is the number of count statements and of SAVEPOINT list_count
// openings since reset.
func (l *countTrace) counts() (count, savepoint int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, s := range l.stmt {
		if countStmt.MatchString(s) {
			count++
		}
		if strings.Contains(s, "SAVEPOINT list_count;") {
			savepoint++
		}
	}
	return count, savepoint
}

// meteredTel is e's telemetry with a meter provider a test reads.
func meteredTel(e *etEnv) (*telemetry.Telemetry, *sdkmetric.ManualReader) {
	reader := sdkmetric.NewManualReader()
	tel := *e.tel
	tel.MeterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	return &tel, reader
}

// tracedEnv is a copy of e whose handler, built with extra and tel (e's when
// nil), runs on a pool of the same database whose statements trace records.
func (e *etEnv) tracedEnv(t *testing.T, tel *telemetry.Telemetry, extra ...server.Option) (*etEnv, *countTrace) {
	t.Helper()
	if tel == nil {
		tel = e.tel
	}
	trace := &countTrace{}
	cfg := e.pool.Config().Copy()
	cfg.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	h, err := server.NewHandler(slog.New(slog.NewTextHandler(testLogWriter{t, e.logs}, &slog.HandlerOptions{Level: slog.LevelError})),
		tel, e.tokens, stubPaths{},
		append([]server.Option{server.WithLimiter(e.lim), server.WithPool(pool), server.WithAllocatorPool(etAllocPool(t, pool)), server.WithAuditReader(etAuditReader(t, pool)), server.WithCursors(kr)}, extra...)...)
	if err != nil {
		t.Fatal(err)
	}
	c := *e
	c.h = h
	return &c, trace
}

// narrowUser publishes an asset policy on tenant A: the admin reads every
// row, user-a (the agent) reads the rows in the type's current states (a
// condition, so the plan is narrowing, but every record is visible), with
// masks.
func narrowUser(t *testing.T, e *etEnv, masks ...authz.Mask) {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `SELECT DISTINCT state FROM records WHERE tenant_id = $1 AND state IS NOT NULL`, e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	states, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(states) == 0 {
		states = []string{"new"}
	}
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "asset", []authz.Rule{
			{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}},
			{ID: "desk", Ops: []string{authz.OpRead}, Who: authz.Subject{Users: []uuid.UUID{e.a.user.PrincipalID}},
				Where: []authz.Cond{{Column: "state", Op: authz.CondIn, Values: states}}},
		}, masks)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// fakeSideRows inserts, as the superuser, n side rows of value for field in
// tenant with no record behind them (replica role: no foreign keys, no
// triggers). value is SQL over g.
func fakeSideRows(t *testing.T, e *etEnv, vt string, tenant, field uuid.UUID, n int, value string) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), "SET session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = e.super.Exec(t.Context(), "SET session_replication_role = origin") }()
	if _, err := e.super.Exec(t.Context(), fmt.Sprintf(`INSERT INTO rfv_%s (tenant_id, field_id, record_id, ord, value)
		SELECT $1, $2, gen_random_uuid(), 0, %s FROM generate_series(1, %d) g`, vt, value, n), tenant, field); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), "ANALYZE rfv_"+vt); err != nil {
		t.Fatal(err)
	}
}

// cloneRecords inserts, as the superuser, n copies of the record seed with a
// new id and number, requester and data patched by patch (a jsonb object over
// the record's columns). Triggers do not run, so a field flagged afterwards
// indexes them (flagField copies the side rows).
func cloneRecords(t *testing.T, e *etEnv, seed string, n int, patch string, args ...any) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), "SET session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = e.super.Exec(t.Context(), "SET session_replication_role = origin") }()
	if _, err := e.super.Exec(t.Context(), `INSERT INTO records
		SELECT (jsonb_populate_record(r, jsonb_build_object('id', gen_random_uuid(), 'number', 'CLONE-' || g || '-' || $2::text) || `+patch+`)).*
		FROM records r, generate_series(1, $3) g WHERE r.id = $1`, append([]any{seed, uuid.NewString(), n}, args...)...); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), "ANALYZE records"); err != nil {
		t.Fatal(err)
	}
}

// AC 1, A2-6 (a): a refused request sends no count statement and opens no
// count savepoint. The control (an accepted list with a next page) shows the
// trace sees the count.
func TestCountBudgetNoCountOnRefusedRequest(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	e.flagField(t, e.a.id, "asset", "qty", "num", "ready", 100)
	e.flagField(t, e.a.id, "asset", "due", "ts", "backfilling", 40)
	adminOnly := authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}
	narrowUser(t, e, authz.Mask{Field: "qty", Read: adminOnly, Write: adminOnly},
		authz.Mask{Field: "assignee", Read: adminOnly, Write: adminOnly})
	c, trace := e.tracedEnv(t, nil)

	// A cursor whose sort key is the masked assignee, signed with the env's
	// keyring for user-a's query (TestListRecordsMaskedSortKeyInCursor's forgery).
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := cursor.ClaimsFor(identity.NewContext(context.Background(), e.tokens["user-a"]), "listRecords",
		url.Values{"_entity": {"asset"}, "filter[name]": {"alpha"}})
	if err != nil {
		t.Fatal(err)
	}
	pos, _ := json.Marshal(map[string]any{"s": "assignee", "d": true, "v": nil, "i": uuid.NewString()})
	maskedCursor, err := kr.Encode(claims, pos)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, query string
		code        int
		problem     string
	}{
		{"masked filter", "filter[qty]=1", http.StatusBadRequest, "masked_field"},
		{"masked filter beside a ready one", "filter[name]=alpha&filter[qty]=1", http.StatusBadRequest, "masked_field"},
		{"backfilling field", "filter[due][gt]=2026-01-01", http.StatusUnprocessableEntity, "field_not_ready"},
		{"invalid value", "filter[name]=" + strings.Repeat("x", 2001), http.StatusBadRequest, "validation_failed"},
		{"invalid cursor", "filter[name]=alpha&cursor=not-a-cursor", http.StatusBadRequest, "invalid_cursor"},
		{"cursor on a masked sort key", "filter[name]=alpha&cursor=" + maskedCursor, http.StatusBadRequest, "masked_field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trace.reset()
			r, _ := c.list(t, "user-a", "asset", tc.query)
			listCode(t, r, tc.code, tc.problem)
			if n, sp := trace.counts(); n != 0 || sp != 0 {
				t.Errorf("refused request sent %d count statements and %d count savepoints, want none", n, sp)
			}
		})
	}

	// Control: alpha, beta, gamma match and limit=2 leaves a next page, so the
	// shortcut does not apply and exactly one budgeted count runs.
	trace.reset()
	p := c.mustList(t, "user-a", "asset", "limit=2&filter[name][in]=alpha,beta,gamma")
	if n, sp := trace.counts(); n != 1 || sp != 1 {
		t.Errorf("control: %d count statements, %d savepoints; want 1 and 1 (is the trace live?)", n, sp)
	}
	if p.State != "exact" || p.Total != 3 || len(p.Items) != 2 || p.Next == nil {
		t.Errorf("control: state %q total %d, %d items, next %v; want exact 3, 2 items and a next cursor", p.State, p.Total, len(p.Items), p.Next)
	}
}

// tenantView is what tenant A's caller learns from one list response.
type tenantView struct {
	status   int
	ids      []string
	total    string // the raw JSON: a number or null
	state    string
	capped   *bool
	hasNext  bool
	bodyKeys []string
}

func viewOf(t *testing.T, e *etEnv, token, query string) tenantView {
	t.Helper()
	r, p := e.list(t, token, "asset", query)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(r.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body: %v: %s", err, r.Body)
	}
	v := tenantView{status: r.Code, total: string(raw["total"]), state: p.State, capped: p.Capped, hasNext: p.Next != nil}
	for _, it := range p.Items {
		v.ids = append(v.ids, it.ID)
	}
	for k := range raw {
		v.bodyKeys = append(v.bodyKeys, k)
	}
	slices.Sort(v.bodyKeys)
	return v
}

// AC 2, A2-6 (b): tenant B's side rows on the value, including rows forged
// under tenant A's field id that point at tenant A's own record ids, change
// nothing tenant A sees, whether the count completes or runs out of budget.
func TestCountBudgetTenantWall(t *testing.T) {
	e := newFilterEnv(t)
	aField := e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	bField := e.flagField(t, e.b.id, "asset", "name", "text", "ready", 100)
	var mine []string
	for i := range 5 {
		mine = append(mine, e.mustCreateRec(t, "admin-a", "asset", fmt.Sprintf("hot%d", i), fmt.Sprintf(`{"short_description": "a%d", "name": "hot"}`, i)).ID)
	}
	cold := e.mustCreateRec(t, "admin-a", "asset", "cold", `{"short_description": "c", "name": "cold"}`)
	// Tenant B's own records with the value, so its side table is not empty
	// of real rows either.
	e.mustCreateRec(t, "admin-b", "asset", "bhot", `{"short_description": "b", "name": "hot"}`)
	narrowUser(t, e)

	exact, exactTrace := e.tracedEnv(t, nil)
	slow, slowTrace := e.tracedEnv(t, nil, server.WithCountDelay(400*time.Millisecond))
	const q = "limit=2&filter[name]=hot"
	type run struct {
		name  string
		env   *etEnv
		trace *countTrace
		token string
		state string
	}
	runs := []run{
		{"agent, count completes", exact, exactTrace, "user-a", "exact"},
		{"admin, count completes", exact, exactTrace, "admin-a", "exact"},
		{"agent, count out of budget", slow, slowTrace, "user-a", "unknown"},
		{"admin, count out of budget", slow, slowTrace, "admin-a", "unknown"},
	}
	look := func(stage string) map[string]tenantView {
		t.Helper()
		out := map[string]tenantView{}
		for _, r := range runs {
			r.trace.reset()
			v := viewOf(t, r.env, r.token, q)
			if v.status != http.StatusOK || v.state != r.state || len(v.ids) != 2 || !v.hasNext {
				t.Fatalf("%s, %s: %d %q, %d items, next %v; want 200 %s, 2 items and a next page", stage, r.name, v.status, v.state, len(v.ids), v.hasNext, r.state)
			}
			if n, _ := r.trace.counts(); n != 1 {
				t.Errorf("%s, %s: %d count statements, want 1 (A has more than limit matches)", stage, r.name, n)
			}
			for _, id := range v.ids {
				if !slices.Contains(mine, id) {
					t.Errorf("%s, %s: item %s is not one of tenant A's hot records", stage, r.name, id)
				}
			}
			switch r.state {
			case "exact":
				if v.total != "5" {
					t.Errorf("%s, %s: total %s, want 5", stage, r.name, v.total)
				}
			case "unknown":
				if v.total != "null" {
					t.Errorf("%s, %s: total %s, want null", stage, r.name, v.total)
				}
			}
			out[r.name] = v
		}
		return out
	}
	before := look("without B's rows")
	// Volume: 100k tenant B side rows on the value with no record behind them,
	// half under B's field id and half under A's.
	fakeSideRows(t, e, "text", e.b.id, bField, 50000, "'hot'")
	fakeSideRows(t, e, "text", e.b.id, aField, 50000, "'hot'")
	// The forgery that can leak: tenant B rows under tenant A's field id whose
	// record_id is each of tenant A's real records, cold included, with the
	// value hot. The registry cannot produce them. They satisfy the EXISTS's
	// v.record_id = r.id, v.field_id and v.value for A's records, so only its
	// v.tenant_id = r.tenant_id predicate and forced RLS on rfv_text keep them
	// out: a leak turns A's total 5 into 6 and surfaces cold, the newest
	// record, in items.
	if _, err := e.super.Exec(t.Context(), "SET session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), `INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value)
		SELECT $1, $2, r.id, 0, 'hot' FROM records r WHERE r.tenant_id = $3`, e.b.id, aField, e.a.id); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), "SET session_replication_role = origin"); err != nil {
		t.Fatal(err)
	}
	var forged, forgedCold int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*), count(*) FILTER (WHERE r.id = $4)
		FROM rfv_text v JOIN records r ON r.tenant_id = $3 AND r.id = v.record_id
		WHERE v.tenant_id = $1 AND v.field_id = $2 AND v.value = 'hot'`, e.b.id, aField, e.a.id, cold.ID).Scan(&forged, &forgedCold); err != nil || forged != 6 || forgedCold != 1 {
		t.Fatalf("fixture: %d tenant B rows under A's field id reference A's records (%d the cold one), want 6 (1): %v", forged, forgedCold, err)
	}
	after := look("with B's rows")
	for _, r := range runs {
		if !reflect.DeepEqual(before[r.name], after[r.name]) {
			t.Errorf("%s: tenant B's rows changed tenant A's response:\nbefore %+v\nafter  %+v", r.name, before[r.name], after[r.name])
		}
	}
}

// AC 3, A2-6 (c): a value on 20k records the agent cannot read looks exactly
// like a value that exists nowhere, and neither runs a count (A2-4).
func TestCountBudgetACLDeniedBroadValueLooksAbsent(t *testing.T) {
	e := newFilterEnv(t)
	// The agent (user-a) reads only the records it requests; the admin reads all.
	e.publishACL(t, e.a.id, "asset",
		authz.Rule{ID: "requester", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	seed := e.mustCreateRec(t, "admin-a", "asset", "deny-seed", fmt.Sprintf(`{"short_description": "s", "name": "hot", "requester": %q}`, e.a.admin.PrincipalID))
	const broad = 20000
	cloneRecords(t, e, seed.ID, broad-1, `jsonb_build_object('requester_id', $4::uuid)`, e.a.admin.PrincipalID)
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	var carried int
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM rfv_text v JOIN records r ON r.tenant_id = v.tenant_id AND r.id = v.record_id
		WHERE v.tenant_id = $1 AND v.value = 'hot' AND r.requester_id IS DISTINCT FROM $2`, e.a.id, e.a.user.PrincipalID).Scan(&carried); err != nil || carried < broad {
		t.Fatalf("fixture: %d records carry the value and are not the agent's, want >= %d: %v", carried, broad, err)
	}
	c, trace := e.tracedEnv(t, nil)

	// Control: the records are real and readable by someone.
	if p := c.mustList(t, "admin-a", "asset", "limit=10&filter[name]=hot"); p.State != "capped" || p.Total != totalCap {
		t.Fatalf("control: admin sees %q total %d, want capped %d", p.State, p.Total, totalCap)
	}

	type answer struct {
		status int
		body   map[string]any
		counts int
	}
	ask := func(value string) answer {
		t.Helper()
		trace.reset()
		r, _ := c.list(t, "user-a", "asset", "filter[name]="+value)
		var body map[string]any
		if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: body %v: %s", value, err, r.Body)
		}
		n, _ := trace.counts()
		return answer{r.Code, body, n}
	}
	denied, absent := ask("hot"), ask("nobody-has-this")
	if denied.status != http.StatusOK || absent.status != http.StatusOK {
		t.Fatalf("status %d / %d, want 200 / 200", denied.status, absent.status)
	}
	if !reflect.DeepEqual(denied.body, absent.body) {
		t.Errorf("an ACL-denied broad value must look absent:\ndenied %v\nabsent %v", denied.body, absent.body)
	}
	if denied.body["total_state"] != "exact" || denied.body["total"] != float64(0) {
		t.Errorf("denied: total %v %v, want exact 0", denied.body["total"], denied.body["total_state"])
	}
	if items, _ := denied.body["items"].([]any); items == nil || len(items) != 0 {
		t.Errorf("denied: items %v, want []", denied.body["items"])
	}
	if denied.counts != 0 || absent.counts != 0 {
		t.Errorf("count statements: denied %d, absent %d; want 0 and 0 (A2-4 shortcut)", denied.counts, absent.counts)
	}
}

// AC 4, A2-7: after an exact, a capped and an unknown indexed list, the
// histogram's points carry exactly acl, selectivity_band and total_state, and
// no attribute holds the tenant id, entity key, field key, field id or filter
// value. Every identifier is a canary string.
func TestCountBudgetMetricLabels(t *testing.T) {
	e := newETEnv(t)
	const (
		entityKey = "cnryentk7q"
		fieldKey  = "cnryfieldk7q"
		exactVal  = "cnryvalexactk7q"
		cappedVal = "cnryvalcapk7q"
	)
	body := strings.Replace(assetTypeBody(entityKey, "CNY"), `"name": {"type": "string"`, `"`+fieldKey+`": {"type": "string"`, 1)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "cny", body); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	for i := range 3 {
		e.mustCreateRec(t, "admin-a", entityKey, fmt.Sprintf("x%d", i), fmt.Sprintf(`{"short_description": "x%d", %q: %q}`, i, fieldKey, exactVal))
	}
	seed := e.mustCreateRec(t, "admin-a", entityKey, "capseed", fmt.Sprintf(`{"short_description": "c", %q: %q}`, fieldKey, cappedVal))
	cloneRecords(t, e, seed.ID, totalCap+50, `'{}'::jsonb`)
	field := e.flagField(t, e.a.id, entityKey, fieldKey, "text", "ready", 100)

	tel, reader := meteredTel(e)
	fast, _ := e.tracedEnv(t, tel)
	slow, _ := e.tracedEnv(t, tel, server.WithCountDelay(400*time.Millisecond))
	for _, l := range []struct {
		env   *etEnv
		value string
		state string
	}{{fast, exactVal, "exact"}, {fast, cappedVal, "capped"}, {slow, exactVal, "unknown"}} {
		if p := l.env.mustList(t, "admin-a", entityKey, "limit=2&"+filterQuery(fieldKey, "eq", l.value)); p.State != l.state {
			t.Fatalf("?%s: total_state %q, want %q", l.value, p.State, l.state)
		}
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(t.Context(), &rm); err != nil {
		t.Fatal(err)
	}
	var pts []metricdata.HistogramDataPoint[float64]
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "nava.records.list.indexed.duration" {
				pts = append(pts, m.Data.(metricdata.Histogram[float64]).DataPoints...)
			}
		}
	}
	canaries := []string{e.a.id.String(), entityKey, fieldKey, field.String(), exactVal, cappedVal, "cnry"}
	states := map[string]bool{}
	for _, pt := range pts {
		var keys []string
		check := func(where string, k, v string) {
			for _, c := range canaries {
				if strings.Contains(k, c) || strings.Contains(v, c) {
					t.Errorf("%s attribute %s=%q carries the canary %q", where, k, v, c)
				}
			}
		}
		for _, kv := range pt.Attributes.ToSlice() {
			keys = append(keys, string(kv.Key))
			check("point", string(kv.Key), kv.Value.String())
		}
		for _, ex := range pt.Exemplars {
			for _, kv := range ex.FilteredAttributes {
				check("exemplar", string(kv.Key), kv.Value.String())
			}
		}
		slices.Sort(keys)
		if !slices.Equal(keys, []string{"acl", "selectivity_band", "total_state"}) {
			t.Errorf("point attributes %v, want exactly acl, selectivity_band, total_state", keys)
		}
		if v, ok := pt.Attributes.Value("total_state"); ok {
			states[v.AsString()] = true
		}
	}
	if !states["exact"] || !states["capped"] || !states["unknown"] || len(states) != 3 {
		t.Errorf("total_state values %v, want exact, capped and unknown", states)
	}
}
