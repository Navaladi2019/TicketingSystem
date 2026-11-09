package authz_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// ReadBranches splits the read predicate into mutually exclusive branches
// (TICK-644): their OR admits what CompileRead admits, and no row matches two.
func TestReadBranches(t *testing.T) {
	mine := pol(rw("mine", authz.Subject{Requester: true, Assignee: true, Shared: true}))
	br, args := authz.ReadBranches(mine, me(), "r", 2)
	if len(br) != 3 {
		t.Fatalf("mine: %d branches, want 3 (requester, assignee, shared): %q", len(br), br)
	}
	for i, want := range []string{"r.requester_id", "r.assignee_id", "record_shares"} {
		if !strings.Contains(br[i], want) {
			t.Errorf("branch %d = %q, want it to lead with %s", i, br[i], want)
		}
	}
	// Later branches exclude the earlier ones, so a row repeats in none.
	if strings.Count(br[2], " AND NOT (") != 2 || strings.Contains(br[0], " AND NOT (") {
		t.Errorf("exclusion: first %q, last %q", br[0], br[2])
	}
	if len(args) == 0 {
		t.Error("no arguments bound")
	}

	// An unconditional rule admits every row: one branch, CompileRead's.
	all := pol(rw("all", everyone()), rw("mine", authz.Subject{Requester: true}))
	if br, _ := authz.ReadBranches(all, me(), "r", 2); len(br) != 1 || br[0] != "true" {
		t.Errorf("everyone rule: %q, want [true]", br)
	}
	// No rule can match: no branch (the caller falls back to CompileRead's "false").
	if br, args := authz.ReadBranches(pol(), identity.Principal{TenantID: tenant()}, "r", 2); len(br) != 0 || len(args) != 0 {
		t.Errorf("no rule: %q %v, want none", br, args)
	}
	// Conditions ride on every branch of their rule.
	cond := pol(rw("c", authz.Subject{Requester: true, Assignee: true}, cond("state", authz.CondEq, "open")))
	br, _ = authz.ReadBranches(cond, me(), "r", 2)
	if len(br) != 2 || !strings.Contains(br[0], "r.state") || !strings.Contains(br[1], "r.state") {
		t.Errorf("conditions: %q", br)
	}
}
