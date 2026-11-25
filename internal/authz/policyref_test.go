package authz

import (
	"testing"

	"github.com/google/uuid"
)

// NamedPolicyAllows (ADR-010 D9, TICK-957): one named policy, one principal,
// one op; every refusal is false.
func TestNamedPolicyAllows(t *testing.T) {
	pol := ptDoc(t, `{"format":2,"conditions":[],"policies":[
		{"name":"mine","ops":["read","write"],"when":`+isRequester()+`},
		{"name":"read_only","ops":["read"],"when":`+isRequester()+`},
		{"name":"broken","ops":["read"],"when":{"ref":"nope"}},
		{"name":"others","ops":["read"],"when":`+inMyGroup()+`}]}`)
	row := Row{State: "open", RequesterID: ptRef(ptCaller())}
	for _, c := range []struct {
		name, policy, op string
		row              Row
		want             bool
	}{
		{"granted", "mine", OpRead, row, true},
		{"granted for write", "mine", OpWrite, row, true},
		{"denied: another requester", "mine", OpRead, Row{RequesterID: ptRef(ptOther())}, false},
		{"denied: NULL requester", "mine", OpRead, Row{}, false},
		{"denied: policy does not grant the op", "read_only", OpWrite, row, false},
		{"denied: unknown op", "mine", "delete", row, false},
		{"denied: another policy would grant", "others", OpRead, row, false},
		{"denied: missing name", "gone", OpRead, row, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := NamedPolicyAllows(pol, c.policy, ptMe(), c.op, c.row); got != c.want {
				t.Errorf("NamedPolicyAllows(%q, %q) = %v, want %v", c.policy, c.op, got, c.want)
			}
		})
	}
	t.Run("nil policy", func(t *testing.T) {
		if NamedPolicyAllows(nil, "mine", ptMe(), OpRead, row) {
			t.Error("a nil policy allowed")
		}
	})
	t.Run("voided is false and counted", func(t *testing.T) {
		before := ptInvalidCount(t, pol.EntityType)
		if NamedPolicyAllows(pol, "broken", ptMe(), OpRead, row) {
			t.Error("a voided policy allowed")
		}
		if after := ptInvalidCount(t, pol.EntityType); after != before+1 {
			t.Errorf("authz_policy_invalid_total went %d -> %d, want +1", before, after)
		}
	})
	t.Run("a voided policy elsewhere in the document does not void this one", func(t *testing.T) {
		before := ptInvalidCount(t, pol.EntityType)
		if !NamedPolicyAllows(pol, "mine", ptMe(), OpRead, row) {
			t.Error("mine denied beside a voided policy")
		}
		if after := ptInvalidCount(t, pol.EntityType); after != before {
			t.Errorf("counted a policy that was not tested: %d -> %d", before, after)
		}
	})
}

// A named policy with an under leaf grants only once BindUnder (what
// ResolveUnder binds) has run; unbound, planRow voids it (TICK-1114).
func TestNamedPolicyAllows_Under(t *testing.T) {
	pol := ptWhen(t, `{"under":{"of":"caller"}}`)
	row := Row{State: "open", OrgUnitID: ptRef(utUnit(2))} // below the caller's unit 1
	bound := BindUnder(pol, utMe(), []uuid.UUID{utUnit(1)}, utClosure())
	if !NamedPolicyAllows(bound, "p", utMe(), OpRead, row) {
		t.Error("a row below the caller's unit, bound: denied")
	}
	if NamedPolicyAllows(pol, "p", utMe(), OpRead, row) {
		t.Error("an unbound under policy allowed")
	}
}
