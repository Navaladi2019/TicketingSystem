//go:build integration

package backfill_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// S22 P7-12: a tenant seeded by onboarding (the flagged templates as nava_onboard,
// then onboard.RegisterSeedIndexes as nava_app) has its three registry rows
// registered backfilling; the backfill coordinator, run on the SDK test
// environment as the worker runs it, makes every one ready.
func TestS22SeedIndexesBecomeReady(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", tenant)
	for _, key := range metadata.TemplateKeys() {
		d, err := metadata.Template(key)
		if err != nil {
			t.Fatal(err)
		}
		if err := tenancy.InTenantTx(ctx, e.onboard, tenant, func(tx pgx.Tx) error {
			_, err := entitytypes.Store{}.Seed(ctx, tx, tenant, entitytypes.SeedInput{
				DefinitionID: onboard.EntityDefinitionID(tenant, key), VersionID: onboard.EntityVersionID(tenant, key, 1),
				Name: key, Definition: d,
			})
			return err
		}); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}
	if err := onboard.RegisterSeedIndexes(ctx, onboard.IndexStore{Pool: e.app}, tenant); err != nil {
		t.Fatal(err)
	}

	rows, err := e.super.Query(ctx, `SELECT id, entity_definition_id, field_key, state FROM record_index_fields WHERE tenant_id = $1 ORDER BY field_key, id`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var fields []fixture
	for rows.Next() {
		f := fixture{tenant: tenant}
		var key, state string
		if err := rows.Scan(&f.field, &f.def, &key, &state); err != nil {
			t.Fatal(err)
		}
		if state != "backfilling" {
			t.Fatalf("%s registered %s, want backfilling", key, state)
		}
		fields = append(fields, f)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("registered %d fields, want 3 (incident.priority, request.priority, change.risk)", len(fields))
	}

	r := newRun(t, tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	for _, f := range fields {
		r.signal(item(f, 1), 0)
	}
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	for _, f := range fields {
		if got := registry(t, e, f); got.state != "ready" || got.progress != 100 {
			t.Errorf("field %s: %+v, want ready at 100", f.field, got)
		}
	}
}
