package workflowdefs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// Reader is interp.DefinitionStore over workflow_definition_versions. It
// reads in workerguard.InTenantTx, so the tenant is the one the activity guard
// verified from the nava-tenant header and RLS scopes the query to it.
type Reader struct {
	Pool *pgxpool.Pool
}

var _ interp.DefinitionStore = Reader{}

// Load returns the published version versionID of tenantID. It returns
// interp.ErrNotFound for an unknown id, for another tenant's id, and when
// tenantID is not the header tenant: the argument is only ever compared with
// the verified tenant, never used to pick one.
func (r Reader) Load(ctx context.Context, tenantID, versionID uuid.UUID) (*interp.Version, error) {
	header, ok := workerguard.TenantFromContext(ctx)
	if !ok || header != tenantID {
		return nil, interp.ErrNotFound
	}
	var (
		body string
		ids  map[string]uuid.UUID
	)
	err := workerguard.InTenantTx(ctx, r.Pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT body::text, step_ids FROM workflow_definition_versions WHERE tenant_id = $1 AND id = $2`,
			tenantID, versionID).Scan(&body, &ids)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, interp.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("workflowdefs: load version: %w", err)
	}
	d, err := def.Parse([]byte(body))
	if err != nil {
		return nil, fmt.Errorf("workflowdefs: stored definition does not parse: %w", err)
	}
	return &interp.Version{Definition: d, StepIDs: ids}, nil
}
