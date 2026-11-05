//go:build integration

// P5-E5-4 (TICK-537; ADR-004-E5 D6, D7, N4, CI-E5-6): a record write that
// changes assignment_group_id to a group the tenant does not have live is
// refused with 422 assignment_group_removed and writes nothing; an unchanged
// value passes; reads return the group with removed: true when it is
// tombstoned.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// groupEnv is newRecEnv plus four groups and a random id:
//   - live: tenant A's live group;
//   - gone: tenant A's tombstoned group;
//   - foreignLive and foreignGone: tenant B's live and tombstoned groups;
//   - random: an id no tenant has.
//
// The groups have no members, so they can be tombstoned (D11).
type groupEnv struct {
	*etEnv
	live, gone, foreignLive, foreignGone, random string
}

func newGroupEnv(t *testing.T) *groupEnv {
	t.Helper()
	e := &groupEnv{etEnv: newRecEnv(t), random: uuid.NewString()}
	e.live = e.addGroup(t, e.a.id, "live", false)
	e.gone = e.addGroup(t, e.a.id, "gone", true)
	e.foreignLive = e.addGroup(t, e.b.id, "b-live", false)
	e.foreignGone = e.addGroup(t, e.b.id, "b-gone", true)
	return e
}

func (e *groupEnv) addGroup(t *testing.T, tenant uuid.UUID, name string, tombstoned bool) string {
	t.Helper()
	id := uuid.New()
	if _, err := e.super.Exec(t.Context(), "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)",
		tenant, id, name+"-"+id.String(), "/t/"+tenant.String()+"/"+name); err != nil {
		t.Fatal(err)
	}
	if tombstoned {
		e.tombstone(t, tenant, id.String())
	}
	return id.String()
}

func (e *groupEnv) tombstone(t *testing.T, tenant uuid.UUID, id string) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tenant, id); err != nil {
		t.Fatal(err)
	}
}

// writes counts what a write leaves behind in tenant A: records, audit_log,
// outbox and record_write_keys rows.
func (e *groupEnv) writes(t *testing.T) [4]int {
	t.Helper()
	var n [4]int
	for i, table := range []string{"records", "audit_log", "outbox", "record_write_keys"} {
		if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", e.a.id).Scan(&n[i]); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
	}
	return n
}

// rowState is a record's version and xmin: an unchanged row keeps both.
func (e *groupEnv) rowState(t *testing.T, id string) string {
	t.Helper()
	var s string
	if err := e.super.QueryRow(t.Context(), "SELECT version || ':' || xmin::text FROM records WHERE tenant_id = $1 AND id = $2", e.a.id, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// patchWithKey sends a merge patch with the given Idempotency-Key.
func (e *groupEnv) patchWithKey(t *testing.T, id, ifMatch, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/records/ticket/"+id, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer user-a")
	req.Header.Set("Content-Type", "application/merge-patch+json")
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("If-Match", ifMatch)
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

var requestInstance = regexp.MustCompile(`"instance":"urn:nava:request:[^"]*"`)

// sansRequestID is the raw response body with only the request id blanked:
// two bodies that differ anywhere else differ here.
func sansRequestID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := rec.Body.String()
	if !requestInstance.MatchString(body) {
		t.Fatalf("no request instance in %s", body)
	}
	return requestInstance.ReplaceAllString(body, `"instance":"urn:nava:request:-"`)
}

func groupData(group string) string {
	return fmt.Sprintf(`{"short_description": "x", "group": %q}`, group)
}

// wantGroupRemoved checks a 422 assignment_group_removed on /data/group that
// names none of the ids.
func wantGroupRemoved(t *testing.T, rec *httptest.ResponseRecorder, ids ...string) {
	t.Helper()
	p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.AssignmentGroupRemoved)
	if got := pointers(p); len(got) != 1 || got[0] != "/data/group" {
		t.Errorf("pointers = %v, want [/data/group]", got)
	}
	for _, leak := range append([]string{"fkey", "constraint", "violates", "foreign key", "deleted_at", "tombstone"}, ids...) {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("the 422 body shows %q: %s", leak, rec.Body)
		}
	}
}

// Scenario 1: a create naming a tombstoned group is refused and writes
// nothing.
func TestCreateRecordWithRemovedGroupIs422AndWritesNothing(t *testing.T) {
	e := newGroupEnv(t)
	before := e.writes(t)
	rec := e.createRec(t, "user-a", "ticket", "gone-1", groupData(e.gone))
	wantGroupRemoved(t, rec, e.gone)
	if after := e.writes(t); after != before {
		t.Errorf("records, audit_log, outbox, record_write_keys = %v, want %v: a refused create writes nothing", after, before)
	}
	// A live group of the tenant is accepted.
	e.mustCreateRec(t, "user-a", "ticket", "live-1", groupData(e.live))
}

// Scenario 4 (CI-E5-6, N4), create: another tenant's live and tombstoned
// group and a random id give the body of scenario 1, byte for byte but the
// request id, and write nothing.
func TestCreateRecordForeignOrUnknownGroupIsTheRemovedGroup422(t *testing.T) {
	e := newGroupEnv(t)
	want := sansRequestID(t, e.createRec(t, "user-a", "ticket", "c-gone", groupData(e.gone)))
	before := e.writes(t)
	for name, id := range map[string]string{"foreign-live": e.foreignLive, "foreign-tombstoned": e.foreignGone, "random": e.random} {
		rec := e.createRec(t, "user-a", "ticket", "c-"+name, groupData(id))
		wantGroupRemoved(t, rec, id)
		if got := sansRequestID(t, rec); got != want {
			t.Errorf("%s: body differs from the tombstoned group's:\n got %s\nwant %s", name, got, want)
		}
	}
	if after := e.writes(t); after != before {
		t.Errorf("writes = %v, want %v", after, before)
	}
}

// Scenarios 2 and 4, update: changing the group to a tombstoned, foreign or
// unknown group is the same 422, the record is unchanged and nothing is
// audited.
func TestPatchRecordToRemovedForeignOrUnknownGroupIs422(t *testing.T) {
	e := newGroupEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "u-seed", groupData(e.live))
	row := e.rowState(t, r.ID)
	before := e.writes(t)
	var want string
	for i, c := range []struct{ name, id string }{
		{"tombstoned", e.gone}, {"foreign live", e.foreignLive}, {"foreign tombstoned", e.foreignGone}, {"random", e.random},
	} {
		rec := e.patchWithKey(t, r.ID, `W/"1"`, fmt.Sprintf("u-%d", i), fmt.Sprintf(`{"data": {"group": %q}}`, c.id))
		wantGroupRemoved(t, rec, c.id)
		got := sansRequestID(t, rec)
		if i == 0 {
			want = got
		} else if got != want {
			t.Errorf("%s: body differs from the tombstoned group's:\n got %s\nwant %s", c.name, got, want)
		}
	}
	if got := e.rowState(t, r.ID); got != row {
		t.Errorf("record version:xmin = %s, want %s: a refused update leaves the row alone", got, row)
	}
	if after := e.writes(t); after != before {
		t.Errorf("writes = %v, want %v", after, before)
	}
	// The create's and the update's 422 are the same body.
	created := sansRequestID(t, e.createRec(t, "user-a", "ticket", "u-create", groupData(e.gone)))
	if created != want {
		t.Errorf("update 422 differs from the create 422:\n got %s\nwant %s", want, created)
	}
}

// Scenarios 3 and 6: an update that leaves a now-tombstoned group unchanged
// passes and writes one audit entry; its replay returns the same response and
// writes nothing more. A replayed refused create is refused again.
func TestPatchKeepingRemovedGroupPassesAndReplays(t *testing.T) {
	e := newGroupEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "k-seed", groupData(e.live))
	e.tombstone(t, e.a.id, e.live)
	audits := e.auditCount(t, r.ID)

	body := `{"data": {"short_description": "y"}}`
	first := e.patchWithKey(t, r.ID, `W/"1"`, "k-1", body)
	if first.Code != http.StatusOK {
		t.Fatalf("patch keeping the removed group = %d: %s", first.Code, first.Body)
	}
	var b recBody
	if err := json.Unmarshal(first.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Version != 2 || b.Data["group"] != e.live || b.Data["short_description"] != "y" {
		t.Errorf("body = %+v, want version 2 with the group kept", b)
	}
	if n := e.auditCount(t, r.ID); n != audits+1 {
		t.Errorf("audit entries = %d, want %d", n, audits+1)
	}

	replay := e.patchWithKey(t, r.ID, `W/"1"`, "k-1", body)
	if replay.Code != first.Code || replay.Body.String() != first.Body.String() || replay.Header().Get("ETag") != first.Header().Get("ETag") {
		t.Errorf("replay = %d %s, want %d %s", replay.Code, replay.Body, first.Code, first.Body)
	}
	if n := e.auditCount(t, r.ID); n != audits+1 {
		t.Errorf("audit entries after replay = %d, want %d", n, audits+1)
	}

	// Naming the same group again is no change either.
	same := e.patchWithKey(t, r.ID, `W/"2"`, "k-2", fmt.Sprintf(`{"data": {"group": %q}}`, strings.ToUpper(e.live)))
	if same.Code != http.StatusOK {
		t.Errorf("patch naming the unchanged group = %d: %s", same.Code, same.Body)
	}

	before := e.writes(t)
	refused := e.createRec(t, "user-a", "ticket", "k-gone", groupData(e.gone))
	wantGroupRemoved(t, refused, e.gone)
	again := e.createRec(t, "user-a", "ticket", "k-gone", groupData(e.gone))
	wantGroupRemoved(t, again, e.gone)
	if after := e.writes(t); after != before {
		t.Errorf("writes = %v, want %v: a replayed refused create writes nothing", after, before)
	}
}

// Scenario 5 (D7): a read returns the group with removed: true once it is
// tombstoned, removed: false while it is live, and no group when there is
// none.
func TestGetRecordShowsRemovedGroup(t *testing.T) {
	e := newGroupEnv(t)
	withGroup := e.mustCreateRec(t, "user-a", "ticket", "g-1", groupData(e.live))
	without := e.mustCreateRec(t, "user-a", "ticket", "g-2", `{"short_description": "x"}`)

	read := func(id string) map[string]any {
		t.Helper()
		rec := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+id, "user-a", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("GET = %d: %s", rec.Code, rec.Body)
		}
		var m map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	want := func(m map[string]any, removed bool) {
		t.Helper()
		g, _ := m["assignment_group"].(map[string]any)
		if g["id"] != e.live || g["removed"] != removed {
			t.Errorf("assignment_group = %v, want {id: %s, removed: %v}", m["assignment_group"], e.live, removed)
		}
		if d, _ := m["data"].(map[string]any); d["group"] != e.live {
			t.Errorf("data.group = %v, want %s: nothing clears the field", d["group"], e.live)
		}
	}
	want(read(withGroup.ID), false)
	if m := read(without.ID); m["assignment_group"] != nil {
		t.Errorf("assignment_group = %v, want absent for a record without a group", m["assignment_group"])
	}
	e.tombstone(t, e.a.id, e.live)
	want(read(withGroup.ID), true)
}
