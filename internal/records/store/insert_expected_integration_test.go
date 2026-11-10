//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TestInsertRecordAsExpected (TICK-692): the INSERT a create sends in its
// COMMIT's batch stores the row whose payload preview the caller built its
// response from, and fails the batch, COMMIT included, with
// ErrStoredRowDiffers when the stored payload is not that preview: nothing
// is stored.
func TestInsertRecordAsExpected(t *testing.T) {
	e := migrated(t)
	fixed := assetFixed()
	tenant, principal := uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, tenant, principal, "kc-"+principal.String())
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Create(ctx, tx, tenant, entitytypes.NewInput{
			Name: "asset", PublishedBy: principal, Definition: assetDefinition(),
		})
		return err
	})
	now := time.Now().UTC().Truncate(time.Microsecond)

	// create inserts a new record in one transaction: the preview first,
	// then the INSERT expecting want(preview), its audit entry and COMMIT in
	// one batch, as a create's last round trip sends them.
	create := func(want func(json.RawMessage) json.RawMessage) (uuid.UUID, error) {
		id := uuid.New()
		err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
			ctx := context.Background()
			def, err := store.DefinitionID(ctx, tx, tenant, "asset")
			if err != nil {
				return err
			}
			rec := &store.Record{TenantID: tenant, ID: id, EntityDefinitionID: def, Version: 1, SchemaVersion: 1,
				Number: "AST-" + id.String()[:7], CreatedBy: &principal, CreatedAt: now, UpdatedAt: now}
			if err := rec.Split(map[string]any{"name": "one", "status": "active", "serial": "SN-" + id.String(),
				"tag": json.Number("7.0"), "owner": principal.String()}, fixed); err != nil {
				return err
			}
			var preview json.RawMessage
			b := &pgx.Batch{}
			if err := store.QueuePayloadPreview(b, rec, fixed, &preview); err != nil {
				return err
			}
			if err := tx.SendBatch(ctx, b).Close(); err != nil {
				return err
			}
			commit, ok := tenancy.QueueAtCommit(tx)
			if !ok {
				t.Fatal("no commit batch")
			}
			if err := store.QueueInsertRecordAsExpected(commit, rec, fixed, want(preview)); err != nil {
				return err
			}
			va := rec.Version
			_, err = audit.QueueAppend(ctx, tx, commit, audit.Entry{Domain: audit.DomainRecord, Action: audit.ActionCreate,
				SubjectType: audit.SubjectRecord, SubjectID: rec.ID, EntityDefinitionID: &def, VersionAfter: &va,
				Actor: audit.Actor{PrincipalID: &principal}, Source: audit.SourceUser})
			return err
		})
		return id, err
	}
	stored := func(id uuid.UUID) int {
		var n int
		if err := e.super.QueryRow(context.Background(), "SELECT count(*) FROM records WHERE id = $1", id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	id, err := create(func(p json.RawMessage) json.RawMessage { return p })
	if err != nil {
		t.Fatalf("create with the preview: %v", err)
	}
	if n := stored(id); n != 1 {
		t.Errorf("%d rows stored, want 1", n)
	}

	for name, want := range map[string]func(json.RawMessage) json.RawMessage{
		"another payload":   func(json.RawMessage) json.RawMessage { return json.RawMessage(`{"name": "two"}`) },
		"reformatted bytes": func(p json.RawMessage) json.RawMessage { return append(json.RawMessage(" "), p...) },
	} {
		id, err := create(want)
		if !errors.Is(err, store.ErrStoredRowDiffers) {
			t.Errorf("%s: err = %v, want ErrStoredRowDiffers", name, err)
		}
		if n := stored(id); n != 0 {
			t.Errorf("%s: %d rows stored, want 0", name, n)
		}
	}
}
