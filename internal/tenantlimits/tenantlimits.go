// Package tenantlimits reads and writes a tenant's limits row, tenant_limits
// (S22, TICK-751, ADR-006 D12, I8): today the per-type filterable-field limit
// an operator may raise from the default 5 up to 10. No row is the default.
//
// Every function takes the caller's transaction, opened by
// tenancy.InTenantTx for the tenant concerned, so row-level security lets it
// see and change that tenant's row and no other. A change writes one config
// audit entry under the tenant_limits subject in the same transaction, which
// the table's commit check requires.
package tenantlimits

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
)

// The bounds of filterable_fields_per_type (ADR-006 D12), as
// tenant_limits_filterable_fields_per_type_check holds them.
const (
	MinPerType = fieldindex.DefaultPerTypeLimit
	MaxPerType = 10
	// MaxReasonBytes bounds the operator's reason.
	MaxReasonBytes = 500
	// maxOperatorBytes matches tenant_limits_updated_by_check.
	maxOperatorBytes = 255
	// actorPrefix names the operator in the audit entry's actor_system.
	actorPrefix = "operator:"
)

var (
	// ErrNotFound means the transaction's tenant has no tenants row.
	ErrNotFound = errors.New("tenantlimits: tenant not found")
	// ErrOutOfRange means the limit is outside MinPerType..MaxPerType.
	ErrOutOfRange = errors.New("tenantlimits: filterable_fields_per_type out of range")
	// ErrNoReason means the change names no reason.
	ErrNoReason = errors.New("tenantlimits: reason required")
	// ErrNoOperator means the change names no operator: the audit actor is
	// never taken from the request body.
	ErrNoOperator = errors.New("tenantlimits: write needs an operator")
)

// Limits is a tenant's limits as they apply: the stored row, or the defaults
// when there is none.
type Limits struct {
	FilterableFieldsPerType int
}

// PerType returns the most flagged fields one entity type of tenantID may
// have: the stored row's value, or fieldindex.DefaultPerTypeLimit when the
// tenant has none. Publish calls it in its own transaction (P7-04).
func PerType(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT filterable_fields_per_type FROM tenant_limits WHERE tenant_id = $1`, tenantID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return fieldindex.DefaultPerTypeLimit, nil
	}
	if err != nil {
		return 0, fmt.Errorf("tenantlimits: read: %w", err)
	}
	return n, nil
}

// SetPerType sets tenantID's per-type limit to n for the reason given, as
// operator, and audits the change. Setting the value already in force writes
// nothing and no audit entry, so a replay is one entry. It returns the limits
// now in force.
func SetPerType(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, n int, reason, operator string) (Limits, error) {
	if n < MinPerType || n > MaxPerType {
		return Limits{}, ErrOutOfRange
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > MaxReasonBytes || !utf8.ValidString(reason) {
		return Limits{}, ErrNoReason
	}
	if operator == "" || len(operator) > maxOperatorBytes || !utf8.ValidString(operator) {
		return Limits{}, ErrNoOperator
	}
	// Two concurrent changes of one tenant serialise on a transaction-level
	// advisory lock (a first INSERT has no row to lock), so the second reads
	// the first's committed value: the audit diff is always the real change.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('tenant_limits:' || $1::text, 0))`, tenantID); err != nil {
		return Limits{}, fmt.Errorf("tenantlimits: lock: %w", err)
	}
	var one int
	err := tx.QueryRow(ctx, `SELECT 1 FROM tenants WHERE id = $1`, tenantID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return Limits{}, ErrNotFound
	}
	if err != nil {
		return Limits{}, fmt.Errorf("tenantlimits: read tenant: %w", err)
	}
	before := fieldindex.DefaultPerTypeLimit
	err = tx.QueryRow(ctx, `SELECT filterable_fields_per_type FROM tenant_limits WHERE tenant_id = $1`, tenantID).Scan(&before)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Limits{}, fmt.Errorf("tenantlimits: lock: %w", err)
	}
	if before == n {
		return Limits{FilterableFieldsPerType: n}, nil
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO tenant_limits (tenant_id, filterable_fields_per_type, updated_at, updated_by)
VALUES ($1, $2, now(), $3)
ON CONFLICT (tenant_id) DO UPDATE SET filterable_fields_per_type = EXCLUDED.filterable_fields_per_type,
    updated_at = EXCLUDED.updated_at, updated_by = EXCLUDED.updated_by`, tenantID, n, operator); err != nil {
		return Limits{}, fmt.Errorf("tenantlimits: write: %w", err)
	}
	if err := auditChange(ctx, tx, tenantID, before, n, reason, operator); err != nil {
		return Limits{}, err
	}
	return Limits{FilterableFieldsPerType: n}, nil
}

// auditChange writes the config update entry of a limit change: the diff is
// {"filterable_fields_per_type": before} to {"filterable_fields_per_type":
// after, "reason": reason}; content_sha256 (R3) hashes the after document. No
// version moves, so both version fields are NULL. The actor is the operator,
// as actor_system operator:<subject>, source system.
func auditChange(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, before, after int, reason, operator string) error {
	b, err := json.Marshal(map[string]any{"filterable_fields_per_type": before})
	if err != nil {
		return fmt.Errorf("tenantlimits: encode: %w", err)
	}
	a, err := json.Marshal(map[string]any{"filterable_fields_per_type": after, "reason": reason})
	if err != nil {
		return fmt.Errorf("tenantlimits: encode: %w", err)
	}
	diff, err := jsondiff.Diff(b, a)
	if err != nil {
		return fmt.Errorf("tenantlimits: diff: %w", err)
	}
	sum := sha256.Sum256(a)
	who := actorPrefix + operator
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        audit.ActionUpdate,
		SubjectType:   audit.SubjectTenantLimits,
		SubjectID:     tenantID,
		Actor:         audit.Actor{System: &who},
		Source:        audit.SourceSystem,
		ContentSHA256: &sum,
		Diff:          diff,
	}
	if id := requestid.FromContext(ctx); id != "" {
		e.RequestID = &id
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("tenantlimits: audit: %w", err)
	}
	return nil
}
