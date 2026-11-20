package interp_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// fakePolicies is a PolicyTester returning one answer and recording its calls.
type fakePolicies struct {
	ok    bool
	err   error
	calls []interp.PolicyTest
	tens  []uuid.UUID
}

func (f *fakePolicies) Test(_ context.Context, tenant uuid.UUID, t interp.PolicyTest) (bool, error) {
	f.calls = append(f.calls, t)
	f.tens = append(f.tens, tenant)
	return f.ok, f.err
}

// policyVersion turns fixture-v1's is_high condition into a policy test.
func policyVersion(t *testing.T) (*interp.Version, fixture) {
	t.Helper()
	v, f := fixtureVersion(t)
	s := mutStep(t, v, "is_high")
	s.Condition = def.Condition{Op: def.OpWrite}
	s.Policy = &def.PolicyRef{EntityType: "ticket", Name: "assignee_edit"}
	s.Who = def.WhoAssignee
	return v, f
}

// TestNextStepPolicyTest: a condition step with a policy test goes to then or
// else on the tester's answer, and the tester gets the run's tenant, the
// record id and the step's literals only.
func TestNextStepPolicyTest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		policies *fakePolicies
		want     string // the step NextStep stops at
	}{
		{"granted takes then", &fakePolicies{ok: true}, "await_approval"},
		{"denied takes else", &fakePolicies{ok: false}, "done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, f := policyVersion(t)
			rec := uuid.New()
			a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{"priority": "high"}, Writer: noWrites{t}, Policies: tc.policies}
			d, err := a.NextStep(tenantCtx(), interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: rec, Step: f.id(t, "is_high")})
			if err != nil {
				t.Fatal(err)
			}
			if d.Step != f.id(t, tc.want) {
				t.Errorf("stopped at %v, want %s", d.Step, tc.want)
			}
			want := interp.PolicyTest{RecordID: rec, EntityType: "ticket", Policy: "assignee_edit", Who: "assignee", Op: "write"}
			if len(tc.policies.calls) != 1 || tc.policies.calls[0] != want || tc.policies.tens[0] != tenantA {
				t.Errorf("tester calls %+v tenants %v, want one %+v for %v", tc.policies.calls, tc.policies.tens, want, tenantA)
			}
		})
	}
}

// TestNextStepPolicyTestFailsClosed: no tester is a false test (never the
// then arm), and a tester error is not turned into a branch.
func TestNextStepPolicyTestFailsClosed(t *testing.T) {
	ref := func(t *testing.T, f fixture) interp.StepRef {
		return interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "is_high")}
	}
	t.Run("no tester", func(t *testing.T) {
		v, f := policyVersion(t)
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{"priority": "high"}, Writer: noWrites{t}}
		d, err := a.NextStep(tenantCtx(), ref(t, f))
		if err != nil || d.Step != f.id(t, "done") {
			t.Errorf("directive %+v err %v, want the else arm (done)", d, err)
		}
	})
	t.Run("record not found", func(t *testing.T) {
		v, f := policyVersion(t)
		p := &fakePolicies{err: fmt.Errorf("%w: record", interp.ErrNotFound)}
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: noWrites{t}, Policies: p}
		_, err := a.NextStep(tenantCtx(), ref(t, f))
		wantRejected(t, err, workerguard.RejectNotFound)
	})
	t.Run("forbidden", func(t *testing.T) {
		v, f := policyVersion(t)
		p := &fakePolicies{err: fmt.Errorf("%w: tenant", interp.ErrForbidden)}
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: noWrites{t}, Policies: p}
		_, err := a.NextStep(tenantCtx(), ref(t, f))
		wantRejected(t, err, workerguard.RejectForbidden)
	})
	t.Run("store failure is retryable and leaks no text", func(t *testing.T) {
		v, f := policyVersion(t)
		boom := errors.New("connection reset")
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: noWrites{t}, Policies: &fakePolicies{err: boom}}
		d, err := a.NextStep(tenantCtx(), ref(t, f))
		if !errors.Is(err, boom) || d.Step != uuid.Nil {
			t.Errorf("directive %+v err %v, want the store error and no directive", d, err)
		}
	})
}
