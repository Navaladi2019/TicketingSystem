//go:build integration

package backfill_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// planTypeRecords is the size of the type the keyset step walks, and
// planOtherTenants the tenants of one record type each beside it: enough
// distinct tenants and types that the generic plan's row estimate for
// "tenant_id = $1 AND entity_definition_id = $2" is a handful of rows.
const (
	planTypeRecords  = 20000
	planOtherTenants = 200
	planExecutions   = 8 // more than the five custom plans plan_cache_mode = auto makes
)

// TICK-990 (P7-07, ADR-006 D10): the backfill's keyset step, prepared once
// on one nava_worker connection as pgx's statement cache does, keeps a
// records_pkey range walk after more than five executions, because db-roles
// gives nava_worker plan_cache_mode = force_custom_plan. Under auto the same
// statement switches to the generic plan, which reads the whole type through
// another index and sorts it (821 ms against 17 ms at 400k records, P7-07):
// the negative control, without which this test could not catch the miss.
func TestTICK990KeysetStepKeepsCustomPlan(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	f := seedPlanFixture(t, e)
	mid := uuid.MustParse(queryString(t, e.super,
		"SELECT id::text FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 ORDER BY id OFFSET $3 LIMIT 1",
		f.tenant, f.def, planTypeRecords/2))

	if got := queryString(t, e.worker, "SHOW plan_cache_mode"); got != "force_custom_plan" {
		t.Fatalf("nava_worker plan_cache_mode = %s, want force_custom_plan from db-roles", got)
	}

	for _, arm := range []struct {
		name, mode string
		wantPkey   bool
	}{
		{"role setting", "", true},
		{"auto, negative control", "auto", false},
	} {
		t.Run(arm.name, func(t *testing.T) {
			stmt := "tick990_" + strings.ReplaceAll(arm.mode, "_", "")
			plan := ""
			err := tenancy.InTenantTx(ctx, e.worker, f.tenant, func(tx pgx.Tx) error {
				if arm.mode != "" {
					if _, err := tx.Exec(ctx, "SET LOCAL plan_cache_mode = "+arm.mode); err != nil {
						return err
					}
				}
				if _, err := tx.Prepare(ctx, stmt, backfill.NextIDsSQL); err != nil {
					return err
				}
				defer func() { _, _ = tx.Exec(ctx, "DEALLOCATE "+stmt) }()
				for range planExecutions {
					rows, err := tx.Query(ctx, stmt, f.tenant, f.def, mid, 1000)
					if err != nil {
						return err
					}
					ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
					if err != nil {
						return err
					}
					if len(ids) != 1000 {
						t.Errorf("batch = %d ids, want 1000", len(ids))
					}
				}
				var raw []byte
				// EXPLAIN EXECUTE takes no bind parameters; uuid.UUID's
				// String is a fixed hex form, safe to splice.
				if err := tx.QueryRow(ctx, fmt.Sprintf("EXPLAIN (FORMAT JSON) EXECUTE %s('%s', '%s', '%s', 1000)",
					stmt, f.tenant, f.def, mid)).Scan(&raw); err != nil {
					return err
				}
				plan = planShape(t, raw)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("after %d executions: %s", planExecutions, plan)
			walk := strings.Contains(plan, "Index Scan using records_pkey") && !strings.Contains(plan, "Sort")
			if walk != arm.wantPkey {
				t.Errorf("plan %s: records_pkey walk = %v, want %v", plan, walk, arm.wantPkey)
			}
		})
	}
}

// seedPlanFixture writes planTypeRecords records of one type and one record
// in each of planOtherTenants other tenants, then analyses records. The
// rows go in as the superuser with session_replication_role = replica (no
// records trigger, no commit check, as audit_commit_check_test seeds): the
// test is about the planner's statistics, not the write path.
func seedPlanFixture(t *testing.T, e itEnv) fixture {
	t.Helper()
	f := newFixture(t, e, "integer", "")
	others := make([]fixture, planOtherTenants)
	for i := range others {
		others[i] = newFixture(t, e, "integer", "")
	}
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, mustConn(t, e), func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data)
			SELECT $1, gen_random_uuid(), $2, 1, 'INC-' || g, 'open', jsonb_build_object('f_integer', g)
			FROM generate_series(1, $3::int) g`, f.tenant, f.def, planTypeRecords); err != nil {
			return err
		}
		for _, o := range others {
			if _, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data)
				VALUES ($1, gen_random_uuid(), $2, 1, 'INC-1', 'open', '{}')`, o.tenant, o.def); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, e.super, "ANALYZE records")
	return f
}

// mustConn is one superuser connection from the pool, released at cleanup.
func mustConn(t *testing.T, e itEnv) *pgx.Conn {
	t.Helper()
	c, err := e.super.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Release)
	return c.Conn()
}

// planShape is the plan's node types and index names, outermost first.
func planShape(t *testing.T, raw []byte) string {
	t.Helper()
	var doc []struct {
		Plan map[string]any `json:"Plan"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc) != 1 {
		t.Fatalf("EXPLAIN output %s: %v", raw, err)
	}
	var parts []string
	var walk func(n map[string]any)
	walk = func(n map[string]any) {
		s, _ := n["Node Type"].(string)
		if idx, ok := n["Index Name"].(string); ok {
			s += " using " + idx
		}
		parts = append(parts, s)
		kids, _ := n["Plans"].([]any)
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok {
				walk(m)
			}
		}
	}
	walk(doc[0].Plan)
	return strings.Join(parts, " > ")
}
