package server

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// ListPageStatement is the page statement the handler runs for sort ("" is
// the default) with or without a cursor, without an ACL or filter, so a plan
// test explains exactly the ORDER BY and keyset predicate the handler builds.
func ListPageStatement(t *testing.T, sort string, withCursor bool, tenant, definition uuid.UUID) (string, []any) {
	t.Helper()
	var s *string
	if sort != "" {
		s = &sort
	}
	pl, perr := planList(listTestDef(t), nil, s, nil)
	if perr != nil {
		t.Fatal(perr)
	}
	q := store.ListQuery{TenantID: tenant, DefinitionID: definition, Where: "true",
		SortColumn: pl.sortCol, Desc: pl.desc, Limit: 51}
	if withCursor {
		val := "new"
		switch pl.sortCol {
		case "created_at", "updated_at":
			val = "2026-11-09 03:39:54.145534+00"
		case "state":
		default:
			val = uuid.NewString()
		}
		q.Keyset, q.KeysetArgs = pl.keyset(listPos{Sort: pl.sortProp, Desc: pl.desc, Val: &val, ID: uuid.NewString()}, 2)
	}
	sql, args, err := store.PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	return sql, args
}
