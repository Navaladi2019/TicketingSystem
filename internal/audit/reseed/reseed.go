// Package reseed is chain:reseed (ADR-004 R17, PLAN-005 C5 as amended on
// TICK-439/TICK-446): it restarts a tenant's dead seal lineage.
//
// Do runs in the caller's nava_app transaction, after the audit.export gate.
// The R17 predicate lives in internal/audit/lineage and is not restated here.
package reseed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/lineage"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

var (
	// ErrChainAlive means the lineage is not dead (R17): 409 chain_alive.
	// Nothing is written.
	ErrChainAlive = errors.New("audit/reseed: the seal lineage is alive")
	// ErrHeadNotFound means the tenant has no audit_chain_head row visible to
	// the transaction: 404, the same answer for another tenant's head.
	ErrHeadNotFound = lineage.ErrHeadNotFound
)

// Result is a completed reseed.
type Result struct {
	// ReseedID is the id of the audit/reseed entry.
	ReseedID uuid.UUID
	// OutboxID is the new lineage's start row, now next_seal_outbox_id.
	OutboxID uuid.UUID
	// PreviousOutboxID is the retired lineage's start row, if it had one.
	PreviousOutboxID *uuid.UUID
}

const (
	headSQL    = `SELECT id, version, next_seal_outbox_id FROM audit_chain_head WHERE tenant_id = $1`
	repointSQL = `UPDATE audit_chain_head SET next_seal_outbox_id = $2, next_seal_at = now() WHERE tenant_id = $1`
)

// Do reseeds tenant's chain in tx, which must come from tenancy.InTenantTx
// for tenant (the validated token's tenant, never request input). Under the
// head lock it refuses with ErrChainAlive unless R17 allows; otherwise it
// writes the audit/reseed entry, enqueues the new start row at the head's
// current version without bumping it (outbox.EnqueueReseed, keyed on the
// entry's id) and repoints next_seal_outbox_id and next_seal_at. On any error
// the caller rolls tx back.
func Do(ctx context.Context, tx pgx.Tx, tenant, principal uuid.UUID) (Result, error) {
	allowed, err := lineage.ReseedAllowed(ctx, tx, tenant) // takes the head lock
	if err != nil {
		return Result{}, err
	}
	if !allowed {
		return Result{}, ErrChainAlive
	}
	var (
		head    uuid.UUID
		version int64
		res     Result
	)
	if err := tx.QueryRow(ctx, headSQL, tenant).Scan(&head, &version, &res.PreviousOutboxID); err != nil {
		return Result{}, fmt.Errorf("audit/reseed: read head: %w", err)
	}
	if res.ReseedID, err = uuid.NewV7(); err != nil {
		return Result{}, fmt.Errorf("audit/reseed: mint id: %w", err)
	}
	if res.OutboxID, err = outbox.EnqueueReseed(ctx, tx, version, res.ReseedID); err != nil {
		return Result{}, fmt.Errorf("audit/reseed: enqueue: %w", err)
	}
	diff, err := json.Marshal([]any{map[string]any{
		"op": "replace", "path": "/next_seal_outbox_id", "value": res.OutboxID,
	}})
	if err != nil {
		return Result{}, fmt.Errorf("audit/reseed: encode diff: %w", err)
	}
	if _, err := audit.Append(ctx, tx, audit.Entry{
		ID: res.ReseedID, Domain: audit.DomainAudit, Action: audit.ActionReseed,
		SubjectType: audit.SubjectAuditChain, SubjectID: head,
		Actor:  audit.Actor{PrincipalID: &principal},
		Source: audit.SourceUser,
		Diff:   diff,
	}); err != nil {
		return Result{}, fmt.Errorf("audit/reseed: write entry: %w", err)
	}
	if _, err := tx.Exec(ctx, repointSQL, tenant, res.OutboxID); err != nil {
		return Result{}, fmt.Errorf("audit/reseed: repoint head: %w", err)
	}
	return res, nil
}
