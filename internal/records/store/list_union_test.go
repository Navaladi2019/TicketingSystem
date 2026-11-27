package store

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func unionQuery() ListQuery {
	return ListQuery{
		TenantID: uuid.New(), DefinitionID: uuid.New(), SortColumn: "updated_at", Desc: true, Limit: 51,
		Where: "w1", UnionWhere: "w2", Args: []any{1, 2},
	}
}

// A UNION ALL page has one arm per predicate, each with the same keyset, and
// the outer query orders and limits again (ADR-009 D12).
func TestPageStatementUnionArms(t *testing.T) {
	q := unionQuery()
	q.Keyset, q.KeysetArgs = "r.id < $5", []any{uuid.New()}
	sql, args, err := PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(sql, "FROM records r") != 2 || !strings.Contains(sql, "w1") || !strings.Contains(sql, "w2") || strings.Count(sql, "r.id < $5") != 2 {
		t.Errorf("page = %s", sql)
	}
	if !strings.Contains(sql, "UNION ALL") || len(args) != 2+2+1+1 {
		t.Errorf("page has %d args, want tenant, definition, args, keyset, limit: %s", len(args), sql)
	}
}

func TestPageStatementUnionWithBranchesAndMaterialize(t *testing.T) {
	q := unionQuery()
	q.Branches = []string{"b1", "b2"}
	q.Materialize = []bool{false, true}
	sql, _, err := PageStatement(q)
	if err != nil {
		t.Fatal(err)
	}
	// 2 predicates x 2 branches
	if strings.Count(sql, "FROM records r") != 4 || strings.Count(sql, "MATERIALIZED") != 2 {
		t.Errorf("page = %s", sql)
	}
}

func TestCountStatementUnionCountsEachArmApart(t *testing.T) {
	q := unionQuery()
	q.UnionCountWhere = "w2count"
	q.CountWhere = "w1count"
	sql := CountStatement(q)
	if !strings.Contains(sql, "w1count") || !strings.Contains(sql, "w2count") || strings.Count(sql, "UNION ALL") != 1 || !strings.Contains(sql, "LIMIT 1001") {
		t.Errorf("count = %s", sql)
	}
	// Without a count-only predicate the page's is used.
	q.UnionCountWhere, q.CountWhere = "", ""
	if sql := CountStatement(q); !strings.Contains(sql, "w1") || !strings.Contains(sql, "w2") {
		t.Errorf("count = %s", sql)
	}
	// With branches: arms are branches x predicates.
	q.Branches = []string{"b1", "b2"}
	if sql := CountStatement(q); strings.Count(sql, "UNION ALL") != 3 {
		t.Errorf("branched count = %s", sql)
	}
	q.CountBranches = []string{"c1", "c2"}
	if sql := CountStatement(q); !strings.Contains(sql, "c1") || strings.Contains(sql, "b1") {
		t.Errorf("count branches ignored: %s", sql)
	}
}
