// Package relay is the claim side of the transactional outbox (ADR-001 A.2
// "Claim order" and "Worst-case bound"; PLAN-001 T30a). It connects as
// nava_relay, which reads every tenant's pending rows through the
// outbox_relay_* policies and may write only the dispatch columns. Dispatch to
// Temporal is T30b, which calls Claim, then MarkDispatched, Backoff or
// MarkDead for each row.
//
// A claim is a lease, not a held lock: Claim commits with the row's
// available_at moved to the end of the lease and attempts incremented. Until
// the lease ends no other claimer sees the row, and its successors on the same
// target are not candidates while it is pending. attempts is the lease's
// fencing token: the Mark* calls and Backoff match it, so a claimer whose
// lease ended and was taken over changes nothing (ErrLeaseLost).
package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric"
)

const (
	// PerTenant is the most rows one poll claims for one tenant (A.2).
	PerTenant = 2
	// ScanCap is the most pending rows one poll scans for one tenant (A.2).
	ScanCap = 200
	// MaxRedeliveryAge is D-P8: a row still failing this long after its
	// redelivery started is marked dead. It must stay shorter than the worker's
	// applied-signal window (workerguard.SignalDedupeWindow, 48 h).
	MaxRedeliveryAge = 24 * time.Hour
	// MaxErrorLen bounds last_error, in bytes.
	MaxErrorLen = 1000
	// TakeoverExpiredError is the last_error Claim writes when it marks a row
	// dead at a lease takeover.
	TakeoverExpiredError = "relay: max redelivery age reached at lease takeover"
)

// ErrLeaseLost means the row is no longer under this claim: its lease ended
// and another claim took it, or it is already dispatched or dead. Nothing was
// written.
var ErrLeaseLost = errors.New("relay: lease lost")

// Querier runs the relay's statements. *pgxpool.Pool and *pgx.Conn satisfy it.
// It must be connected as nava_relay; no tenant is set.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Claimed is one claimed outbox row: every column nava_relay may read.
type Claimed struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Kind           string
	KindVersion    int16
	TargetKey      string
	TargetSeq      int64
	CausationID    *uuid.UUID
	CausationDepth int16
	Payload        json.RawMessage
	TraceContext   *string // W3C traceparent, or nil
	CreatedAt      time.Time
	// Attempts counts claims, this one included, so it is at least 1. It is
	// the lease's fencing token.
	Attempts int32
	// LeaseUntil is when the row becomes claimable again if nothing is marked.
	LeaseUntil time.Time
}

// IsStart reports whether the row starts a workflow. A row is a start exactly
// when its target_key is its kind, a colon and its own id in canonical
// lowercase form (outbox.Enqueue writes it so); every other row is a signal
// to the workflow its target_key names. Nothing else is parsed from it.
func (c Claimed) IsStart() bool {
	return c.TargetKey == c.Kind+":"+c.ID.String()
}

// Options tune a Relay. Zero fields take the defaults.
type Options struct {
	// Lease is how long a claim hides its row. Default 30 s.
	Lease time.Duration
	// BackoffBase is the first retry delay; each later attempt doubles it.
	// Default 1 s.
	BackoffBase time.Duration
	// BackoffMax caps the retry delay. Default 5 min.
	BackoffMax time.Duration
}

// Relay claims and settles outbox rows. Build it with New.
type Relay struct {
	db      Querier
	log     *slog.Logger
	opts    Options
	scanned metric.Int64Histogram
}

// scanBuckets are the boundaries of outbox_relay_rows_scanned (TICK-94 item
// 7). Buckets are upper-inclusive, so a capped scan (200) is in le="200" and
// anything above the cap is not.
func scanBuckets() []float64 { return []float64{0, 1, 10, 50, 100, 150, 200, 250, 500} }

// New registers outbox_relay_rows_scanned on mp, which must come from
// internal/platform/telemetry in production.
func New(db Querier, mp metric.MeterProvider, log *slog.Logger, opts Options) (*Relay, error) {
	if opts.Lease <= 0 {
		opts.Lease = 30 * time.Second
	}
	if opts.BackoffBase <= 0 {
		opts.BackoffBase = time.Second
	}
	if opts.BackoffMax <= 0 {
		opts.BackoffMax = 5 * time.Minute
	}
	h, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/relay").Int64Histogram(
		"outbox_relay_rows_scanned",
		metric.WithDescription("Pending outbox rows one relay poll scanned for one tenant. One observation per tenant per poll; never labelled by tenant."),
		metric.WithExplicitBucketBoundaries(scanBuckets()...))
	if err != nil {
		return nil, fmt.Errorf("relay: %w", err)
	}
	return &Relay{db: db, log: log, opts: opts, scanned: h}, nil
}

// claimSQL is one statement, so its scan, locks and lease are one snapshot
// and one transaction.
//
//   - tenants walks the pending index's distinct tenant_ids with a loose index
//     scan (nava_relay cannot read tenants).
//   - window is each tenant's available pending rows, oldest first, at most
//     $1 (ScanCap): the rows the poll scans.
//   - picked takes at most $2 (PerTenant) rows per tenant from its window
//     that have no lower pending target_seq on their target (FIFO per
//     target), locking them FOR UPDATE SKIP LOCKED. The pending and available
//     tests sit on the locked row, so a row a concurrent claimer leased
//     since this snapshot fails the recheck and is skipped.
//   - stranded finds heads that sort behind their own successors (TICK-248).
//     A claim moves a head's available_at to the lease end and leaves its
//     successors where they are, so a head whose lease ended with no settle
//     (relay crash, settle_failed) sorts after them. With ScanCap of them
//     older, the window holds only blocked successors and never the head.
//     For a tenant whose window is full and that picked fewer than $2 rows,
//     stranded takes the rest from the heads of the window's targets that
//     sort outside it (a pending target_seq below the target's lowest in the
//     window): one outbox_pending_target probe per target, at most $1,
//     oldest target first, with the same locks and rechecks as picked. A
//     head inside the window is picked's, not stranded's.
//   - claimed moves each picked or stranded row's available_at to the lease
//     end and increments attempts. One that was attempted before (its lease
//     ended or its backoff is over) and is $4 (MaxRedeliveryAge) old is
//     marked dead instead, with $5 as last_error (killed). A crash between
//     dispatch and MarkDispatched never reaches Backoff, so without this the
//     row would be redelivered past the worker's signal window. A row never
//     attempted is never marked dead.
//
// The result has one row per scanned tenant and claimed or killed row; a
// tenant with neither has one row with a NULL id.
const claimSQL = `WITH RECURSIVE tenants AS (
    (SELECT tenant_id FROM outbox WHERE dispatched_at IS NULL AND dead_at IS NULL
     ORDER BY tenant_id LIMIT 1)
    UNION ALL
    SELECT (SELECT o.tenant_id FROM outbox o
            WHERE o.tenant_id > t.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL
            ORDER BY o.tenant_id LIMIT 1)
    FROM tenants t WHERE t.tenant_id IS NOT NULL
),
window_rows AS MATERIALIZED (
    SELECT w.tenant_id, w.id, w.target_key, w.target_seq, w.available_at
    FROM tenants t
    CROSS JOIN LATERAL (
        SELECT o.tenant_id, o.id, o.target_key, o.target_seq, o.available_at FROM outbox o
        WHERE o.tenant_id = t.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL
          AND o.available_at <= now()
        ORDER BY o.available_at, o.id
        LIMIT $1
    ) w
    WHERE t.tenant_id IS NOT NULL
),
scan AS (
    SELECT t.tenant_id, count(w.id) AS scanned
    FROM tenants t LEFT JOIN window_rows w ON w.tenant_id = t.tenant_id
    WHERE t.tenant_id IS NOT NULL
    GROUP BY t.tenant_id
),
picked AS (
    SELECT p.tenant_id, p.id, p.expired
    FROM scan s
    CROSS JOIN LATERAL (
        SELECT o.tenant_id, o.id,
               o.attempts > 0 AND o.created_at <= now() - make_interval(secs => $4) AS expired
        FROM window_rows w
        JOIN outbox o ON o.tenant_id = w.tenant_id AND o.id = w.id
        WHERE w.tenant_id = s.tenant_id
          AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
          AND NOT EXISTS (
              SELECT 1 FROM outbox h
              WHERE h.tenant_id = o.tenant_id AND h.target_key = o.target_key
                AND h.target_seq < o.target_seq AND h.dispatched_at IS NULL AND h.dead_at IS NULL)
        ORDER BY w.available_at, w.id
        LIMIT $2
        FOR UPDATE OF o SKIP LOCKED
    ) p
),
room AS (
    SELECT s.tenant_id, $2 - count(p.id) AS room
    FROM scan s LEFT JOIN picked p ON p.tenant_id = s.tenant_id
    WHERE s.scanned >= $1
    GROUP BY s.tenant_id
    HAVING count(p.id) < $2
),
stranded AS (
    SELECT p.tenant_id, p.id, p.expired
    FROM room r
    CROSS JOIN LATERAL (
        SELECT o.tenant_id, o.id,
               o.attempts > 0 AND o.created_at <= now() - make_interval(secs => $4) AS expired
        FROM (SELECT w.target_key, min(w.target_seq) AS low, min(w.available_at) AS since
              FROM window_rows w WHERE w.tenant_id = r.tenant_id
              GROUP BY w.target_key) k
        CROSS JOIN LATERAL (
            SELECT h.id FROM outbox h
            WHERE h.tenant_id = r.tenant_id AND h.target_key = k.target_key AND h.target_seq < k.low
              AND h.dispatched_at IS NULL AND h.dead_at IS NULL
            ORDER BY h.target_seq
            LIMIT 1
        ) hd
        JOIN outbox o ON o.tenant_id = r.tenant_id AND o.id = hd.id
        WHERE o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
        ORDER BY k.since, k.target_key
        LIMIT r.room
        FOR UPDATE OF o SKIP LOCKED
    ) p
),
candidates AS (
    SELECT * FROM picked UNION ALL SELECT * FROM stranded
),
claimed AS (
    UPDATE outbox o
    SET available_at = now() + make_interval(secs => $3), attempts = o.attempts + 1
    FROM candidates p
    WHERE o.tenant_id = p.tenant_id AND o.id = p.id AND NOT p.expired
    RETURNING o.tenant_id, o.id, o.kind, o.kind_version, o.target_key, o.target_seq, o.causation_id,
              o.causation_depth, o.payload, o.trace_context, o.created_at, o.attempts, o.available_at,
              false AS expired
),
-- A separate UPDATE, not a CASE in claimed: keeping last_error on a claimed
-- row would mean reading it, and nava_relay has no SELECT on last_error.
killed AS (
    UPDATE outbox o
    SET dead_at = now(), last_error = $5
    FROM candidates p
    WHERE o.tenant_id = p.tenant_id AND o.id = p.id AND p.expired
    RETURNING o.tenant_id, o.id, o.kind, o.kind_version, o.target_key, o.target_seq, o.causation_id,
              o.causation_depth, o.payload, o.trace_context, o.created_at, o.attempts, o.available_at,
              true AS expired
),
settled AS (
    SELECT * FROM claimed UNION ALL SELECT * FROM killed
)
SELECT s.tenant_id, s.scanned, c.id, c.kind, c.kind_version, c.target_key, c.target_seq, c.causation_id,
       c.causation_depth, c.payload, c.trace_context, c.created_at, c.attempts, c.available_at, c.expired
FROM scan s LEFT JOIN settled c ON c.tenant_id = s.tenant_id
ORDER BY s.tenant_id, c.target_key, c.target_seq`

// Claim takes at most PerTenant rows of every tenant with pending rows,
// scanning at most ScanCap of each tenant's rows, and never a row that has a
// lower pending target_seq on its target. It records one
// outbox_relay_rows_scanned observation per tenant, and logs with tenant_id
// each tenant whose scan reached ScanCap. A row it would take over that was
// attempted before and is MaxRedeliveryAge old is marked dead, logged and
// not returned.
func (r *Relay) Claim(ctx context.Context) ([]Claimed, error) {
	rows, err := r.db.Query(ctx, claimSQL, ScanCap, PerTenant, r.opts.Lease.Seconds(), MaxRedeliveryAge.Seconds(),
		TakeoverExpiredError)
	if err != nil {
		return nil, fmt.Errorf("relay: claim: %w", err)
	}
	defer rows.Close()

	type tenantScan struct {
		tenant  uuid.UUID
		scanned int64
	}
	var (
		out   []Claimed
		scans []tenantScan
	)
	for rows.Next() {
		var (
			tenant  uuid.UUID
			scanned int64
			id      *uuid.UUID
			c       Claimed
			kind    *string
			kindV   *int16
			key     *string
			seq     *int64
			depth   *int16
			created *time.Time
			att     *int32
			lease   *time.Time
			expired *bool
		)
		if err := rows.Scan(&tenant, &scanned, &id, &kind, &kindV, &key, &seq, &c.CausationID, &depth,
			&c.Payload, &c.TraceContext, &created, &att, &lease, &expired); err != nil {
			return nil, fmt.Errorf("relay: claim: %w", err)
		}
		if len(scans) == 0 || scans[len(scans)-1].tenant != tenant {
			scans = append(scans, tenantScan{tenant, scanned})
		}
		if id == nil {
			continue
		}
		if *expired {
			r.log.WarnContext(ctx, "outbox row passed its max redelivery age and was marked dead",
				slog.String("tenant_id", tenant.String()), slog.String("outbox_id", id.String()),
				slog.Int("attempts", int(*att)))
			continue
		}
		c.ID, c.TenantID, c.Kind, c.KindVersion, c.TargetKey, c.TargetSeq = *id, tenant, *kind, *kindV, *key, *seq
		c.CausationDepth, c.CreatedAt, c.Attempts, c.LeaseUntil = *depth, *created, *att, *lease
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("relay: claim: %w", err)
	}

	for _, s := range scans {
		r.scanned.Record(ctx, s.scanned)
		if s.scanned >= ScanCap {
			r.log.WarnContext(ctx, "outbox relay scan reached its per-tenant cap",
				slog.String("tenant_id", s.tenant.String()), slog.Int64("scanned", s.scanned))
		}
	}
	return out, nil
}

// MarkDispatched records that c reached Temporal. It returns ErrLeaseLost if
// c's lease was taken over or the row is no longer pending.
func (r *Relay) MarkDispatched(ctx context.Context, c Claimed) error {
	return r.settle(ctx, "mark dispatched", `UPDATE outbox SET dispatched_at = now()
WHERE tenant_id = $1 AND id = $2 AND attempts = $3 AND dispatched_at IS NULL AND dead_at IS NULL
RETURNING id`, c.TenantID, c.ID, c.Attempts)
}

// MarkDead records that c will never be dispatched, with cause as last_error
// (for example a signal whose workflow is NotFound or already completed). It
// returns ErrLeaseLost as MarkDispatched does. cause must hold no record data.
func (r *Relay) MarkDead(ctx context.Context, c Claimed, cause string) error {
	return r.settle(ctx, "mark dead", `UPDATE outbox SET dead_at = now(), last_error = $4
WHERE tenant_id = $1 AND id = $2 AND attempts = $3 AND dispatched_at IS NULL AND dead_at IS NULL
RETURNING id`, c.TenantID, c.ID, c.Attempts, truncate(cause))
}

// backoffSQL backs a head off and pushes its pending successors on the same
// target to at least the head's new available_at, in one statement, so no
// successor is ever claimable before its head (A.2 "Worst-case bound"). A
// row past MaxRedeliveryAge is marked dead instead and pushes nothing: once
// it is dead its successor is the head. $4 is the delay and $6 the age, in
// seconds.
const backoffSQL = `WITH head AS (
    UPDATE outbox o SET
        available_at = CASE WHEN o.created_at <= now() - make_interval(secs => $6) THEN o.available_at
                            ELSE now() + make_interval(secs => $4) END,
        dead_at      = CASE WHEN o.created_at <= now() - make_interval(secs => $6) THEN now() END,
        last_error   = $5
    WHERE o.tenant_id = $1 AND o.id = $2 AND o.attempts = $3 AND o.dispatched_at IS NULL AND o.dead_at IS NULL
    RETURNING o.tenant_id, o.target_key, o.target_seq, o.available_at, o.dead_at IS NOT NULL AS dead
),
pushed AS (
    UPDATE outbox s SET available_at = h.available_at
    FROM head h
    WHERE NOT h.dead AND s.tenant_id = h.tenant_id AND s.target_key = h.target_key
      AND s.target_seq > h.target_seq AND s.dispatched_at IS NULL AND s.dead_at IS NULL
      AND s.available_at < h.available_at
    RETURNING s.id
)
SELECT h.dead, (SELECT count(*) FROM pushed) FROM head h`

// Backoff records a failed dispatch of c with cause as last_error and makes
// it claimable again after the retry delay for c.Attempts, pushing its
// successors with it. If c is MaxRedeliveryAge old it is marked dead instead
// and dead is true. It returns ErrLeaseLost as MarkDispatched does. cause
// must hold no record data.
func (r *Relay) Backoff(ctx context.Context, c Claimed, cause string) (dead bool, err error) {
	var pushed int64
	err = r.db.QueryRow(ctx, backoffSQL, c.TenantID, c.ID, c.Attempts, r.delay(c.Attempts).Seconds(),
		truncate(cause), MaxRedeliveryAge.Seconds()).Scan(&dead, &pushed)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return false, ErrLeaseLost
	case err != nil:
		return false, fmt.Errorf("relay: backoff: %w", err)
	}
	return dead, nil
}

func (r *Relay) settle(ctx context.Context, op, sql string, args ...any) error {
	var id uuid.UUID
	switch err := r.db.QueryRow(ctx, sql, args...).Scan(&id); {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrLeaseLost
	case err != nil:
		return fmt.Errorf("relay: %s: %w", op, err)
	}
	return nil
}

// delay is BackoffBase doubled for every attempt after the first, capped at
// BackoffMax.
func (r *Relay) delay(attempts int32) time.Duration {
	d := r.opts.BackoffBase
	for i := int32(1); i < attempts && d < r.opts.BackoffMax; i++ {
		d *= 2
	}
	return min(d, r.opts.BackoffMax)
}

// truncate makes s storable as text and cuts it to at most MaxErrorLen bytes
// on a rune boundary. PostgreSQL rejects invalid UTF-8 and NUL in text, which
// would fail the whole Backoff or MarkDead, so those bytes are replaced first.
func truncate(s string) string {
	s = strings.ReplaceAll(strings.ToValidUTF8(s, "�"), "\x00", "�")
	if len(s) <= MaxErrorLen {
		return s
	}
	s = s[:MaxErrorLen]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
