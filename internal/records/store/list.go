package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/google/uuid"
)

// ListQuery is one page of a list (P3-34). Where is a boolean SQL predicate
// over the records alias r whose placeholders are numbered from $3 (the
// store binds $1 = tenant and $2 = entity definition) and Args are its
// arguments: the caller puts the ACL predicate (authz.CompileRead) and the
// filters in it, and Total counts with exactly this predicate.
type ListQuery struct {
	TenantID, DefinitionID uuid.UUID
	Where                  string
	Args                   []any
	// SortColumn is a column listable accepts.
	SortColumn string
	Desc       bool
	// Keyset is a predicate over r placed after Where for the page only, with
	// placeholders numbered after Args; KeysetArgs are its arguments.
	Keyset     string
	KeysetArgs []any
	Limit      int

	// Branches, when it has two or more entries, are mutually exclusive
	// predicates over r (authz.ReadBranches) whose OR is the ACL; Where then
	// holds the filters alone. Each branch is read in index order with its own
	// LIMIT and the branches are merged, so a caller that sees a small share of
	// the tenant does not walk the whole sort index (TICK-644). Their rows and
	// order are those of Where AND (OR of Branches).
	Branches []string
	// CountDrivers, when set, has one entry per branch (or one for Where when
	// there are no branches): a CountDriver with a From drives that term's
	// count from the closure (authz.RegionDriver), so the count reads at most
	// TotalCap+1 records for it whatever the size of the region. The page
	// never uses them.
	CountDrivers []CountDriver
	// CountWhere, when set, replaces Where in the count statement, and
	// CountBranches, when set, replaces Branches (one per branch): the same
	// terms with a region leaf in its count form (authz.ListRead), binding
	// the same placeholders as the page's.
	CountWhere    string
	CountBranches []string
	// Materialize, when set, has one entry per page term (per branch, or one
	// for Where): true renders that term C3 (ADR-010-A2 A2-5), its rows read
	// into a MATERIALIZED CTE and then sorted, instead of walked in index
	// order. The rows are the same either way.
	Materialize []bool

	// BudgetCount is set for a list with a data filter (read through its side
	// table): its count runs under CountBudget in SAVEPOINT list_count.
	BudgetCount bool
	// CountDelay and SkipCount are test seams (ADR-006-A2 D-7A2-5): CountDelay
	// sleeps inside every count statement, SkipCount runs no count and leaves the
	// total unknown. The handler copies them only from fields that
	// internal/server/export_test.go sets.
	CountDelay time.Duration
	SkipCount  bool
}

// CountDriver drives one count term (authz.RegionDriver): From is a FROM
// item, Key the column of it that each driven record's org_unit_id equals.
// Its placeholders are among the term's own, so it binds nothing.
type CountDriver struct {
	From, Key string
}

// TieBreaksOnUpdatedAt reports whether a list sorted by col orders by
// (col, updated_at, id) rather than (col, id): the sort indexes of state and
// the foreign keys end in updated_at DESC, id DESC.
func TieBreaksOnUpdatedAt(col string) bool { return col != "created_at" && col != "updated_at" }

// listable reports whether col is a column a list may order by.
func listable(col string) bool {
	switch col {
	case "created_at", "updated_at", "state", "requester_id", "assignee_id", "assignment_group_id":
		return true
	}
	return false
}

// ListPage is a page of records and, for each, the sort column's text value
// (nil for NULL) that a keyset cursor carries.
type ListPage struct {
	Records  []*Record
	SortKeys []*string
	Total    int64
	// TotalCapped is true when more than TotalCap rows match and Total is TotalCap.
	TotalCapped bool
	// TotalState says what Total means: TotalExact or TotalCapped here.
	TotalState TotalState
}

// TotalState is how trustworthy ListPage.Total is (ADR-006-A2 A2-1).
type TotalState string

// The states of a list total. TotalUnknown (the count ran out of its time
// budget) is not produced yet.
const (
	TotalExact   TotalState = "exact"
	TotalCapped  TotalState = "capped"
	TotalUnknown TotalState = "unknown"
)

// TotalCap is the most the list counts: an exact count of a large tenant's
// matches alone misses the list p95 (P3-36), so past it Total is TotalCap and
// TotalCapped is set.
const TotalCap = 1000

// CountBudget is the time a list's count may take before its total is reported
// unknown instead (ADR-006-A2 A2-2). Not a setting.
const CountBudget = 140 * time.Millisecond

// List returns up to Limit records of the entity type matching Where and
// Keyset, ordered by SortColumn then id, and the count of rows matching Where
// alone. Row-level security still scopes both statements to the tenant.
func List(ctx context.Context, tx pgx.Tx, q ListQuery) (*ListPage, error) {
	if !listable(q.SortColumn) {
		return nil, fmt.Errorf("records/store: sort column %q is not listable", q.SortColumn)
	}
	// The page runs first and is read to the end (A2-3): a count that runs out
	// of budget then costs the caller only its total.
	sql, pageArgs, err := PageStatement(q)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("records/store: list records: %w", err)
	}
	defer rows.Close()
	page := &ListPage{}
	for rows.Next() {
		var key *string
		rec, err := scan(rowWithExtra{rows, &key})
		if err != nil {
			return nil, err
		}
		page.Records = append(page.Records, rec)
		page.SortKeys = append(page.SortKeys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("records/store: list records: %w", err)
	}
	rows.Close()

	switch {
	case q.SkipCount:
		page.TotalState = TotalUnknown
	case q.Keyset == "" && len(page.Records) <= q.Limit-1:
		// First-page shortcut (A2-4): Limit is the fetch size, limit+1, so at
		// most limit rows means no next page and the page is the whole result.
		page.Total, page.TotalState = int64(len(page.Records)), TotalExact
	default:
		if err := countInto(ctx, tx, q, page); err != nil {
			return nil, err
		}
	}
	return page, nil
}

// countInto runs the count of q and fills the page's total. With BudgetCount
// the count runs under CountBudget inside SAVEPOINT list_count (A2-2): a 57014
// on a live request leaves the total unknown instead of failing the list.
func countInto(ctx context.Context, tx pgx.Tx, q ListQuery, page *ListPage) error {
	args := append([]any{q.TenantID, q.DefinitionID}, q.Args...)
	// The count reads records alone: the joins of the page only add columns (one
	// definition, at most one group per row), so they cannot change the count.
	stmt := CountStatement(q)
	if q.CountDelay > 0 {
		stmt = `SELECT c.n FROM (` + stmt + `) c(n) CROSS JOIN LATERAL (SELECT pg_sleep(` +
			strconv.FormatFloat(q.CountDelay.Seconds(), 'f', 6, 64) + `)) s`
	}
	var total int64
	if !q.BudgetCount {
		if err := tx.QueryRow(ctx, stmt, args...).Scan(&total); err != nil {
			return fmt.Errorf("records/store: count records: %w", err)
		}
		setTotal(page, total)
		return nil
	}
	// One no-argument Exec: SET LOCAL cannot take a parameter and the value is
	// whole milliseconds.
	if _, err := tx.Exec(ctx, "SAVEPOINT list_count; SET LOCAL statement_timeout = '"+
		strconv.FormatInt(CountBudget.Milliseconds(), 10)+"'"); err != nil {
		return fmt.Errorf("records/store: open count budget: %w", err)
	}
	countErr := tx.QueryRow(ctx, stmt, args...).Scan(&total)
	// Always ROLLBACK TO, never RELEASE: it undoes the SET LOCAL and, after a
	// 57014, returns the aborted transaction to a usable state.
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT list_count"); err != nil {
		if countErr != nil {
			return fmt.Errorf("records/store: count records: %w", countErr)
		}
		return fmt.Errorf("records/store: close count budget: %w", err)
	}
	if countErr != nil {
		var pe *pgconn.PgError
		if ctx.Err() == nil && errors.As(countErr, &pe) && pe.Code == "57014" {
			page.Total, page.TotalState = 0, TotalUnknown
			return nil
		}
		return fmt.Errorf("records/store: count records: %w", countErr)
	}
	setTotal(page, total)
	return nil
}

func setTotal(page *ListPage, total int64) {
	page.Total, page.TotalCapped, page.TotalState = min(total, TotalCap), total > TotalCap, TotalExact
	if page.TotalCapped {
		page.TotalState = TotalCapped
	}
}

// rowWithExtra scans the record columns plus one trailing text column.
type rowWithExtra struct {
	rows  pgx.Rows
	extra **string
}

func (r rowWithExtra) Scan(dest ...any) error { return r.rows.Scan(append(dest, r.extra)...) }

// PageStatement is the SQL and arguments of the page statement of q. The
// tests EXPLAIN it, so what they check is what List runs.
func PageStatement(q ListQuery) (string, []any, error) {
	if !listable(q.SortColumn) {
		return "", nil, fmt.Errorf("records/store: sort column %q is not listable", q.SortColumn)
	}
	args := append(append(append([]any{q.TenantID, q.DefinitionID}, q.Args...), q.KeysetArgs...), q.Limit)
	keyset := ""
	if q.Keyset != "" {
		keyset = " AND " + q.Keyset
	}
	// The only ORDER BY shapes the sort indexes serve (P3-34a/b): descending is
	// the index forward, ascending the same index backward. created_at and
	// updated_at end in id; state and the foreign keys tie-break on updated_at.
	dir, nulls := "ASC", "NULLS LAST"
	if q.Desc {
		dir, nulls = "DESC", "NULLS FIRST"
	}
	order := dir + " " + nulls
	if TieBreaksOnUpdatedAt(q.SortColumn) {
		order += ", r.updated_at " + dir
	}
	order += ", r.id " + dir
	if branched(q) {
		return branchedPage(q, args, keyset, order), args, nil
	}
	if q.materialize(0) {
		// C3: the term's rows, then sorted; acl_m keeps the alias r so the
		// select list and the ORDER BY read as the walked form's.
		return `WITH acl_m AS MATERIALIZED (SELECT r.* FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where + keyset + `)
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.` + q.SortColumn + `::text FROM acl_m r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
ORDER BY r.` + q.SortColumn + ` ` + order + ` LIMIT $` + fmt.Sprint(len(args)), args, nil
	}
	return `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.` + q.SortColumn + `::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where + keyset +
		` ORDER BY r.` + q.SortColumn + ` ` + order + ` LIMIT $` + fmt.Sprint(len(args)), args, nil
}

// branched reports whether q is answered branch by branch.
func branched(q ListQuery) bool { return len(q.Branches) > 1 }

// CountStatement counts the rows of q (the tenant, definition, Where and ACL
// predicate), up to TotalCap+1; the tests and the bench EXPLAIN it. With Branches each branch counts apart,
// capped by itself, and the exclusive branches add up. A term with a
// CountDriver (the region branch) is counted from its driver
// (countArm), always, so it reads at most TotalCap+1 records (SP-1d decision
// 1 on TICK-946: K = 1,000 = TotalCap).
func CountStatement(q ListQuery) string {
	w := q.Where
	if q.CountWhere != "" {
		w = q.CountWhere
	}
	where := ` WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + w
	cap1 := fmt.Sprint(TotalCap + 1)
	if !branched(q) {
		return `SELECT count(*) FROM ` + countArm(q.driver(0), where, cap1) + ` c`
	}
	branches := q.Branches
	if len(q.CountBranches) == len(q.Branches) {
		branches = q.CountBranches
	}
	arms := make([]string, len(branches))
	for i, b := range branches {
		arms[i] = countArm(q.driver(i), where+` AND `+b, cap1)
	}
	return `SELECT count(*) FROM (SELECT 1 FROM (` + strings.Join(arms, ` UNION ALL `) + `) u LIMIT ` + cap1 + `) c`
}

// countArm is one capped count term over records filtered by where. Driven,
// it is P3: the driver's units and closure, then for each closure descendant
// a LATERAL read of that unit's records (records_org_unit), every one also
// meeting where, so the term's rows are exactly where's and the cap stops
// the read early.
func countArm(d CountDriver, where, cap1 string) string {
	if d.From == "" {
		return `(SELECT 1 FROM records r` + where + ` LIMIT ` + cap1 + `)`
	}
	return `(SELECT 1 FROM ` + d.From + ` CROSS JOIN LATERAL (SELECT 1 FROM records r` + where +
		` AND r.org_unit_id = ` + d.Key + ` LIMIT ` + cap1 + `) acl_x LIMIT ` + cap1 + `)`
}

// driver is the count driver of term i, or none.
func (q ListQuery) driver(i int) CountDriver {
	if i < len(q.CountDrivers) {
		return q.CountDrivers[i]
	}
	return CountDriver{}
}

// materialize reports whether page term i renders C3.
func (q ListQuery) materialize(i int) bool { return i < len(q.Materialize) && q.Materialize[i] }

// branchedPage is the page statement of a branched q: one arm per branch, each
// ordered and limited by the index it can use, merged by the outer ORDER BY
// and LIMIT, then joined to the definition and group of the at most Limit rows.
func branchedPage(q ListQuery, args []any, keyset, order string) string {
	limit := `LIMIT $` + fmt.Sprint(len(args))
	cols := `r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.` + q.SortColumn + ` AS sort_key`
	arms := make([]string, len(q.Branches))
	for i, b := range q.Branches {
		term := `SELECT ` + cols + ` FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where + ` AND ` + b + keyset
		if q.materialize(i) {
			// C3: the branch's rows, then sorted, never walked.
			arms[i] = `(WITH acl_m AS MATERIALIZED (` + term + `) SELECT * FROM acl_m r ORDER BY r.` + q.SortColumn + ` ` + order + ` ` + limit + `)`
			continue
		}
		arms[i] = `(` + term + ` ORDER BY r.` + q.SortColumn + ` ` + order + ` ` + limit + `)`
	}
	outer := strings.NewReplacer("r.updated_at", "updated_at", "r.id", "id").Replace(order)
	// The sort column's ORDER BY term leads; order starts with its direction.
	return `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
` + strings.Join(arms, "\nUNION ALL\n") + `
ORDER BY sort_key ` + outer + ` ` + limit + `
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key ` + strings.NewReplacer("updated_at", "x.updated_at", "id", "x.id").Replace(outer)
}
