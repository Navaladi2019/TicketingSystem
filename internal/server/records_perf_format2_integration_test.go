//go:build integration && perf

// S30 P11-12 (TICK-944): the 800k benchmark on format 2 (ADR-010).
//
// NAVA_PERF_ACL_FORMAT=2 stores every tenant's ACL as the format-2 document
// the format-1 converter makes of perfACL (ConvertFormat1), published as a
// format-2 body through authz.Prepare and PublishPrepared, the checks and
// insert the access route uses. Unset or 1 keeps acltest.Publish (format 1),
// the P3-36 baseline.
//
// The "composed:" scenarios run last. Before the first one, the large
// tenant's ticket ACL is republished with the agents' policy written as the
// S30 composed policy, in_assigned_group AND NOT is_requester, beside the
// converted "mine" and "admins" policies (perfComposedDoc). Each scenario logs
// the document's ReadBranchBound.

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// perfACLFormat is NAVA_PERF_ACL_FORMAT: 2 stores format 2, anything else
// format 1.
func perfACLFormat() int {
	if os.Getenv("NAVA_PERF_ACL_FORMAT") == "2" {
		return authz.FormatV2
	}
	return 1
}

// perfAgentsGroup is tenant i's agents group (seed's id scheme).
func perfAgentsGroup(i int) uuid.UUID {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(fmt.Sprintf("agents/%d", i)))
}

// perfComposedDoc is perfACL with the agents' policy written as S30's
// "agents see tickets in their group that they did not raise".
func perfComposedDoc(agents, admins uuid.UUID) authz.Document {
	d := authz.ConvertFormat1(perfACL(agents, admins))
	d.Conditions = []authz.Condition{
		{Name: "in_assigned_group", Expr: authz.Node{Cmp: &authz.Cond{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		{Name: "is_requester", Expr: authz.Node{Cmp: &authz.Cond{Column: "requester_id", Op: authz.CondEq, Caller: authz.CallerSelf}}},
	}
	for i, p := range d.Policies {
		if p.Name == "desk" {
			d.Policies[i].When = authz.Node{All: []authz.Node{
				{MemberOf: &authz.MemberOf{Groups: []uuid.UUID{agents}}},
				{Ref: "in_assigned_group"},
				{Not: &authz.Node{Ref: "is_requester"}},
			}}
		}
	}
	return d
}

// perfPublishDoc publishes d as a format-2 body, through Prepare (the
// route's database-free checks) and PublishPrepared, in tx.
func perfPublishDoc(ctx context.Context, tx pgx.Tx, d authz.Document) (*authz.Prepared, error) {
	enc, err := authz.Encode(d)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]json.RawMessage{"rules": enc})
	if err != nil {
		return nil, err
	}
	prep, problems := authz.Prepare(body, authz.DefaultMaxDocumentBytes)
	if len(problems) > 0 {
		return nil, fmt.Errorf("perf: format-2 document refused: %v", problems)
	}
	if prep.Format != authz.FormatV2 {
		return nil, fmt.Errorf("perf: prepared format %d, want 2", prep.Format)
	}
	store := authz.PolicyStore{}
	cur, err := store.Newest(ctx, tx, "ticket")
	if err != nil {
		return nil, err
	}
	have := 0
	if cur != nil {
		have = cur.Version
	}
	_, err = store.PublishPrepared(ctx, tx, "ticket", have, authz.Publisher{System: "acl-perf", Source: audit.SourceSystem}, prep)
	return prep, err
}

// publishPerfACL publishes tenant tn's ticket ACL in the NAVA_PERF_ACL_FORMAT.
func (e *perfEnv) publishPerfACL(t *testing.T, tn, agents, admins uuid.UUID) {
	t.Helper()
	if err := tenancy.InTenantTx(t.Context(), e.pool, tn, func(tx pgx.Tx) error {
		if perfACLFormat() == authz.FormatV2 {
			_, err := perfPublishDoc(t.Context(), tx, authz.ConvertFormat1(perfACL(agents, admins)))
			return err
		}
		_, err := acltest.Publish(t.Context(), tx, "ticket", perfACL(agents, admins)...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// composedBefore republishes the large tenant's ticket ACL as perfComposedDoc
// once, before the first composed scenario that runs.
func (e *perfEnv) composedBefore(t *testing.T) func() {
	return func() {
		if e.composed {
			return
		}
		d := perfComposedDoc(perfAgentsGroup(0), e.large.adminGroup)
		if err := tenancy.InTenantTx(t.Context(), e.pool, e.large.id, func(tx pgx.Tx) error {
			prep, err := perfPublishDoc(t.Context(), tx, d)
			if err == nil {
				for _, w := range prep.Warnings {
					t.Logf("COMPOSED lint %s at %s (accepted)", w.Code, w.Pointer)
				}
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		e.composed = true
		t.Logf("COMPOSED published: ReadBranchBound=%d (converted S10 document: %d)",
			authz.ReadBranchBound(d), authz.ReadBranchBound(authz.ConvertFormat1(perfACL(perfAgentsGroup(0), e.large.adminGroup))))
	}
}
