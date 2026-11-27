//go:build integration && perf

package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestPerfCountPlanSummary pins the A4-10 plan read-out on the 2026-11-27
// smoke run's m = 0 count-only count plan (TICK-1105): the regional arm is
// an Index Scan of records_number_key, its Index Cond starts with tenant_id,
// and it read the type's 33,366 rows (0 returned, 33,366 removed by filter).
// No database: it parses a saved EXPLAIN (ANALYZE) text.
func TestPerfCountPlanSummary(t *testing.T) {
	b, err := os.ReadFile("testdata/perfplan/wide-m0-count-only.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := summariseCountPlan(string(b), perfWideType, 8)
	for _, want := range []string{
		"stmt 8, count 42.564 ms",
		"IOS of records_list_updated: no",
		"regional arm: Index Scan using records_number_key on records; Index Cond starts with tenant_id: yes; rows read 33366 (N̂_e 33366)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary %q\nlacks %q", got, want)
		}
	}
	if n := strings.Count(got, "rows read"); n != 1 {
		t.Errorf("summary lists %d regional arms, want 1 (the requester and assignee arms carry no region term): %s", n, got)
	}
}

func TestPerfIsListCount(t *testing.T) {
	for sql, want := range map[string]bool{
		"SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r) UNION ALL (SELECT 1)) u LIMIT 1001) c": true,
		"SELECT count(*) FROM (SELECT 1 FROM x CROSS JOIN LATERAL (SELECT 1) acl_x LIMIT 1044) acl_p":          false,
		"SELECT acl_s.n, CASE WHEN acl_s.n >= $4::bigint THEN NULL ELSE (SELECT count(*) FROM y) END":          false,
	} {
		if got := isListCount(sql); got != want {
			t.Errorf("isListCount(%.40q) = %v, want %v", sql, got, want)
		}
	}
}
