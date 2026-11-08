package authz

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// The record ACL (PLAN-003 S10, P3-31; ADR-001 I5: one authorization layer,
// inside the query). An entity type's published ACL is a Policy: allow-only
// rules that OR together. A rule admits a principal for an op when its
// subject names the principal and every one of its conditions holds on the
// record. No policy, an empty policy and a malformed rule all deny (fail
// closed).
//
// The same decision has two forms: Allows checks one record in Go, and
// CompileRead returns a SQL predicate a query puts in its WHERE. They must
// agree for every policy, principal and row (I5; P3-32 fuzzes it). Both are
// built on one resolution step, plan: it validates the policy, drops every
// rule that cannot match the principal and the op, and resolves everything
// that depends only on the principal (its groups, its id, self, my_groups)
// into fixed sets. What is left depends only on the row, and has exactly two
// renderings: plannedRule.allows (Go) and plannedRule.sql (SQL). Every case
// either rendering handles must be handled the same way in the other.

// The ops a rule grants.
const (
	OpRead  = "read"
	OpWrite = "write"
)

// The comparison operators of a Cond.
const (
	CondEq = "eq"
	CondIn = "in"
)

// The caller references of a Cond.
const (
	CallerSelf     = "self"      // the caller's principal id
	CallerMyGroups = "my_groups" // the caller's platform group ids
)

// Errors of the authorizer.
var (
	// ErrNotVisible means the principal may not read the record. The handler
	// answers it with the same 404 as a missing record.
	ErrNotVisible = errors.New("authz: record not visible")
)

// Subject is who a rule or mask names; any match counts.
type Subject struct {
	// Groups are platform group ids of this tenant; they match
	// identity.Principal.Groups only.
	Groups []uuid.UUID `json:"groups,omitempty"`
	Users  []uuid.UUID `json:"users,omitempty"`
	// Requester matches when the caller is records.requester_id.
	Requester bool `json:"requester,omitempty"`
	// Assignee matches when the caller is records.assignee_id.
	Assignee bool `json:"assignee,omitempty"`
	// Shared matches when a record_shares row names the caller (a user share)
	// or one of the caller's groups (a group share).
	Shared bool `json:"shared,omitempty"`
}

// Cond is one condition on a fixed column (the PLAN-003 cut: no data
// properties in M1).
//
// Column is state, requester_id, assignee_id or assignment_group_id. Op is eq
// (exactly one value) or in (any of the values; none matches nothing). With
// Caller set, Values must be empty and the column is compared against the
// caller instead: self (requester_id or assignee_id equals the caller's id)
// or my_groups (assignment_group_id is one of the caller's groups); Op is then
// eq or in, with the same meaning. A NULL column matches nothing. A Cond that
// breaks any of these rules makes its rule match nothing.
type Cond struct {
	Column string   `json:"column"`
	Op     string   `json:"op"`
	Values []string `json:"values,omitempty"`
	Caller string   `json:"caller,omitempty"`
}

// Rule is one allow rule: Who may perform Ops on the records where every
// condition in Where holds (an empty Where is every record).
type Rule struct {
	ID    string   `json:"id"`
	Ops   []string `json:"ops"`
	Who   Subject  `json:"who"`
	Where []Cond   `json:"where,omitempty"`
}

// Mask is a field mask (P3-33 enforces it): a top-level schema property hidden
// from everyone not in Read and unwritable for everyone not in Write.
type Mask struct {
	Field string  `json:"field"`
	Read  Subject `json:"read"`
	Write Subject `json:"write"`
}

// Policy is an entity type's published ACL version.
type Policy struct {
	EntityDefinitionID uuid.UUID
	Version            int
	Rules              []Rule
	Masks              []Mask
}

// Row is what a rule can see of one record: its fixed columns and its shares.
//
// SharedWith is pair-matched (A1, AR-S10-01): it holds only the principal ids
// of the record's shares whose (principal_kind, principal_id) names the
// caller, a user share for the caller's id or a group share for one of the
// caller's groups. SharesFor loads it that way; a test or fuzzer that builds a
// Row by hand must follow the same rule, because CompileRead matches the
// pair, never the id alone.
type Row struct {
	ID                                         uuid.UUID
	State                                      string
	RequesterID, AssigneeID, AssignmentGroupID *uuid.UUID
	SharedWith                                 []uuid.UUID
}

// Allows reports whether pol lets p perform op on r, and the id of the first
// rule, in policy order, that does. A nil policy allows nothing.
func Allows(pol *Policy, p identity.Principal, op string, r Row) (ruleID string, ok bool) {
	for _, pr := range plan(pol, p, op) {
		if pr.allows(r) {
			return pr.id, true
		}
	}
	return "", false
}

// CompileRead returns a boolean SQL predicate over the records row aliased
// alias that is true exactly for the rows Allows(pol, p, "read", row) admits,
// and its arguments. Placeholders are numbered from argBase+1, so argBase is
// the number of arguments the surrounding query already binds. It returns
// "false" and no arguments when no rule can match.
//
// Every id the predicate binds comes from pol or from p, never from a
// request. alias must be a plain SQL identifier; anything else panics (a
// programming error).
func CompileRead(pol *Policy, p identity.Principal, alias string, argBase int) (sql string, args []any) {
	if !identRE.MatchString(alias) {
		panic(fmt.Sprintf("authz: CompileRead alias %q is not a plain identifier", alias))
	}
	c := &compiler{alias: alias, base: argBase}
	var ors []string
	for _, pr := range plan(pol, p, OpRead) {
		ors = append(ors, pr.sql(c))
	}
	if len(ors) == 0 {
		return "false", nil
	}
	return "(" + strings.Join(ors, " OR ") + ")", c.args
}

var identRE = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// plannedRule is a rule resolved against one principal and op: what remains
// depends only on the row.
type plannedRule struct {
	id string
	// everyone: the rule's Groups or Users name the caller, so the subject
	// matches every row. Otherwise at least one of the row-dependent subjects
	// below is set.
	everyone bool
	// caller is the caller's principal id; the row-dependent subjects compare
	// against it, and are dropped when the caller has none.
	caller    uuid.UUID
	groups    []uuid.UUID // the caller's groups, for shared
	requester bool
	assignee  bool
	shared    bool
	conds     []plannedCond
}

// plannedCond is a condition resolved to a non-empty set: the column must be
// non-NULL and one of the values. A uuid column has ids, state has strs.
type plannedCond struct {
	column string
	ids    []uuid.UUID
	strs   []string
}

// plan resolves pol for p and op, keeping the policy's rule order. A rule that
// does not grant op, names nobody p can be, or has a condition that is
// invalid or resolves to an empty set is dropped: it matches no row.
func plan(pol *Policy, p identity.Principal, op string) []plannedRule {
	if pol == nil || (op != OpRead && op != OpWrite) {
		return nil
	}
	var out []plannedRule
	for _, r := range pol.Rules {
		pr, ok := planRule(r, p, op)
		if ok {
			out = append(out, pr)
		}
	}
	return out
}

func planRule(r Rule, p identity.Principal, op string) (plannedRule, bool) {
	if !slices.Contains(r.Ops, op) {
		return plannedRule{}, false
	}
	pr := plannedRule{id: r.ID, caller: p.PrincipalID, groups: p.Groups}
	pr.everyone = intersects(r.Who.Groups, p.Groups) ||
		(p.PrincipalID != uuid.Nil && slices.Contains(r.Who.Users, p.PrincipalID))
	if !pr.everyone && p.PrincipalID != uuid.Nil {
		pr.requester, pr.assignee, pr.shared = r.Who.Requester, r.Who.Assignee, r.Who.Shared
	}
	if !pr.everyone && !pr.requester && !pr.assignee && !pr.shared {
		return plannedRule{}, false
	}
	for _, c := range r.Where {
		pc, ok := planCond(c, p)
		if !ok {
			return plannedRule{}, false
		}
		pr.conds = append(pr.conds, pc)
	}
	return pr, true
}

func planCond(c Cond, p identity.Principal) (plannedCond, bool) {
	pc := plannedCond{column: c.Column}
	switch c.Op {
	case CondEq, CondIn:
	default:
		return pc, false
	}
	isState := c.Column == metadata.ColumnState
	switch c.Column {
	case metadata.ColumnState, metadata.ColumnRequesterID, metadata.ColumnAssigneeID, metadata.ColumnAssignmentGroupID:
	default:
		return pc, false
	}
	switch c.Caller {
	case "":
		if c.Op == CondEq && len(c.Values) != 1 {
			return pc, false
		}
		for _, v := range c.Values {
			if isState {
				pc.strs = append(pc.strs, v)
				continue
			}
			id, err := uuid.Parse(v)
			if err != nil {
				return pc, false
			}
			pc.ids = append(pc.ids, id)
		}
	case CallerSelf:
		if len(c.Values) > 0 || (c.Column != metadata.ColumnRequesterID && c.Column != metadata.ColumnAssigneeID) || p.PrincipalID == uuid.Nil {
			return pc, false
		}
		pc.ids = []uuid.UUID{p.PrincipalID}
	case CallerMyGroups:
		if len(c.Values) > 0 || c.Column != metadata.ColumnAssignmentGroupID {
			return pc, false
		}
		pc.ids = slices.Clone(p.Groups)
	default:
		return pc, false
	}
	if len(pc.ids) == 0 && len(pc.strs) == 0 {
		return pc, false // an empty set matches no row
	}
	return pc, true
}

// allows is the Go rendering: the subject matches the row, and so does every
// condition.
func (pr plannedRule) allows(r Row) bool {
	who := pr.everyone ||
		(pr.requester && eqID(r.RequesterID, pr.caller)) ||
		(pr.assignee && eqID(r.AssigneeID, pr.caller)) ||
		(pr.shared && sharedWithCaller(r.SharedWith, pr.caller, pr.groups))
	if !who {
		return false
	}
	for _, c := range pr.conds {
		if !c.allows(r) {
			return false
		}
	}
	return true
}

func (c plannedCond) allows(r Row) bool {
	var col *uuid.UUID
	switch c.column {
	case metadata.ColumnState:
		return slices.Contains(c.strs, r.State)
	case metadata.ColumnRequesterID:
		col = r.RequesterID
	case metadata.ColumnAssigneeID:
		col = r.AssigneeID
	case metadata.ColumnAssignmentGroupID:
		col = r.AssignmentGroupID
	}
	return col != nil && slices.Contains(c.ids, *col)
}

// sharedWithCaller is the Go side of the share subject. SharedWith is already
// pair-matched by SharesFor; matching it against the caller's id and groups
// again keeps a hand-built Row from granting more than the SQL does.
func sharedWithCaller(shared []uuid.UUID, caller uuid.UUID, groups []uuid.UUID) bool {
	for _, id := range shared {
		if id == caller || slices.Contains(groups, id) {
			return true
		}
	}
	return false
}

// compiler numbers placeholders and collects their arguments.
type compiler struct {
	alias string
	base  int
	args  []any
}

func (c *compiler) arg(v any) string {
	c.args = append(c.args, v)
	return "$" + strconv.Itoa(c.base+len(c.args))
}

// sql is the SQL rendering of allows, term for term. records.state is NOT
// NULL; a NULL uuid column is excluded explicitly, so no term is ever NULL
// and the predicate needs no three-valued reasoning.
func (pr plannedRule) sql(c *compiler) string {
	var who []string
	if pr.everyone {
		who = []string{"true"}
	} else {
		if pr.requester {
			who = append(who, c.colEq(metadata.ColumnRequesterID, pr.caller))
		}
		if pr.assignee {
			who = append(who, c.colEq(metadata.ColumnAssigneeID, pr.caller))
		}
		if pr.shared {
			who = append(who, c.shared(pr.caller, pr.groups))
		}
	}
	terms := []string{"(" + strings.Join(who, " OR ") + ")"}
	for _, cd := range pr.conds {
		terms = append(terms, cd.sql(c))
	}
	return "(" + strings.Join(terms, " AND ") + ")"
}

func (c *compiler) col(name string) string { return c.alias + "." + name }

func (c *compiler) colEq(column string, id uuid.UUID) string {
	return "(" + c.col(column) + " IS NOT NULL AND " + c.col(column) + " = " + c.arg(id) + "::uuid)"
}

// shared matches a share by its (principal_kind, principal_id) pair, never by
// the id alone (AR-S10-01), under the row's own tenant.
func (c *compiler) shared(caller uuid.UUID, groups []uuid.UUID) string {
	return "EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = " + c.col("tenant_id") +
		" AND acl_share.record_id = " + c.col("id") +
		" AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = " + c.arg(caller) + "::uuid)" +
		" OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY(" + c.arg(UUIDTexts(groups)) + "::uuid[]))))"
}

func (cd plannedCond) sql(c *compiler) string {
	if cd.column == metadata.ColumnState {
		return "(" + c.col(cd.column) + " = ANY(" + c.arg(cd.strs) + "::text[]))"
	}
	return "(" + c.col(cd.column) + " IS NOT NULL AND " + c.col(cd.column) + " = ANY(" + c.arg(UUIDTexts(cd.ids)) + "::uuid[]))"
}

// UUIDTexts is ids as the text array a uuid[] parameter is bound as. The
// server's pools run in QueryExecModeExec, where a parameter has no declared
// type and pgx cannot encode a []uuid.UUID; a []string with an explicit
// ::uuid[] cast always encodes, and an empty or nil ids is the empty array.
func UUIDTexts(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func eqID(col *uuid.UUID, id uuid.UUID) bool { return col != nil && *col == id }

func intersects(a, b []uuid.UUID) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}
	return false
}

// PolicyReader reads an entity type's current ACL in the caller's
// transaction. A nil policy means none is published: deny all (fail closed).
type PolicyReader interface {
	Current(ctx context.Context, tx pgx.Tx, entityKey string) (*Policy, error)
}

// Authorizer decides record reads and writes (A1). The pipeline and the
// records handlers call it inside the caller's tenancy.InTenantTx.
type Authorizer interface {
	// CanRead returns nil when p may read r, ErrNotVisible when it may not
	// (the handler's missing-record 404), and a wrapped error when it could
	// not decide (500).
	CanRead(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, r Row) error
	// CanWrite returns nil when p may write both before (nil on a create) and
	// after, ErrForbidden when it may not, and a wrapped error when it could
	// not decide. Requiring both stops a write from moving a record out of
	// the caller's write scope.
	CanWrite(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, before, after *Row) error
}
