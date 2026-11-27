package onboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// SeedIndexer registers the field-index rows of the seeded entity types (S22
// P7-12). The seed (nava_onboard) cannot write the registry, so a second step
// as nava_app does.
type SeedIndexer interface {
	// RegisterSeedFilterable writes the registry rows and the backfill outbox
	// rows for the fields the seeded type key flags. Idempotent.
	RegisterSeedFilterable(ctx context.Context, tenantID uuid.UUID, key string) error
}

// IndexStore is the PostgreSQL SeedIndexer. Its pool logs in as nava_app, the
// role that writes the registry (P7-01); every call runs in the tenant's own
// transaction.
type IndexStore struct{ Pool *pgxpool.Pool }

var _ SeedIndexer = IndexStore{}

// RegisterSeedFilterable implements SeedIndexer through
// entitytypes.Store.RegisterSeedFilterable.
func (s IndexStore) RegisterSeedFilterable(ctx context.Context, tenantID uuid.UUID, key string) error {
	err := tenancy.InTenantTx(ctx, s.Pool, tenantID, func(tx pgx.Tx) error {
		return entitytypes.Store{}.RegisterSeedFilterable(ctx, tx, tenantID, key)
	})
	if err != nil {
		return fmt.Errorf("onboard: register field index of %s: %w", key, err)
	}
	return nil
}

// RegisterSeedIndexes registers the field-index rows of every seed template,
// one type at a time. Onboarding calls it on every attempt, after the seed and
// before Activate, so a retry after a crash between the two steps finishes it.
func RegisterSeedIndexes(ctx context.Context, idx SeedIndexer, tenantID uuid.UUID) error {
	for _, key := range metadata.TemplateKeys() {
		if err := idx.RegisterSeedFilterable(ctx, tenantID, key); err != nil {
			return err
		}
	}
	return nil
}
