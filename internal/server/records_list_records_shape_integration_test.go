//go:build integration

package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// recordsShapeList is a list page of the route.
type recordsShapeList struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Total       int64   `json:"total"`
	TotalCapped bool    `json:"total_capped"`
	NextCursor  *string `json:"next_cursor"`
}

// TICK-1104 (ADR-010-A3) through the route, with R chosen by the real
// selector: the estimate 33,100 gives a cap of 10,031 units, and the policy's
// subtree has 10,057 (five digits each, so no group of a uuid equals them).
// The list answers 200 with the rows, the order and the capped total of the
// reference (the same request with no estimate, so A2's shapes) at 1,000, 1,001 and 1,002 readable records; the response and the
// log lines carry neither n, the cap, α, U_cert, U_hash nor the shape
// (A3-7 item 1). A probe that fails is a 500 that carries none of them either.
// covers: listRecords 200 500
func TestListRecordsChosenByTheSelector(t *testing.T) {
	e := newPatchEnv(t)
	ctx := t.Context()
	const estimate, capUnits, size = 33_100, 10_031, 10_057
	in := e.mustCreateRec(t, "user-a", "job", "tick1104-in", `{"short_description": "in"}`)
	out := e.mustCreateRec(t, "user-a", "job", "tick1104-out", `{"short_description": "out"}`)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := e.super.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	root, child := uuid.New(), uuid.New()
	exec(`INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, NULL, 'tick1104-root', 'root'), ($1, $3, $2, 'tick1104-child', 'child')`,
		e.a.id, root, child)
	rebuildOrgUnitClosure(t, e.super)
	doc := `{"format": 2, "conditions": [], "policies": [{"name": "unit", "ops": ["read"], "when": {"under": {"of": ["` + root.String() + `"]}}}]}`
	if got := e.access(t, http.MethodPut, accessPath, "admin-a", s31Key("tick1104-under"), `W/"1"`, `{"rules": `+doc+`}`); got.Code != http.StatusOK {
		t.Fatalf("publish under of the root = %d %s", got.Code, got.Body)
	}
	var def uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT entity_definition_id FROM records WHERE tenant_id = $1 AND id = $2`, e.a.id, in.ID).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// replica runs sql past the audit commit check.
	replica := func(sql string, args ...any) {
		t.Helper()
		tx, err := e.super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// The other units and their closure rows, written directly: the rebuild
	// is quadratic on a table with no statistics (TICK-1054), and the foreign
	// keys' checks keep the plan of the small tables they first ran on.
	replica(`INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
		SELECT $1, gen_random_uuid(), $2, 'tick1104-u' || g, 'u' || g FROM generate_series(1, $3::int) g`, e.a.id, child, size-2)
	replica(`INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
		SELECT u.tenant_id, a.id, u.id FROM org_units u CROSS JOIN LATERAL (VALUES (u.id), ($2::uuid), ($3::uuid)) a (id)
		WHERE u.tenant_id = $1 AND u.keycloak_group_id LIKE 'tick1104-u%'`, e.a.id, root, child)
	exec(`ANALYZE org_units`)
	exec(`ANALYZE org_unit_closure`)
	// add copies the in record n times under the child unit.
	copies := 0
	add := func(n int) {
		t.Helper()
		replica(`INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state,
				requester_id, assignee_id, assignment_group_id, org_unit_id, created_by, created_at, updated_at, data)
			SELECT r.tenant_id, gen_random_uuid(), r.entity_definition_id, r.version, r.schema_version, r.number || '-copy' || ($3::int + g), r.state,
				r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at - ($3::int + g) * interval '1 ms', r.data
			FROM records r, generate_series(1, $4::int) g WHERE r.tenant_id = $1 AND r.id = $2`, e.a.id, in.ID, copies, n)
		copies += n
	}
	replica(`UPDATE records SET org_unit_id = $3 WHERE tenant_id = $1 AND id = $2`, e.a.id, in.ID, child)
	setEstimate := func(n int64) {
		t.Helper()
		exec(`DELETE FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2`, e.a.id, def)
		if n >= 0 {
			exec(`INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at) VALUES ($1, $2, $3, now())`, e.a.id, def, n)
		}
	}
	// chosen reports whether the route's selector picks R for user-a now.
	user := identity.Principal{TenantID: e.a.id, PrincipalID: e.a.user.PrincipalID}
	chosen := func() (records bool) {
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
			records = strings.Contains(authz.ReadListShaped(bound, user, "r", 2, shapes).Where, ") IS TRUE)")
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return records
	}
	// leaks checks a response and the log lines it wrote for the values and
	// names A3-7 item 1 never exposes, and returns how often the word records
	// stands in them: the shape's label is also the route's own noun, so it is
	// compared with the reference's count instead.
	leaks := func(what string, rec *httptest.ResponseRecorder, logs string, words bool) int {
		t.Helper()
		var headers []string
		for h, vs := range rec.Header() {
			headers = append(headers, strings.ToLower(h))
			if !strings.EqualFold(h, "x-request-id") {
				headers = append(headers, vs...)
			}
		}
		n := 0
		for where, text := range map[string]string{"body": rec.Body.String(), "headers": strings.Join(headers, " "), "logs": logs} {
			toks := leakTokens(text)
			for _, v := range []int{size, capUnits, estimate, 10_111, 20_222, 33_366} {
				if toks[strconv.Itoa(v)] {
					t.Errorf("%s: the %s carry %d", what, where, v)
				}
			}
			bad := []string{"alpha", "α", "ucert", "u_cert", "uhash", "u_hash", "acl_s", "acl_z", "acl_p", "kstar", "c3", "live_records", "entity_record_estimates"}
			if words {
				bad = append(bad, leakWords()...)
			}
			for _, w := range bad {
				if toks[w] {
					t.Errorf("%s: the %s name %q", what, where, w)
				}
			}
			n += len(regexp.MustCompile(`\brecords\b`).FindAllString(text, -1))
		}
		return n
	}
	// list reads the first page, or all pages, checks each for leaks, and
	// returns the ids in order and the first page.
	list := func(what string, all bool) (ids []string, first recordsShapeList, records int) {
		t.Helper()
		path := "/api/v1/records/job?limit=100"
		for next := ""; ; {
			before := e.logs.String()
			rec := e.do(t, http.MethodGet, path+next, "user-a", "", "")
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: list = %d %s", what, rec.Code, rec.Body)
			}
			records += leaks(what, rec, strings.TrimPrefix(e.logs.String(), before), true)
			var l recordsShapeList
			if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
				t.Fatal(err)
			}
			if next == "" {
				first = l
			}
			for _, it := range l.Items {
				ids = append(ids, it.ID)
			}
			if l.NextCursor == nil || !all {
				return ids, first, records
			}
			next = "&cursor=" + *l.NextCursor
		}
	}

	add(999) // with the in record, 1,000 under the subtree
	for _, n := range []int{1000, 1001, 1002} {
		if n > 1000 {
			add(1)
		}
		exec(`ANALYZE records`)
		setEstimate(-1)
		if chosen() {
			t.Fatalf("%d records: no estimate, but the selector picks R", n)
		}
		refIDs, ref, refRecords := list("reference", n == 1002)
		setEstimate(estimate)
		if !chosen() {
			t.Fatalf("%d records: the selector does not pick R", n)
		}
		ids, got, records := list("R", n == 1002)
		rows := 100
		if n == 1002 {
			rows = n
		}
		if len(ids) != rows || slices.Contains(ids, out.ID) || !slices.Equal(ids, refIDs) {
			t.Errorf("%d records: R lists %d rows, the reference %d, in the same order %v", n, len(ids), len(refIDs), slices.Equal(ids, refIDs))
		}
		if got.Total != 1000 && n <= 1000 || got.Total != ref.Total || got.TotalCapped != (n > 1000) || ref.TotalCapped != (n > 1000) {
			t.Errorf("%d records: R total %d capped %v, the reference %d capped %v", n, got.Total, got.TotalCapped, ref.Total, ref.TotalCapped)
		}
		if records != refRecords {
			t.Errorf("%d records: the word records stands %d times in R's responses and logs, %d in the reference's", n, records, refRecords)
		}
	}

	// The probe fails: 500, no other shape, and the same body and log line
	// whichever probe it was.
	failed := func(what string) (body string, records int) {
		t.Helper()
		exec(`REVOKE SELECT ON org_unit_closure FROM nava_app`)
		defer exec(`GRANT SELECT ON org_unit_closure TO nava_app`)
		before := e.logs.String()
		rec := e.do(t, http.MethodGet, "/api/v1/records/job?limit=50", "user-a", "", "")
		logs := strings.TrimPrefix(e.logs.String(), before)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), in.ID) {
			t.Fatalf("%s: a failed probe = %d %s, want 500 with no row", what, rec.Code, rec.Body)
		}
		if !strings.Contains(logs, authz.ErrProbe.Error()) {
			t.Errorf("%s: the 500 is not the probe's: %s", what, logs)
		}
		n := leaks(what, rec, logs, false)
		return patchProblemCode(t, rec), n
	}
	setEstimate(-1)
	refCode, refRecords := failed("reference 500")
	setEstimate(estimate)
	code, records := failed("R 500")
	if code != refCode || records != refRecords {
		t.Errorf("a failed folded probe answers %q with the word records %d times, a failed P3 %q with %d", code, records, refCode, refRecords)
	}
	if !chosen() {
		t.Error("R is not chosen again after the grant")
	}
}
