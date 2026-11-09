package onboard

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// fakeACL records the tenants it was asked to publish for.
type fakeACL struct {
	mu        sync.Mutex
	calls     []uuid.UUID
	err       map[uuid.UUID]error
	published map[uuid.UUID]bool
	hook      func(uuid.UUID)
}

func (f *fakeACL) PublishDefaultIncident(_ context.Context, id uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, id)
	if f.hook != nil {
		f.hook(id)
	}
	if err := f.err[id]; err != nil {
		return false, err
	}
	return f.published[id], nil
}

func TestDefaultIncidentRulesAreTheContractsDefault(t *testing.T) {
	t.Parallel()
	admins, desk := uuid.New(), uuid.New()
	rules := DefaultIncidentRules(admins, []uuid.UUID{desk})
	want := []authz.Rule{
		{ID: "default-admins", Ops: []string{"read", "write"}, Who: authz.Subject{Groups: []uuid.UUID{admins}}},
		{ID: "default-service-desk", Ops: []string{"read", "write"}, Who: authz.Subject{Groups: []uuid.UUID{desk}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: "in", Caller: "my_groups"}}},
		{ID: "default-requester", Ops: []string{"read"}, Who: authz.Subject{Requester: true}},
		{ID: "default-requester-new", Ops: []string{"read", "write"}, Who: authz.Subject{Requester: true},
			Where: []authz.Cond{{Column: "state", Op: "eq", Values: []string{"new"}}}},
	}
	got, _ := json.Marshal(rules)
	exp, _ := json.Marshal(want)
	if string(got) != string(exp) {
		t.Fatalf("rules = %s\nwant    %s", got, exp)
	}
}

func TestDefaultIncidentRulesWithoutAServiceDeskGroupHaveNoDeadRule(t *testing.T) {
	t.Parallel()
	rules := DefaultIncidentRules(uuid.New(), nil)
	ids := make([]string, len(rules))
	for i, r := range rules {
		ids[i] = r.ID
	}
	if want := []string{"default-admins", "default-requester", "default-requester-new"}; !slices.Equal(ids, want) {
		t.Fatalf("rule ids = %v, want %v", ids, want)
	}
}

// The default is a policy the publish validation accepts, so an admin can
// download it with GET, edit it and PUT it back.
func TestDefaultIncidentRulesPassPublishValidation(t *testing.T) {
	t.Parallel()
	body, err := json.Marshal(map[string]any{"rules": DefaultIncidentRules(uuid.New(), []uuid.UUID{uuid.New()}), "masks": []authz.Mask{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ps := authz.ParsePolicy(body); len(ps) > 0 {
		t.Fatalf("the default ACL is refused by the publish validation: %v", ps)
	}
}

// The default decides as the contract says: admins everything, the service
// desk the incidents of its own groups, a requester its own (read always,
// write while new),
// everyone else nothing.
func TestDefaultIncidentRulesDecide(t *testing.T) {
	t.Parallel()
	adminsG, deskG, otherG := uuid.New(), uuid.New(), uuid.New()
	pol := &authz.Policy{Version: 1, Rules: DefaultIncidentRules(adminsG, []uuid.UUID{deskG})}
	admin := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{adminsG}}
	agent := identity.Principal{PrincipalID: uuid.New(), Groups: []uuid.UUID{deskG}}
	user := identity.Principal{PrincipalID: uuid.New()}
	rowOf := func(requester *uuid.UUID, group *uuid.UUID) authz.Row {
		return authz.Row{ID: uuid.New(), State: "new", RequesterID: requester, AssignmentGroupID: group}
	}
	mine, theirs := rowOf(&user.PrincipalID, &deskG), rowOf(nil, &otherG)
	raised := rowOf(&user.PrincipalID, nil) // the requester's own, not assigned yet
	closed := raised
	closed.State = "closed"
	for _, tc := range []struct {
		name  string
		p     identity.Principal
		op    string
		row   authz.Row
		allow bool
	}{
		{"admin reads", admin, authz.OpRead, theirs, true},
		{"admin writes", admin, authz.OpWrite, theirs, true},
		{"agent reads its group's", agent, authz.OpRead, mine, true},
		{"agent writes its group's", agent, authz.OpWrite, mine, true},
		{"agent reads another group's", agent, authz.OpRead, theirs, false},
		{"agent reads an unassigned one", agent, authz.OpRead, rowOf(nil, nil), false},
		{"requester reads its own", user, authz.OpRead, mine, true},
		{"requester writes its own while new", user, authz.OpWrite, mine, true},
		{"requester raises an unassigned incident", user, authz.OpWrite, raised, true},
		{"requester reads its own once closed", user, authz.OpRead, closed, true},
		{"requester does not write its own once closed", user, authz.OpWrite, closed, false},
		{"requester does not write another's", user, authz.OpWrite, theirs, false},
		{"agent does not write an unassigned one", agent, authz.OpWrite, raised, false},
		{"admin triages an unassigned one", admin, authz.OpWrite, raised, true},
		{"stranger reads nothing", user, authz.OpRead, theirs, false},
	} {
		if _, ok := authz.Allows(pol, tc.p, tc.op, tc.row); ok != tc.allow {
			t.Errorf("%s: allowed = %v, want %v", tc.name, ok, tc.allow)
		}
	}
}

func TestReconcileDefaultACLsCountsEachOutcomeAndKeepsGoing(t *testing.T) {
	t.Parallel()
	pub, same, bad, after := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f := &fakeACL{
		published: map[uuid.UUID]bool{pub: true, after: true},
		err:       map[uuid.UUID]error{bad: errors.New("boom")},
	}
	sum, err := ReconcileDefaultACLs(t.Context(), f, slog.New(slog.DiscardHandler), []uuid.UUID{pub, same, bad, after})
	if err == nil {
		t.Fatal("a failed tenant must make the job fail")
	}
	if want := (ACLReconcileSummary{Published: 2, Unchanged: 1, Failed: 1}); sum != want {
		t.Fatalf("summary = %+v, want %+v", sum, want)
	}
	if want := []uuid.UUID{pub, same, bad, after}; !slices.Equal(f.calls, want) {
		t.Fatalf("calls = %v, want every tenant in order, a failure not stopping the rest", f.calls)
	}
}

func TestReconcileDefaultACLsSucceedsWhenNothingFails(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	sum, err := ReconcileDefaultACLs(t.Context(), &fakeACL{}, slog.New(slog.DiscardHandler), []uuid.UUID{id})
	if err != nil || sum.Unchanged != 1 {
		t.Fatalf("summary %+v, error %v, want one unchanged and no error", sum, err)
	}
}

func newSvcWithACL(t *testing.T, acl ACLPublisher) (*Service, *fakeStore) {
	t.Helper()
	st := newStore()
	specs := NewSpecs()
	dir := &fakeDir{orgs: map[uuid.UUID]string{}, specs: specs}
	svc, err := New(Options{Store: st, Directory: dir, Finder: dir, Domains: st, ACL: acl, Specs: specs, Resolver: okResolver,
		Issuer: "https://kc/realms/nava", Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st
}

// Onboarding publishes the default ACL once, for the new tenant, after the
// admins group is recorded (the default names it) and before the tenant is
// active (a tenant never serves a request with no ACL).
func TestOnboardPublishesTheDefaultACLBeforeActivation(t *testing.T) {
	var st *fakeStore
	var activeAtCall, adminAtCall = true, false
	acl := &fakeACL{hook: func(id uuid.UUID) {
		activeAtCall = st.active[id]
		_, adminAtCall = st.adminGroups[id]
	}}
	var svc *Service
	svc, st = newSvcWithACL(t, acl)
	res, err := svc.Onboard(t.Context(), op1, "key-acl", []byte("h"), validReq())
	if err != nil {
		t.Fatal(err)
	}
	if want := []uuid.UUID{res.TenantID}; !slices.Equal(acl.calls, want) {
		t.Fatalf("publish calls = %v, want one for %v", acl.calls, res.TenantID)
	}
	if activeAtCall || !adminAtCall {
		t.Errorf("at the call: active = %v, admin group recorded = %v, want inactive and recorded", activeAtCall, adminAtCall)
	}
}

// A failed publish stops onboarding before the tenant is activated, and a
// retry publishes again (the publish itself is idempotent).
func TestOnboardStopsBeforeActivationWhenTheDefaultACLFails(t *testing.T) {
	tenant := TenantID("op-1", "key-acl-fail")
	acl := &fakeACL{err: map[uuid.UUID]error{tenant: errors.New("boom")}}
	svc, st := newSvcWithACL(t, acl)
	if _, err := svc.Onboard(t.Context(), op1, "key-acl-fail", []byte("h"), validReq()); err == nil {
		t.Fatal("onboarding succeeded though the default ACL failed")
	}
	if st.active[tenant] {
		t.Fatal("the tenant was activated with no ACL")
	}
	delete(acl.err, tenant)
	if _, err := svc.Onboard(t.Context(), op1, "key-acl-fail", []byte("h"), validReq()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !st.active[tenant] || len(acl.calls) != 2 {
		t.Fatalf("after the retry: active = %v, publish calls = %d, want active and 2", st.active[tenant], len(acl.calls))
	}
}

func TestDefaultIncidentRulesWithoutAnAdminsGroupHaveNoAdminsRule(t *testing.T) {
	t.Parallel()
	for _, r := range DefaultIncidentRules(uuid.Nil, []uuid.UUID{uuid.New()}) {
		if r.ID == RuleAdmins {
			t.Fatalf("an admins rule naming the nil group was published: %+v", r)
		}
		if slices.Contains(r.Who.Groups, uuid.Nil) {
			t.Fatalf("rule %s names the nil group", r.ID)
		}
	}
}
