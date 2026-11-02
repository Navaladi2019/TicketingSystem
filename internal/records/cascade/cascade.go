// Package cascade is the start key of ADR-001 amendment A5: a workflow
// definition starts at most once per (tenant, origin, record).
//
// The EventSink (P3-42) calls ClaimStart in the triggering write's
// transaction before it enqueues a nava.interpreter start of definition D on
// record R. A claimed Claim is the only value outbox.EnqueueStart takes, so a
// start cannot be enqueued without its key (A5-6). On a repeat the sink
// enqueues nothing, calls pipeline.AuditSuppressed, and the write commits
// (A5-4). This package writes no record table: the audit row goes through the
// pipeline, the only writer of audit_log (PLAN-003 D-3-6).
//
// The tenant, origin and depth of a claim are never arguments: ClaimStart
// reads them from R's records row, which row-level security lets it see only
// for the transaction's tenant, as the pipeline stored them for this write.
// No outbox payload or definition data can name the key (A5, I9).
package cascade

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Audit actions and reasons of the cascade audit rows the pipeline writes
// (audit_log.action; the reason is in the row's after, under DetailReason).
const (
	// ActionCycle is a suppressed start (A5-4): the write committed and D
	// did not start again on R under this origin.
	ActionCycle = "cascade_cycle"
	// ActionRejected is a write the pipeline refused for its depth (I9); the
	// write rolled back.
	ActionRejected = "cascade_rejected"

	ReasonCycle   = "cascade_cycle"
	ReasonTooDeep = "cascade_too_deep"

	// DetailReason and DetailDefinition are the keys of a cascade audit
	// row's detail: its reason, and the suppressed definition.
	DetailReason     = "reason"
	DetailDefinition = "definition_id"

	pkeyConstraint = "cascade_start_pkey"
)

var (
	// ErrRecordNotFound means the transaction's tenant has no such record,
	// or the transaction has no tenant.
	ErrRecordNotFound = errors.New("cascade: record not found")
	// ErrNoOrigin means the record carries no causation: a row the write
	// pipeline never wrote. Its writes cannot be keyed (A5-3).
	ErrNoOrigin = errors.New("cascade: record has no causation origin")
	// ErrInvalidKey means the key names no record or no definition.
	ErrInvalidKey = errors.New("cascade: invalid key")
)

// Key names a start: definition DefinitionID on record RecordID. The origin
// and tenant are read from the record, never given.
type Key struct {
	RecordID     uuid.UUID
	DefinitionID uuid.UUID
}

// Claim is the result of ClaimStart. Its fields are unexported so only
// ClaimStart can build one (A5-6): a claimed Claim is the token
// outbox.EnqueueStart requires, and its OutboxID is the start row's id.
type Claim struct {
	outboxID     uuid.UUID // uuid.Nil unless claimed
	tenantID     uuid.UUID
	originID     uuid.UUID
	depth        int16
	recordID     uuid.UUID
	definitionID uuid.UUID
}

// Claimed reports whether this claim holds the key.
func (c Claim) Claimed() bool { return c.outboxID != uuid.Nil }

// OutboxID is the id the start row must take; uuid.Nil unless claimed.
func (c Claim) OutboxID() uuid.UUID { return c.outboxID }

// TenantID is the record's tenant.
func (c Claim) TenantID() uuid.UUID { return c.tenantID }

// RecordID is R.
func (c Claim) RecordID() uuid.UUID { return c.recordID }

// DefinitionID is D.
func (c Claim) DefinitionID() uuid.UUID { return c.definitionID }

// Causation is the origin and depth stored on R by the triggering write.
func (c Claim) Causation() (origin uuid.UUID, depth int16) { return c.originID, c.depth }

const (
	// The triggering write already holds this lock (A5-1); taking it again
	// is free and keeps the order when ClaimStart is called on its own.
	lockRecordSQL = `SELECT tenant_id, causation_origin_id, causation_depth FROM records WHERE id = $1 FOR UPDATE`
	claimSQL      = `INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ON CONSTRAINT ` + pkeyConstraint + ` DO NOTHING
RETURNING outbox_id`
)

// ClaimStart claims the start of k.DefinitionID on k.RecordID under the
// origin stored on the record, in tx. claimed is true when this is the
// first start of the key: enqueue it with outbox.EnqueueStart(ctx, tx, c, ...).
// claimed is false on a cycle: enqueue nothing and call
// pipeline.AuditSuppressed.
//
// A concurrent claim of the same key waits on R's row lock and then on the
// uncommitted key: it gets claimed = false if the first commits and claims
// the key if the first rolls back. If tx rolls back the claim is released.
//
// tx must come from tenancy.InTenantTx. On an error the caller rolls tx back.
func ClaimStart(ctx context.Context, tx pgx.Tx, k Key) (c Claim, claimed bool, err error) {
	if k.RecordID == uuid.Nil || k.DefinitionID == uuid.Nil {
		return Claim{}, false, fmt.Errorf("%w: no record or no definition", ErrInvalidKey)
	}
	var (
		origin *uuid.UUID
		depth  *int16
	)
	c = Claim{recordID: k.RecordID, definitionID: k.DefinitionID}
	switch err := tx.QueryRow(ctx, lockRecordSQL, k.RecordID).Scan(&c.tenantID, &origin, &depth); {
	case errors.Is(err, pgx.ErrNoRows):
		return Claim{}, false, ErrRecordNotFound
	case err != nil:
		return Claim{}, false, fmt.Errorf("cascade: lock record: %w", err)
	}
	if origin == nil || *origin == uuid.Nil || depth == nil {
		return Claim{}, false, ErrNoOrigin
	}
	c.originID, c.depth = *origin, *depth

	id, err := uuid.NewV7()
	if err != nil {
		return Claim{}, false, fmt.Errorf("cascade: new outbox id: %w", err)
	}
	var got uuid.UUID
	switch err := tx.QueryRow(ctx, claimSQL, c.tenantID, c.originID, c.recordID, c.definitionID, id).Scan(&got); {
	case errors.Is(err, pgx.ErrNoRows):
		return c, false, nil
	case err != nil:
		return Claim{}, false, fmt.Errorf("cascade: claim start: %w", err)
	}
	c.outboxID = got
	return c, true, nil
}
