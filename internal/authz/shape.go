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

// The list's shape for a region term (ADR-010-A2 A2-5, A2-5a, A2-9, TICK-1036;
// ADR-010-A3, TICK-1104). A list term driven by an under leaf (RegionDriver)
// has its page statement picked per request from a capped probe P3 of that
// leaf:
//
//   - C3, when the probe counts m ≤ K* records under U: the term's rows are
//     read into a MATERIALIZED CTE (the leaf as r.org_unit_id = ANY(ARRAY(
//     closure descendants of U)), then sorted. It cannot walk the sort index,
//     so it reads exactly m records.
//   - W0, above K*: the ordered walk of the sort index with one closure-PK
//     probe per row, EXISTS (… OFFSET 0); the OFFSET 0 keeps the probe a
//     SubPlan, never a semi-join.
//
// The count is TICK-1034's CC (the region driver, capped at TotalCap+1, the
// leaf in its predicate form), whatever P3 says, with two exceptions that have
// no driver:
//
//   - A2-5a: a type whose estimate N̂_e is at most K* skips the probe and
//     renders W0 for the page and the count.
//   - R (A3-1, A3-2): a type that is small next to the subtree reads its own
//     records, page and count, and keeps those whose unit is in the subtree's
//     hashed set. A closure count n of U, folded into P3's statement, picks it:
//     ⌈N̂_e/α⌉ ≤ n < U_hash.
//
// Which rows a caller sees never depends on any of this: every shape computes
// the same predicate (A2-6), so a stale estimate or a wrong probe count costs
// time, never rows. A probe error is a 500 and never falls back to another
// shape (fails closed).
//
// Security (A2-6, review 01a1fcad; A3-7): P3 and the folded probe run only for
// an ACL policy's under leaves, never for a request filter; the shape, the
// probe count, the closure count n, R's cap, K*, N̂_e, α and U_cert never reach
// a response, a log line or an error text; and the tenant every statement
// binds is the transaction's own (tenancy.TenantOf), the value set in
// app.tenant_id. N̂_e is a per-type count of records the caller may not be
// allowed to see: it only sets K* and R's cap.

// ListShape is a region term's page shape.
type ListShape uint8

const (
	// ShapeWalk is W0, the OFFSET 0 walk: the zero value, and the shape of a
	// leaf that was not probed.
	ShapeWalk ListShape = iota
	// ShapeC3 is the materialized CTE.
	ShapeC3
	// ShapeRecords is R, the records-driven page and count (A3-1).
	ShapeRecords
)

// The bounds of R (ADR-010-A3 A3-3). Like K* they guard a budget every tenant
// shares, so they are computed, never configured.
const (
	// alpha is where R and A2's shapes cost the same: N̂_e = α·|S| (SP-1f).
	alpha = 3.3
	// uCert is the unit count ADR-010-A2 is certified to (A2-7), the threshold
	// of the alert GroupsyncOrgUnitsAboveCertified. A type above α·uCert never
	// gets R, so it does not pay the closure count.
	uCert = 10111
	// uHash is the subtree size from which R never runs: its closure set must
	// stay a hashed SubPlan.
	uHash = 2 * uCert

	// alphaTenths is α in integers, and maxRecordsEstimate the largest N̂_e
	// with N̂_e ≤ α·uCert; comparing with it cannot overflow.
	alphaTenths        int64 = alpha * 10
	maxRecordsEstimate       = alphaTenths * uCert / 10
)

// recordsCap is ⌈N̂_e/α⌉, the fewest units for which R is chosen.
func recordsCap(estimate int64) int64 { return (10*estimate + alphaTenths - 1) / alphaTenths }

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
	// chosen is the shape of each probed leaf; a leaf not in it is W0.
	chosen map[string]ListShape
}

// FixedListShapes renders every region term of a list as s, with that shape's
// count (driven, or R's own), as if every probe had chosen it. It is for the agreement tests and
// the bench, which compare each shape's rows with Allows; the list route
// uses ChooseListShapes. Like every shape, it changes SQL text, not rows.
func FixedListShapes(s ListShape) ListShapes { return ListShapes{fixed: true, shape: s} }

// of is the shape of the region term of leaf key.
func (s ListShapes) of(key string) ListShape {
	switch {
	case s.rare:
		return ShapeWalk
	case s.fixed:
		return s.shape
	}
	return s.chosen[key]
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

// FoldedProbeStatement is the folded probe (A3-2 step 4) for K* = k: the
// closure rows n of the canonical U ($2) of tenant $1, counted up to U_hash
// ($5), and P3 of entity definition $3, only when n picks no R. R is
// $4 ≤ n < $5, with $4 = ⌈N̂_e/α⌉. P3 is an InitPlan that only the ELSE reads,
// so it never executes on an R path. It is exported so the tests run and
// EXPLAIN what ChooseListShapes runs.
func FoldedProbeStatement(k int) string {
	return `SELECT acl_s.n, CASE WHEN acl_s.n >= $4::bigint AND acl_s.n < $5::bigint THEN NULL ELSE (` + ProbeStatement(k) +
		`) END FROM (SELECT count(*) AS n FROM (SELECT 1 FROM ` + regionFrom(underArgs{tenant: "$1", units: "$2"}) +
		` LIMIT $5::bigint) acl_z) acl_s`
}

// ErrProbe wraps a failed P3 or folded probe: the list answers 500 and never falls back to
// another shape (A2-6). Its text names no count, threshold or estimate.
var ErrProbe = errors.New("authz: under list probe failed")

// ChooseListShapes picks the shape of each region term of pol's list for p
// (A2-5, A2-5a, A3-2): it reads N̂_e for (tenant, def), computes K* from the
// page statement's LIMIT, and either applies A2-5a or runs one probe per
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
	out := ListShapes{chosen: make(map[string]ListShape, len(leaves))}
	for _, l := range leaves {
		start := time.Now()
		shape, err := probeLeaf(ctx, tx, l, def, estimate, known, k)
		observeProbe(ctx, time.Since(start))
		if err != nil {
			return ListShapes{}, err
		}
		out.chosen[l.key] = shape
		countShape(ctx, shape)
	}
	return out, nil
}

// probeLeaf is A3-2 steps 2 to 4 for a leaf of a type that is not rare: the
// folded probe when R is possible, else P3 alone, as in A2-5. l is bound for
// the transaction's tenant (ChooseListShapes checked it).
func probeLeaf(ctx context.Context, tx pgx.Tx, l regionLeaf, def uuid.UUID, estimate int64, known bool, k int) (ListShape, error) {
	var m int64
	if !known || estimate > maxRecordsEstimate {
		if err := tx.QueryRow(ctx, ProbeStatement(k), l.tenant, UUIDTexts(l.units), def).Scan(&m); err != nil {
			return ShapeWalk, fmt.Errorf("%w: %w", ErrProbe, err)
		}
	} else {
		atLeast := recordsCap(estimate)
		n, p3, err := foldedProbe(ctx, tx, l, def, k, atLeast, uHash)
		switch {
		case err != nil:
			return ShapeWalk, err
		case atLeast <= n && n < uHash:
			return ShapeRecords, nil
		case p3 == nil:
			return ShapeWalk, fmt.Errorf("%w: the probe returned no value", ErrProbe)
		}
		m = *p3
	}
	if m <= int64(k) {
		return ShapeC3, nil
	}
	return ShapeWalk, nil
}

// foldedProbe runs FoldedProbeStatement for leaf l: the closure count n,
// stopped at below, and P3's value, nil when atLeast ≤ n < below.
func foldedProbe(ctx context.Context, tx pgx.Tx, l regionLeaf, def uuid.UUID, k int, atLeast, below int64) (n int64, m *int64, err error) {
	if err := tx.QueryRow(ctx, FoldedProbeStatement(k), l.tenant, UUIDTexts(l.units), def, atLeast, below).Scan(&n, &m); err != nil {
		return 0, nil, fmt.Errorf("%w: %w", ErrProbe, err)
	}
	return n, m, nil
}
