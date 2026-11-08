//go:build integration

// P3-31 (TICK-616): the record ACL through the real handler. An unreadable
// record is the missing record's 404, byte for byte, on GET, PATCH (with no
// If-Match, a stale one and the current one: never 428 or 412) and history;
// a write that moves a record out of the caller's write scope is 403.

package server_test

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// perRequestHeader reports a header whose value differs on every response by
// design: sameResponse checks it is present on both, not its value.
func perRequestHeader(k string) bool { return k == "X-Request-Id" || k == "Date" }

// sameResponse fails unless got and want have the same status, body bytes,
// header keys, and header values except perRequestHeader's.
func sameResponse(t *testing.T, name string, got, want *httptest.ResponseRecorder) {
	t.Helper()
	if got.Code != want.Code || got.Body.String() != want.Body.String() {
		t.Errorf("%s = %d %s; want the missing record's %d %s", name, got.Code, got.Body, want.Code, want.Body)
	}
	gh, wh := got.Header(), want.Header()
	if !slices.Equal(slices.Sorted(maps.Keys(gh)), slices.Sorted(maps.Keys(wh))) {
		t.Errorf("%s headers = %v; want %v", name, gh, wh)
	}
	for k := range wh {
		if perRequestHeader(k) {
			continue
		}
		if !slices.Equal(gh[k], wh[k]) {
			t.Errorf("%s header %s = %q; want %q", name, k, gh[k], wh[k])
		}
	}
}

// covers: getRecord 404
// covers: patchRecord 404
// covers: getRecordHistory 404
func TestUnreadableRecordIsTheMissingRecord(t *testing.T) {
	e := newRecEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "acl-c1", `{"short_description": "private"}`)
	// From now on only tenant A's admin may read or write tickets.
	e.publishACL(t, e.a.id, "ticket", acltest.AllowUsers(e.a.admin.PrincipalID))
	audits := e.auditCount(t, r.ID)
	missing := uuid.Must(uuid.NewV7()).String()

	if got := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID, "admin-a", "", ""); got.Code != http.StatusOK {
		t.Fatalf("control: the admin's GET = %d: %s", got.Code, got.Body)
	}
	sameResponse(t, "GET",
		e.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID, "user-a", "", ""),
		e.do(t, http.MethodGet, "/api/v1/records/ticket/"+missing, "user-a", "", ""))
	sameResponse(t, "history",
		e.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID+"/history", "user-a", "", ""),
		e.do(t, http.MethodGet, "/api/v1/records/ticket/"+missing+"/history", "user-a", "", ""))
	for name, ifMatch := range map[string]string{"no If-Match": "", "star": "*", "stale If-Match": `W/"9"`, "current If-Match": `W/"1"`} {
		sameResponse(t, "PATCH, "+name,
			e.patch(t, "user-a", "ticket", r.ID, ifMatch, `{"data": {"short_description": "x"}}`),
			e.patch(t, "user-a", "ticket", missing, ifMatch, `{"data": {"short_description": "x"}}`))
	}
	if got := e.auditCount(t, r.ID); got != audits {
		t.Errorf("audit entries = %d, want %d: a refused PATCH wrote", got, audits)
	}
	if got := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID, "admin-a", "", ""); got.Header().Get("ETag") != `W/"1"` {
		t.Errorf("record after the refusals: ETag %q, want W/\"1\"", got.Header().Get("ETag"))
	}
}

// covers: patchRecord 403
func TestWriteOutOfScopeIsForbidden(t *testing.T) {
	e := newRecEnv(t)
	desk, other := e.a.adminGroup, e.a.lookalikeGroup
	// The admin works the records of its own groups; the user leads both
	// groups.
	e.publishACL(t, e.a.id, "ticket",
		authz.Rule{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{desk}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		authz.Rule{ID: "lead", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{e.a.user.PrincipalID}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Values: []string{desk.String(), other.String()}}}},
	)
	r := e.mustCreateRec(t, "admin-a", "ticket", "acl-s1", fmt.Sprintf(`{"short_description": "s", "group": %q}`, desk))

	// Create out of scope: the admin is not in the other group.
	if rec := e.createRec(t, "admin-a", "ticket", "acl-s2", fmt.Sprintf(`{"short_description": "s", "group": %q}`, other)); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
		t.Errorf("create out of scope = %d: %s, want 403 forbidden", rec.Code, rec.Body)
	}
	// Move out of scope: before is the admin's, after is not.
	audits := e.auditCount(t, r.ID)
	move := fmt.Sprintf(`{"data": {"group": %q}}`, other)
	if rec := e.patch(t, "admin-a", "ticket", r.ID, `W/"1"`, move); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
		t.Fatalf("move out of scope = %d: %s, want 403 forbidden", rec.Code, rec.Body)
	}
	if got := e.auditCount(t, r.ID); got != audits {
		t.Fatalf("audit entries = %d, want %d after the refusal", got, audits)
	}
	// A write that stays in scope succeeds for the admin.
	if rec := e.patch(t, "admin-a", "ticket", r.ID, `W/"1"`, `{"data": {"short_description": "t"}}`); rec.Code != http.StatusOK {
		t.Fatalf("in-scope write = %d: %s", rec.Code, rec.Body)
	}
	// The same move by the lead succeeds: both states are in its scope.
	if rec := e.patch(t, "user-a", "ticket", r.ID, `W/"2"`, move); rec.Code != http.StatusOK {
		t.Fatalf("the lead's move = %d: %s", rec.Code, rec.Body)
	}
	// Now the record is outside the admin's read scope: the missing 404.
	if rec := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+r.ID, "admin-a", "", ""); rec.Code != http.StatusNotFound {
		t.Errorf("admin GET after the move = %d, want 404: %s", rec.Code, rec.Body)
	}
}

// covers: createRecord 403
// covers: getRecord 404
func TestNoPublishedACLDeniesEveryone(t *testing.T) {
	e := newETEnv(t)
	e.noTestACL = true
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "acl-t1", recTypeBody("bare", "BAR")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	for _, token := range []string{"admin-a", "user-a"} {
		if rec := e.createRec(t, token, "bare", "acl-n-"+token, `{"short_description": "s"}`); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
			t.Errorf("%s create with no ACL = %d: %s, want 403 forbidden (fail closed)", token, rec.Code, rec.Body)
		}
	}
	if n := e.recordCount(t, "a"); n != 0 {
		t.Fatalf("tenant A has %d records", n)
	}
	// A record a system writer made is still unreadable.
	id := e.seedSystemRecord(t, "bare")
	sameResponse(t, "GET with no ACL",
		e.do(t, http.MethodGet, "/api/v1/records/bare/"+id, "admin-a", "", ""),
		e.do(t, http.MethodGet, "/api/v1/records/bare/"+uuid.Must(uuid.NewV7()).String(), "admin-a", "", ""))
}

// seedSystemRecord creates a record of entity in tenant A through the
// pipeline as a system writer, which skips the authorizer (A1).
func (e *etEnv) seedSystemRecord(t *testing.T, entity string) string {
	t.Helper()
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, pipeline.PoolRunner{Pool: e.pool}, pipeline.SystemClock{})
	var res pipeline.Result
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		var err error
		res, err = p.Write(t.Context(), tx, pipeline.Command{Op: pipeline.OpCreate, EntityKey: entity,
			Data: map[string]any{"short_description": "by the system"}, Actor: identity.Principal{TenantID: e.a.id}, Source: pipeline.SourceSystem})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return res.RecordID.String()
}
