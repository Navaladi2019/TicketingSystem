package entitytypes

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// addsDefault reports whether next has a top-level default that prev lacks
// (ADR-009 D1): old records start reading it, so they need a backfill.
func addsDefault(prev, next *metadata.Compiled) bool {
	had := prev.DefaultKeys()
	for _, k := range next.DefaultKeys() {
		if !slices.Contains(had, k) {
			return true
		}
	}
	return false
}

// queueBackfill writes, in the publish transaction, the schema_backfills row
// (pending, default_keys = every top-level default of the new version,
// causation = the publish's config audit entry) and its outbox start row
// (ADR-009 D1, contract C3) when the new version adds a default. A version
// that adds none writes neither. A replay of the same publish finds both rows
// already there (same primary key and dedupe key) and writes nothing more.
func queueBackfill(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, version int, auditID uuid.UUID, prev, next *metadata.Compiled, after func() error) error {
	if !addsDefault(prev, next) {
		return nil
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO schema_backfills (tenant_id, entity_definition_id, target_version, default_keys, causation_audit_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, entity_definition_id, target_version) DO NOTHING`,
		tenantID, defID, version, next.DefaultKeys(), auditID); err != nil {
		return fmt.Errorf("entitytypes: insert schema backfill: %w", err)
	}
	if _, err := outbox.EnqueueSchemaBackfill(ctx, tx, defID, version); err != nil {
		return fmt.Errorf("entitytypes: enqueue schema backfill: %w", err)
	}
	if after != nil {
		return after()
	}
	return nil
}
