package entitytypes

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

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
// that is not needs_backfill and adds none writes neither (D1: every
// needs_backfill publish writes the job, even one that adds no new default,
// such as a property with a default becoming required). A replay of the same publish finds both rows
// already there (same primary key and dedupe key) and writes nothing more.
func queueBackfill(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, version int, auditID uuid.UUID, class metadata.Compat, prev, next *metadata.Compiled, after func() error) error {
	if class != metadata.NeedsBackfill && !addsDefault(prev, next) {
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

// RemainingCap is the most a backfill's remaining count reads: 1,001 means
// more than 1,000 (ADR-009 D10, D12).
const RemainingCap = 1001

// BackfillStatus is a type's newest schema backfill job, the schema_backfills
// row with the highest target_version, as GET /entity-types/{key} and the
// D12 list filter read it (ADR-009 D10).
type BackfillStatus struct {
	State         string
	TargetVersion int
	DefaultKeys   []string
	Rewritten     int64
	Skipped       int64
	Failed        int64
	// Remaining is the job's pending rows plus its failure rows, at most
	// RemainingCap.
	Remaining  int64
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// Open reports whether the job's window is open (ADR-009 Terms): the newest
// job is not yet done, so a record may still lack a default in storage.
func (b *BackfillStatus) Open() bool {
	switch b.State {
	case "pending", "filling", "running", "failed_records":
		return true
	}
	return false
}

// newestBackfillSQL reads the newest job of one type and its remaining count
// in one statement. The count reads at most 1,001 rows (RemainingCap) of the queue and
// the failures together. nava_app holds SELECT only on the three tables.
const newestBackfillSQL = `
SELECT b.state, b.target_version, b.default_keys, b.rewritten, b.skipped, b.failed, b.started_at, b.finished_at,
       (SELECT count(*) FROM (
            SELECT 1 FROM schema_backfill_pending p
             WHERE p.tenant_id = b.tenant_id AND p.entity_definition_id = b.entity_definition_id AND p.target_version = b.target_version
            UNION ALL
            SELECT 1 FROM schema_backfill_failures f
             WHERE f.tenant_id = b.tenant_id AND f.entity_definition_id = b.entity_definition_id AND f.target_version = b.target_version
            LIMIT 1001) q)
FROM schema_backfills b
WHERE b.tenant_id = $1 AND b.entity_definition_id = $2
ORDER BY b.target_version DESC
LIMIT 1`

// NewestBackfill returns the newest job of the type entityKey (by
// target_version), or nil when the type has none or the tenant has no such
// type. It reads in tx, under the transaction's tenant.
func (Store) NewestBackfill(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, entityKey string) (*BackfillStatus, error) {
	var defID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, tenantID, entityKey).Scan(&defID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil //nolint:nilnil // no such type has no job
		}
		return nil, fmt.Errorf("entitytypes: read entity type: %w", err)
	}
	return newestBackfill(ctx, tx, tenantID, defID)
}

func newestBackfill(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID) (*BackfillStatus, error) {
	var b BackfillStatus
	err := tx.QueryRow(ctx, newestBackfillSQL, tenantID, defID).Scan(
		&b.State, &b.TargetVersion, &b.DefaultKeys, &b.Rewritten, &b.Skipped, &b.Failed, &b.StartedAt, &b.FinishedAt, &b.Remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // no job is a nil status
	}
	if err != nil {
		return nil, fmt.Errorf("entitytypes: read newest backfill: %w", err)
	}
	return &b, nil
}

// NewestBackfillOf is NewestBackfill for a caller that already holds the
// definition id (the list handler reads it once per request).
func (Store) NewestBackfillOf(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID) (*BackfillStatus, error) {
	return newestBackfill(ctx, tx, tenantID, defID)
}

// BackfillFailure is one record the job could not rewrite: its id and a stable
// code, never a value (ADR-009 D7).
type BackfillFailure struct {
	RecordID uuid.UUID
	Code     string
}

// BackfillFailures returns up to limit failure rows of the type's newest job
// with record_id after the given id (keyset, ascending). It returns no rows
// for a type that has no job, and ErrNotFound for a key the tenant does not
// have. It reads in tx, under the transaction's tenant.
func (Store) BackfillFailures(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, entityKey string, after uuid.UUID, limit int) ([]BackfillFailure, error) {
	var defID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, tenantID, entityKey).Scan(&defID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("entitytypes: read entity type: %w", err)
	}
	rows, err := tx.Query(ctx, `
SELECT f.record_id, f.code
FROM schema_backfill_failures f
WHERE f.tenant_id = $1 AND f.entity_definition_id = $2
  AND f.target_version = (SELECT max(b.target_version) FROM schema_backfills b WHERE b.tenant_id = $1 AND b.entity_definition_id = $2)
  AND f.record_id > $3
ORDER BY f.record_id
LIMIT $4`, tenantID, defID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: read backfill failures: %w", err)
	}
	defer rows.Close()
	var out []BackfillFailure
	for rows.Next() {
		var f BackfillFailure
		if err := rows.Scan(&f.RecordID, &f.Code); err != nil {
			return nil, fmt.Errorf("entitytypes: read backfill failures: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
