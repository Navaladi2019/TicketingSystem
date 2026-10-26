//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// TICK-146: every batch DELETE, run as nava_purge under its *_purge policies,
// can be served by its 00005 index. The seed is tiny, so seq scans are
// switched off: a statement or policy change that no index can serve plans a
// Seq Scan anyway and fails here. The plans at volume are in
// docs/data/purge-indexes.md.
func TestT146PurgeStatementsUseTheirIndexes(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPurgeable(t, super, tenantA, tenantB)

	want := map[string][]string{
		purge.TableIdempotencyKeys: {"idempotency_keys_purge_expires"},
		purge.TableOutbox:          {"outbox_purge_dispatched", "outbox_purge_dead"},
		purge.TableWorkflowRef:     {"workflow_ref_purge_closed"},
	}
	ctx := context.Background()
	conn := connect(t, roleConfig(t, "nava_purge", "nava"))
	for _, s := range purge.Statements(purge.MaxBatch, purge.DefaultWindows()) {
		t.Run(s.Table, func(t *testing.T) {
			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
				t.Fatal(err)
			}
			rows, err := tx.Query(ctx, "EXPLAIN (COSTS OFF) "+s.SQL, s.Args...)
			if err != nil {
				t.Fatal(err)
			}
			var lines []string
			for rows.Next() {
				var l string
				if err := rows.Scan(&l); err != nil {
					t.Fatal(err)
				}
				lines = append(lines, l)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			plan := strings.Join(lines, "\n")
			if strings.Contains(plan, "Seq Scan") {
				t.Errorf("plan has a Seq Scan:\n%s", plan)
			}
			for _, idx := range want[s.Table] {
				if !strings.Contains(plan, idx) {
					t.Errorf("plan does not use %s:\n%s", idx, plan)
				}
			}
		})
	}
}
