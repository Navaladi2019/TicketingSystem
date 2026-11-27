//go:build integration

// TICK-920 (ADR-009 D1, contract C3): EnqueueSchemaBackfill against a real
// PostgreSQL as nava_app, under RLS and outbox_payload_check.

package outbox_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The row the relay reads: kind schema.backfill, a start (target_key
// schema.backfill:<own id>), target_seq the version, causation its own id at
// depth 0, payload {entity_definition_id}, dedupe
// schema-backfill:<def>:<version>, and the transaction's tenant. A retry of
// the same publish returns the row it wrote; another version writes another.
func TestEnqueueSchemaBackfill(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant, def := uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	enqueue := func(version int) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
			var err error
			id, err = outbox.EnqueueSchemaBackfill(ctx, tx, def, version)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	id := enqueue(4)
	var (
		rowTenant, causation             uuid.UUID
		kind, dedupe, targetKey, payload string
		targetSeq                        int64
		depth                            int16
	)
	if err := e.super.QueryRow(ctx, `SELECT tenant_id, kind, dedupe_key, target_key, target_seq, causation_id, causation_depth, payload::text
		FROM outbox WHERE id = $1`, id).Scan(&rowTenant, &kind, &dedupe, &targetKey, &targetSeq, &causation, &depth, &payload); err != nil {
		t.Fatal(err)
	}
	if rowTenant != tenant || kind != outbox.SchemaBackfillKind || dedupe != outbox.SchemaBackfillDedupeKey(def, 4) ||
		targetKey != outbox.SchemaBackfillKind+":"+id.String() || targetSeq != 4 || causation != id || depth != 0 ||
		payload != `{"entity_definition_id": "`+def.String()+`"}` {
		t.Errorf("row = %s %s %s %s %d %s/%d %s", rowTenant, kind, dedupe, targetKey, targetSeq, causation, depth, payload)
	}
	if again := enqueue(4); again != id {
		t.Errorf("retry wrote %s, want the first row %s", again, id)
	}
	if next := enqueue(5); next == id {
		t.Error("version 5 returned version 4's row")
	}
	var n int
	if err := e.super.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant).Scan(&n); err != nil || n != 2 {
		t.Errorf("rows = %d, %v; want 2", n, err)
	}

	// A transaction with no tenant writes nothing.
	tx, err := e.app.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := outbox.EnqueueSchemaBackfill(ctx, tx, def, 6); err == nil {
		t.Error("a transaction with no tenant enqueued a row")
	}
}
