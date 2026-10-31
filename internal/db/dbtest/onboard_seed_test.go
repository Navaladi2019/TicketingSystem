//go:build integration

package dbtest

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// P3-06 (TICK-344): onboarding seeds incident, request and change at version 1
// as the real nava_onboard role, through the production seeding code.

var seedWant = []struct{ key, prefix, name string }{
	{"change", "CHG", "Change"},
	{"incident", "INC", "Incident"},
	{"request", "REQ", "Request"},
}

func TestP306OnboardSeedsThreeTypesAtVersionOne(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err != nil {
		t.Fatalf("SeedEntityTypes: %v", err)
	}
	super := superNava(t)
	for _, w := range seedWant {
		defID, verID := onboard.EntityDefinitionID(id, w.key), onboard.EntityVersionID(id, w.key, 1)
		got := queryString(t, super, `SELECT d.id::text || ' ' || d.number_prefix || ' ' || d.name || ' ' || d.current_version
			FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $2`, id, w.key)
		if want := defID.String() + " " + w.prefix + " " + w.name + " 1"; got != want {
			t.Errorf("%s definition = %q, want %q", w.key, got, want)
		}
		got = queryString(t, super, `SELECT v.id::text || ' ' || v.version || ' ' || v.compat || ' ' || coalesce(v.published_by::text, 'null')
			FROM entity_definition_versions v WHERE v.tenant_id = $1 AND v.definition_id = $2`, id, defID)
		if want := verID.String() + " 1 initial null"; got != want {
			t.Errorf("%s version = %q, want %q", w.key, got, want)
		}
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1", id); n != 3 {
		t.Errorf("versions = %d, want 3", n)
	}
}

// A second run (the retry) inserts nothing, and a crash that left a
// definition without its version, or one type of three, is finished.
func TestP306SeedRetryInsertsNothingAndFinishesWhatIsMissing(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err != nil {
		t.Fatal(err)
	}
	count := func() (int, int) {
		return countRows(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", id),
			countRows(t, "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1", id)
	}
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err != nil {
		t.Fatalf("second SeedEntityTypes: %v", err)
	}
	if d, v := count(); d != 3 || v != 3 {
		t.Fatalf("after the second run: %d types, %d versions, want 3 and 3", d, v)
	}
	// The store's own report: a repeat of a seeded type inserts nothing.
	d, err := metadata.Template("incident")
	if err != nil {
		t.Fatal(err)
	}
	inserted, err := s.SeedEntityType(t.Context(), id, entitytypes.SeedInput{
		DefinitionID: onboard.EntityDefinitionID(id, "incident"), VersionID: onboard.EntityVersionID(id, "incident", 1),
		Name: "Incident", Definition: d,
	})
	if err != nil || inserted {
		t.Errorf("SeedEntityType of a seeded type = %v, %v; want false, nil", inserted, err)
	}

	// Simulate a crash: one type gone entirely, another with no version.
	super := superNava(t)
	exec(t, super, "DELETE FROM entity_definition_versions WHERE tenant_id = $1 AND definition_id IN ($2, $3)",
		id, onboard.EntityDefinitionID(id, "change"), onboard.EntityDefinitionID(id, "request"))
	exec(t, super, "DELETE FROM entity_definitions WHERE tenant_id = $1 AND id = $2", id, onboard.EntityDefinitionID(id, "change"))
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err != nil {
		t.Fatalf("SeedEntityTypes after a partial seed: %v", err)
	}
	if d, v := count(); d != 3 || v != 3 {
		t.Errorf("after the retry: %d types, %d versions, want 3 and 3", d, v)
	}
}

// TestOnboardCannotSeedActiveTenant is the CI-9 fixture: once a tenant is
// active, nava_onboard cannot seed it again, whatever it asks for (the
// provisioning WITH CHECK refuses the insert, even one that would conflict).
func TestOnboardCannotSeedActiveTenant(t *testing.T) {
	migratedPlatform(t)
	s := onboardStore(t, 4)
	id := uuid.New()
	if _, err := s.Begin(t.Context(), id, []byte("h"), nil); err != nil {
		t.Fatal(err)
	}
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err != nil {
		t.Fatal(err)
	}
	recordAdmin(t, s, id)
	if err := s.Activate(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	before := countRows(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", id)

	// The same ids as the seed (a pure retry) and a fresh type are both refused.
	err := onboard.SeedEntityTypes(t.Context(), s, id)
	if err == nil {
		t.Fatal("SeedEntityTypes on an active tenant succeeded, want a refusal")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("SeedEntityTypes on an active tenant = %v, want SQLSTATE 42501", err)
	}
	super := superNava(t)
	exec(t, super, "DELETE FROM entity_definition_versions WHERE tenant_id = $1", id)
	exec(t, super, "DELETE FROM entity_definitions WHERE tenant_id = $1", id)
	if err := onboard.SeedEntityTypes(t.Context(), s, id); err == nil {
		t.Fatal("SeedEntityTypes into an emptied active tenant succeeded, want a refusal")
	}
	if n := countRows(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", id); n != 0 || before != 3 {
		t.Errorf("active tenant holds %d types after the refused seeds (had %d), want 0 inserted", n, before)
	}
}
