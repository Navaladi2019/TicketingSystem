package authz

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Explanation is the answer of the access explainer (P3-35): whether a user
// may read a record under the entity type's newest ACL, and why.
type Explanation struct {
	Allow bool
	// RuleID names the first policy, in declared order, that grants read
	// (for a converted format-1 rule, its stored rule id); empty for a deny.
	RuleID string
	// Policy is the name of the first policy, in declared order, that grants
	// read; empty for a deny (D10). RuleID is its format-1 id when it has one.
	Policy string
	// Trace is the value on the row of each named condition reached: those of
	// the granting policy on an allow, those of every valid policy that grants
	// read on a deny (why none passed). The planner orders them.
	Trace []TraceStep
	// Invalid lists the policies granting read that the planner voided; they
	// grant nothing.
	Invalid []InvalidPolicy
	// Version is the ACL version evaluated; 0 when none is published.
	Version int
	// MaskedFields are the fields the policy hides from the target, leaving
	// out every field the asking admin cannot read itself.
	MaskedFields []string
}

// InvalidPolicy is a policy voided by the planner: its name, the problem code
// and the JSON pointer of the policy in the document.
type InvalidPolicy struct {
	Policy  string
	Code    string
	Pointer string
}

// Explain decides read for target on row under pol with the same Allows a
// read uses (so the explanation can never disagree with the enforcement), and
// lists target's masked fields that admin may read. A field admin cannot read
// is never listed: the explainer must not tell the admin that it exists.
func Explain(pol *Policy, admin, target identity.Principal, row Row) Explanation {
	var out Explanation
	if pol != nil {
		out.Version = pol.Version
	}
	// One plan feeds the decision, the trace and the invalid list, so they
	// describe the same trees Allows judged. A voided policy is counted once.
	planned := plan(pol, target, OpRead)
	countVoided(pol, planned)
	out.Policy, out.Trace, out.Allow = firstGrant(planned, row)
	out.RuleID = ruleID(pol, out.Policy)
	out.Invalid = []InvalidPolicy{}
	for _, pp := range planned {
		if pp.err != nil {
			out.Invalid = append(out.Invalid, invalidOf(pol, pp))
		} else if !out.Allow {
			out.Trace = append(out.Trace, pp.trace(row)...)
		}
	}
	out.MaskedFields = []string{}
	for _, f := range MaskedFields(pol, target) {
		if Readable(pol, admin, f) {
			out.MaskedFields = append(out.MaskedFields, f)
		}
	}
	return out
}

// invalidOf describes a voided policy; the pointer is its index in the
// document, never a value from it.
func invalidOf(pol *Policy, pp plannedPolicy) InvalidPolicy {
	var pe *PlanError
	out := InvalidPolicy{Policy: pp.name}
	if errors.As(pp.err, &pe) {
		out.Code = pe.Rule
	}
	for i, np := range pol.Document.Policies {
		if np.Name == pp.name {
			out.Pointer = fmt.Sprintf("/policies/%d", i)
			break
		}
	}
	return out
}

// ruleID is the explainer's name for a granting policy (D10: RuleID stays for
// one release): the stored rule id of a policy converted from a format-1 rule
// whose id is not a valid name, the policy name otherwise.
func ruleID(pol *Policy, name string) string {
	if pol == nil || name == "" {
		return ""
	}
	for _, np := range pol.Document.Policies {
		if np.Name == name && np.LegacyID != "" {
			return np.LegacyID
		}
	}
	return name
}

const targetSQL = `
SELECT p.id,
       ARRAY(SELECT gm.group_id FROM group_members gm
             JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id AND g.deleted_at IS NULL
             ORDER BY gm.group_id)
FROM principals p
WHERE p.tenant_id = $1 AND p.id = $2 AND p.deleted_at IS NULL`

// PrincipalOf is the principal a user of the transaction's tenant would be
// when signed in: its id and its live groups, read in tx. found is false for
// a user the tenant does not have (unknown, another tenant's, or removed from
// the directory).
func PrincipalOf(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (p identity.Principal, found bool, err error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return p, false, fmt.Errorf("authz: %w", err)
	}
	p.TenantID = tenant
	err = tx.QueryRow(ctx, targetSQL, tenant, userID).Scan(&p.PrincipalID, &p.Groups)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Principal{}, false, nil
	}
	if err != nil {
		return identity.Principal{}, false, fmt.Errorf("authz: read principal: %w", err)
	}
	return p, true, nil
}

const (
	groupsInTenantSQL = `SELECT id FROM groups WHERE tenant_id = $1 AND deleted_at IS NULL AND id = ANY($2::uuid[])`
	usersInTenantSQL  = `SELECT id FROM principals WHERE tenant_id = $1 AND deleted_at IS NULL AND id = ANY($2::uuid[])`
)

// NotInTenant returns a Problem for every ref whose id is not a live group (or
// user) of the transaction's tenant. A tombstoned id is refused like an unknown
// one. It filters by tenant_id itself, on top of row-level security, and
// answers the same way for an id that does not exist and an id of another
// tenant: the publish route must not become an oracle for ids of other
// tenants.
func NotInTenant(ctx context.Context, tx pgx.Tx, refs []TenantRef) ([]Problem, error) {
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("authz: %w", err)
	}
	var groups, users []uuid.UUID
	for _, r := range refs {
		switch r.Kind {
		case KindGroup:
			groups = append(groups, r.ID)
		case KindUser:
			users = append(users, r.ID)
		}
	}
	found := map[Kind]map[uuid.UUID]bool{KindGroup: {}, KindUser: {}}
	for _, q := range []struct {
		kind Kind
		sql  string
		ids  []uuid.UUID
	}{{KindGroup, groupsInTenantSQL, groups}, {KindUser, usersInTenantSQL, users}} {
		if len(q.ids) == 0 {
			continue
		}
		rows, err := tx.Query(ctx, q.sql, tenant, UUIDTexts(q.ids))
		if err != nil {
			return nil, fmt.Errorf("authz: check tenant ids: %w", err)
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, fmt.Errorf("authz: check tenant ids: %w", err)
			}
			found[q.kind][id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("authz: check tenant ids: %w", err)
		}
	}
	var out []Problem
	for _, r := range refs {
		if !found[r.Kind][r.ID] {
			out = append(out, Problem{r.Pointer, RuleNotTenant})
		}
	}
	return out, nil
}
