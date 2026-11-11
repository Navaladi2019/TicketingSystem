package authz_test

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The fixture ids: functions, not globals (gochecknoglobals).
func tenant() uuid.UUID { return uuid.MustParse("0a000000-0000-4000-8000-000000000001") }
func caller() uuid.UUID { return uuid.MustParse("0a000000-0000-4000-8000-0000000000c1") }
func other() uuid.UUID  { return uuid.MustParse("0a000000-0000-4000-8000-0000000000c2") }
func g1() uuid.UUID     { return uuid.MustParse("0a000000-0000-4000-8000-0000000000a1") }
func g2() uuid.UUID     { return uuid.MustParse("0a000000-0000-4000-8000-0000000000a2") }
func g3() uuid.UUID     { return uuid.MustParse("0a000000-0000-4000-8000-0000000000a3") }

// me is the caller, in groups g1 and g2.
func me() identity.Principal {
	return identity.Principal{TenantID: tenant(), PrincipalID: caller(), Groups: []uuid.UUID{g1(), g2()}}
}

func ptr(id uuid.UUID) *uuid.UUID { return &id }

func pol(rules ...authz.Rule) *authz.Policy { return &authz.Policy{Version: 1, Rules: rules} }

func rw(id string, who authz.Subject, where ...authz.Cond) authz.Rule {
	return authz.Rule{ID: id, Ops: []string{authz.OpRead, authz.OpWrite}, Who: who, Where: where}
}

func everyone() authz.Subject { return authz.Subject{Users: []uuid.UUID{caller()}} }

func cond(column, op string, values ...string) authz.Cond {
	return authz.Cond{Column: column, Op: op, Values: values}
}

func callerCond(column, op, ref string) authz.Cond {
	return authz.Cond{Column: column, Op: op, Caller: ref}
}

// TestAllows covers each Subject kind, each Cond op and caller reference,
// rule OR, condition AND, ops, and every malformed rule (which matches
// nothing). Each case also checks CompileRead agrees on "no rule can match".
func TestAllows(t *testing.T) {
	open := authz.Row{ID: uuid.New(), State: "open"}
	withReq := func(id uuid.UUID) authz.Row { r := open; r.RequesterID = ptr(id); return r }
	withAsg := func(id uuid.UUID) authz.Row { r := open; r.AssigneeID = ptr(id); return r }
	withGroup := func(id uuid.UUID) authz.Row { r := open; r.AssignmentGroupID = ptr(id); return r }
	shared := func(ids ...uuid.UUID) authz.Row { r := open; r.SharedWith = ids; return r }
	nobody := identity.Principal{TenantID: tenant()} // a principal with no id

	cases := []struct {
		name   string
		pol    *authz.Policy
		p      identity.Principal
		op     string
		row    authz.Row
		wantID string // "" = denied
	}{
		{"nil policy denies", nil, me(), authz.OpRead, open, ""},
		{"empty policy denies", pol(), me(), authz.OpRead, open, ""},
		{"empty subject names nobody", pol(rw("r", authz.Subject{})), me(), authz.OpRead, open, ""},

		{"groups: a caller's group", pol(rw("r", authz.Subject{Groups: []uuid.UUID{g3(), g2()}})), me(), authz.OpRead, open, "r"},
		{"groups: none of the caller's", pol(rw("r", authz.Subject{Groups: []uuid.UUID{g3()}})), me(), authz.OpRead, open, ""},
		{"users: the caller", pol(rw("r", authz.Subject{Users: []uuid.UUID{other(), caller()}})), me(), authz.OpRead, open, "r"},
		{"users: someone else", pol(rw("r", authz.Subject{Users: []uuid.UUID{other()}})), me(), authz.OpRead, open, ""},
		{"users: a principal with no id is nobody", pol(rw("r", authz.Subject{Users: []uuid.UUID{uuid.Nil}})), nobody, authz.OpRead, open, ""},

		{"requester: the caller", pol(rw("r", authz.Subject{Requester: true})), me(), authz.OpRead, withReq(caller()), "r"},
		{"requester: someone else", pol(rw("r", authz.Subject{Requester: true})), me(), authz.OpRead, withReq(other()), ""},
		{"requester: none", pol(rw("r", authz.Subject{Requester: true})), me(), authz.OpRead, open, ""},
		{"requester: a principal with no id", pol(rw("r", authz.Subject{Requester: true})), nobody, authz.OpRead, withReq(uuid.Nil), ""},
		{"assignee: the caller", pol(rw("r", authz.Subject{Assignee: true})), me(), authz.OpRead, withAsg(caller()), "r"},
		{"assignee: someone else", pol(rw("r", authz.Subject{Assignee: true})), me(), authz.OpRead, withAsg(other()), ""},
		{"assignee is not requester", pol(rw("r", authz.Subject{Assignee: true})), me(), authz.OpRead, withReq(caller()), ""},

		{"shared: a user share for the caller", pol(rw("r", authz.Subject{Shared: true})), me(), authz.OpRead, shared(caller()), "r"},
		{"shared: a group share for a caller's group", pol(rw("r", authz.Subject{Shared: true})), me(), authz.OpRead, shared(g2()), "r"},
		{"shared: none", pol(rw("r", authz.Subject{Shared: true})), me(), authz.OpRead, open, ""},
		{"shared: a hand-built row naming someone else", pol(rw("r", authz.Subject{Shared: true})), me(), authz.OpRead, shared(other(), g3()), ""},
		{"shared: a principal with no id", pol(rw("r", authz.Subject{Shared: true})), nobody, authz.OpRead, shared(uuid.Nil), ""},

		{"eq on state", pol(rw("r", everyone(), cond("state", "eq", "open"))), me(), authz.OpRead, open, "r"},
		{"eq on state, other value", pol(rw("r", everyone(), cond("state", "eq", "closed"))), me(), authz.OpRead, open, ""},
		{"eq needs exactly one value", pol(rw("r", everyone(), cond("state", "eq", "open", "closed"))), me(), authz.OpRead, open, ""},
		{"eq with no value", pol(rw("r", everyone(), cond("state", "eq"))), me(), authz.OpRead, open, ""},
		{"in on state", pol(rw("r", everyone(), cond("state", "in", "closed", "open"))), me(), authz.OpRead, open, "r"},
		{"in, no match", pol(rw("r", everyone(), cond("state", "in", "closed"))), me(), authz.OpRead, open, ""},
		{"in with no values matches nothing", pol(rw("r", everyone(), cond("state", "in"))), me(), authz.OpRead, open, ""},
		{"eq on a uuid column", pol(rw("r", everyone(), cond("assignee_id", "eq", other().String()))), me(), authz.OpRead, withAsg(other()), "r"},
		{"eq on a uuid column, upper case literal", pol(rw("r", everyone(), cond("assignee_id", "eq", strings.ToUpper(other().String())))), me(), authz.OpRead, withAsg(other()), "r"},
		{"in on assignment_group_id", pol(rw("r", everyone(), cond("assignment_group_id", "in", g3().String(), g1().String()))), me(), authz.OpRead, withGroup(g1()), "r"},
		{"a NULL column matches no literal", pol(rw("r", everyone(), cond("requester_id", "in", other().String()))), me(), authz.OpRead, open, ""},
		{"a uuid literal that does not parse", pol(rw("r", everyone(), cond("requester_id", "eq", "not-a-uuid"))), me(), authz.OpRead, withReq(caller()), ""},

		{"self on requester_id", pol(rw("r", authz.Subject{Groups: []uuid.UUID{g1()}}, callerCond("requester_id", "eq", "self"))), me(), authz.OpRead, withReq(caller()), "r"},
		{"self on requester_id, someone else", pol(rw("r", authz.Subject{Groups: []uuid.UUID{g1()}}, callerCond("requester_id", "eq", "self"))), me(), authz.OpRead, withReq(other()), ""},
		{"self on assignee_id", pol(rw("r", everyone(), callerCond("assignee_id", "in", "self"))), me(), authz.OpRead, withAsg(caller()), "r"},
		{"self on state is invalid", pol(rw("r", everyone(), callerCond("state", "eq", "self"))), me(), authz.OpRead, open, ""},
		{"self on assignment_group_id is invalid", pol(rw("r", everyone(), callerCond("assignment_group_id", "eq", "self"))), me(), authz.OpRead, withGroup(caller()), ""},
		{"self with values is invalid", pol(rw("r", everyone(), authz.Cond{Column: "requester_id", Op: "eq", Caller: "self", Values: []string{caller().String()}})), me(), authz.OpRead, withReq(caller()), ""},
		{"my_groups on assignment_group_id", pol(rw("r", everyone(), callerCond("assignment_group_id", "in", "my_groups"))), me(), authz.OpRead, withGroup(g2()), "r"},
		{"my_groups, another group", pol(rw("r", everyone(), callerCond("assignment_group_id", "in", "my_groups"))), me(), authz.OpRead, withGroup(g3()), ""},
		{"my_groups, no group", pol(rw("r", everyone(), callerCond("assignment_group_id", "in", "my_groups"))), me(), authz.OpRead, open, ""},
		{"my_groups for a caller in no group", pol(rw("r", everyone(), callerCond("assignment_group_id", "in", "my_groups"))), identity.Principal{TenantID: tenant(), PrincipalID: caller()}, authz.OpRead, withGroup(g1()), ""},
		{"my_groups on requester_id is invalid", pol(rw("r", everyone(), callerCond("requester_id", "in", "my_groups"))), me(), authz.OpRead, withReq(g1()), ""},

		{"unknown column", pol(rw("r", everyone(), cond("data.priority", "eq", "high"))), me(), authz.OpRead, open, ""},
		{"unknown cond op", pol(rw("r", everyone(), cond("state", "ne", "closed"))), me(), authz.OpRead, open, ""},
		{"unknown caller reference", pol(rw("r", everyone(), callerCond("requester_id", "eq", "boss"))), me(), authz.OpRead, withReq(caller()), ""},

		{"where ANDs", pol(rw("r", everyone(), cond("state", "eq", "open"), callerCond("assignment_group_id", "in", "my_groups"))), me(), authz.OpRead, open, ""},
		{"where ANDs, both hold", pol(rw("r", everyone(), cond("state", "eq", "open"), callerCond("assignment_group_id", "in", "my_groups"))), me(), authz.OpRead, withGroup(g1()), "r"},
		{"rules OR: the first that matches", pol(
			rw("r1", everyone(), cond("state", "eq", "closed")),
			rw("r2", authz.Subject{Requester: true}),
			rw("r3", everyone()),
		), me(), authz.OpRead, withReq(caller()), "r2"},
		{"an invalid rule does not hide a valid one", pol(
			rw("bad", everyone(), cond("nope", "eq", "x")),
			rw("good", everyone()),
		), me(), authz.OpRead, open, "good"},
		{"subject OR: requester or assignee", pol(rw("r", authz.Subject{Requester: true, Assignee: true})), me(), authz.OpRead, withAsg(caller()), "r"},

		{"read rule does not grant write", pol(authz.Rule{ID: "r", Ops: []string{"read"}, Who: everyone()}), me(), authz.OpWrite, open, ""},
		{"write rule does not grant read", pol(authz.Rule{ID: "w", Ops: []string{"write"}, Who: everyone()}), me(), authz.OpRead, open, ""},
		{"write rule grants write", pol(authz.Rule{ID: "w", Ops: []string{"write"}, Who: everyone()}), me(), authz.OpWrite, open, "w"},
		{"an unknown op is never granted", pol(authz.Rule{ID: "x", Ops: []string{"delete"}, Who: everyone()}), me(), "delete", open, ""},
		{"no ops", pol(authz.Rule{ID: "x", Who: everyone()}), me(), authz.OpRead, open, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id, ok := authz.Allows(c.pol, c.p, c.op, c.row)
			if ok != (c.wantID != "") || id != c.wantID {
				t.Fatalf("Allows = %q, %v; want %q, %v", id, ok, c.wantID, c.wantID != "")
			}
			if c.op != authz.OpRead {
				return
			}
			// "no rule can match" must be the same decision on both sides.
			if sql, _ := authz.CompileRead(c.pol, c.p, "r", 0); c.wantID != "" && sql == "false" {
				t.Fatalf("CompileRead = false for a row Allows admits")
			}
		})
	}
}

var placeholder = regexp.MustCompile(`\$(\d+)`)

// CompileRead numbers its placeholders from argBase+1, binds one argument
// per placeholder, binds the caller's ids and the policy's literals only,
// and matches shares on (principal_kind, principal_id).
func TestCompileReadShape(t *testing.T) {
	p := pol(
		rw("a", authz.Subject{Requester: true, Assignee: true, Shared: true}, cond("state", "in", "open", "new")),
		rw("b", authz.Subject{Groups: []uuid.UUID{g1()}}, callerCond("assignment_group_id", "in", "my_groups"), cond("requester_id", "eq", other().String())),
	)
	sql, args := authz.CompileRead(p, me(), "rec", 2)
	nums := placeholder.FindAllStringSubmatch(sql, -1)
	if len(nums) != len(args) {
		t.Fatalf("%d placeholders, %d args: %s", len(nums), len(args), sql)
	}
	for i, m := range nums {
		if n, _ := strconv.Atoi(m[1]); n != 3+i {
			t.Fatalf("placeholder %d is $%s, want $%d: %s", i, m[1], 3+i, sql)
		}
	}
	for _, want := range []string{
		"rec.requester_id", "rec.assignee_id", "rec.state", "rec.assignment_group_id",
		"acl_share.tenant_id = rec.tenant_id", "acl_share.record_id = rec.id",
		"acl_share.principal_kind = 'user' AND acl_share.principal_id =",
		"acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY(",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("predicate lacks %q: %s", want, sql)
		}
	}
	for _, a := range args {
		switch v := a.(type) {
		case uuid.UUID:
			if v != caller() {
				t.Errorf("bound id %s is not the caller's", v)
			}
		case []string:
			// A uuid set is bound as text (exec mode); state values are text.
			for _, s := range v {
				if id, err := uuid.Parse(s); err == nil && id != g1() && id != g2() && id != other() {
					t.Errorf("bound id %s is neither the caller's group nor a policy literal", id)
				}
			}
		default:
			t.Errorf("unexpected argument %T", a)
		}
	}
}

func TestCompileReadNoRuleIsFalse(t *testing.T) {
	for name, p := range map[string]*authz.Policy{
		"nil":          nil,
		"empty":        pol(),
		"write only":   pol(authz.Rule{ID: "w", Ops: []string{"write"}, Who: everyone()}),
		"not me":       pol(rw("r", authz.Subject{Users: []uuid.UUID{other()}})),
		"invalid cond": pol(rw("r", everyone(), cond("state", "like", "%"))),
	} {
		if sql, args := authz.CompileRead(p, me(), "r", 0); sql != "false" || args != nil {
			t.Errorf("%s: CompileRead = %q, %v; want false and no args", name, sql, args)
		}
	}
	if sql, args := authz.CompileRead(pol(rw("r", everyone())), me(), "r", 0); sql != "(((true)))" || len(args) != 0 {
		t.Errorf("everyone, no conditions: %q %v", sql, args)
	}
}

func TestCompileReadRefusesABadAlias(t *testing.T) {
	for _, alias := range []string{"", "r;", "r.x", "R", "1r", "r id"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("alias %q: no panic", alias)
				}
			}()
			authz.CompileRead(pol(), me(), alias, 0)
		}()
	}
}

// tenantTx answers tenancy.TenantOf with its tenant; any other statement
// panics through the nil pgx.Tx.
type tenantTx struct {
	pgx.Tx
	tenant string
}

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

func (x tenantTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowFunc(func(dest ...any) error { *dest[0].(*string) = x.tenant; return nil })
}

type fakePolicies struct {
	pol *authz.Policy
	err error
	key *string
}

func (f fakePolicies) Current(_ context.Context, _ pgx.Tx, key string) (*authz.Policy, error) {
	if f.key != nil {
		*f.key = key
	}
	return f.pol, f.err
}

func TestACLCanRead(t *testing.T) {
	ctx := context.Background()
	tx := tenantTx{tenant: tenant().String()}
	row := authz.Row{ID: uuid.New(), State: "open", RequesterID: ptr(caller())}
	var key string
	a := authz.ACL{Policies: fakePolicies{pol: pol(rw("req", authz.Subject{Requester: true})), key: &key}}
	if err := a.CanRead(ctx, tx, me(), "incident", row); err != nil || key != "incident" {
		t.Fatalf("requester: %v (key %q)", err, key)
	}
	if err := a.CanRead(ctx, tx, identity.Principal{TenantID: tenant(), PrincipalID: other()}, "incident", row); !errors.Is(err, authz.ErrNotVisible) {
		t.Fatalf("someone else: %v, want ErrNotVisible", err)
	}
	// A principal of another tenant is refused before the policy is read:
	// its ids and groups mean nothing here.
	foreign := me()
	foreign.TenantID = uuid.New()
	if err := a.CanRead(ctx, tx, foreign, "incident", row); !errors.Is(err, authz.ErrNotVisible) {
		t.Fatalf("other tenant: %v, want ErrNotVisible", err)
	}
	if err := (authz.ACL{Policies: fakePolicies{}}).CanRead(ctx, tx, me(), "incident", row); !errors.Is(err, authz.ErrNotVisible) {
		t.Fatalf("no policy: %v, want ErrNotVisible (fail closed)", err)
	}
	boom := errors.New("boom")
	err := (authz.ACL{Policies: fakePolicies{err: boom}}).CanRead(ctx, tx, me(), "incident", row)
	if !errors.Is(err, boom) || errors.Is(err, authz.ErrNotVisible) {
		t.Fatalf("reader failure: %v, want it wrapped, not ErrNotVisible", err)
	}
}

func TestACLCanWrite(t *testing.T) {
	ctx := context.Background()
	tx := tenantTx{tenant: tenant().String()}
	// The service desk writes what is assigned to one of its groups.
	desk := authz.ACL{Policies: fakePolicies{pol: pol(rw("desk", authz.Subject{Groups: []uuid.UUID{g1()}},
		callerCond("assignment_group_id", "in", "my_groups")))}}
	mine := &authz.Row{ID: uuid.New(), State: "open", AssignmentGroupID: ptr(g2())}
	alsoMine := &authz.Row{ID: mine.ID, State: "open", AssignmentGroupID: ptr(g1())}
	notMine := &authz.Row{ID: mine.ID, State: "open", AssignmentGroupID: ptr(g3())}

	if err := desk.CanWrite(ctx, tx, me(), "incident", nil, mine); err != nil {
		t.Fatalf("create in scope: %v", err)
	}
	if err := desk.CanWrite(ctx, tx, me(), "incident", mine, alsoMine); err != nil {
		t.Fatalf("update within scope: %v", err)
	}
	if err := desk.CanWrite(ctx, tx, me(), "incident", mine, notMine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("moving the record out of scope: %v, want ErrForbidden", err)
	}
	if err := desk.CanWrite(ctx, tx, me(), "incident", notMine, mine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("pulling a record into scope: %v, want ErrForbidden", err)
	}
	if err := desk.CanWrite(ctx, tx, me(), "incident", nil, notMine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("create out of scope: %v, want ErrForbidden", err)
	}
	readOnly := authz.ACL{Policies: fakePolicies{pol: pol(authz.Rule{ID: "r", Ops: []string{"read"}, Who: everyone()})}}
	if err := readOnly.CanWrite(ctx, tx, me(), "incident", mine, mine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("read-only rule: %v, want ErrForbidden", err)
	}
	if err := (authz.ACL{Policies: fakePolicies{}}).CanWrite(ctx, tx, me(), "incident", nil, mine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("no policy: %v, want ErrForbidden (fail closed)", err)
	}
	foreign := me()
	foreign.TenantID = uuid.New()
	if err := desk.CanWrite(ctx, tx, foreign, "incident", nil, mine); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("other tenant: %v, want ErrForbidden", err)
	}
	if err := desk.CanWrite(ctx, tx, me(), "incident", mine, nil); err == nil || errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("no after row: %v, want a contract error", err)
	}
	boom := errors.New("boom")
	if err := (authz.ACL{Policies: fakePolicies{err: boom}}).CanWrite(ctx, tx, me(), "incident", nil, mine); !errors.Is(err, boom) {
		t.Fatalf("reader failure: %v, want it wrapped", err)
	}
}

// A uuid set binds as text: never nil, so an empty set is the empty array.
func TestUUIDTexts(t *testing.T) {
	for _, ids := range [][]uuid.UUID{nil, {}} {
		if got := authz.UUIDTexts(ids); got == nil || len(got) != 0 {
			t.Errorf("UUIDTexts(%v) = %#v, want an empty non-nil slice", ids, got)
		}
	}
	if got := authz.UUIDTexts([]uuid.UUID{g1(), g2()}); len(got) != 2 || got[0] != g1().String() || got[1] != g2().String() {
		t.Errorf("UUIDTexts = %v", got)
	}
}

// Every argument CompileRead binds encodes in pgx's exec mode, which has no
// parameter types: never a []uuid.UUID, including for an empty group set.
func TestCompileReadArgsEncodeWithoutTypes(t *testing.T) {
	noGroups := identity.Principal{TenantID: tenant(), PrincipalID: caller()}
	p := pol(rw("a", authz.Subject{Shared: true}, callerCond("assignment_group_id", "in", "my_groups")),
		rw("b", authz.Subject{Requester: true, Shared: true}, cond("requester_id", "in", other().String())))
	for _, principal := range []identity.Principal{me(), noGroups} {
		_, args := authz.CompileRead(p, principal, "r", 0)
		for _, a := range args {
			switch a.(type) {
			case uuid.UUID, []string:
			default:
				t.Errorf("argument of type %T does not encode in exec mode", a)
			}
		}
	}
}
