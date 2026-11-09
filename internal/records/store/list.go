package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

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
}

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
}

// List returns up to Limit records of the entity type matching Where and
// Keyset, ordered by SortColumn then id, and the count of rows matching Where
// alone. Row-level security still scopes both statements to the tenant.
func List(ctx context.Context, tx pgx.Tx, q ListQuery) (*ListPage, error) {
	if !listable(q.SortColumn) {
		return nil, fmt.Errorf("records/store: sort column %q is not listable", q.SortColumn)
	}
	args := append([]any{q.TenantID, q.DefinitionID}, q.Args...)
	// The count reads records alone: the joins below only add columns (one
	// definition, at most one group per row), so they cannot change the count.
	where := ` WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ` + q.Where

	var total int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM records r`+where, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("records/store: count records: %w", err)
	}

	sql, pageArgs, err := PageStatement(q)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, sql, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("records/store: list records: %w", err)
	}
	defer rows.Close()
	page := &ListPage{Total: total}
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
	// The only ORDER BY shapes the sort indexes serve (P3-34a): descending is
	// the index forward, ascending the same index backward.
	order := "ASC NULLS LAST, r.id ASC"
	if q.Desc {
		order = "DESC NULLS FIRST, r.id DESC"
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
