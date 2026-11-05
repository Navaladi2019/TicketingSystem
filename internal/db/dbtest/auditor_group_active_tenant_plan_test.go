//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// P5-23 (TICK-568), Security review: every predicate the four nava_onboard
// policies add is index-backed. Each statement onboarding's upgrade reconcile
// runs (onboard.PGStore.recordGroup, plus the no-op read) is planned as
// nava_onboard under A's context with sequential scans priced out: a "Seq
// Scan" left in the plan means no index can serve that predicate, so at scale
// it reads every tenant's rows. The policies' subqueries on tenants go through
// tenants_onboard_select (id = app.tenant_id, tenants_pkey); the probes on
// tenant_auditor_group and groups are equalities on their primary keys.
func TestP523OnboardPoliciesAreIndexBacked(t *testing.T) {
	_ = activeAB(t)
	ob := asRole(t, "nava_onboard")
	a := uuid.MustParse(tenantA)
	for name, stmt := range map[string]struct {
		sql  string
		args []any
	}{
		"groups INSERT (WITH CHECK)": {
			"INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			[]any{a, onboard.AuditorGroupID(a), "kc-plan", "/t/" + tenantA + "/auditors"},
		},
		"groups read-back by Keycloak id (USING)": {
			"SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2 AND deleted_at IS NULL)",
			[]any{a, "kc-plan"},
		},
		"tenant_auditor_group INSERT (WITH CHECK)": {
			`INSERT INTO tenant_auditor_group (tenant_id, group_id)
			 SELECT tenant_id, id FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2 AND deleted_at IS NULL
			 ON CONFLICT DO NOTHING`,
			[]any{a, "kc-plan"},
		},
		"tenant_auditor_group no-op probe (USING)": {
			"SELECT EXISTS (SELECT 1 FROM tenant_auditor_group WHERE tenant_id = $1)",
			[]any{a},
		},
	} {
		var plan strings.Builder
		err := inTenantTx(ob, tenantA, func(tx pgx.Tx) error {
			ctx := context.Background()
			if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
				return err
			}
			rows, err := tx.Query(ctx, "EXPLAIN (COSTS OFF) "+stmt.sql, stmt.args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var line string
				if err := rows.Scan(&line); err != nil {
					return err
				}
				plan.WriteString(line + "\n")
			}
			return rows.Err()
		})
		if err != nil {
			t.Fatalf("%s: EXPLAIN: %v", name, err)
		}
		t.Logf("%s:\n%s", name, plan.String())
		if strings.Contains(plan.String(), "Seq Scan") {
			t.Errorf("%s: plan has a sequential scan, a policy predicate is not index-backed:\n%s", name, plan.String())
		}
	}
}
