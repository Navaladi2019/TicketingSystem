// Package lineage holds ADR-004 R17's one predicate on audit_chain_head: a
// seal lineage is dead once next_seal_at < now() - 10 min. Two readers use it
// and nothing else may restate it:
//
//   - the chain:reseed gate (ReseedAllowed, nava_app, under the head lock),
//     which answers 409 chain_alive while the predicate is false;
//   - the dead-lineage gauge (Monitor, nava_purge, every tenant), which
//     raises AuditDeadLineage while any head matches it.
//
// So the alert fires exactly when a reseed becomes allowed (TICK-590). The
// nava_purge SELECT policy on audit_chain_head is the same predicate;
// internal/db/catalog pins its deparsed form, PolicyQual.
package lineage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DeadAfter is R17's threshold, the interval Predicate spells out.
const DeadAfter = 10 * time.Minute

// Predicate is R17's test on an audit_chain_head row. A head with no
// next_seal_at (no lineage yet) does not match: NULL is not less than now.
const Predicate = "next_seal_at < now() - interval '10 minutes'"

// PolicyQual is Predicate as PostgreSQL deparses it into pg_policies.qual,
// for the audit_chain_head_purge_select policy (internal/db/catalog).
const PolicyQual = "(next_seal_at < (now() - '00:10:00'::interval))"

// ReseedGateSQL locks the tenant's head and evaluates Predicate on it, in
// the reseed transaction. nava_app's tenant policy limits it to its tenant.
const ReseedGateSQL = "SELECT " + Predicate + " FROM audit_chain_head WHERE tenant_id = $1 FOR UPDATE"

// DeadLineagesSQL lists every dead head, in tenant order. As nava_purge the
// policy already limits the rows to Predicate; the WHERE keeps the query
// correct on its own.
const DeadLineagesSQL = "SELECT tenant_id, next_seal_at FROM audit_chain_head WHERE " + Predicate + " ORDER BY tenant_id"

// ErrHeadNotFound means the tenant has no audit_chain_head row.
var ErrHeadNotFound = errors.New("lineage: audit chain head not found")

// ReseedAllowed reports whether R17 lets chain:reseed proceed for tenant:
// true when the lineage is dead. It takes the head lock (FOR UPDATE), so call
// it first in the transaction that repoints the head, opened as nava_app with
// the tenant set.
func ReseedAllowed(ctx context.Context, tx pgx.Tx, tenant uuid.UUID) (bool, error) {
	var dead *bool
	err := tx.QueryRow(ctx, ReseedGateSQL, tenant).Scan(&dead)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrHeadNotFound
	}
	if err != nil {
		return false, fmt.Errorf("lineage: read head: %w", err)
	}
	return dead != nil && *dead, nil
}

// Dead is one dead lineage: its tenant and when its next run was due.
type Dead struct {
	TenantID   uuid.UUID
	NextSealAt time.Time
}

// Querier is the part of a pool DeadLineages needs. *pgxpool.Pool and
// *pgx.Conn satisfy it.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// DeadLineages returns every dead lineage, across tenants. Connect as
// nava_purge, the only role whose policy lets it read other tenants' heads.
func DeadLineages(ctx context.Context, db Querier) ([]Dead, error) {
	rows, err := db.Query(ctx, DeadLineagesSQL)
	if err != nil {
		return nil, fmt.Errorf("lineage: read dead lineages: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Dead, error) {
		var d Dead
		err := r.Scan(&d.TenantID, &d.NextSealAt)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("lineage: read dead lineages: %w", err)
	}
	return out, nil
}
