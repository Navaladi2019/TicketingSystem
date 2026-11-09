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
	// DataFilter is true when Where filters on a data property: such a list has
	// no index to lean on, so both statements run under Timeout.
	DataFilter bool
	// Timeout bounds a DataFilter list; zero means DataFilterTimeout.
	Timeout time.Duration

	// Branches, when it has two or more entries, are mutually exclusive
	// predicates over r (authz.ReadBranches) whose OR is the ACL; Where then
	// holds the filters alone. Each branch is read in index order with its own
	// LIMIT and the branches are merged, so a caller that sees a small share of
	// the tenant does not walk the whole sort index (TICK-644). Their rows and
	// order are those of Where AND (OR of Branches).
	Branches []string
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
}

// TotalCap is the most the list counts: an exact count of a large tenant's
// matches alone misses the list p95 (P3-36), so past it Total is TotalCap and
// TotalCapped is set.
const TotalCap = 1000

// DataFilterTimeout is the statement timeout of a list with a data filter (TICK-647).
const DataFilterTimeout = 2 * time.Second

// ErrStatementTimeout is returned when a DataFilter list ran past its timeout.
var ErrStatementTimeout = errors.New("records/store: list statement timed out")

// List returns up to Limit records of the entity type matching Where and
// Keyset, ordered by SortColumn then id, and the count of rows matching Where
// alone. Row-level security still scopes both statements to the tenant.
func List(ctx context.Context, tx pgx.Tx, q ListQuery) (*ListPage, error) {
	if !listable(q.SortColumn) {
		return nil, fmt.Errorf("records/store: sort column %q is not listable", q.SortColumn)
	}
	restore, err := boundDataFilter(ctx, tx, q)
	if err != nil {
		return nil, err
	}
	args := append([]any{q.TenantID, q.DefinitionID}, q.Args...)
	// The count reads records alone: the joins below only add columns (one
	// definition, at most one group per row), so they cannot change the count.
	var total int64
	if err := tx.QueryRow(ctx, CountStatement(q), args...).Scan(&total); err != nil {
		return nil, timedOut(ctx, fmt.Errorf("records/store: count records: %w", err))
	}

	sql, pageArgs, err := PageStatement(q)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, pageArgs...)
	if err != nil {
		return nil, timedOut(ctx, fmt.Errorf("records/store: list records: %w", err))
	}
	defer rows.Close()
	page := &ListPage{Total: min(total, TotalCap), TotalCapped: total > TotalCap}
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
		return nil, timedOut(ctx, fmt.Errorf("records/store: list records: %w", err))
	}
	rows.Close()
	if err := restore(); err != nil {
		return nil, err
	}
	return page, nil
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
	return `SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.` + q.SortColumn + `::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where + keyset +
		` ORDER BY r.` + q.SortColumn + ` ` + order + ` LIMIT $` + fmt.Sprint(len(args)), args, nil
}

// boundDataFilter sets the transaction's statement_timeout for a DataFilter
// list and returns the func that puts the earlier value back, so the handler's
// later statements in the same transaction are not bound. SET LOCAL ends with
// the transaction either way.
func boundDataFilter(ctx context.Context, tx pgx.Tx, q ListQuery) (func() error, error) {
	if !q.DataFilter {
		return func() error { return nil }, nil
	}
	d := q.Timeout
	if d <= 0 {
		d = DataFilterTimeout
	}
	var prior string
	if err := tx.QueryRow(ctx, `SELECT current_setting('statement_timeout')`).Scan(&prior); err != nil {
		return nil, fmt.Errorf("records/store: read statement_timeout: %w", err)
	}
	set := func(v string) error {
		// SET cannot take a parameter. v is whole milliseconds or the server's own
		// earlier setting, quoted as a literal.
		_, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+strings.ReplaceAll(v, "'", "''")+"'")
		return err
	}
	if err := set(strconv.FormatInt(d.Milliseconds(), 10)); err != nil {
		return nil, fmt.Errorf("records/store: set statement_timeout: %w", err)
	}
	return func() error {
		if err := set(prior); err != nil {
			return fmt.Errorf("records/store: restore statement_timeout: %w", err)
		}
		return nil
	}, nil
}

// timedOut wraps err with ErrStatementTimeout when Postgres cancelled the
// statement (57014) while the request itself is still live.
func timedOut(ctx context.Context, err error) error {
	var pe *pgconn.PgError
	if ctx.Err() == nil && errors.As(err, &pe) && pe.Code == "57014" {
		return fmt.Errorf("%w: %w", ErrStatementTimeout, err)
	}
	return err
}

// branched reports whether q is answered branch by branch.
func branched(q ListQuery) bool { return len(q.Branches) > 1 }

// CountStatement counts the rows of q (the tenant, definition, Where and ACL
// predicate), up to TotalCap+1; the tests and the bench EXPLAIN it. With Branches each branch counts apart,
// capped by itself, and the exclusive branches add up.
func CountStatement(q ListQuery) string {
	where := ` WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where
	cap1 := fmt.Sprint(TotalCap + 1)
	if !branched(q) {
		return `SELECT count(*) FROM (SELECT 1 FROM records r` + where + ` LIMIT ` + cap1 + `) c`
	}
	arms := make([]string, len(q.Branches))
	for i, b := range q.Branches {
		arms[i] = `(SELECT 1 FROM records r` + where + ` AND ` + b + ` LIMIT ` + cap1 + `)`
	}
	return `SELECT count(*) FROM (SELECT 1 FROM (` + strings.Join(arms, ` UNION ALL `) + `) u LIMIT ` + cap1 + `) c`
}

// branchedPage is the page statement of a branched q: one arm per branch, each
// ordered and limited by the index it can use, merged by the outer ORDER BY
// and LIMIT, then joined to the definition and group of the at most Limit rows.
func branchedPage(q ListQuery, args []any, keyset, order string) string {
	limit := `LIMIT $` + fmt.Sprint(len(args))
	cols := `r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.` + q.SortColumn + ` AS sort_key`
	arms := make([]string, len(q.Branches))
	for i, b := range q.Branches {
		arms[i] = `(SELECT ` + cols + ` FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where + ` AND ` + b + keyset +
			` ORDER BY r.` + q.SortColumn + ` ` + order + ` ` + limit + `)`
	}
	outer := strings.NewReplacer("r.updated_at", "updated_at", "r.id", "id").Replace(order)
	// The sort column's ORDER BY term leads; order starts with its direction.
	return `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
` + strings.Join(arms, "\nUNION ALL\n") + `
ORDER BY sort_key ` + outer + ` ` + limit + `
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key ` + strings.NewReplacer("updated_at", "x.updated_at", "id", "x.id").Replace(outer)
}
