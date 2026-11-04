//go:build integration

// Security review of P5-E5-4 (TICK-537; ADR-004-E5 D6, N4, CI-E5-6):
// adversarial equal-422 checks. A tenant A user holding a valid token names
// tenant A's tombstoned group, tenant B's live group, tenant B's tombstoned
// group and an id no tenant has, on create and on update. Status, headers and
// body must be the same in all eight responses except for the request id, and
// the lookup that decides the answer must do the same work for every
// cross-tenant and unknown id, so timing does not show whether B's group
// exists.

package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var requestURN = regexp.MustCompile(`urn:nava:request:([^"]*)`)

// wireShape is the status, every header and the body of a response, with the
// request id blanked wherever it appears. Two responses with the same
// wireShape differ only in the request id.
func wireShape(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := rec.Body.String()
	m := requestURN.FindStringSubmatch(body)
	if m == nil || m[1] == "" {
		t.Fatalf("no request id in %s", body)
	}
	rid := m[1]
	blank := func(s string) string { return strings.ReplaceAll(s, rid, "-") }
	var hs []string
	for k, vs := range rec.Header() {
		hs = append(hs, k+": "+blank(strings.Join(vs, ", ")))
	}
	slices.Sort(hs)
	return fmt.Sprintf("%d\n%s\n\n%s", rec.Code, strings.Join(hs, "\n"), blank(body))
}

// Create and update, four ids each: the eight responses differ only in the
// request id.
func TestSecGroupRemoved422SameStatusHeadersAndBody(t *testing.T) {
	e := newGroupEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "sec-seed", groupData(e.live))
	cases := []struct{ name, id string }{
		{"tombstoned", e.gone}, {"foreign live", e.foreignLive}, {"foreign tombstoned", e.foreignGone}, {"random", e.random},
	}
	var want, wantFrom string
	check := func(label string, rec *httptest.ResponseRecorder) {
		t.Helper()
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s = %d, want 422: %s", label, rec.Code, rec.Body)
		}
		got := wireShape(t, rec)
		if want == "" {
			want, wantFrom = got, label
			return
		}
		if got != want {
			t.Errorf("%s differs from %s:\n got %s\nwant %s", label, wantFrom, got, want)
		}
	}
	for i, c := range cases {
		check("create/"+c.name, e.createRec(t, "user-a", "ticket", fmt.Sprintf("sec-c-%d", i), groupData(c.id)))
		check("update/"+c.name, e.patchWithKey(t, r.ID, `W/"1"`, fmt.Sprintf("sec-u-%d", i), fmt.Sprintf(`{"data": {"group": %q}}`, c.id)))
	}
}

// The lookup that decides the 422 (store.GroupLive, in tenant A's RLS
// transaction) reads false for B's live group, B's tombstoned group and an
// unknown id, and runs the same plan touching the same rows for all three: the
// (tenant_id, id) primary key never reaches B's row, so no heap fetch or
// filter step tells them apart.
func TestSecGroupLiveLookupIsTheSameForForeignAndUnknown(t *testing.T) {
	e := newGroupEnv(t)
	var want, wantFrom string
	for _, c := range []struct{ name, id string }{
		{"foreign live", e.foreignLive}, {"foreign tombstoned", e.foreignGone}, {"random", e.random},
	} {
		err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			live, err := store.GroupLive(t.Context(), tx, e.a.id, uuid.MustParse(c.id))
			if err != nil {
				return err
			}
			if live {
				t.Errorf("%s: GroupLive = true under tenant A", c.name)
			}
			rows, err := tx.Query(t.Context(),
				`EXPLAIN (ANALYZE, COSTS OFF, TIMING OFF, SUMMARY OFF, BUFFERS OFF)
				 SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)`,
				e.a.id, uuid.MustParse(c.id))
			if err != nil {
				return err
			}
			var lines []string
			for rows.Next() {
				var l string
				if err := rows.Scan(&l); err != nil {
					return err
				}
				lines = append(lines, l)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			// Only the inlined id may differ.
			plan := strings.ReplaceAll(strings.Join(lines, "\n"), c.id, "<id>")
			if strings.Contains(plan, "Rows Removed by Filter: 1") {
				t.Errorf("%s: the lookup reached the row and filtered it:\n%s", c.name, plan)
			}
			if want == "" {
				want, wantFrom = plan, c.name
			} else if plan != want {
				t.Errorf("%s: plan differs from %s:\n got %s\nwant %s", c.name, wantFrom, plan, want)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
