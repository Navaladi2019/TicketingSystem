//go:build integration

// P3-31 (TICK-616) against a real PostgreSQL as nava_app, under RLS:
//
//   - agreement (I5): for a set of fixed policies and principals,
//     SELECT ... WHERE <authz.CompileRead> returns exactly the rows
//     authz.Allows admits, with Row built as the pipeline builds it
//     (pipeline.RowOf: fixed columns and the caller's pair-matched shares).
//     P3-32 makes the policies random;
//   - a group share and a user share carrying the same UUID: only the share
//     of the matching kind admits, on both sides (AR-S10-01);
//   - the write scope: a write that moves a record out of the caller's
//     scope is refused, the same write by a caller whose scope holds both
//     states succeeds; an unreadable record is ErrNotFound even at a stale
//     version; no published ACL refuses every user write.

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

const aclKey = "aclcase"

// aclSchema maps every fixed column.
const aclSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["title"],
  "properties": {
    "title": {"type": "string"},
    "status": {"type": "string", "enum": ["open", "closed"]},
    "requester": {"type": "string", "format": "uuid"},
    "assignee": {"type": "string", "format": "uuid"},
    "group": {"type": "string", "format": "uuid"}
  }
}`

func aclDefinition() metadata.Definition {
	return metadata.Definition{
		Key: aclKey, Schema: json.RawMessage(aclSchema), NumberPrefix: "ACL",
		FixedColumns: map[string]string{
			"status": metadata.ColumnState, "requester": metadata.ColumnRequesterID,
			"assignee": metadata.ColumnAssigneeID, "group": metadata.ColumnAssignmentGroupID,
		},
		States: metadata.StateModel{Initial: "open", States: []string{"open", "closed"},
			Transitions: []metadata.Transition{{From: "open", To: "closed"}}},
	}
}

// aclWorld is one tenant with the aclcase type, two more principals, three
// groups and a grid of records written by a system writer.
type aclWorld struct {
	f                  tenantFixture
	p                  *pipeline.Pipeline
	u1, u2             uuid.UUID // u1 is f.principal
	g1, g2, g3         uuid.UUID
	records            []uuid.UUID
	sameUUIDGroupShare uuid.UUID // shared with group g1
	sameUUIDUserShare  uuid.UUID // a "user" share of the principal whose id is g1's
	userShare          uuid.UUID // shared with u1
	mislabelledUser    uuid.UUID // a "group" share of the group whose id is u1's
}

func seedACLWorld(t *testing.T, e env) aclWorld {
	t.Helper()
	ctx := context.Background()
	w := aclWorld{f: seedTenantWith(t, e, aclDefinition()), g1: uuid.New(), g2: uuid.New(), g3: uuid.New()}
	w.p = newPipeline(e, nil)
	w.u1, w.u2 = w.f.principal.PrincipalID, uuid.New()
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		w.f.id, w.u2, "kc-"+w.u2.String())
	for i, g := range []uuid.UUID{w.g1, w.g2, w.g3} {
		mustExec(t, e.super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)`,
			w.f.id, g, fmt.Sprintf("kc-g%d-%s", i, g), fmt.Sprintf("/t/%s/g%d", w.f.id, i))
	}
	sys := identity.Principal{TenantID: w.f.id}
	create := func(data map[string]any) uuid.UUID {
		t.Helper()
		data["title"] = "t"
		res, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Data: data, Actor: sys, Source: pipeline.SourceSystem})
		if err != nil {
			t.Fatalf("seed %v: %v", data, err)
		}
		return res.RecordID
	}
	ids := func(xs ...uuid.UUID) []any {
		out := []any{nil}
		for _, x := range xs {
			out = append(out, x.String())
		}
		return out
	}
	for _, closed := range []bool{false, true} {
		for _, req := range ids(w.u1, w.u2) {
			for _, asg := range ids(w.u1) {
				for _, grp := range ids(w.g1, w.g2) {
					data := map[string]any{}
					for k, v := range map[string]any{"requester": req, "assignee": asg, "group": grp} {
						if v != nil {
							data[k] = v
						}
					}
					id := create(data)
					if closed {
						if _, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: aclKey, RecordID: id,
							ExpectedVersion: 1, Data: map[string]any{"status": "closed"}, Actor: sys, Source: pipeline.SourceSystem}); err != nil {
							t.Fatal(err)
						}
					}
					w.records = append(w.records, id)
				}
			}
		}
	}
	// The same-UUID cases need a principal that carries g1's id and a group
	// that carries u1's id: since TICK-625 a share names its user or group
	// through a foreign key (user_id, group_id; principal_id is generated
	// from them), so each target must exist. The caller is in group g1 and is
	// u1; the principal g1 and the group u1 are someone else.
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, $2, $3, true, now())`,
		w.f.id, w.g1, "kc-same-"+w.g1.String())
	mustExec(t, e.super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)`,
		w.f.id, w.u1, "kc-same-"+w.u1.String(), fmt.Sprintf("/t/%s/same-as-u1", w.f.id))
	share := func(kind string, principal uuid.UUID) uuid.UUID {
		id := create(map[string]any{})
		w.records = append(w.records, id)
		column := "user_id"
		if kind == "group" {
			column = "group_id"
		}
		inTx(t, e, w.f.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO record_shares (tenant_id, record_id, `+column+`, principal_kind) VALUES ($1, $2, $3, $4)`,
				w.f.id, id, principal, kind)
			return err
		})
		return id
	}
	w.sameUUIDGroupShare = share("group", w.g1)
	w.sameUUIDUserShare = share("user", w.g1)
	w.userShare = share("user", w.u1)
	w.mislabelledUser = share("group", w.u1)
	share("user", w.u2) // someone else's share
	share("group", w.g3)
	return w
}

// visible returns the records the SQL predicate admits and those Allows
// admits, each sorted, read in one tenant transaction.
func (w aclWorld) visible(t *testing.T, e env, pol *authz.Policy, p identity.Principal) (bySQL, byGo []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	inTx(t, e, w.f.id, func(tx pgx.Tx) error {
		pred, args := authz.CompileRead(pol, p, "r", 2)
		rows, err := tx.Query(ctx, `SELECT r.id FROM records r JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
			WHERE r.tenant_id = $1 AND d.key = $2 AND `+pred+` ORDER BY r.id`, append([]any{w.f.id, aclKey}, args...)...)
		if err != nil {
			return fmt.Errorf("predicate %s: %w", pred, err)
		}
		if bySQL, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
			return err
		}
		for _, id := range w.records {
			rec, err := pipeline.Get(ctx, tx, w.f.id, id)
			if err != nil {
				return err
			}
			row, err := pipeline.RowOf(ctx, tx, p, rec)
			if err != nil {
				return err
			}
			if _, ok := authz.Allows(pol, p, authz.OpRead, row); ok {
				byGo = append(byGo, id)
			}
		}
		return nil
	})
	slices.SortFunc(byGo, func(a, b uuid.UUID) int { return compareUUID(a, b) })
	return bySQL, byGo
}

func compareUUID(a, b uuid.UUID) int {
	for i := range a {
		if a[i] != b[i] {
			return int(a[i]) - int(b[i])
		}
	}
	return 0
}

func TestCompileReadAgreesWithAllows(t *testing.T) {
	e := migrated(t)
	w := seedACLWorld(t, e)
	rd := []string{authz.OpRead}
	rule := func(id string, who authz.Subject, where ...authz.Cond) authz.Rule {
		return authz.Rule{ID: id, Ops: rd, Who: who, Where: where}
	}
	in := func(col string, vals ...string) authz.Cond {
		return authz.Cond{Column: col, Op: authz.CondIn, Values: vals}
	}
	ref := func(col, caller string) authz.Cond { return authz.Cond{Column: col, Op: authz.CondEq, Caller: caller} }
	policies := map[string]*authz.Policy{
		"nil":       nil,
		"empty":     {},
		"requester": {Rules: []authz.Rule{rule("r", authz.Subject{Requester: true})}},
		"assignee":  {Rules: []authz.Rule{rule("r", authz.Subject{Assignee: true})}},
		"shared":    {Rules: []authz.Rule{rule("r", authz.Subject{Shared: true})}},
		"service desk": {Rules: []authz.Rule{
			rule("desk", authz.Subject{Groups: []uuid.UUID{w.g1}}, authz.Cond{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}),
			rule("req", authz.Subject{Requester: true}),
		}},
		"open, mine":      {Rules: []authz.Rule{rule("r", authz.Subject{Users: []uuid.UUID{w.u1, w.u2}}, in("state", "open"), ref("requester_id", authz.CallerSelf))}},
		"literal group":   {Rules: []authz.Rule{rule("r", authz.Subject{Groups: []uuid.UUID{w.g2, w.g3}}, in("assignment_group_id", w.g2.String()), in("state", "closed", "open"))}},
		"literal people":  {Rules: []authz.Rule{rule("r", authz.Subject{Users: []uuid.UUID{w.u1}}, in("requester_id", w.u2.String()), in("assignee_id", w.u1.String()))}},
		"everything":      {Rules: []authz.Rule{rule("r", authz.Subject{Users: []uuid.UUID{w.u1, w.u2}})}},
		"write only":      {Rules: []authz.Rule{{ID: "w", Ops: []string{authz.OpWrite}, Who: authz.Subject{Users: []uuid.UUID{w.u1}}}}},
		"invalid ignored": {Rules: []authz.Rule{rule("bad", authz.Subject{Users: []uuid.UUID{w.u1}}, in("data", "x")), rule("ok", authz.Subject{Assignee: true, Shared: true}, in("state", "closed"))}},
	}
	principals := map[string]identity.Principal{
		"u1 in g1":       {TenantID: w.f.id, PrincipalID: w.u1, Groups: []uuid.UUID{w.g1}},
		"u1 in no group": {TenantID: w.f.id, PrincipalID: w.u1},
		"u2 in g2, g3":   {TenantID: w.f.id, PrincipalID: w.u2, Groups: []uuid.UUID{w.g2, w.g3}},
		"no principal":   {TenantID: w.f.id, Groups: []uuid.UUID{w.g1}},
	}
	nonEmpty := 0
	for pn, pol := range policies {
		for qn, p := range principals {
			bySQL, byGo := w.visible(t, e, pol, p)
			if !slices.Equal(bySQL, byGo) {
				t.Errorf("policy %q, principal %q: SQL admits %d rows %v, Allows %d rows %v", pn, qn, len(bySQL), bySQL, len(byGo), byGo)
			}
			if len(byGo) > 0 && len(byGo) < len(w.records) {
				nonEmpty++
			}
		}
	}
	// The table must actually discriminate, or agreement proves nothing.
	if nonEmpty < 10 {
		t.Fatalf("only %d policy/principal pairs admit a proper subset of the rows", nonEmpty)
	}
}

// AR-S10-01: a share matches on (principal_kind, principal_id). A "user"
// share carrying a group's UUID does not admit that group's members, and a
// "group" share carrying a user's UUID does not admit that user, in Allows
// (through RowOf) and in CompileRead alike.
func TestSharesMatchOnKindAndID(t *testing.T) {
	e := migrated(t)
	w := seedACLWorld(t, e)
	shared := &authz.Policy{Rules: []authz.Rule{{ID: "s", Ops: []string{authz.OpRead}, Who: authz.Subject{Shared: true}}}}
	p := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1, Groups: []uuid.UUID{w.g1}}
	bySQL, byGo := w.visible(t, e, shared, p)
	want := []uuid.UUID{w.sameUUIDGroupShare, w.userShare}
	slices.SortFunc(want, compareUUID)
	if !slices.Equal(bySQL, want) || !slices.Equal(byGo, want) {
		t.Fatalf("shared rows: SQL %v, Allows %v; want only the group share of g1 and the user share of u1 %v (not %s or %s)",
			bySQL, byGo, want, w.sameUUIDUserShare, w.mislabelledUser)
	}
}

// The write scope through the real pipeline and the published ACL.
func TestWriteScopeIsEnforcedBeforeAndAfter(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	w := seedACLWorld(t, e)
	agent := identity.Principal{TenantID: w.f.id, PrincipalID: w.u1, Groups: []uuid.UUID{w.g1}}
	lead := identity.Principal{TenantID: w.f.id, PrincipalID: w.u2, Groups: []uuid.UUID{w.g1, w.g2}}
	desk := authz.Rule{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{w.g1, w.g2}},
		Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}}
	inTx(t, e, w.f.id, func(tx pgx.Tx) error { _, err := acltest.Publish(ctx, tx, aclKey, desk); return err })

	upd := func(actor identity.Principal, id uuid.UUID, version int64, patch map[string]any) error {
		_, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: aclKey, RecordID: id,
			ExpectedVersion: version, Data: patch, Actor: actor, Source: pipeline.SourceUser})
		return err
	}
	res, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Actor: agent, Source: pipeline.SourceUser,
		Data: map[string]any{"title": "t", "group": w.g1.String()}})
	if err != nil {
		t.Fatalf("agent's create in scope: %v", err)
	}
	if _, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Actor: agent, Source: pipeline.SourceUser,
		Data: map[string]any{"title": "t", "group": w.g2.String()}}); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("agent's create out of scope: %v, want ErrForbidden", err)
	}
	move := map[string]any{"group": w.g2.String()}
	if err := upd(agent, res.RecordID, 1, move); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("agent moving the record to g2: %v, want ErrForbidden", err)
	}
	if _, _, _, v, _ := storedData(t, e, res.RecordID); v != 1 {
		t.Fatalf("record at version %d after the refusal", v)
	}
	if err := upd(lead, res.RecordID, 1, move); err != nil {
		t.Fatalf("lead moving the record to g2: %v", err)
	}
	// Now outside the agent's read scope: ErrNotFound, at the current and
	// at a stale version alike.
	for _, v := range []int64{2, 1} {
		if err := upd(agent, res.RecordID, v, map[string]any{"title": "x"}); !errors.Is(err, pipeline.ErrNotFound) {
			t.Fatalf("agent writing an unreadable record at version %d: %v, want ErrNotFound", v, err)
		}
	}
	// An AI write is authorized as its principal.
	ref := uuid.New().String()
	if _, err := write(ctx, e, w.p, w.f.id, pipeline.Command{Op: pipeline.OpCreate, EntityKey: aclKey, Actor: agent, Source: pipeline.SourceAI,
		SourceRef: &ref, Data: map[string]any{"title": "t", "group": w.g3.String()}}); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("AI create out of its principal's scope: %v, want ErrForbidden", err)
	}
}

// No published ACL: every user write and read is refused (fail closed); a
// system writer is not affected.
func TestNoPublishedACLRefusesUserWrites(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	f := seedTenant(t, e)
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		d := aclDefinition()
		d.Key, d.NumberPrefix = "bare", "BAR"
		_, err := entitytypes.Store{}.Create(ctx, tx, f.id, entitytypes.NewInput{Name: d.Key, PublishedBy: f.principal.PrincipalID, Definition: d})
		return err
	})
	p := newPipeline(e, nil)
	cmd := pipeline.Command{Op: pipeline.OpCreate, EntityKey: "bare", Actor: f.principal, Source: pipeline.SourceUser, Data: map[string]any{"title": "t"}}
	if _, err := write(ctx, e, p, f.id, cmd); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("user create with no ACL: %v, want ErrForbidden", err)
	}
	cmd.Source, cmd.Actor = pipeline.SourceSystem, f.system()
	res, err := write(ctx, e, p, f.id, cmd)
	if err != nil {
		t.Fatalf("system create with no ACL: %v", err)
	}
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		pol, err := authz.PolicyStore{}.Current(ctx, tx, "bare")
		if err != nil || pol != nil {
			return fmt.Errorf("Current = %+v, %v; want nil, nil", pol, err)
		}
		rec, err := pipeline.Get(ctx, tx, f.id, res.RecordID)
		if err != nil {
			return err
		}
		row, err := pipeline.RowOf(ctx, tx, f.principal, rec)
		if err != nil {
			return err
		}
		if err := (authz.ACL{Policies: authz.PolicyStore{}}).CanRead(ctx, tx, f.principal, "bare", row); !errors.Is(err, authz.ErrNotVisible) {
			return fmt.Errorf("CanRead with no ACL: %v, want ErrNotVisible", err)
		}
		return nil
	})
}

// PolicyStore reads the newest version, in the transaction's tenant only.
func TestPolicyStoreReadsTheNewestVersionOfItsTenant(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	a, b := seedTenant(t, e), seedTenant(t, e)
	inTx(t, e, a.id, func(tx pgx.Tx) error {
		_, err := acltest.Publish(ctx, tx, entityKey, authz.Rule{ID: "v2", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true}})
		return err
	})
	inTx(t, e, a.id, func(tx pgx.Tx) error {
		pol, err := authz.PolicyStore{}.Current(ctx, tx, entityKey)
		if err != nil {
			return err
		}
		if pol == nil || pol.Version != 2 || len(pol.Rules) != 1 || pol.Rules[0].ID != "v2" || !pol.Rules[0].Who.Requester {
			return fmt.Errorf("tenant A's policy = %+v, want version 2 with rule v2", pol)
		}
		return nil
	})
	inTx(t, e, b.id, func(tx pgx.Tx) error {
		pol, err := authz.PolicyStore{}.Current(ctx, tx, entityKey)
		if err != nil {
			return err
		}
		if pol == nil || pol.Version != 1 || pol.Rules[0].ID != "test-allow-users" {
			return fmt.Errorf("tenant B's policy = %+v, want its own version 1", pol)
		}
		return nil
	})
	// A rule list the decoder would refuse never reaches it: the shape CHECK
	// (TICK-629) refuses it at insert, the superuser's too, and the newest
	// version stays 2.
	_, err := e.super.Exec(ctx, `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
		SELECT tenant_id, id, 3, '[{"id": "x", "ops": ["read"], "who": {}, "colour": "red"}]', '[]' FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, a.id, entityKey)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "acl_versions_rules_check" {
		t.Fatalf("inserting a malformed rule list: %v, want 23514 on acl_versions_rules_check", err)
	}
	inTx(t, e, a.id, func(tx pgx.Tx) error {
		pol, err := authz.PolicyStore{}.Current(ctx, tx, entityKey)
		if err != nil {
			return err
		}
		if pol == nil || pol.Version != 2 {
			return fmt.Errorf("tenant A's policy after the refused insert = %+v, want version 2", pol)
		}
		return nil
	})
}
