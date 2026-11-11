// Package export is the audit log's JSONL export (ADR-004 R19-R21, PLAN-005
// C4, audit-export-v1.md): the lease and the audit/export entry that commit
// before the first byte, then the anchored, paged stream.
//
// Begin runs in the caller's nava_app transaction (ADR-004-A1 A1-7), after the
// audit.export gate. Write then streams on the masking reader's pool, one
// short transaction per page.
package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The R21 numbers (spike S1, P5-01): page size, per-page statement_timeout
// and the cap on one export.
const (
	DefaultPageSize         = 5000
	DefaultStatementTimeout = 30 * time.Second
	DefaultCap              = 15 * time.Minute
	// leaseSlack keeps the lease alive a little past the cap, so a slow last
	// page cannot let a second export start.
	leaseSlack = time.Minute
)

// ErrInProgress means the tenant's export lease is held (R19): 409
// export_in_progress.
var ErrInProgress = errors.New("audit/export: an export is already in progress for this tenant")

// ErrNoHead means the tenant has no audit_chain_head row: a wiring or
// onboarding bug, answered 500.
var ErrNoHead = errors.New("audit/export: the tenant has no audit chain head")

// Request is what the caller asked for.
type Request struct {
	Filter read.ExportFilter
	// Stubs is the request's stubs flag: true emits every non-matching seq as
	// a stub line. It is recorded in the header as received (ADR-004-E2).
	Stubs bool
	// CSV is the requested file format: false is JSONL (the default). It is
	// recorded in the audit/export entry's request diff.
	CSV bool
}

// Session is a started export: the lease is held and the audit/export entry
// has committed with the caller's transaction.
type Session struct {
	TenantID   uuid.UUID
	ExportID   uuid.UUID
	ExportedAt time.Time
	HeadSeq    int64
	HeadHash   [32]byte
	Request    Request
}

// Begin takes the tenant's export lease and writes the audit/export entry, in
// tx. The caller commits tx before it calls Write, so the entry is durable
// before the first byte (R19). One statement reads the head and takes the
// lease: zero rows means the lease is held (ErrInProgress) or there is no head
// (ErrNoHead). head_seq is read here, once, and bounds the whole export.
func Begin(ctx context.Context, tx pgx.Tx, tenantID, principalID uuid.UUID, req Request, cap time.Duration) (Session, error) {
	if cap <= 0 {
		cap = DefaultCap
	}
	s := Session{TenantID: tenantID, Request: req}
	var hash []byte
	err := tx.QueryRow(ctx, `UPDATE audit_chain_head
SET export_lease_until = now() + make_interval(secs => $2)
WHERE tenant_id = $1 AND (export_lease_until IS NULL OR export_lease_until < now())
RETURNING seq, entry_hash`, tenantID, (cap+leaseSlack).Seconds()).Scan(&s.HeadSeq, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		var one int
		if e := tx.QueryRow(ctx, `SELECT 1 FROM audit_chain_head WHERE tenant_id = $1`, tenantID).Scan(&one); errors.Is(e, pgx.ErrNoRows) {
			return Session{}, ErrNoHead
		} else if e != nil {
			return Session{}, fmt.Errorf("audit/export: probe head: %w", e)
		}
		return Session{}, ErrInProgress
	}
	if err != nil {
		return Session{}, fmt.Errorf("audit/export: take lease: %w", err)
	}
	if len(hash) != 32 {
		return Session{}, fmt.Errorf("audit/export: head hash has %d bytes", len(hash))
	}
	copy(s.HeadHash[:], hash)

	id, err := uuid.NewV7()
	if err != nil {
		return Session{}, fmt.Errorf("audit/export: mint id: %w", err)
	}
	diff, err := requestDiff(req)
	if err != nil {
		return Session{}, err
	}
	if _, err := audit.Append(ctx, tx, audit.Entry{
		ID: id, Domain: audit.DomainAudit, Action: audit.ActionExport,
		SubjectType: audit.SubjectAuditExport, SubjectID: id,
		Actor:  audit.Actor{PrincipalID: &principalID},
		Source: audit.SourceUser,
		Diff:   diff,
	}); err != nil {
		return Session{}, fmt.Errorf("audit/export: write entry: %w", err)
	}
	s.ExportID = id
	if err := tx.QueryRow(ctx, `SELECT occurred_at FROM audit_log WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&s.ExportedAt); err != nil {
		return Session{}, fmt.Errorf("audit/export: read entry time: %w", err)
	}
	return s, nil
}

// requestDiff records the request as one RFC 6902 add op, so the audit trail
// shows what was exported. It holds filter values and ids, never entry data.
func requestDiff(req Request) ([]byte, error) {
	format := "jsonl"
	if req.CSV {
		format = "csv"
	}
	b, err := json.Marshal([]any{map[string]any{
		"op": "add", "path": "/request",
		"value": map[string]any{"format": format, "stubs": req.Stubs, "filter": filterJSON(req.Filter)},
	}})
	if err != nil {
		return nil, fmt.Errorf("audit/export: encode request: %w", err)
	}
	return b, nil
}

// Release clears the tenant's lease. It runs on a context that survives the
// request's cancellation, and a failure is not fatal: the lease expires on its
// own after the cap.
func Release(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return tenancy.InTenantTx(ctx, pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE audit_chain_head SET export_lease_until = NULL WHERE tenant_id = $1`, tenantID)
		if err != nil {
			return fmt.Errorf("audit/export: release lease: %w", err)
		}
		return nil
	})
}
