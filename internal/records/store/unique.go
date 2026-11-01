package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// LockUnique takes the transaction-scoped advisory lock of one unique value
// (D-3-22): pg_advisory_xact_lock(hashtextextended(tenant ‖ entity type ‖
// field ‖ canonical value, 0)). Writers of the same value queue here until
// the holder's transaction ends, so the plain SELECT in UniqueTaken that
// follows sees the holder's committed row. The lock is released at commit or
// rollback, never held on a pooled connection, so it is safe under PgBouncer
// transaction pooling; two values whose keys collide only wait for each
// other. canonical is the value's canonical form (the pipeline's canonical);
// the separators keep a field name from running into its value.
func LockUnique(ctx context.Context, tx pgx.Tx, tenantID, entityDefinitionID uuid.UUID, field, canonical string) error {
	_, err := tx.Exec(ctx, `
SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text || ':' || $3 || ':' || $4, 0))`,
		tenantID, entityDefinitionID, field, canonical)
	if err != nil {
		return fmt.Errorf("records/store: lock unique value: %w", err)
	}
	return nil
}

// UniqueTaken reports whether another record of the tenant's entity type
// (any record but except) holds value in field. A field mapped to a fixed
// column (fixed maps property to column) is compared on that column, any
// other field on its key in data, with jsonb equality (so 1 and 1.0 are the
// same number); a uuid column's value must be the lowercase form uuid.String
// gives. There is no index behind this (D25): it scans the type's
// records in the tenant.
func UniqueTaken(ctx context.Context, tx pgx.Tx, tenantID, entityDefinitionID, except uuid.UUID,
	field string, value any, fixed map[string]string) (bool, error) {
	v, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("records/store: encode unique value: %w", err)
	}
	args := []any{tenantID, entityDefinitionID, except, v}
	var match string
	switch fixed[field] {
	case "":
		match, args = `r.data -> $5 = $4::jsonb`, append(args, field)
	case metadata.ColumnState:
		match = `to_jsonb(r.state) = $4::jsonb`
	case metadata.ColumnRequesterID:
		match = `to_jsonb(r.requester_id) = $4::jsonb`
	case metadata.ColumnAssigneeID:
		match = `to_jsonb(r.assignee_id) = $4::jsonb`
	case metadata.ColumnAssignmentGroupID:
		match = `to_jsonb(r.assignment_group_id) = $4::jsonb`
	default:
		return false, fmt.Errorf("records/store: unknown fixed column %q", fixed[field])
	}
	var taken bool
	err = tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM records r
    WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.id <> $3 AND `+match+`)`,
		args...).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("records/store: check unique value: %w", err)
	}
	return taken, nil
}
