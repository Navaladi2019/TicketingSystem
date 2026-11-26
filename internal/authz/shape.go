package authz

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The list page's shape for a region term (ADR-010-A2 A2-5, A2-5a, A2-9;
// TICK-1036). A list term driven by an under leaf (RegionDriver) has its page
// statement picked per request from a capped probe P3 of that leaf:
//
//   - C3, when the probe counts m ≤ K* records under U: the term's rows are
//     read into a MATERIALIZED CTE (the leaf as r.org_unit_id = ANY(ARRAY(
//     closure descendants of U)), then sorted. It cannot walk the sort index,
//     so it reads exactly m records.
//   - W0, above K*: the ordered walk of the sort index with one closure-PK
//     probe per row, EXISTS (… OFFSET 0); the OFFSET 0 keeps the probe a
//     SubPlan, never a semi-join.
//
// The count is always TICK-1034's CC (the region driver, capped at
// TotalCap+1, the leaf in its predicate form), whatever the probe says,
// except A2-5a: a type whose estimate N̂_e is at most K* skips the probe and
// renders W0 for the page and the count, with no driver.
//
// Which rows a caller sees never depends on any of this: every shape computes
// the same predicate (A2-6), so a stale estimate or a wrong probe count costs
// time, never rows. A probe error is a 500 and never falls back to another
// shape (fails closed).
//
// Security (A2-6, review 01a1fcad): P3 runs only for an ACL policy's under
// leaves, never for a request filter; the shape, the probe count, K* and N̂_e
// never reach a response, a log line or an error text; and the tenant every
// statement binds is the transaction's own (tenancy.TenantOf), the value set
// in app.tenant_id. N̂_e is a per-type count of records the caller may not be
// allowed to see: it only sets K*.

// ListShape is a region term's page shape.
type ListShape uint8

const (
	// ShapeWalk is W0, the OFFSET 0 walk: the zero value, and the shape of a
	// leaf that was not probed.
	ShapeWalk ListShape = iota
	// ShapeC3 is the materialized CTE.
	ShapeC3
)

// The clamp of K* (A2-9) and its coefficient c, measured by the A2-12 (a)
// crossover cells. K* is computed, never configured.
const (
	KStarMin = 1000
	KStarMax = 20000
	kStarC   = 0.8
)

// KStar is the density threshold of A2-9: clamp(0.8 × √(page × N̂_e), 1,000,
// 20,000), where page is the page statement's LIMIT. A missing estimate
// (known false) is the floor.
func KStar(page int, estimate int64, known bool) int {
	if !known || page <= 0 || estimate <= 0 {
		return KStarMin
	}
	k := math.Floor(kStarC * math.Sqrt(float64(page)*float64(estimate)))
	return int(min(max(k, KStarMin), KStarMax))
}

// rareType is A2-5a's test: a type with an estimate at most K* is rare. A
// missing estimate is not rare, so the probe runs.
func rareType(estimate int64, known bool, k int) bool {
	return known && estimate <= int64(k)
}

// ListShapes is the shape of each region term of one list request. It lives
// for that request only (A2-5): nothing is cached across requests, so it is
// keyed, by underKey, on the tenant and the bound U alone. The zero value
// renders every region term W0 with the count driven (no probe ran).
type ListShapes struct {
	// rare is A2-5a: W0 for the page and the count, no count driver.
	rare bool
	// fixed renders every region term as shape, whatever was probed.
	fixed bool
	shape ListShape
	c3    map[string]bool
}

// FixedListShapes renders every region term of a list as s, with the count
// driven, as if every probe had chosen it. It is for the agreement tests and
// the bench, which compare each shape's rows with Allows; the list route
// uses ChooseListShapes. Like every shape, it changes SQL text, not rows.
func FixedListShapes(s ListShape) ListShapes { return ListShapes{fixed: true, shape: s} }

// c3For reports whether the region term of leaf key renders C3.
func (s ListShapes) c3For(key string) bool {
	switch {
	case s.rare:
		return false
	case s.fixed:
		return s.shape == ShapeC3
	}
	return s.c3[key]
}

// regionLeaf is an under leaf that drives a list term: its tenant and U.
type regionLeaf struct {
	key    string
	tenant uuid.UUID
	units  []uuid.UUID
}

// estimateSQL reads N̂_e (A2-9, TICK-1070) in the list's transaction. The
// tenant is filtered explicitly on top of row-level security.
const estimateSQL = `SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2`

// ProbeStatement is P3 (A2-5) for K = k: the records under the canonical,
// distinct U ($2, a text array) of tenant $1 and entity definition $3, read
// through the closure one unit at a time with a per-unit LATERAL LIMIT, and
// capped at k + 1 in all. It reads at most k + 1 records whatever the size of
// the subtree. It takes no request filter: it measures the region alone
// (A2-6 requirement 1). It is exported so the tests EXPLAIN what runs.
func ProbeStatement(k int) string {
	a := underArgs{tenant: "$1", units: "$2"}
	lim := strconv.Itoa(k + 1)
	return `SELECT count(*) FROM (SELECT 1 FROM ` + regionFrom(a) +
		` CROSS JOIN LATERAL (SELECT 1 FROM records r WHERE r.tenant_id = $1::uuid AND r.entity_definition_id = $3::uuid` +
		` AND r.org_unit_id IS NOT NULL AND r.org_unit_id = acl_d.descendant_id LIMIT ` + lim + `) acl_x LIMIT ` + lim + `) acl_p`
}

// ErrProbe wraps a failed P3: the list answers 500 and never falls back to
// another shape (A2-6). Its text names no count, threshold or estimate.
var ErrProbe = errors.New("authz: under list probe failed")

// ChooseListShapes picks the page shape of each region term of pol's list
// for p (A2-5, A2-5a): it reads N̂_e for (tenant, def), computes K* from the
// page statement's LIMIT, and either applies A2-5a or runs P3 once per
// distinct region leaf, in tx, before the page and count statements. pol is
// the policy ResolveUnder bound. With no region term it reads nothing.
//
// It plans pol without counting a voided policy: ReadListShaped, from the
// same policy, counts it once (D5b).
func ChooseListShapes(ctx context.Context, tx pgx.Tx, pol *Policy, p identity.Principal, def uuid.UUID, page int) (ListShapes, error) {
	_, _, leaves := readList(plan(pol, p, OpRead), "r", 0, ListShapes{})
	if len(leaves) == 0 {
		return ListShapes{}, nil
	}
	// The tenant every statement binds is the transaction's, set in
	// app.tenant_id (A2-6 requirement 3); a leaf bound for another tenant
	// is refused, never probed.
	tenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return ListShapes{}, fmt.Errorf("authz: %w", err)
	}
	for _, l := range leaves {
		if l.tenant != tenant {
			return ListShapes{}, fmt.Errorf("%w: a region leaf is bound for another tenant", ErrProbe)
		}
	}
	var estimate int64
	known := true
	if err := tx.QueryRow(ctx, estimateSQL, tenant, def).Scan(&estimate); errors.Is(err, pgx.ErrNoRows) {
		known = false
	} else if err != nil {
		return ListShapes{}, fmt.Errorf("authz: read the record estimate: %w", err)
	}
	k := KStar(page, estimate, known)
	if rareType(estimate, known, k) {
		for range leaves {
			countShape(ctx, ShapeWalk)
		}
		return ListShapes{rare: true}, nil
	}
	out := ListShapes{c3: make(map[string]bool, len(leaves))}
	sql := ProbeStatement(k)
	for _, l := range leaves {
		start := time.Now()
		var m int64
		err := tx.QueryRow(ctx, sql, tenant, UUIDTexts(l.units), def).Scan(&m)
		observeProbe(ctx, time.Since(start))
		if err != nil {
			return ListShapes{}, fmt.Errorf("%w: %w", ErrProbe, err)
		}
		shape := ShapeWalk
		if m <= int64(k) {
			shape = ShapeC3
			out.c3[l.key] = true
		}
		countShape(ctx, shape)
	}
	return out, nil
}
