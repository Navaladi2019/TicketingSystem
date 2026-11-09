//go:build integration

// I5 agreement for the branch form (TICK-644, P3-34c): the UNION ALL of
// authz.ReadBranches returns, for random policies, principals and records,
// exactly {r | authz.Allows(pol, p, "read", r)} and returns each such row once.
// The single-statement form is TestI5Agreement; a row it hides (another
// tenant's, a revoked share's) is hidden here too because the world holds both.

package isolation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

func TestI5BranchesAgreement(t *testing.T) {
	seed, err := i5Seed()
	if err != nil {
		t.Fatal(err)
	}
	w, cases := i5Generate(seed)
	err = i5InWorld(t, w, func(ctx context.Context, tx pgx.Tx) error {
		split := 0
		for pi, c := range cases {
			for ki, p := range c.Principals {
				branches, bargs := authz.ReadBranches(c.Policy, p, "r", 2)
				if len(branches) > 1 {
					split++
				}
				got, err := i5BranchRows(ctx, tx, w, branches, bargs)
				if err != nil {
					return fmt.Errorf("policy #%d principal #%d: %w", pi, ki, err)
				}
				for _, rec := range w.Records {
					_, want := authz.Allows(c.Policy, p, authz.OpRead, rec.row(p))
					switch n := got[rec.Row.ID]; {
					case n > 1:
						t.Errorf("seed %d policy #%d principal #%d: record %s returned %d times by %q", seed, pi, ki, rec.Row.ID, n, branches)
					case want != (n == 1):
						t.Errorf("seed %d policy #%d principal #%d: record %s Allows=%v branches=%d times; %q", seed, pi, ki, rec.Row.ID, want, n, branches)
					}
				}
			}
		}
		if split == 0 {
			t.Error("no case split into branches; the run proves nothing")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
}

// i5BranchRows runs the branches as a UNION ALL under the tenant's RLS and
// counts how often each record id comes back. No branch is the "false" of
// CompileRead; one branch is the statement itself.
func i5BranchRows(ctx context.Context, tx pgx.Tx, w *i5World, branches []string, bargs []any) (map[uuid.UUID]int, error) {
	if len(branches) == 0 {
		return map[uuid.UUID]int{}, nil
	}
	arms := make([]string, len(branches))
	for i, b := range branches {
		arms[i] = "SELECT r.id FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND " + b
	}
	rows, err := tx.Query(ctx, strings.Join(arms, "\nUNION ALL\n"), append([]any{w.Tenant, w.Definition}, bargs...)...)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]int{}
	var id uuid.UUID
	_, err = pgx.ForEachRow(rows, []any{&id}, func() error { out[id]++; return nil })
	return out, err
}
