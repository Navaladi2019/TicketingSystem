package authz

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The under leaf (ADR-010 D7, D5a; PLAN-011 D-11-5; TICK-950 P11-17).
//
// {"under": {"of": "caller"}} holds when the record's org unit is in the live
// subtree of one of the caller's units (org_unit_members); {"under": {"of":
// [ids]}} when it is in the live subtree of one of the listed units. That
// subtree is set S. The planner is pure, so S is resolved before it runs:
// ExpandUnder plans once without S to find the under leaves that survive
// folding, and only if one does, runs one recursive query over live units for
// every root those leaves name, then binds the result to a copy of the policy.
// The leaf renders "org_unit_id IS NOT NULL AND org_unit_id = ANY($S)" in SQL
// and membership of Row.OrgUnitID in S in Go: one array argument, no join
// inside the row predicate, and a NULL org_unit_id is false on both sides.
//
// A tombstoned unit drops out of S with its whole subtree (D7), live
// descendants included (parent_id keeps linking tombstoned units into the
// tree): the walk down stops at it, and a seed with a tombstoned ancestor
// (a caller's own live unit, or a listed one, below a removed unit) is no seed
// at all. Whatever needs under to be true is removed (fails closed), and
// `not under` reads those records as outside the region.
//
// A policy reaching an under leaf that was not resolved for the principal it
// is planned for (ExpandUnder not called, or called for another principal) is
// voided with RuleUnderUnresolved: it grants nothing and is counted in
// authz_policy_invalid_total, never read as false under a not. The principal
// is its tenant, its id and its groups: which leaves fold away depends on the
// groups (member_of, shared), so a binding made for groups G1 and read for G2
// could read a leaf that survives under G2 as rest's empty S (TICK-1029).

// MaxUnderSet is the most org units one under leaf's set S may hold (D7).
// Above it ExpandUnder returns *UnderSetTooLargeError, which the API answers
// with 500 access_scope_too_large, and counts authz_under_set_too_large_total.
const MaxUnderSet = 10_000

// RuleUnderUnresolved voids a policy whose under leaf has no set S bound for
// the principal it is planned for. It is a planner code, never a publish
// problem: it means a caller skipped ExpandUnder.
const RuleUnderUnresolved = "under_unresolved"

// UnderSetTooLargeError is ExpandUnder's refusal: an under leaf's set S has
// more than MaxUnderSet units. It carries no id and no tenant.
type UnderSetTooLargeError struct {
	EntityType string
}

func (e *UnderSetTooLargeError) Error() string {
	return fmt.Sprintf("authz: an under set of entity type %q holds more than %d org units", e.EntityType, MaxUnderSet)
}

// underScope is set S resolved for one principal: the caller's live subtree
// and the live subtree of each listed root. A root that is not a live unit of
// the tenant maps to an empty subtree; a root missing from units was not
// resolved.
type underScope struct {
	tenant, principal uuid.UUID
	groups            [sha256.Size]byte // groupsFingerprint of the principal's groups
	callerOK          bool
	caller            []uuid.UUID
	units             map[uuid.UUID][]uuid.UUID
	// rest: every leaf not covered above (a root missing from units, the
	// caller when not callerOK) folded away in ExpandUnder's probe plan, so
	// its value cannot change any policy's result; it reads as an empty S.
	// The planner plans every child before folding, and would otherwise void
	// the policy over a leaf that does not matter (TICK-950 review).
	rest bool
}

// BindUnder returns a copy of pol with set S bound for p: caller is the live
// subtree of p's units, units the live subtree of each root an under leaf
// lists (an empty slice for a root that is not a live unit). It is what
// ExpandUnder binds after its query, exported so a test without a database can
// give the planner S from its own model of the tree. A nil pol stays nil.
func BindUnder(pol *Policy, p identity.Principal, caller []uuid.UUID, units map[uuid.UUID][]uuid.UUID) *Policy {
	if pol == nil {
		return nil
	}
	out := *pol
	sc := &underScope{tenant: p.TenantID, principal: p.PrincipalID, groups: groupsFingerprint(p.Groups), callerOK: true, caller: sortedIDs(caller), units: map[uuid.UUID][]uuid.UUID{}}
	for root, ids := range units {
		sc.units[root] = sortedIDs(ids)
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
// leaf that survives folding, so ExpandUnder will run its query.
func UnderNeeded(pol *Policy, p identity.Principal) bool {
	n := needs(pol, p)
	return len(n.leaves) > 0
}

// needs plans pol for p, read and write, without S, and collects the under
// leaves that survive folding in a valid policy.
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

// expandSQL is D7's one recursive query: seeds are the caller's units
// ($2, NULL for none) and the listed roots ($3). up walks each seed's
// ancestors; a seed that is tombstoned or has a tombstoned ancestor is
// dropped. The rest are walked down over live units only, so a tombstoned
// unit ends its branch. The tenant is filtered explicitly on top of
// row-level security. org_units_pkey serves the walk up, org_units_parent
// (tenant_id, parent_id) the walk down, org_unit_members_principal the
// caller's seed. Depth is bounded (groupsync refuses a chain deeper than 8),
// and UNION ends any cycle.
const expandSQL = `
WITH RECURSIVE seed(caller, root) AS (
    SELECT true, m.org_unit_id FROM org_unit_members m
    WHERE m.tenant_id = $1 AND m.principal_id = $2::uuid
    UNION
    SELECT false, x FROM unnest($3::uuid[]) x
), up(root, id, dead) AS (
    SELECT u.id, u.parent_id, u.deleted_at IS NOT NULL
    FROM org_units u WHERE u.tenant_id = $1 AND u.id IN (SELECT root FROM seed)
    UNION
    SELECT up.root, a.parent_id, a.deleted_at IS NOT NULL
    FROM up JOIN org_units a ON a.tenant_id = $1 AND a.id = up.id
), s(caller, root, id) AS (
    SELECT seed.caller, seed.root, seed.root
    FROM seed
    WHERE EXISTS (SELECT 1 FROM up WHERE up.root = seed.root)
      AND NOT EXISTS (SELECT 1 FROM up WHERE up.root = seed.root AND up.dead)
    UNION
    SELECT s.caller, s.root, c.id
    FROM s JOIN org_units c ON c.tenant_id = $1 AND c.parent_id = s.id AND c.deleted_at IS NULL
)
SELECT caller, root, id FROM s`

// ExpandUnder resolves set S for every under leaf of pol that survives
// folding for p, read or write, and returns pol with S bound. When no such
// leaf exists it returns pol unchanged and runs no query. It runs in tx, under
// the transaction's tenant; a principal of another tenant gets a binding the
// planner refuses (fail closed). A leaf whose S holds more than MaxUnderSet
// units returns *UnderSetTooLargeError and counts
// authz_under_set_too_large_total.
func ExpandUnder(ctx context.Context, tx pgx.Tx, pol *Policy, p identity.Principal) (*Policy, error) {
	need := needs(pol, p)
	if len(need.leaves) == 0 {
		// No leaf survives folding: S cannot change any result, so every
		// leaf reads as empty and no query runs.
		return bindRest(pol, p, nil, nil), nil
	}
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var principal any // NULL: no caller seed
	if need.caller && p.PrincipalID != uuid.Nil {
		principal = p.PrincipalID.String()
	}
	rows, err := tx.Query(ctx, expandSQL, tenant, principal, UUIDTexts(need.roots))
	if err != nil {
		return nil, fmt.Errorf("authz: expand under: %w", err)
	}
	var caller []uuid.UUID
	units := make(map[uuid.UUID][]uuid.UUID, len(need.roots))
	for _, r := range need.roots {
		units[r] = nil
	}
	var (
		isCaller bool
		root, id uuid.UUID
	)
	_, err = pgx.ForEachRow(rows, []any{&isCaller, &root, &id}, func() error {
		if isCaller {
			caller = append(caller, id)
		} else {
			units[root] = append(units[root], id)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("authz: expand under: %w", err)
	}
	bound := bindRest(pol, identity.Principal{TenantID: tenant, PrincipalID: p.PrincipalID, Groups: p.Groups}, caller, units)
	if !need.caller {
		bound.under.callerOK = false // folded away: rest, not a resolved set
	}
	for _, leaf := range need.leaves {
		if s, _ := bound.under.set(leaf); len(s) > MaxUnderSet {
			countUnderTooLarge(pol)
			return nil, &UnderSetTooLargeError{EntityType: pol.EntityType}
		}
	}
	return bound, nil
}

// bindRest is BindUnder for ExpandUnder: the sets it queried, and every
// other leaf (folded away in the probe plan) read as empty (underScope.rest).
func bindRest(pol *Policy, p identity.Principal, caller []uuid.UUID, units map[uuid.UUID][]uuid.UUID) *Policy {
	out := BindUnder(pol, p, caller, units)
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

// set is leaf's set S, sorted, and whether it was resolved. A caller leaf for
// a principal with no id is resolved and empty without a binding.
func (sc *underScope) set(leaf Under) ([]uuid.UUID, bool) {
	if sc == nil {
		return nil, false
	}
	if leaf.Caller {
		if !sc.callerOK {
			return nil, sc.rest
		}
		return sc.caller, true
	}
	var out []uuid.UUID
	for _, root := range leaf.Units {
		ids, ok := sc.units[root]
		if !ok {
			if !sc.rest {
				return nil, false
			}
			continue // folded away: empty
		}
		out = append(out, ids...)
	}
	return sortedIDs(out), true
}

// sortedIDs is ids sorted and without repeats, a new slice.
func sortedIDs(ids []uuid.UUID) []uuid.UUID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(out)
}

// inSet reports whether id is in s, which sortedIDs built.
func inSet(s []uuid.UUID, id uuid.UUID) bool {
	_, ok := slices.BinarySearchFunc(s, id, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return ok
}
