package authz

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The under leaf (ADR-010 D7 as amended by the R1 ruling on TICK-946, the
// formal text ADR-010-A2; PLAN-011; TICK-1034).
//
// {"under": {"of": "caller"}} holds when the record's org unit is at or below
// one of the caller's units (org_unit_members); {"under": {"of": [ids]}} when
// it is at or below one of the listed units. Those start units are U. Both
// renderings read the closure (org_unit_closure: for each live unit, every
// live unit at or below it, itself included), so nothing builds the subtree
// in Go and nothing walks the tree per request:
//
//   - SQL (CompileRead, ReadBranches): the record's unit is one of the
//     closure's descendants of U (compiler.under). The list count's region
//     term is driven from U through the closure instead (CC, RegionDriver),
//     capped at TotalCap + 1 records, and the list page's is C3 or W0 as the
//     capped probe P3 picks (shape.go, ADR-010-A2 A2-5).
//   - Go (Allows): the record's unit has a closure row whose ancestor is in U.
//     ResolveUnder reads, in one statement, the caller's units and those rows
//     for the records about to be checked, and binds them to a copy of the
//     policy.
//
// The closure holds live paths only: a tombstoned unit has no row, and
// neither has any unit below it. So a tombstoned unit and its subtree are
// outside every U's region, on both sides, and `not under` reads them as
// outside. A NULL org_unit_id is never under.
//
// A policy reaching an under leaf that was not resolved for the principal it
// is planned for (ResolveUnder not called, or called for another principal),
// or checked in Go against a record whose unit ResolveUnder did not read, is
// voided with RuleUnderUnresolved: it grants nothing and is counted in
// authz_policy_invalid_total, never read as false under a not. The principal
// is its tenant, its id and its groups: which leaves fold away depends on the
// groups (member_of, shared), so a binding made for groups G1 and read for G2
// could read a leaf that survives under G2 as rest's empty U (TICK-1029).

// RuleUnderUnresolved voids a policy whose under leaf has no binding for the
// principal it is planned for, or whose record's unit was not read. It is a
// planner code, never a publish problem: it means a caller skipped
// ResolveUnder or did not pass it the record.
const RuleUnderUnresolved = "under_unresolved"

// underScope is what ResolveUnder read for one principal: the caller's units
// (U of a caller leaf) and, for each record unit it was asked about, that
// unit's closure ancestors among every U the policy names. A listed leaf's U
// is the leaf's own ids and needs no read.
type underScope struct {
	tenant, principal uuid.UUID
	groups            [sha256.Size]byte // groupsFingerprint of the principal's groups
	callerOK          bool
	caller            []uuid.UUID
	// ancestors maps a record's org unit to its closure ancestors in the
	// union of every U. A unit missing from it was not read.
	ancestors map[uuid.UUID][]uuid.UUID
	// rest: the caller leaf, when not callerOK, folded away in ResolveUnder's
	// probe plan, so its value cannot change any policy's result; it reads as
	// an empty U. The planner plans every child before folding, and would
	// otherwise void the policy over a leaf that does not matter (TICK-950
	// review).
	rest bool
}

// BindUnder returns a copy of pol bound for p: caller is p's units (U of a
// caller leaf), and closure maps a record's org unit to its live ancestors,
// itself included (none for a tombstoned unit or one below a tombstone), as
// org_unit_closure holds them. It is what ResolveUnder binds after its query,
// exported so a test without a database can give the planner its own model
// of the tree. closure is copied, its lists are not: the caller must not
// change them afterwards. A nil pol stays nil.
func BindUnder(pol *Policy, p identity.Principal, caller []uuid.UUID, closure map[uuid.UUID][]uuid.UUID) *Policy {
	if pol == nil {
		return nil
	}
	out := *pol
	sc := &underScope{tenant: p.TenantID, principal: p.PrincipalID, groups: groupsFingerprint(p.Groups), callerOK: true,
		caller: sortedIDs(caller), ancestors: make(map[uuid.UUID][]uuid.UUID, len(closure))}
	for unit, ids := range closure {
		sc.ancestors[unit] = ids // only intersected, never ordered
	}
	out.under = sc
	return &out
}

// underNeeds is what the under leaves of a plan need resolved.
type underNeeds struct {
	caller bool
	roots  []uuid.UUID
	leaves []Under
}

// UnderNeeded reports whether a read or a write of pol by p reaches an under
// leaf that survives folding.
func UnderNeeded(pol *Policy, p identity.Principal) bool {
	n := needs(pol, p)
	return len(n.leaves) > 0
}

// needs plans pol for p, read and write, without a binding, and collects the
// under leaves that survive folding in a valid policy.
func needs(pol *Policy, p identity.Principal) underNeeds {
	var out underNeeds
	if pol == nil {
		return out
	}
	for _, op := range []string{OpRead, OpWrite} {
		pl := newPlanner(pol.Document, p)
		pl.probe = true
		for _, np := range pol.Document.Policies {
			if !slices.Contains(np.Ops, op) {
				continue
			}
			pp := pl.policy(np)
			if pp.err != nil {
				continue
			}
			pp.tree.walk(func(n pnode) {
				if n.op != pUnder || n.leaf == nil {
					return
				}
				out.leaves = append(out.leaves, *n.leaf)
				if n.leaf.Caller {
					out.caller = true
				}
				for _, id := range n.leaf.Units {
					if !slices.Contains(out.roots, id) {
						out.roots = append(out.roots, id)
					}
				}
			})
		}
	}
	return out
}

// resolveSQL reads, in one statement, the caller's units ($2, NULL for no
// caller leaf) and the closure rows from any U (the caller's units or the
// listed roots, $3) down to the record units $4: the Allows probe, a PK probe
// on org_unit_closure_pkey per (ancestor, record unit). The tenant is
// filtered explicitly on top of row-level security; the closure's RLS policy
// is the same tenant.
const resolveSQL = `
WITH m AS (
    SELECT org_unit_id FROM org_unit_members
    WHERE tenant_id = $1 AND principal_id = $2::uuid
)
SELECT true, org_unit_id, NULL::uuid FROM m
UNION ALL
SELECT false, c.ancestor_id, c.descendant_id FROM org_unit_closure c
WHERE c.tenant_id = $1 AND c.descendant_id = ANY($4::uuid[])
  AND (c.ancestor_id = ANY($3::uuid[]) OR c.ancestor_id IN (SELECT org_unit_id FROM m))`

// ResolveUnder binds what the under leaves of pol that survive folding for p,
// read or write, need: the caller's units for a caller leaf, and, for each
// row's org unit, whether it is at or below a U (the Allows probe). A list
// read passes no row. When no leaf survives, or there is nothing to read, it
// runs no query. It reads in tx, under the transaction's tenant; a principal
// of another tenant gets a binding the planner refuses (fail closed).
func ResolveUnder(ctx context.Context, tx pgx.Tx, pol *Policy, p identity.Principal, rows ...Row) (*Policy, error) {
	need := needs(pol, p)
	if len(need.leaves) == 0 {
		// No leaf survives folding: U cannot change any result, so every
		// leaf reads as empty and no query runs.
		return bindRest(pol, p, nil, nil), nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var units []uuid.UUID
	for _, r := range rows {
		if r.OrgUnitID != nil {
			units = append(units, *r.OrgUnitID)
		}
	}
	units = sortedIDs(units)
	var principal any // NULL: no caller seed
	if need.caller && p.PrincipalID != uuid.Nil {
		principal = p.PrincipalID.String()
	}
	ancestors := make(map[uuid.UUID][]uuid.UUID, len(units))
	for _, u := range units {
		ancestors[u] = nil // read: no row means outside every U
	}
	var caller []uuid.UUID
	if principal != nil || len(units) > 0 {
		q, err := tx.Query(ctx, resolveSQL, tenant, principal, UUIDTexts(need.roots), UUIDTexts(units))
		if err != nil {
			return nil, fmt.Errorf("authz: resolve under: %w", err)
		}
		var (
			isCaller   bool
			id         uuid.UUID
			descendant *uuid.UUID
		)
		_, err = pgx.ForEachRow(q, []any{&isCaller, &id, &descendant}, func() error {
			if isCaller {
				caller = append(caller, id)
			} else if descendant != nil {
				ancestors[*descendant] = append(ancestors[*descendant], id)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("authz: resolve under: %w", err)
		}
	}
	bound := bindRest(pol, identity.Principal{TenantID: tenant, PrincipalID: p.PrincipalID, Groups: p.Groups}, caller, ancestors)
	if !need.caller {
		bound.under.callerOK = false // folded away: rest, not a resolved set
	}
	return bound, nil
}

// bindRest is BindUnder for ResolveUnder: what it read, and a caller leaf
// folded away in the probe plan read as empty (underScope.rest).
func bindRest(pol *Policy, p identity.Principal, caller []uuid.UUID, closure map[uuid.UUID][]uuid.UUID) *Policy {
	out := BindUnder(pol, p, caller, closure)
	if out != nil {
		out.under.rest = true
	}
	return out
}

// scopeFor is pol's binding when it was made for p (tenant, id and groups),
// else nil.
func scopeFor(pol *Policy, p identity.Principal) *underScope {
	if pol == nil || pol.under == nil || pol.under.tenant != p.TenantID || pol.under.principal != p.PrincipalID ||
		pol.under.groups != groupsFingerprint(p.Groups) {
		return nil
	}
	return pol.under
}

// groupsFingerprint identifies a set of groups: SHA-256 over the ids sorted
// and without repeats, so the same groups in another order or listed twice
// keep the binding. member_of and shared read the groups as a set.
func groupsFingerprint(groups []uuid.UUID) [sha256.Size]byte {
	h := sha256.New()
	for _, g := range sortedIDs(groups) {
		h.Write(g[:])
	}
	var out [sha256.Size]byte
	h.Sum(out[:0])
	return out
}

// start is leaf's U, sorted, and whether it is resolved. A listed leaf's U is
// its ids; a caller leaf's is the caller's units, bound by ResolveUnder.
func (sc *underScope) start(leaf Under) ([]uuid.UUID, bool) {
	if sc == nil {
		return nil, false
	}
	if !leaf.Caller {
		return sortedIDs(leaf.Units), true
	}
	if !sc.callerOK {
		return nil, sc.rest
	}
	return sc.caller, true
}

// readRow reports whether the binding holds the closure answer for r: a NULL
// org unit needs none.
func (sc *underScope) readRow(r Row) bool {
	if r.OrgUnitID == nil {
		return true
	}
	if sc == nil {
		return false
	}
	_, ok := sc.ancestors[*r.OrgUnitID]
	return ok
}

// voidUnreadRow voids, for the Go check of r, every valid policy of planned
// that reaches an under leaf when the binding does not hold r's unit
// (RuleUnderUnresolved): reading it as false would make `not under` true.
func voidUnreadRow(pol *Policy, p identity.Principal, planned []plannedPolicy, r Row) {
	if scopeFor(pol, p).readRow(r) {
		return
	}
	for i := range planned {
		if planned[i].err == nil && planned[i].tree.hasUnder() {
			planned[i].err = &PlanError{Policy: planned[i].name, Rule: RuleUnderUnresolved}
		}
	}
}

// hasUnder reports whether n holds an under leaf.
func (n pnode) hasUnder() bool {
	found := false
	n.walk(func(k pnode) { found = found || k.op == pUnder })
	return found
}

// sortedIDs is ids sorted and without repeats, a new slice. Byte order is
// the order of the ids' canonical text (lower-case hex at fixed positions),
// so the bound arrays read the same as before, without formatting each id.
func sortedIDs(ids []uuid.UUID) []uuid.UUID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return slices.Compact(out)
}
