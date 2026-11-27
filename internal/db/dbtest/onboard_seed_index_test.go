//go:build integration

package dbtest

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// S22 P7-12: onboarding seeds each type at version 1 with its filterable
// fields flagged (published_by_system 'onboarding'), then registers the
// field-index rows as nava_app before it activates the tenant.

// seedFlags are the fields the seed templates flag, by type key.
var seedFlags = map[string]string{"incident": "priority", "request": "priority", "change": "risk"} //nolint:gochecknoglobals // read-only test fixture

// registrySnapshot counts the tenant's registry rows and field-index outbox
// rows, and returns the "type.field state" of each registry row, sorted.
func registrySnapshot(t *testing.T, tenant uuid.UUID) (rows string, backfills int) {
	t.Helper()
	super := superNava(t)
	rows = queryString(t, super, `SELECT coalesce(string_agg(d.key || '.' || f.field_key || ' ' || f.state, ', ' ORDER BY d.key, f.field_key), '')
		FROM record_index_fields f JOIN entity_definitions d ON d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id
		WHERE f.tenant_id = $1`, tenant)
	return rows, countRows(t, `SELECT count(*) FROM outbox WHERE tenant_id = $1 AND kind = 'field_index.backfill'`, tenant)
}

const wantRegistry = "change.risk backfilling, incident.priority backfilling, request.priority backfilling"

func TestS22OnboardingRegistersTheSeedFieldIndexes(t *testing.T) {
	migratedPlatform(t)
	svc, _ := onboardService(t)
	op := identity.Operator{PrincipalID: "op-" + uuid.NewString()}
	key := uuid.NewString()
	id := onboard.TenantID(op.PrincipalID, key)
	if _, err := svc.Onboard(t.Context(), op, key, []byte("h"), onboardRequest("s22-"+id.String()[:8]+".test")); err != nil {
		t.Fatalf("Onboard: %v", err)
	}
	// Each seed type's version 1 names the onboarding system as publisher.
	for k := range seedFlags {
		got := queryString(t, superNava(t), `SELECT coalesce(v.published_by::text, 'null') || ' ' || coalesce(v.published_by_system, 'null')
			FROM entity_definition_versions v WHERE v.tenant_id = $1 AND v.definition_id = $2 AND v.version = 1`, id, onboard.EntityDefinitionID(id, k))
		if got != "null onboarding" {
			t.Errorf("%s v1 publisher = %q, want null onboarding", k, got)
		}
	}
	// Registered backfilling at v1 with one backfill each: the worker makes
	// them ready (TestS22SeedIndexesBecomeReady in the backfill package).
	rows, backfills := registrySnapshot(t, id)
	if rows != wantRegistry || backfills != 3 {
		t.Errorf("registry = %q with %d backfill rows, want %q and 3", rows, backfills, wantRegistry)
	}
}

// A crash between the seed and the registration leaves a provisioning tenant
// with seeded types and no registry rows; the retry finishes it, with the same
// rows as an uninterrupted run, and nothing twice.
func TestS22OnboardingRetryAfterACrashBetweenSeedAndRegistration(t *testing.T) {
	migratedPlatform(t)
	real := onboard.IndexStore{Pool: rolePool(t, "nava_app", 2)}
	var calls atomic.Int32
	crashing := indexFunc(func(ctx context.Context, tenant uuid.UUID, k string) error {
		if calls.Add(1) == 2 { // the second type's registration fails once
			return errors.New("simulated crash")
		}
		return real.RegisterSeedFilterable(ctx, tenant, k)
	})
	svc, _ := onboardServiceWithIndex(t, crashing)
	op := identity.Operator{PrincipalID: "op-" + uuid.NewString()}
	key := uuid.NewString()
	id := onboard.TenantID(op.PrincipalID, key)
	req := onboardRequest("s22r-" + id.String()[:8] + ".test")

	if _, err := svc.Onboard(t.Context(), op, key, []byte("h"), req); err == nil {
		t.Fatal("Onboard succeeded though a registration failed")
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", id); st == "active" {
		t.Fatal("the tenant was activated with an unregistered seed type")
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1", id); n != 3 {
		t.Fatalf("the seed left %d versions, want 3", n)
	}
	if _, err := svc.Onboard(t.Context(), op, key, []byte("h"), req); err != nil {
		t.Fatalf("retry: %v", err)
	}
	rows, backfills := registrySnapshot(t, id)
	if rows != wantRegistry || backfills != 3 {
		t.Errorf("after the retry: registry %q with %d backfill rows, want %q and 3", rows, backfills, wantRegistry)
	}
	// A further full attempt adds nothing.
	if err := onboard.RegisterSeedIndexes(t.Context(), real, id); err != nil {
		t.Fatal(err)
	}
	if rows2, backfills2 := registrySnapshot(t, id); rows2 != rows || backfills2 != backfills {
		t.Errorf("a repeated registration changed the rows: %q/%d -> %q/%d", rows, backfills, rows2, backfills2)
	}
	if st := queryString(t, superNava(t), "SELECT state FROM tenants WHERE id = $1", id); st != "active" {
		t.Errorf("tenant state = %s, want active", st)
	}
}

// A v1 with no publisher at all, as a build before TICK-1172 seeded it, flags
// nothing: RegisterSeedFilterable returns nil and writes no registry row, so a
// retried onboarding can go on to Activate.
func TestS22RegisterSeedFilterableIgnoresAPreColumnSeed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	tenant := uuid.NewString()
	seedTenants(t, super, []string{tenant}, nil)
	exec(t, super, insertType, tenant, "oldseed", "OLD")
	exec(t, super, insertVersion, tenant, tenant, tenant, 1, testSchema, schemaSum(testSchema), "initial", nil)
	if err := (onboard.IndexStore{Pool: rolePool(t, "nava_app", 2)}).RegisterSeedFilterable(t.Context(), uuid.MustParse(tenant), "oldseed"); err != nil {
		t.Fatalf("RegisterSeedFilterable of a pre-column v1 = %v, want nil", err)
	}
	if n := countRows(t, "SELECT count(*) FROM record_index_fields WHERE tenant_id = $1", tenant); n != 0 {
		t.Errorf("registry rows = %d, want 0", n)
	}
}

type indexFunc func(ctx context.Context, tenant uuid.UUID, key string) error

func (f indexFunc) RegisterSeedFilterable(ctx context.Context, tenant uuid.UUID, key string) error {
	return f(ctx, tenant, key)
}
