//go:build integration

package server_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-1036 acceptance test 9 and the TICK-1070 security condition (ADR-010-A2
// A2-6 requirement 2): a list response under each page shape, C3, W0 and
// A2-5a's rare-type W0, carries no header or body field that names the shape,
// K*, the estimate N̂_e or the probe count, and the handler's log lines carry
// none of them either. The response's fields and headers are the same set
// under every shape; only the rows a caller may read differ, never by shape.
// covers: listRecords 200
func TestListResponseNamesNoShapeEstimateOrProbe(t *testing.T) {
	e := newPatchEnv(t)
	ctx := t.Context()
	in := e.mustCreateRec(t, "user-a", "job", "tick1036-in", `{"short_description": "in"}`)
	ouA, ouA1 := uuid.New(), uuid.New()
	for _, u := range []struct {
		id     uuid.UUID
		parent any
		key    string
	}{{ouA, nil, "tick1036-ou-a"}, {ouA1, ouA, "tick1036-ou-a1"}} {
		if _, err := e.super.Exec(ctx, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, $4)`,
			e.a.id, u.id, u.parent, u.key); err != nil {
			t.Fatal(err)
		}
	}
	rebuildOrgUnitClosure(t, e.super)
	doc := `{"format": 2, "conditions": [], "policies": [{"name": "unit", "ops": ["read"], "when": {"under": {"of": ["` + ouA.String() + `"]}}}]}`
	if got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("tick1036-under"), `W/"1"`, `{"rules": `+doc+`}`); got.Code != http.StatusOK {
		t.Fatalf("publish under of A = %d %s", got.Code, got.Body)
	}
	var def uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT entity_definition_id FROM records WHERE tenant_id = $1 AND id = $2`, e.a.id, in.ID).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// add puts n copies of the in record under A1, past the audit commit
	// check (replica role); n = 0 moves in itself under A1.
	add := func(n int) {
		t.Helper()
		tx, err := e.super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, e.a.id, in.ID, ouA1); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state,
					requester_id, assignee_id, assignment_group_id, org_unit_id, created_by, created_at, updated_at, data)
				SELECT r.tenant_id, gen_random_uuid(), r.entity_definition_id, r.version, r.schema_version, r.number || '-copy' || g, r.state,
					r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at - g * interval '1 ms', r.data
				FROM records r, generate_series(1, $3::int) g WHERE r.tenant_id = $1 AND r.id = $2`, e.a.id, in.ID, n); err != nil {
				t.Fatal(err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	setEstimate := func(n int64) {
		t.Helper()
		if _, err := e.super.Exec(ctx, `DELETE FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2`, e.a.id, def); err != nil {
			t.Fatal(err)
		}
		if n < 0 {
			return
		}
		if _, err := e.super.Exec(ctx, `INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at) VALUES ($1, $2, $3, now())`,
			e.a.id, def, n); err != nil {
			t.Fatal(err)
		}
	}
	// shapeNow is what the route will choose for user-a at limit 50 (page
	// 51), read the same way in its own transaction.
	user := identity.Principal{TenantID: e.a.id, PrincipalID: e.a.user.PrincipalID}
	shapeNow := func() (c3, driven bool) {
		t.Helper()
		if err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
			d, err := authz.Decode([]byte(doc))
			if err != nil {
				return err
			}
			bound, err := authz.ResolveUnder(ctx, tx, &authz.Policy{Version: 1, EntityType: "job", Document: d}, user)
			if err != nil {
				return err
			}
			shapes, err := authz.ChooseListShapes(ctx, tx, bound, user, def, 51)
			if err != nil {
				return err
			}
			read := authz.ReadListShaped(bound, user, "r", 2, shapes)
			c3 = slices.Contains(read.Materialize, true)
			driven = slices.ContainsFunc(read.Drivers, func(d authz.RegionDriver) bool { return d.From != "" })
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return c3, driven
	}

	// The estimates are distinctive numbers, so their text can be looked for.
	// Every step holds 1,002 records under A, past the CC cap, so every
	// response carries total_capped and the field sets compare across shapes.
	const big, rare = int64(98765431), int64(931)
	add(1001)
	// The fixture itself must not hold a banned token, or the body checks
	// below fail on the fixture and not on a leak (a '-c' || g suffix made
	// record 3's number end in the token c3).
	var numbers []string
	if err := e.super.QueryRow(ctx, `SELECT array_agg(number) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, e.a.id, def).Scan(&numbers); err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 1002 {
		t.Fatalf("the fixture holds %d records, want 1,002", len(numbers))
	}
	for _, word := range leakWords() {
		if leakTokens(strings.Join(numbers, " "))[word] {
			t.Fatalf("a fixture record number holds the banned token %q", word)
		}
	}
	type seen struct {
		headers []string
		keys    []string
		item    []string
		capped  bool
	}
	var first *seen
	for _, step := range []struct {
		name     string
		estimate int64
		c3       bool
		driven   bool
	}{
		{"W0: m = 1,002 > K* = 1,000, no estimate", -1, false, true},
		{"C3: m = 1,002 <= K* = 20,000, a large estimate", big, true, true},
		{"A2-5a: a rare estimate", rare, false, false},
	} {
		setEstimate(step.estimate)
		if c3, driven := shapeNow(); c3 != step.c3 || driven != step.driven {
			t.Fatalf("%s: the route would choose C3 = %v, driven = %v; want %v, %v", step.name, c3, driven, step.c3, step.driven)
		}
		before := e.logs.String()
		rec := e.do(t, http.MethodGet, "/api/v1/records/job?limit=50", "user-a", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: list = %d %s", step.name, rec.Code, rec.Body)
		}
		logs := strings.TrimPrefix(e.logs.String(), before)
		var top map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &top); err != nil {
			t.Fatal(err)
		}
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(top["items"], &items); err != nil || len(items) == 0 {
			t.Fatalf("%s: items %v, %v", step.name, items, err)
		}
		s := seen{keys: slices.Sorted(maps.Keys(top)), item: slices.Sorted(maps.Keys(items[0]))}
		_ = json.Unmarshal(top["total_capped"], &s.capped)
		if !s.capped {
			t.Errorf("%s: total_capped is not set for 1,002 readable records", step.name)
		}
		var values []string
		for h, vs := range rec.Header() {
			lh := strings.ToLower(h)
			s.headers = append(s.headers, lh)
			for _, bad := range []string{"server-timing", "shape", "probe", "estimate", "kstar", "k-star", "debug"} {
				if strings.Contains(lh, bad) {
					t.Errorf("%s: header %q", step.name, h)
				}
			}
			if lh != "x-request-id" { // a random id: any short token turns up in it by chance
				values = append(values, vs...)
			}
		}
		slices.Sort(s.headers)
		for _, k := range s.keys {
			if !slices.Contains([]string{"items", "next_cursor", "total", "total_capped"}, k) {
				t.Errorf("%s: body field %q", step.name, k)
			}
		}
		// Whole tokens only, so an id or a timestamp that merely contains
		// "c3" or the digits does not count. The rare estimate (3 digits)
		// is looked for in the headers and logs only: a timestamp's
		// fraction can be the same token by chance.
		for _, text := range []struct {
			where  string
			s      string
			values []int64
		}{{"body", rec.Body.String(), []int64{big}}, {"headers", strings.Join(values, " "), []int64{big, rare}}, {"logs", logs, []int64{big, rare}}} {
			toks := leakTokens(text.s)
			for _, v := range text.values {
				if toks[strconv.FormatInt(v, 10)] {
					t.Errorf("%s: the %s carry the estimate %d", step.name, text.where, v)
				}
			}
			for _, word := range leakWords() {
				if toks[word] {
					t.Errorf("%s: the %s name %q", step.name, text.where, word)
				}
			}
		}
		if first == nil {
			first = &s
		} else if !slices.Equal(s.headers, first.headers) || !slices.Equal(s.keys, first.keys) || !slices.Equal(s.item, first.item) || s.capped != first.capped {
			t.Errorf("%s: the response's fields or headers differ by shape: %+v, first %+v", step.name, s, *first)
		}
	}
}

// leakWords are the tokens that would name the shape, K*, the estimate or
// the probe in a response or a log line.
func leakWords() []string {
	return []string{"c3", "k*", "kstar", "probe", "shape", "materialized", "live_records", "entity_record_estimates", "acl_p"}
}

// leakTokens is the set of s's lower-case tokens: runs of letters, digits,
// '_' and '*'.
func leakTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range regexp.MustCompile(`[A-Za-z0-9_*]+`).FindAllString(strings.ToLower(s), -1) {
		out[tok] = true
	}
	return out
}
