//go:build integration

package dbtest

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Security review of P3-06 (TICK-344). The seed is INSERT ... ON CONFLICT DO
// NOTHING, with no conflict target, and nava_onboard has no SELECT on either
// table. The risk this guards: a seed run in tenant X's transaction that names
// tenant Y must fail with 42501. It must not be silently absorbed as "nothing
// inserted" because Y's row with the same key already exists. If it were
// absorbed, the inserted flag would also tell nava_onboard whether Y's row
// exists. Postgres checks the RLS INSERT WITH CHECK on the proposed row before
// the conflict, and this test pins that order through the production Seed.
func TestP306SeedIntoAnotherTenantIsRefusedEvenOnConflict(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	pool := rolePool(t, "nava_onboard", 2)
	x, y := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{x, y} {
		if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
			t.Fatal(err)
		}
	}
	// Y is seeded (provisioning). X is provisioning and has no rows yet.
	if err := onboard.SeedEntityTypes(t.Context(), s, y); err != nil {
		t.Fatal(err)
	}
	d, err := metadata.Template("incident")
	if err != nil {
		t.Fatal(err)
	}
	seedAs := func(txTenant, target uuid.UUID, in entitytypes.SeedInput) (bool, error) {
		var inserted bool
		err := tenancy.InTenantTx(t.Context(), pool, txTenant, func(tx pgx.Tx) error {
			var err error
			inserted, err = entitytypes.Store{}.Seed(t.Context(), tx, target, in)
			return err
		})
		return inserted, err
	}
	refused := func(what string, inserted bool, err error) {
		t.Helper()
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
			t.Errorf("%s = inserted %v, err %v; want SQLSTATE 42501", what, inserted, err)
		}
	}

	// 1. The same ids as Y's seed, so the row conflicts. It must still be refused.
	inserted, err := seedAs(x, y, entitytypes.SeedInput{
		DefinitionID: onboard.EntityDefinitionID(y, "incident"), VersionID: onboard.EntityVersionID(y, "incident", 1),
		Name: "Incident", Definition: d,
	})
	refused("tenant X's tx seeds Y's existing incident (conflicting ids)", inserted, err)

	// 2. Fresh ids, so the only conflict is (tenant_id, key) on Y's row.
	inserted, err = seedAs(x, y, entitytypes.SeedInput{
		DefinitionID: uuid.New(), VersionID: uuid.New(), Name: "Incident", Definition: d,
	})
	refused("tenant X's tx seeds Y's key with fresh ids", inserted, err)

	// 3. No conflict at all: a key Y has not seeded.
	other := d
	other.Key, other.NumberPrefix = "problem", "PRB"
	inserted, err = seedAs(x, y, entitytypes.SeedInput{
		DefinitionID: uuid.New(), VersionID: uuid.New(), Name: "Problem", Definition: other,
	})
	refused("tenant X's tx seeds a new type into Y", inserted, err)

	if n := countRows(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", y); n != 3 {
		t.Errorf("tenant Y holds %d types, want its own 3", n)
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", x); n != 0 {
		t.Errorf("tenant X holds %d types, want 0", n)
	}

	// The seeded ids differ per tenant, so tenant X's own seed never conflicts
	// with Y's rows.
	if onboard.EntityDefinitionID(x, "incident") == onboard.EntityDefinitionID(y, "incident") {
		t.Fatal("seeded definition ids collide across tenants")
	}
	if err := onboard.SeedEntityTypes(t.Context(), s, x); err != nil {
		t.Fatalf("control: X seeds itself: %v", err)
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1", x); n != 3 {
		t.Errorf("tenant X holds %d versions after its own seed, want 3", n)
	}
}
