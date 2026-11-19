//go:build integration

// P7-06 (TICK-746, ADR-006 D6, D7): GET /api/v1/records/{entity}?filter[f]...
// on a data field through the real handler on real PostgreSQL. The tests set
// registry rows to ready directly (no dependency on the backfill workflow) and
// let the P7-02 trigger fill the side tables as the records are written.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// sideColumn is the record_field_coerce column of each value type.
var sideColumn = map[string]string{"text": "value_text", "num": "value_num", "ts": "value_ts", "bool": "value_bool"}

// flagField writes a registry row for field key of the entity type, as the
// publish and the backfill would, and, for a ready field, copies the values
// the type's existing records hold through record_field_coerce, the function
// the trigger and the backfill share. Records written afterwards are indexed
// by the trigger. It returns the field id.
func (e *etEnv) flagField(t *testing.T, tenant uuid.UUID, entity, key, valueType, state string, progress int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := e.super.Exec(t.Context(), `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		SELECT d.tenant_id, $2, d.id, $4, $5, $6, $7, 1 FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $3`,
		tenant, id, entity, key, valueType, state, progress); err != nil {
		t.Fatalf("flag %s.%s: %v", entity, key, err)
	}
	if state == "ready" {
		if _, err := e.super.Exec(t.Context(), fmt.Sprintf(`INSERT INTO rfv_%s (tenant_id, field_id, record_id, ord, value)
			SELECT r.tenant_id, $2, r.id, c.ord, c.%s FROM records r
			JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id AND d.key = $3,
			LATERAL record_field_coerce(r.data -> $4, $5) c WHERE r.tenant_id = $1`, valueType, sideColumn[valueType]),
			tenant, id, entity, key, valueType); err != nil {
			t.Fatalf("index %s.%s: %v", entity, key, err)
		}
	}
	return id
}

// assetTypeBody is recSchema plus one flagged-able field of each value type
// and an integer priority (a name the legacy allow-list knows for seed types).
func assetTypeBody(key, prefix string) string {
	return strings.Replace(recTypeBody(key, prefix), `"requester": {"type": "string", "format": "uuid"}`,
		`"requester": {"type": "string", "format": "uuid"},
    "name": {"type": "string", "maxLength": 100},
    "qty": {"type": "number"},
    "due": {"type": "string", "format": "date-time"},
    "vip": {"type": "boolean"},
    "scores": {"type": "array", "items": {"type": "number"}},
    "priority": {"type": "integer"}`, 1)
}

// filterEnv: tenant A and B each have an "asset" type.
func newFilterEnv(t *testing.T) *etEnv {
	t.Helper()
	e := newETEnv(t)
	for _, c := range []struct{ tok, key string }{{"admin-a", "fa"}, {"admin-b", "fb"}} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", c.tok, c.key, assetTypeBody("asset", "AST")); rec.Code != http.StatusCreated {
			t.Fatalf("seed asset = %d: %s", rec.Code, rec.Body)
		}
	}
	return e
}

type filterProblem struct {
	Code          string                           `json:"code"`
	IndexProgress *int                             `json:"index_progress"`
	Errors        []struct{ Pointer, Rule string } `json:"errors"`
	Raw           map[string]any                   `json:"-"`
}

func decodeFilterProblem(t *testing.T, rec *httptest.ResponseRecorder) filterProblem {
	t.Helper()
	var p filterProblem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("problem body: %v: %s", err, rec.Body)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p.Raw); err != nil {
		t.Fatal(err)
	}
	return p
}

// assetRow is what the recompute oracle reads: the values a record was
// created with.
type assetRow struct {
	name *string
	qty  *float64
	due  *string
	vip  *bool
	rec  recBody
}

func (e *etEnv) createAssets(t *testing.T, token string) []assetRow {
	t.Helper()
	s := func(v string) *string { return &v }
	f := func(v float64) *float64 { return &v }
	b := func(v bool) *bool { return &v }
	rows := []assetRow{
		{name: s("alpha"), qty: f(1), due: s("2026-03-01T10:00:00Z"), vip: b(true)},
		{name: s("beta"), qty: f(2.5), due: s("2026-03-02T00:00:00+05:30"), vip: b(false)},
		{name: s("gamma"), qty: f(5), due: s("2026-03-05T00:00:00+00:00"), vip: b(true)},
		{name: s("Alpha"), qty: f(5), due: s("2026-03-05T00:00:00Z"), vip: b(false)},
		{name: s("delta"), qty: f(-3), due: s("2025-12-31T23:59:59Z")},
		{name: s("")},
		{}, // none of the four fields
	}
	for i := range rows {
		data := map[string]any{"short_description": fmt.Sprintf("a%d", i)}
		if rows[i].name != nil {
			data["name"] = *rows[i].name
		}
		if rows[i].qty != nil {
			data["qty"] = *rows[i].qty
		}
		if rows[i].due != nil {
			data["due"] = *rows[i].due
		}
		if rows[i].vip != nil {
			data["vip"] = *rows[i].vip
		}
		raw, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		rows[i].rec = e.mustCreateRec(t, token, "asset", fmt.Sprintf("asset-%s-%d", token, i), string(raw))
	}
	return rows
}

func parseTS(t *testing.T, v string) time.Time {
	t.Helper()
	if d, err := time.Parse(time.DateOnly, v); err == nil {
		return d
	}
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func filterQuery(field, op string, vals ...string) string {
	enc := make([]string, len(vals))
	for i, v := range vals {
		enc[i] = url.QueryEscape(v)
	}
	switch op {
	case "eq":
		return "filter[" + field + "]=" + enc[0]
	case "in":
		return "filter[" + field + "][in]=" + strings.Join(enc, ",")
	}
	return "filter[" + field + "][" + op + "]=" + enc[0]
}

// covers: listRecords 200
//
// Every operator on each type it applies to returns exactly the rows a
// recompute from the created values gives, and total is that count.
func TestListFilterReadyOperatorsMatchTheRecomputeOracle(t *testing.T) {
	e := newFilterEnv(t)
	rows := e.createAssets(t, "admin-a")
	for _, f := range []struct{ key, vt string }{{"name", "text"}, {"qty", "num"}, {"due", "ts"}, {"vip", "bool"}} {
		e.flagField(t, e.a.id, "asset", f.key, f.vt, "ready", 100)
	}
	type tc struct {
		field, op string
		vals      []string
		match     func(r assetRow) bool
	}
	numCmp := func(op string, v float64) func(assetRow) bool {
		return func(r assetRow) bool {
			if r.qty == nil {
				return false
			}
			switch op {
			case "eq":
				return *r.qty == v
			case "lt":
				return *r.qty < v
			case "lte":
				return *r.qty <= v
			case "gt":
				return *r.qty > v
			}
			return *r.qty >= v
		}
	}
	tsCmp := func(op string, v string) func(assetRow) bool {
		return func(r assetRow) bool {
			if r.due == nil {
				return false
			}
			a, b := parseTS(t, *r.due), parseTS(t, v)
			switch op {
			case "eq":
				return a.Equal(b)
			case "lt":
				return a.Before(b)
			case "lte":
				return !a.After(b)
			case "gt":
				return a.After(b)
			}
			return !a.Before(b)
		}
	}
	cases := []tc{
		{"name", "eq", []string{"alpha"}, func(r assetRow) bool { return r.name != nil && *r.name == "alpha" }},
		{"name", "eq", []string{"ALPHA"}, func(r assetRow) bool { return false }}, // bytewise: case matters
		{"name", "eq", []string{""}, func(r assetRow) bool { return r.name != nil && *r.name == "" }},
		{"name", "in", []string{"alpha", "Alpha", "nope"}, func(r assetRow) bool { return r.name != nil && (*r.name == "alpha" || *r.name == "Alpha") }},
		{"vip", "eq", []string{"true"}, func(r assetRow) bool { return r.vip != nil && *r.vip }},
		{"vip", "eq", []string{"false"}, func(r assetRow) bool { return r.vip != nil && !*r.vip }},
		{"vip", "in", []string{"true", "false"}, func(r assetRow) bool { return r.vip != nil }},
		{"qty", "in", []string{"1", "5", "99"}, func(r assetRow) bool { return r.qty != nil && (*r.qty == 1 || *r.qty == 5) }},
		{"due", "in", []string{"2026-03-05", "2026-03-01T10:00:00Z"}, func(r assetRow) bool {
			return tsCmp("eq", "2026-03-05")(r) || tsCmp("eq", "2026-03-01T10:00:00Z")(r)
		}},
	}
	for _, op := range []string{"eq", "lt", "lte", "gt", "gte"} {
		for _, v := range []string{"5", "2.5", "-3", "0"} {
			var fv float64
			if _, err := fmt.Sscan(v, &fv); err != nil {
				t.Fatal(err)
			}
			cases = append(cases, tc{"qty", op, []string{v}, numCmp(op, fv)})
		}
		for _, v := range []string{"2026-03-05", "2026-03-01T10:00:00Z", "2026-03-01T15:30:00+05:30", "2026-03-01T04:30:00Z"} {
			cases = append(cases, tc{"due", op, []string{v}, tsCmp(op, v)})
		}
	}
	for _, c := range cases {
		q := filterQuery(c.field, c.op, c.vals...)
		var want []recBody
		for _, r := range rows {
			if c.match(r) {
				want = append(want, r.rec)
			}
		}
		p := e.mustList(t, "admin-a", "asset", q+"&limit=200")
		wantIDs(t, q, p.Items, want...)
		if p.Total != len(want) {
			t.Errorf("?%s: total %d, want %d", q, p.Total, len(want))
		}
	}
	// Two filters are ANDed: a range from two keys.
	p := e.mustList(t, "admin-a", "asset", "filter[qty][gte]=2&filter[qty][lt]=5")
	wantIDs(t, "range", p.Items, rows[1].rec)

	// A fixed-column filter and a data filter share one statement and one
	// placeholder numbering: a wrong offset would return the wrong rows with
	// no error. rows[0] (qty 1) is closed, rows[1] (qty 2.5) stays new.
	if rec := e.patch(t, "admin-a", "asset", rows[0].rec.ID, `W/"1"`, `{"state": "closed"}`); rec.Code != http.StatusOK {
		t.Fatalf("close = %d: %s", rec.Code, rec.Body)
	}
	for q, want := range map[string][]recBody{
		"filter[state]=closed&filter[qty][lte]=2.5":                               {rows[0].rec},
		"filter[qty][lte]=2.5&filter[state]=new":                                  {rows[1].rec, rows[4].rec},
		"filter[state][in]=new,closed&filter[name][in]=alpha,beta":                {rows[0].rec, rows[1].rec},
		"filter[state]=closed&filter[name]=beta":                                  nil,
		"filter[state]=new&filter[vip]=true&filter[qty][gt]=1&filter[name]=gamma": {rows[2].rec},
	} {
		p := e.mustList(t, "admin-a", "asset", q)
		wantIDs(t, q, p.Items, want...)
		if p.Total != len(want) {
			t.Errorf("?%s: total %d, want %d", q, p.Total, len(want))
		}
	}
}

// covers: listRecords 422
func TestListFilterBackfillingAnswers422WithOnlyAPercentage(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "qty", "num", "backfilling", 37)
	for _, q := range []string{"filter[qty]=1", "filter[qty][in]=1,2", "filter[qty][gt]=0", "filter[qty]=notanumber"} {
		rec, _ := e.list(t, "admin-a", "asset", q)
		listCode(t, rec, http.StatusUnprocessableEntity, "field_not_ready")
		p := decodeFilterProblem(t, rec)
		if p.IndexProgress == nil || *p.IndexProgress != 37 {
			t.Errorf("?%s: index_progress = %v, want 37", q, p.IndexProgress)
		}
		for k := range p.Raw {
			switch k {
			case "type", "title", "status", "code", "detail", "instance", "index_progress":
			default:
				t.Errorf("?%s: body member %q: field_not_ready says a percentage and nothing else about the field", q, k)
			}
		}
		if len(p.Errors) != 0 {
			t.Errorf("?%s: errors = %v, want none", q, p.Errors)
		}
	}
	// Another field of the type still answers by its own state.
	rec, _ := e.list(t, "admin-a", "asset", "filter[name]=alpha")
	listCode(t, rec, http.StatusBadRequest, "field_not_filterable")
}

// covers: listRecords 400
//
// The D6 order: masked, then not flagged, then backfilling, then ready.
func TestListFilterRefusalsFollowTheD6Order(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "vip", "bool", "removing", 100)
	// qty is masked from everyone but the admin.
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "asset",
			[]authz.Rule{acltest.AllowUsers(e.a.admin.PrincipalID, e.a.user.PrincipalID)},
			[]authz.Mask{{Field: "qty", Read: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}, Write: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	qtyID := e.flagField(t, e.a.id, "asset", "qty", "num", "backfilling", 10)

	// Masked wins over backfilling (and over everything below it).
	rec, _ := e.list(t, "user-a", "asset", "filter[qty]=1")
	listCode(t, rec, http.StatusBadRequest, "masked_field")
	if body := rec.Body.String(); strings.Contains(body, "index_progress") {
		t.Errorf("a masked field must not leak its progress: %s", body)
	}
	rec, _ = e.list(t, "user-a", "asset", "filter[qty]=1&filter[name]=x")
	listCode(t, rec, http.StatusBadRequest, "masked_field")
	// The caller the mask admits sees the registry's answer.
	rec, _ = e.list(t, "admin-a", "asset", "filter[qty]=1")
	listCode(t, rec, http.StatusUnprocessableEntity, "field_not_ready")

	// Not flagged, and flag gone (removing): 400 field_not_filterable with the pointer only.
	for _, q := range []string{"filter[name]=alpha", "filter[due][gt]=2026-01-01", "filter[vip]=true", "filter[short_description]=a1"} {
		rec, _ := e.list(t, "admin-a", "asset", q)
		listCode(t, rec, http.StatusBadRequest, "field_not_filterable")
		if p := decodeFilterProblem(t, rec); len(p.Errors) != 1 || !strings.HasPrefix(p.Errors[0].Pointer, "/filter/") {
			t.Errorf("?%s: errors = %v", q, p.Errors)
		}
	}
	// A property the schema does not have stays a validation error.
	rec, _ = e.list(t, "admin-a", "asset", "filter[nope]=1")
	listCode(t, rec, http.StatusBadRequest, "validation_failed")

	// Once ready, the same query is 200.
	if _, err := e.super.Exec(t.Context(), `UPDATE record_index_fields SET state = 'ready', progress = 100 WHERE tenant_id = $1 AND id = $2`, e.a.id, qtyID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.super.Exec(t.Context(), `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value)
		SELECT r.tenant_id, $2, r.id, c.ord, c.value_num FROM records r, LATERAL record_field_coerce(r.data -> 'qty', 'num') c WHERE r.tenant_id = $1`, e.a.id, qtyID); err != nil {
		t.Fatal(err)
	}
	if p := e.mustList(t, "admin-a", "asset", "filter[qty]=5"); p.Total != 2 {
		t.Errorf("ready qty=5: total %d, want 2", p.Total)
	}
}

// covers: listRecords 400
func TestListFilterValueErrorsAreValidationFailed(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	for _, f := range []struct{ key, vt string }{{"name", "text"}, {"qty", "num"}, {"due", "ts"}, {"vip", "bool"}} {
		e.flagField(t, e.a.id, "asset", f.key, f.vt, "ready", 100)
	}
	vals := func(n int) string { return strings.TrimSuffix(strings.Repeat("a,", n), ",") }
	nums := strings.TrimSuffix(strings.Repeat("1,", 101), ",")
	for name, q := range map[string]string{
		"101 values":              "filter[name][in]=" + vals(101),
		"101 numbers":             "filter[qty][in]=" + nums,
		"num given text":          "filter[qty]=abc",
		"num given hex":           "filter[qty]=0x10",
		"num out of range":        "filter[qty]=1e400",
		"in with one bad number":  "filter[qty][in]=1,x",
		"ts given text":           "filter[due]=yesterday",
		"ts without an offset":    "filter[due]=2026-03-01T10:00:00",
		"ts bad day":              "filter[due]=2026-13-40",
		"bool given yes":          "filter[vip]=yes",
		"text longer than 2000":   "filter[name]=" + strings.Repeat("a", 2001),
		"range on text":           "filter[name][gt]=a",
		"range on bool":           "filter[vip][lte]=true",
		"unknown operator":        "filter[qty][like]=1",
		"repeated key":            "filter[qty]=1&filter[qty]=2",
		"range on a fixed column": "filter[state][gt]=new",
	} {
		rec, _ := e.list(t, "admin-a", "asset", q)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s, want 400 validation_failed", name, rec.Code, rec.Body)
			continue
		}
		listCode(t, rec, http.StatusBadRequest, "validation_failed")
	}
	// 100 values is the most.
	if p := e.mustList(t, "admin-a", "asset", "filter[name][in]="+vals(100)); p.Total != 0 {
		t.Errorf("100 values: total %d, want 0", p.Total)
	}
}

// CI-6c: a value that only an ACL-denied record holds gives the same hits and
// total as a value no record holds: the row ACL is in the same statement.
func TestListFilterRowACLOracle(t *testing.T) {
	e := newFilterEnv(t)
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	e.flagField(t, e.a.id, "asset", "qty", "num", "ready", 100)
	e.publishACL(t, e.a.id, "asset",
		authz.Rule{ID: "requester", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}},
		authz.Rule{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}})
	user := e.a.user.PrincipalID
	secret := e.mustCreateRec(t, "admin-a", "asset", "o1", `{"short_description": "s", "name": "only-the-secret-has-this", "qty": 4242}`)
	mine := e.mustCreateRec(t, "admin-a", "asset", "o2", fmt.Sprintf(`{"short_description": "m", "name": "shared", "qty": 1, "requester": %q}`, user))
	e.mustCreateRec(t, "admin-a", "asset", "o3", `{"short_description": "x", "name": "shared", "qty": 2}`)
	for _, pair := range [][2]string{
		{"filter[name]=only-the-secret-has-this", "filter[name]=nobody-has-this"},
		{"filter[name][in]=only-the-secret-has-this,zzz", "filter[name][in]=yyy,zzz"},
		{"filter[qty]=4242", "filter[qty]=4243"},
		{"filter[qty][gt]=100", "filter[qty][gt]=100000"},
	} {
		a, b := e.mustList(t, "user-a", "asset", pair[0]), e.mustList(t, "user-a", "asset", pair[1])
		if len(a.Items) != 0 || len(b.Items) != 0 || a.Total != b.Total || a.Total != 0 {
			t.Errorf("%v: %d items/total %d vs %d items/total %d; an ACL-denied value must look absent", pair, len(a.Items), a.Total, len(b.Items), b.Total)
		}
		if ids(a.Items)[secret.ID] {
			t.Errorf("%v: the denied record is in the page", pair)
		}
	}
	// The requester still finds what it may read, and only that.
	p := e.mustList(t, "user-a", "asset", "filter[name]=shared")
	wantIDs(t, "own shared", p.Items, mine)
	if p.Total != 1 {
		t.Errorf("own shared: total %d, want 1", p.Total)
	}
	// The admin sees the secret's value.
	if p := e.mustList(t, "admin-a", "asset", "filter[qty]=4242"); p.Total != 1 {
		t.Errorf("admin qty=4242: total %d, want 1", p.Total)
	}
}

// Tenant A's filter never matches tenant B's rows, including a value only B has
// and a field id both tenants share.
func TestListFilterCrossTenant(t *testing.T) {
	e := newFilterEnv(t)
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)
	e.flagField(t, e.b.id, "asset", "name", "text", "ready", 100)
	a := e.mustCreateRec(t, "admin-a", "asset", "x1", `{"short_description": "a", "name": "both"}`)
	b := e.mustCreateRec(t, "admin-b", "asset", "x2", `{"short_description": "b", "name": "both"}`)
	e.mustCreateRec(t, "admin-b", "asset", "x3", `{"short_description": "b", "name": "only-b"}`)
	if p := e.mustList(t, "admin-a", "asset", "filter[name]=only-b"); p.Total != 0 || len(p.Items) != 0 {
		t.Errorf("tenant A found tenant B's value: total %d, %d items", p.Total, len(p.Items))
	}
	p := e.mustList(t, "admin-a", "asset", "filter[name]=both")
	wantIDs(t, "A", p.Items, a)
	p = e.mustList(t, "admin-b", "asset", "filter[name]=both")
	wantIDs(t, "B", p.Items, b)
	if p.Total != 1 {
		t.Errorf("B total %d, want 1", p.Total)
	}
}

// Invalid filters write nothing: no audit row and no outbox row.
func TestListFilterRefusalsWriteNothing(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	e.flagField(t, e.a.id, "asset", "qty", "num", "ready", 100)
	e.flagField(t, e.a.id, "asset", "due", "ts", "backfilling", 5)
	count := func() [2]int {
		var n [2]int
		for i, q := range []string{"SELECT count(*) FROM audit_log WHERE tenant_id = $1", "SELECT count(*) FROM outbox WHERE tenant_id = $1"} {
			if err := e.super.QueryRow(t.Context(), q, e.a.id).Scan(&n[i]); err != nil {
				t.Fatal(err)
			}
		}
		return n
	}
	before := count()
	for _, q := range []string{"filter[name]=x", "filter[due]=2026-01-01", "filter[qty]=abc", "filter[qty][in]=" + strings.Repeat("1,", 101) + "1", "filter[qty][like]=1", "filter[nope]=1"} {
		if rec, _ := e.list(t, "admin-a", "asset", q); rec.Code < 400 {
			t.Fatalf("?%s = %d, want a refusal", q, rec.Code)
		}
	}
	if after := count(); after != before {
		t.Errorf("audit/outbox rows %v -> %v: a refused filter wrote", before, after)
	}
}

// PLAN-007 D-7-6 and D-7-14: the legacy scan is the fixed allow-list keyed by
// (seed template type, field key), and replaces only step 2 of the D6 order.
func TestListFilterLegacyAllowList(t *testing.T) {
	e := newFilterEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "inc", incidentTypeBody()); rec.Code != http.StatusCreated {
		t.Fatalf("seed incident = %d: %s", rec.Code, rec.Body)
	}
	one := e.mustCreateRec(t, "admin-a", "incident", "i1", `{"short_description": "p1", "priority": 1}`)
	e.mustCreateRec(t, "admin-a", "incident", "i2", `{"short_description": "p2", "priority": 2}`)
	e.mustCreateRec(t, "admin-a", "asset", "t1", `{"short_description": "t", "priority": 1}`)

	// A tenant-defined type with its own priority never reaches the scan.
	rec, _ := e.list(t, "admin-a", "asset", "filter[priority]=1")
	listCode(t, rec, http.StatusBadRequest, "field_not_filterable")
	// An unflagged seed field on the list goes through the scan.
	p := e.mustList(t, "admin-a", "incident", "filter[priority]=1")
	wantIDs(t, "scan eq", p.Items, one)
	if p.Total != 1 {
		t.Errorf("scan total %d, want 1", p.Total)
	}
	if p := e.mustList(t, "admin-a", "incident", "filter[priority][in]=1,2"); p.Total != 2 {
		t.Errorf("scan in: total %d, want 2", p.Total)
	}
	// The scan takes equality and in only; another field of the seed type is not on the list.
	rec, _ = e.list(t, "admin-a", "incident", "filter[priority][gt]=0")
	listCode(t, rec, http.StatusBadRequest, "validation_failed")
	// A value of the wrong type is refused before the scan runs.
	for _, q := range []string{"filter[priority][in]=1,abc", "filter[priority]=abc"} {
		rec, _ = e.list(t, "admin-a", "incident", q)
		listCode(t, rec, http.StatusBadRequest, "validation_failed")
	}
	rec, _ = e.list(t, "admin-a", "incident", "filter[short_description]=p1")
	listCode(t, rec, http.StatusBadRequest, "field_not_filterable")

	// Masked still answers first.
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "incident",
			[]authz.Rule{acltest.AllowUsers(e.a.admin.PrincipalID, e.a.user.PrincipalID)},
			[]authz.Mask{{Field: "priority", Read: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}, Write: authz.Subject{Users: []uuid.UUID{e.a.admin.PrincipalID}}}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rec, _ = e.list(t, "user-a", "incident", "filter[priority]=1")
	listCode(t, rec, http.StatusBadRequest, "masked_field")

	// A listed field that is flagged follows the registry: backfilling is 422, never the scan.
	e.flagField(t, e.a.id, "incident", "priority", "num", "backfilling", 60)
	rec, _ = e.list(t, "admin-a", "incident", "filter[priority]=1")
	listCode(t, rec, http.StatusUnprocessableEntity, "field_not_ready")
	if p := decodeFilterProblem(t, rec); p.IndexProgress == nil || *p.IndexProgress != 60 {
		t.Errorf("index_progress = %v, want 60", p.IndexProgress)
	}
}

// CI-1: as nava_app under forced RLS, each operator on each type shows the
// side-table predicate as an Index Cond, so it is pushed below the policy
// (every comparison is leakproof, ADR-006 CI-2).
func TestListFilterPushdownIsAnIndexCond(t *testing.T) {
	e := newFilterEnv(t)
	fields := map[fieldindex.ValueType]string{fieldindex.TypeText: "name", fieldindex.TypeNum: "qty", fieldindex.TypeTS: "due", fieldindex.TypeBool: "vip"}
	ids := map[fieldindex.ValueType]uuid.UUID{}
	for _, vt := range fieldindex.ValueTypes() {
		ids[vt] = e.flagField(t, e.a.id, "asset", fields[vt], string(vt), "ready", 100)
	}
	// 20,000 values per field with no record behind them (replica role: no
	// foreign keys), so the value index is clearly cheaper than the primary
	// key's (tenant, field) prefix for the selective values probed below.
	base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := e.super.Exec(t.Context(), "SET session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = e.super.Exec(t.Context(), "SET session_replication_role = origin") }()
	for _, q := range []struct{ vt, expr string }{
		{"text", "'v' || g"}, {"num", "g::float8"}, {"ts", "$3::timestamptz + g * interval '1 second'"},
		{"bool", "(g = 1)"},
	} {
		sql := fmt.Sprintf(`INSERT INTO rfv_%s (tenant_id, field_id, record_id, ord, value)
			SELECT $1, $2, gen_random_uuid(), 0, %s FROM generate_series(1, 20000) g`, q.vt, q.expr)
		args := []any{e.a.id, ids[fieldindex.ValueType(q.vt)]}
		if q.vt == "ts" {
			args = append(args, base)
		}
		if _, err := e.super.Exec(t.Context(), sql, args...); err != nil {
			t.Fatal(err)
		}
		if _, err := e.super.Exec(t.Context(), "ANALYZE rfv_"+q.vt); err != nil {
			t.Fatal(err)
		}
	}
	// A value near the low end for eq, in, lt and lte, and near the high end
	// for gt and gte, so every probe is selective.
	valueFor := func(vt fieldindex.ValueType, op string) any {
		high := op == "gt" || op == "gte"
		switch vt {
		case fieldindex.TypeText:
			return "v5"
		case fieldindex.TypeNum:
			if high {
				return 19995.0
			}
			return 5.0
		case fieldindex.TypeTS:
			if high {
				return base.Add(19995 * time.Second)
			}
			return base.Add(5 * time.Second)
		}
		return true
	}
	indexCond := regexp.MustCompile(`Index Cond:.*\bvalue\b`)
	for _, vt := range fieldindex.ValueTypes() {
		id := ids[vt]
		ops := []string{"eq", "in"}
		if vt == fieldindex.TypeNum || vt == fieldindex.TypeTS {
			ops = append(ops, "lt", "lte", "gt", "gte")
		}
		for _, op := range ops {
			sql, args := server.ExistsTermForTest(vt, id, op, valueFor(vt, op))
			stmt := fmt.Sprintf("EXPLAIN (COSTS OFF) SELECT 1 FROM records r WHERE r.tenant_id = '%s' AND %s", e.a.id, sql)
			var plan []string
			if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
				// The join is driven from the side table, as it is for a selective
				// value on a large tenant; on these few rows the planner would
				// otherwise probe the primary key per record, which is a plan
				// choice, not a pushdown question.
				for _, g := range []string{"enable_seqscan", "enable_nestloop", "enable_mergejoin"} {
					if _, err := tx.Exec(t.Context(), "SET LOCAL "+g+" = off"); err != nil {
						return err
					}
				}
				rows, err := tx.Query(t.Context(), stmt, args...)
				if err != nil {
					return err
				}
				defer rows.Close()
				for rows.Next() {
					var line string
					if err := rows.Scan(&line); err != nil {
						return err
					}
					plan = append(plan, line)
				}
				return rows.Err()
			}); err != nil {
				t.Fatalf("%s %s: %v", vt, op, err)
			}
			if !slices.ContainsFunc(plan, indexCond.MatchString) {
				t.Errorf("%s %s: the value predicate is not an Index Cond:\n%s", vt, op, strings.Join(plan, "\n"))
			}
		}
	}
}

// A failed registry read is a 500, never a fall back to the legacy scan or an
// unchecked filter.
func TestListFilterRegistryReadFailureIsA500(t *testing.T) {
	e := newFilterEnv(t)
	e.createAssets(t, "admin-a")
	if _, err := e.super.Exec(t.Context(), "ALTER TABLE record_index_fields RENAME TO record_index_fields_gone"); err != nil {
		t.Fatal(err)
	}
	// A data filter needs the registry; the same list without one does not.
	rec, _ := e.list(t, "admin-a", "asset", "filter[name]=alpha")
	listCode(t, rec, http.StatusInternalServerError, "internal_error")
	if rec, _ := e.list(t, "admin-a", "asset", "filter[state]=new"); rec.Code != http.StatusOK {
		t.Errorf("a list with no data filter = %d, want 200", rec.Code)
	}
}

// covers: listRecords 200
//
// ADR-006 D7: all comparisons on one field sit inside one EXISTS, so on an
// array field ONE element must satisfy every bound. [1, 9] has an element >= 5
// and another < 3 but none in between, and is not in [2, 6).
func TestListFilterRangeOnArrayFieldNeedsOneElementForEveryBound(t *testing.T) {
	e := newFilterEnv(t)
	mk := func(i int, scores string) recBody {
		return e.mustCreateRec(t, "admin-a", "asset", fmt.Sprintf("arr-%d", i),
			fmt.Sprintf(`{"short_description": "s%d", "scores": %s}`, i, scores))
	}
	split, inside, outside := mk(0, "[1, 9]"), mk(1, "[1, 4]"), mk(2, "[7, 8]")
	e.flagField(t, e.a.id, "asset", "scores", "num", "ready", 100)
	p := e.mustList(t, "admin-a", "asset", "filter[scores][gte]=2&filter[scores][lt]=6")
	wantIDs(t, "array range", p.Items, inside)
	if p.Total != 1 {
		t.Errorf("total %d, want 1", p.Total)
	}
	_, _ = split, outside
}

// covers: listRecords 200, 400
//
// Two comparisons on one array field are one EXISTS, so one element must meet
// both: eq and in with different values match nothing, with a shared value they
// match; the same key twice is refused as repeated.
func TestListFilterTwoEqualitiesOnOneFieldArePinned(t *testing.T) {
	e := newFilterEnv(t)
	both := e.mustCreateRec(t, "admin-a", "asset", "eq-0", `{"short_description": "s0", "scores": [1, 2]}`)
	e.flagField(t, e.a.id, "asset", "scores", "num", "ready", 100)
	if p := e.mustList(t, "admin-a", "asset", "filter[scores]=1&filter[scores][in]=2,3"); len(p.Items) != 0 || p.Total != 0 {
		t.Errorf("eq 1 + in (2,3) on [1,2]: %d items, total %d, want none", len(p.Items), p.Total)
	}
	wantIDs(t, "eq+in shared", e.mustList(t, "admin-a", "asset", "filter[scores]=1&filter[scores][in]=1,3").Items, both)
	rec, _ := e.list(t, "admin-a", "asset", "filter[scores]=1&filter[scores]=2")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("repeated key = %d: %s", rec.Code, rec.Body)
	}
}
