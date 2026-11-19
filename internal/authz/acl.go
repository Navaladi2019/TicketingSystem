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
)

// The record ACL (PLAN-003 S10, P3-31; ADR-001 I5: one authorization layer,
// inside the query). An entity type's published ACL is a Policy whose
// Document holds allow-only policies that OR together (ADR-010 D1): a policy
// admits a principal for an op when it grants the op and its when expression
// holds on the record. No policy, an empty document and an invalid policy all
// deny (fail closed).
//
// The same decision has two forms: Allows checks one record in Go, and
// CompileRead returns a SQL predicate a query puts in its WHERE. They must
// agree for every policy, principal and row (I5; P3-32 fuzzes it). Both are
// built on one resolution step, plan (plan.go): it expands every ref, voids
// every invalid policy, drops every policy that does not grant the op, and
// resolves everything that depends only on the principal (its groups, its id,
// self, my_groups, member_of) into constants and fixed sets. What is left
// depends only on the row, and has exactly two renderings: pnode.eval (Go) and
// pnode.sql (SQL). Every case either rendering handles must be handled the
// same way in the other, and no SQL term may be NULL (plan.go, D5a).

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

// Mask is a field mask (P3-33, mask.go): a top-level schema property hidden
// from everyone not in Read and unwritable for everyone not in Write. Read and
// Write match on Groups and Users only (A2): Requester, Assignee and Shared in
// a mask never match.
type Mask struct {
	Field string  `json:"field"`
	Read  Subject `json:"read"`
	Write Subject `json:"write"`
}

// Policy is an entity type's published ACL version. Document is the decoded
// rules (format 2, or format 1 converted on read). EntityType is the entity
// type's key, the label of authz_policy_invalid_total; PolicyStore sets it.
type Policy struct {
	EntityDefinitionID uuid.UUID
	EntityType         string
	Version            int
	Document           Document
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

// Allows reports whether pol lets p perform op on r: ok, the name of the
// first policy, in declared order, that grants it, and the value on r of each
// named condition that policy reaches (TraceStep). The answer is the OR over
// the policies granting op (D1); the order only decides which name is
// reported. A voided policy grants nothing and is counted. A nil policy
// allows nothing.
func Allows(pol *Policy, p identity.Principal, op string, r Row) (policy string, trace []TraceStep, ok bool) {
	for _, pp := range plan(pol, p, op) {
		if pp.err != nil {
			countInvalid(pol, pp)
			continue
		}
		if !ok && pp.tree.eval(r) {
			policy, trace, ok = pp.name, pp.trace(r), true
		}
	}
	return policy, trace, ok
}

// CompileRead returns a boolean SQL predicate over the records row aliased
// alias that is true exactly for the rows Allows(pol, p, "read", row) admits,
// and its arguments. Placeholders are numbered from argBase+1, so argBase is
// the number of arguments the surrounding query already binds. It returns
// "false" and no arguments when no policy can match.
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
	for _, pp := range plan(pol, p, OpRead) {
		if pp.err != nil {
			countInvalid(pol, pp)
			continue
		}
		ors = append(ors, pp.tree.sql(c))
	}
	if len(ors) == 0 {
		return "false", nil
	}
	return "(" + strings.Join(ors, " OR ") + ")", c.args
}

// ReadBranches is CompileRead's predicate split into branches for a query that
// answers each branch from its own index and merges them (TICK-644). Their OR
// admits exactly the rows CompileRead admits and no row matches two of them:
// each branch carries the negation of the branches before it, so a UNION ALL of
// the branches repeats no row. The placeholders and arguments follow
// CompileRead's contract (numbered from argBase+1, every id from pol or p).
//
// The split (ADR-010 D6) is on the OR across policies, then on each policy's
// folded tree: a top-level any gives one branch per child, and a top-level all
// with exactly one any child gives one branch per child of that any, each
// ANDed with the all's other children (today's per-subject disjuncts of a
// converted format-1 rule). A policy whose tree folds to true admits every
// row, so the whole predicate is the single branch "true". No branch means no
// policy can match; callers fall back to CompileRead when there are fewer
// than two branches. The SP-2 total-branch threshold is TICK-938 P11-06's.
//
// Exclusion needs every term to be two-valued: the planner never renders a
// term that can be NULL (plan.go), so NOT d_j is true exactly when d_j is false
// and no row is dropped by "AND NOT". A term that could be NULL would silently
// remove its rows from every later branch (Security review risk 2); keep new
// terms two-valued. The text grows with the square of the number of branches
// (branch i repeats the i-1 before it).
func ReadBranches(pol *Policy, p identity.Principal, alias string, argBase int) (branches []string, args []any) {
	if !identRE.MatchString(alias) {
		panic(fmt.Sprintf("authz: ReadBranches alias %q is not a plain identifier", alias))
	}
	c := &compiler{alias: alias, base: argBase}
	planned := plan(pol, p, OpRead)
	for _, pp := range planned {
		if pp.err == nil && pp.tree.op == pTrue {
			for _, q := range planned {
				if q.err != nil {
					countInvalid(pol, q)
				}
			}
			return []string{"true"}, nil
		}
	}
	var plain []string
	for _, pp := range planned {
		if pp.err != nil {
			countInvalid(pol, pp)
			continue
		}
		plain = append(plain, pp.tree.disjuncts(c)...)
	}
	for i, d := range plain {
		b := d
		for _, prev := range plain[:i] {
			b += " AND NOT " + prev
		}
		branches = append(branches, "("+b+")")
	}
	return branches, c.args
}

// disjuncts is the tree's SQL split as ReadBranches describes; their OR is
// n.sql. The other children of a distributed all are rendered, and their
// arguments bound, once, and their text shared.
func (n pnode) disjuncts(c *compiler) []string {
	switch n.op {
	case pAny:
		out := make([]string, len(n.kids))
		for i, k := range n.kids {
			out[i] = k.sql(c)
		}
		return out
	case pAll:
		at := -1
		for i, k := range n.kids {
			if k.op == pAny {
				if at >= 0 {
					return []string{n.sql(c)}
				}
				at = i
			}
		}
		if at < 0 {
			return []string{n.sql(c)}
		}
		subjects := make([]string, len(n.kids[at].kids))
		for i, k := range n.kids[at].kids {
			subjects[i] = k.sql(c)
		}
		rest := make([]string, len(n.kids))
		for i, k := range n.kids {
			if i != at {
				rest[i] = k.sql(c)
			}
		}
		out := make([]string, len(subjects))
		for i, s := range subjects {
			terms := slices.Clone(rest)
			terms[at] = s
			out[i] = "(" + strings.Join(terms, " AND ") + ")"
		}
		return out
	}
	return []string{n.sql(c)}
}

var identRE = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

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
	// CanWriteFields is the field-mask write check (P3-33), a separate
	// method so the A1 row check keeps its shape. before is the stored
	// payload (nil on a create), after the caller's data without defaults,
	// patch the caller's merge patch (nil on a create; see CheckWriteFields).
	// It returns nil, *ErrFieldNotWritable naming the fields p may not write,
	// ErrForbidden, or a wrapped error when it could not decide.
	CanWriteFields(ctx context.Context, tx pgx.Tx, p identity.Principal, entity string, before, after, patch map[string]any) error
}
