//go:build integration

// ADR-006-A2 (TICK-1022): the bounded count probe. A narrowing list pre-counts
// each ready filtered field's side rows in the tenant up to broadCap; a broad
// filter on a visible set of at most visibleCap records is probed per record
// in the count only. These tests are the Security & Multi-Tenancy Reviewer's
// conditions on the change (TICK-1022, ruling on A2-6 / R13) and A2-4/A2-5.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// probeRecorder counts the pre-counts and the probe decisions of the lists
// an env built with its option runs, under caps a test may change.
type probeRecorder struct {
	mu                   sync.Mutex
	pre, probe           []bool
	broadCap, visibleCap int
}

// newProbeEnv is a filter env whose handler reports to a recorder set to the
// production caps.
func newProbeEnv(t *testing.T) (*etEnv, *probeRecorder) {
	t.Helper()
	r := &probeRecorder{}
	r.broadCap, r.visibleCap = server.DefaultCountProbeCaps()
	e := newFilterEnvWith(t, server.WithCountProbe(
		func() (int, int) { r.mu.Lock(); defer r.mu.Unlock(); return r.broadCap, r.visibleCap },
		func(b bool) { r.mu.Lock(); r.pre = append(r.pre, b); r.mu.Unlock() },
		func(p bool) { r.mu.Lock(); r.probe = append(r.probe, p); r.mu.Unlock() }))
	return e, r
}

// set changes the caps for the lists that follow.
func (r *probeRecorder) set(broadCap, visibleCap int) {
	r.mu.Lock()
	r.broadCap, r.visibleCap = broadCap, visibleCap
	r.mu.Unlock()
}

// take returns and clears what the recorder saw.
func (r *probeRecorder) take() (pre, probe []bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pre, probe, r.pre, r.probe = r.pre, r.probe, nil, nil
	return pre, probe
}

// narrowUser publishes an asset policy on tenant A: the admin reads every
// row, user-a reads the rows in the type's current states (a condition, so
// the plan is narrowing, but every record is visible), and qty is masked
// from everyone but the admin.
func narrowUser(t *testing.T, e *etEnv, masks ...authz.Mask) {
	t.Helper()
	var states []string
	rows, err := e.super.Query(t.Context(), `SELECT DISTINCT state FROM records WHERE tenant_id = $1 AND state IS NOT NULL`, e.a.id)
	if err != nil {
		t.Fatal(err)
	}
	if states, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
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

// fakeSideRows inserts n side rows of value v for field id in tenant, with no
// record behind them (replica role: no foreign keys). value is SQL over g.
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

// Security condition 1: a request refused before the SQL, and a list by an
// actor who reads every row, run no pre-count; a narrowing list runs one per
// filtered field.
func TestListCountProbeNoPreCountOnRefusals(t *testing.T) {
	e, rec := newProbeEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	e.flagField(t, e.a.id, "asset", "qty", "num", "ready", 100)
	e.flagField(t, e.a.id, "asset", "due", "ts", "backfilling", 40)
	adminOnly := authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}
	narrowUser(t, e, authz.Mask{Field: "qty", Read: adminOnly, Write: adminOnly},
		authz.Mask{Field: "assignee", Read: adminOnly, Write: adminOnly})
	// A cursor whose sort key is the masked assignee, signed with the env's
	// keyring for user-a's query: refused 400 masked_field after the filters
	// resolve (TestListRecordsMaskedSortKeyInCursor's forgery).
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
	for _, c := range []struct {
		name, token, query string
		code               int
		problem            string
	}{
		{"masked", "user-a", "filter[qty]=1", http.StatusBadRequest, "masked_field"},
		{"masked with a ready filter", "user-a", "filter[name]=alpha&filter[qty]=1", http.StatusBadRequest, "masked_field"},
		{"backfilling", "user-a", "filter[due][gt]=2026-01-01", http.StatusUnprocessableEntity, "field_not_ready"},
		{"invalid value", "user-a", "filter[name]=" + strings.Repeat("x", 2001), http.StatusBadRequest, "validation_failed"},
		{"invalid cursor", "user-a", "filter[name]=alpha&cursor=not-a-cursor", http.StatusBadRequest, "invalid_cursor"},
		{"masked cursor sort", "user-a", "filter[name]=alpha&cursor=" + maskedCursor, http.StatusBadRequest, "masked_field"},
	} {
		r, _ := e.list(t, c.token, "asset", c.query)
		listCode(t, r, c.code, c.problem)
		if pre, _ := rec.take(); len(pre) != 0 {
			t.Errorf("%s: %d pre-counts on a refused request", c.name, len(pre))
		}
	}
	// The admin reads every row: no pre-count.
	e.mustList(t, "admin-a", "asset", "filter[name]=alpha&filter[qty]=1")
	if pre, _ := rec.take(); len(pre) != 0 {
		t.Errorf("admin: %d pre-counts", len(pre))
	}
	// The control: a narrowing list pre-counts each filtered field once.
	e.mustList(t, "user-a", "asset", "filter[name]=alpha&filter[name][in]=alpha,beta")
	if pre, _ := rec.take(); len(pre) != 1 || pre[0] {
		t.Errorf("narrowing, one field: pre-counts %v, want one, not broad", pre)
	}
}

// Security condition 2: the pre-count is the tenant's own. Tenant B holds 50k
// side rows of the value; tenant A's filter on it is not broad, and A's count
// is the selective one.
func TestListCountProbePreCountIsTenantScoped(t *testing.T) {
	e, rec := newProbeEnv(t)
	aField := e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	bField := e.flagField(t, e.b.id, "asset", "name", "text", "ready", 100)
	mine := e.mustCreateRec(t, "admin-a", "asset", "ts1", `{"short_description": "a", "name": "hot"}`)
	fakeSideRows(t, e, "text", e.b.id, bField, 50000, "'hot'")
	// The same field id in tenant B cannot happen through the registry; forge it
	// too, so the tenant predicate alone is what keeps the rows out.
	fakeSideRows(t, e, "text", e.b.id, aField, 50000, "'hot'")
	narrowUser(t, e)
	p := e.mustList(t, "user-a", "asset", "filter[name]=hot")
	wantIDs(t, "A", p.Items, mine)
	if p.Total != 1 {
		t.Errorf("total %d, want 1", p.Total)
	}
	if pre, probe := rec.take(); !slices.Equal(pre, []bool{false}) || len(probe) != 0 {
		t.Errorf("tenant B's rows reached A's pre-count: pre %v, probe %v", pre, probe)
	}
}

// Security condition 4 (and CI-6c's broad variant): a value whose side rows
// are all ACL-denied or have no record behind them gives the same hits,
// total, status and body shape as a value that exists nowhere, through the
// probing count; the caller's own record with it is found.
func TestListCountProbeACLDeniedBroadValueLooksAbsent(t *testing.T) {
	e, rec := newProbeEnv(t)
	field := e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	// user-a reads only the records it requests; the admin reads all.
	e.publishACL(t, e.a.id, "asset",
		authz.Rule{ID: "requester", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	secret := e.mustCreateRec(t, "admin-a", "asset", "d1", `{"short_description": "s", "name": "hot"}`)
	broadCap, _ := server.DefaultCountProbeCaps()
	fakeSideRows(t, e, "text", e.a.id, field, broadCap+5000, "'hot'")

	ra, a := e.list(t, "user-a", "asset", "filter[name]=hot")
	preA, probeA := rec.take()
	rb, b := e.list(t, "user-a", "asset", "filter[name]=nobody-has-this")
	preB, _ := rec.take()
	if ra.Code != http.StatusOK || rb.Code != http.StatusOK {
		t.Fatalf("status %d / %d", ra.Code, rb.Code)
	}
	if !slices.Equal(preA, []bool{true}) || !slices.Equal(probeA, []bool{true}) || !slices.Equal(preB, []bool{false}) {
		t.Fatalf("the broad value did not take the probe: pre %v probe %v, absent value pre %v", preA, probeA, preB)
	}
	if len(a.Items) != 0 || len(b.Items) != 0 || a.Total != 0 || b.Total != 0 || ids(a.Items)[secret.ID] {
		t.Errorf("an ACL-denied broad value must look absent: %d items/total %d vs %d items/total %d", len(a.Items), a.Total, len(b.Items), b.Total)
	}
	if ka, kb := bodyKeys(t, ra.Body.Bytes()), bodyKeys(t, rb.Body.Bytes()); !slices.Equal(ka, kb) {
		t.Errorf("body shapes differ: %v vs %v", ka, kb)
	}
	// The requester's own record with the value is found by the probing count.
	own := e.mustCreateRec(t, "admin-a", "asset", "d2", fmt.Sprintf(`{"short_description": "m", "name": "hot", "requester": %q}`, e.a.user.PrincipalID))
	p := e.mustList(t, "user-a", "asset", "filter[name]=hot")
	wantIDs(t, "own", p.Items, own)
	if p.Total != 1 {
		t.Errorf("own: total %d, want 1", p.Total)
	}
	if _, probe := rec.take(); !slices.Equal(probe, []bool{true}) {
		t.Errorf("own: probe %v", probe)
	}
}

func bodyKeys(t *testing.T, b []byte) []string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Guard A: with more visible records than visibleCap the count keeps its
// plain form even for a broad filter, and the rows are the same.
func TestListCountProbeVisibleSetGuard(t *testing.T) {
	e, rec := newProbeEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	narrowUser(t, e)
	rec.set(1, 2) // every matching value is broad; 7 visible > 2
	p := e.mustList(t, "user-a", "asset", "filter[name][in]=alpha,beta,gamma")
	if pre, probe := rec.take(); !slices.Equal(pre, []bool{true}) || !slices.Equal(probe, []bool{false}) {
		t.Errorf("guard: pre %v probe %v, want broad and no probe", pre, probe)
	}
	if p.Total != 3 {
		t.Errorf("total %d, want 3", p.Total)
	}
}

// A2-5: the probing count gives the same total as the plain count on the
// edge values of every type: collation ("alpha" against "Alpha"), the empty
// string, in with repeats, negative and fractional numbers, time-zone
// offsets, inclusive and exclusive bounds, and bool.
func TestListCountProbeSameTotalsAsThePlainCount(t *testing.T) {
	e, rec := newProbeEnv(t)
	e.createAssets(t, "admin-a")
	for _, f := range []struct{ key, vt string }{{"name", "text"}, {"qty", "num"}, {"due", "ts"}, {"vip", "bool"}} {
		e.flagField(t, e.a.id, "asset", f.key, f.vt, "ready", 100)
	}
	narrowUser(t, e)
	queries := []string{
		"filter[name]=alpha", "filter[name]=Alpha", "filter[name]=", "filter[name][in]=alpha,alpha,Alpha,beta",
		"filter[qty]=5", "filter[qty][lt]=0", "filter[qty][lte]=2.5", "filter[qty][gt]=-3", "filter[qty][gte]=-3",
		"filter[qty][in]=1,1,5", "filter[qty][gte]=1&filter[qty][lt]=5",
		"filter[due][gte]=" + fiEscForProbe("2026-03-01T18:30:00Z") + "&filter[due][lt]=" + fiEscForProbe("2026-03-05T00:00:00Z"),
		"filter[due]=" + fiEscForProbe("2026-03-05T00:00:00+00:00"), "filter[due][gt]=2026-03-01",
		"filter[vip]=true", "filter[vip]=false",
		"filter[name][in]=alpha,gamma&filter[vip]=true&filter[qty][gte]=1",
	}
	plain := map[string]int{}
	rec.set(1<<30, 30000) // nothing is broad
	for _, q := range queries {
		plain[q] = e.mustList(t, "user-a", "asset", q).Total
	}
	rec.take()
	rec.set(1, 30000) // every matching value is broad; all 7 visible
	for _, q := range queries {
		got := e.mustList(t, "user-a", "asset", q).Total
		_, probe := rec.take()
		if got > 0 && !slices.Equal(probe, []bool{true}) {
			t.Errorf("?%s: probe %v, want the probing count", q, probe)
		}
		if got != plain[q] {
			t.Errorf("?%s: probing count total %d, plain %d", q, got, plain[q])
		}
	}
}

func fiEscForProbe(s string) string { return strings.NewReplacer(":", "%3A", "+", "%2B").Replace(s) }

// A2-4 (the plan is the gate, extending CI-1): as nava_app under forced RLS,
// for each type and operator, a probing count reaches the side table through
// a per-record SubPlan whose Index Cond seeks record_id = r.id, with no Hash
// or Merge join anywhere; and the pre-count is an index scan on the value
// index that stops at the cap: an ordered Index (Only) Scan, never a Seq
// Scan, a Bitmap scan or a Sort (Technical Architect, A2 rev 0, guard C). Run before
// and after VACUUM, so both visibility-map states are covered.
func TestListCountProbePlans(t *testing.T) {
	e := newFilterEnv(t)
	fields := map[fieldindex.ValueType]string{fieldindex.TypeText: "name", fieldindex.TypeNum: "qty", fieldindex.TypeTS: "due", fieldindex.TypeBool: "vip"}
	ids := map[fieldindex.ValueType]uuid.UUID{}
	for _, vt := range fieldindex.ValueTypes() {
		ids[vt] = e.flagField(t, e.a.id, "asset", fields[vt], string(vt), "ready", 100)
	}
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	// Records the count walks, and side rows for them; plus 40k side rows per
	// field with no record behind them, so the value is broad in the tenant.
	for i := 0; i < 50; i++ {
		e.mustCreateRec(t, "admin-a", "asset", fmt.Sprintf("p%d", i),
			fmt.Sprintf(`{"short_description": "p", "name": "v%d", "qty": %d, "due": %q, "vip": %t}`, i%5, i%5, base.Add(time.Duration(i%5)*time.Second).Format(time.RFC3339), i%2 == 0))
	}
	for vt, expr := range map[string]string{"text": "'v' || (g % 5)", "num": "(g % 5)::float8", "ts": "'2030-01-01T00:00:00Z'::timestamptz + (g % 5) * interval '1 second'", "bool": "(g % 2 = 0)"} {
		fakeSideRows(t, e, vt, e.a.id, ids[fieldindex.ValueType(vt)], 40000, expr)
	}
	valueFor := func(vt fieldindex.ValueType) any {
		switch vt {
		case fieldindex.TypeText:
			return "v1"
		case fieldindex.TypeNum:
			return 1.0
		case fieldindex.TypeTS:
			return base.Add(time.Second)
		}
		return true
	}
	explain := func(stmt string, args ...any) string {
		t.Helper()
		var plan []string
		if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			rows, err := tx.Query(t.Context(), "EXPLAIN (COSTS OFF) "+stmt, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var l string
				if err := rows.Scan(&l); err != nil {
					return err
				}
				plan = append(plan, l)
			}
			return rows.Err()
		}); err != nil {
			t.Fatalf("explain %.60s: %v", stmt, err)
		}
		return strings.Join(plan, "\n")
	}
	seek := regexp.MustCompile(`Index Cond:.*record_id = r\.id`)
	for _, state := range []string{"as written", "after VACUUM"} {
		if state == "after VACUUM" {
			if _, err := e.super.Exec(t.Context(), "VACUUM (ANALYZE) rfv_text, rfv_num, rfv_ts, rfv_bool, records"); err != nil {
				t.Fatal(err)
			}
		}
		for _, vt := range fieldindex.ValueTypes() {
			ops := []string{"eq", "in"}
			if vt == fieldindex.TypeNum || vt == fieldindex.TypeTS {
				ops = append(ops, "lt", "lte", "gt", "gte")
			}
			for _, op := range ops {
				v := valueFor(vt)
				vals := []any{v}
				if op == "in" {
					vals = []any{v, v}
				}
				probe, args := server.ProbeTermForTest(vt, ids[vt], op, vals...)
				count := fmt.Sprintf("SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = '%s' AND %s LIMIT 1001) c", e.a.id, probe)
				plan := explain(count, args...)
				if !strings.Contains(plan, "SubPlan") || !seek.MatchString(plan) || strings.Contains(plan, "Hash") || strings.Contains(plan, "Merge") {
					t.Errorf("%s, %s %s: the probing count is not a per-record seek:\n%s", state, vt, op, plan)
				}
				pre, pargs := server.PreCountForTest(vt, e.a.id, ids[vt], op, vals...)
				plan = explain(pre, pargs...)
				table, _ := vt.SideTable()
				if strings.Contains(plan, "Seq Scan") || strings.Contains(plan, "Bitmap") || strings.Contains(plan, "Sort") ||
					!strings.Contains(plan, "Index Only Scan using "+table+"_value") && !strings.Contains(plan, "Index Scan using "+table+"_value") {
					t.Errorf("%s, %s %s: the pre-count is not an index scan on %s_value:\n%s", state, vt, op, table, plan)
				}
			}
		}
	}
}

// A2-12: a request pre-counts at most three fields, the first in field-id
// order; a fourth keeps today's SQL in the count, and the rows are the same.
func TestListCountProbePreCountsAtMostThreeFields(t *testing.T) {
	e, rec := newProbeEnv(t)
	e.createAssets(t, "admin-a")
	for _, f := range []struct{ key, vt string }{{"name", "text"}, {"qty", "num"}, {"due", "ts"}, {"vip", "bool"}} {
		e.flagField(t, e.a.id, "asset", f.key, f.vt, "ready", 100)
	}
	narrowUser(t, e)
	q := "filter[name][in]=alpha,gamma&filter[qty][gte]=1&filter[due][gte]=2026-01-01&filter[vip]=true"
	rec.set(1<<30, 30000)
	plain := e.mustList(t, "user-a", "asset", q).Total
	if pre, _ := rec.take(); len(pre) != 3 {
		t.Errorf("four fields: %d pre-counts, want 3", len(pre))
	}
	rec.set(1, 30000)
	if got := e.mustList(t, "user-a", "asset", q).Total; got != plain {
		t.Errorf("probing total %d, plain %d", got, plain)
	}
	if pre, probe := rec.take(); len(pre) != 3 || !slices.Equal(probe, []bool{true}) {
		t.Errorf("four broad fields: pre %v probe %v", pre, probe)
	}
}
