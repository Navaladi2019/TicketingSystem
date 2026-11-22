//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/estimates"
)

// TICK-1071 adversarial (Security & Multi-Tenancy Reviewer): the refresh job
// binds the same $1 to set_config and to its upsert and zero statements, and
// filters records by tenant_id = $1 itself. These tests take that agreement
// away and check that RLS alone still keeps every tenant's estimates in that
// tenant: a job whose set_config names another tenant, or no tenant at all,
// neither reads the target's records nor writes or zeroes the target's rows.

// desyncTenant rewrites the tenant a refresh transaction sets: when the job
// sets `from`, the session gets `to` instead, while the job's own statements
// still bind `from`. This simulates a regression that sets the wrong tenant.
type desyncTenant struct {
	estimates.Store
	from, to string
}

// InTenant scopes the transaction to `to` when the job asks for `from`; fn
// still binds `from`.
func (b desyncTenant) InTenant(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	if tenant.String() == b.from {
		tenant = uuid.MustParse(b.to)
	}
	return b.Store.InTenant(ctx, tenant, fn)
}

// Tenant B's refresh runs under tenant A's app.tenant_id (or a tenant with no data): B's
// records are invisible, so the upsert selects nothing; B's existing row is
// invisible to the zero UPDATE, so it is not set to 0 even though, under A, no
// record of B "exists". Neither tenant's rows change.
func TestTICK1071DesyncedTenantWritesNothingAcross(t *testing.T) {
	for _, tc := range []struct{ name, session string }{
		{"other tenant set", tenantA},
		{"tenant with no data set", "0190a000-0000-7000-8000-0000000000ee"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedEstimateTypes(t, super, tenantA, tenantB)
			addRecords(t, super, tenantA, tenantA, "TCK", 3)
			addRecords(t, super, tenantB, tenantB, "TCK", 5)
			worker := estimates.PoolStore{Pool: workerPool(t)}

			clean, _ := newJob(t, worker, nil)
			if n := clean.Cycle(context.Background()); n != 0 {
				t.Fatalf("clean cycle: failed tenants = %d, want 0", n)
			}
			beforeB := refreshedAt(t, super, tenantB)

			// More of B's records, then a cycle whose B transaction runs
			// under the wrong (or no) tenant.
			addRecords(t, super, tenantB, tenantB, "TCX", 4)
			bad, _ := newJob(t, desyncTenant{worker, tenantB, tc.session}, nil)
			bad.Cycle(context.Background())

			// Ground truth, read as the superuser (bypasses RLS): B keeps its
			// old count and time; A has no row of B's type and still its own.
			if got := queryString(t, super,
				"SELECT string_agg(tenant_id::text || '=' || live_records::text, ',' ORDER BY tenant_id) FROM entity_record_estimates"); got != tenantA+"=3,"+tenantB+"=5" {
				t.Errorf("rows = %s, want A=3 and B=5 untouched", got)
			}
			if !refreshedAt(t, super, tenantB).Equal(beforeB) {
				t.Error("B's refreshed_at moved under a desynced tenant")
			}
			if got := queryString(t, super,
				"SELECT count(*)::text FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2", tenantA, tenantB); got != "0" {
				t.Errorf("A holds %s rows of B's type, want 0", got)
			}
		})
	}
}
