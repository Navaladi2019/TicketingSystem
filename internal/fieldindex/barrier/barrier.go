// Package barrier is the ADR-006 D9 snapshot barrier, built by S21 for the
// field-index backfill (TICK-745) and reused by the schema backfill (ADR-009
// D7): it waits, inside one activity attempt, until every transaction that was
// in progress when it started has ended. It reads no tenant data, so it takes
// no tenant: transaction ids are not tenant data.
package barrier

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
)

// State is the barrier's heartbeat details: the bounds of the wait set
// taken once on the first attempt (D9), when that was, and whether the slow
// alert has fired. The wait set is XIP ∪ [Lo, Hi): the snapshot's xip, and
// every xid from its xmax (Lo) up to the fence xid (Hi), exclusive. The range
// is kept as its bounds, never expanded, so the details stay a fixed size
// however many xids were handed out between the snapshot and the fence
// (ADR-006-E1 E1-1). Lo and Hi are empty in details written before E1-1,
// whose XIP already held the expanded range; an empty range waits on nothing.
type State struct {
	XIP   []string  `json:"xip"`
	Lo    string    `json:"lo,omitempty"`
	Hi    string    `json:"hi,omitempty"`
	Since time.Time `json:"since"`
	Slow  bool      `json:"slow"`
}

// The poll interval: 1 s, doubling to 10 s (D9).
const (
	firstPoll = time.Second
	maxPoll   = 10 * time.Second
)

const (
	// snapshotSQL and fenceSQL, run in that order in one transaction, give
	// the set of transactions in progress at the snapshot. A transaction
	// that started after it sees every change committed before it (writers
	// run READ COMMITTED).
	//
	// pg_snapshot_xip alone is not that set: it lists only xids below the
	// snapshot's xmax, and when the newest assigned xid is still running,
	// xmax is that xid, so it is not listed (found in TICK-745 testing: a
	// snapshot 1869:1869: with 1869 in progress). fenceSQL assigns this
	// transaction an xid M, after the snapshot, and the barrier also waits on
	// every xid in [xmax, M) still in progress: the xids handed out between
	// the two statements, which include any such newest one.
	snapshotSQL = `SELECT (SELECT coalesce(array_agg(x::text), '{}') FROM pg_snapshot_xip(s.s) x), pg_snapshot_xmax(s.s)::text
		FROM (SELECT pg_current_snapshot() AS s) s`
	fenceSQL = `SELECT pg_current_xact_id()::text`
	// openSQL counts the xids of XIP ∪ [Lo, Hi) still in progress; the range
	// is expanded here, per poll, never in Go. An empty Lo or Hi is an empty
	// range. pg_xact_status needs no grant.
	openSQL = `SELECT (SELECT count(*) FROM unnest($1::text[]) x WHERE pg_xact_status(x::xid8) = 'in progress')
		+ (SELECT count(*) FROM generate_series(nullif($2, '')::bigint, nullif($3, '')::bigint - 1) g(n)
			WHERE pg_xact_status(n::text::xid8) = 'in progress')`
)

// Options tune Wait. Every field may be zero.
type Options struct {
	// SlowAfter is how long Wait waits before it calls OnSlow, once. Zero
	// never calls it.
	SlowAfter time.Duration
	// OnSlow raises the caller's slow alert (metric and log, no tenant
	// data). open is how many transactions of the wait set are still open.
	OnSlow func(ctx context.Context, open int64, waited time.Duration)
	// StopIf, checked before each poll, ends the wait early when it reports
	// true: the caller's work no longer needs the barrier.
	StopIf func(ctx context.Context) (bool, error)
	// AfterSnapshot runs between the snapshot and the fence (test-only: it
	// widens [Lo, Hi)).
	AfterSnapshot func(ctx context.Context)
}

// Wait is the snapshot barrier (D9, R11). It takes pg_current_snapshot() and
// a fence xid once and keeps the bounds of the wait set in the heartbeat
// details, then polls it with pg_xact_status, once per poll, 1 s apart
// backing off to 10 s, until none is still open. Nothing sleeps in SQL, and
// no connection is held between polls. Past SlowAfter it calls OnSlow once
// and keeps waiting.
//
// The polls run inside one activity attempt, heartbeating after each, not as
// failed attempts on a retry timer (TICK-745 design deviation): the worker
// guard records a non-allow-listed activity failure as a record_data deny
// with alert=true, so a retry-driven poll would page on every barrier. A
// worker lost mid-wait times the heartbeat out, and the next attempt resumes
// from the snapshot in the heartbeat details, never a new one. ctx must be an
// activity's.
func Wait(ctx context.Context, pool *pgxpool.Pool, o Options) error {
	var st State
	if !activity.HasHeartbeatDetails(ctx) || activity.GetHeartbeatDetails(ctx, &st) != nil || st.Since.IsZero() {
		var err error
		if st, err = takeSnapshot(ctx, pool, o.AfterSnapshot); err != nil {
			return fmt.Errorf("barrier: snapshot: %w", err)
		}
		activity.RecordHeartbeat(ctx, st)
	}
	wait := firstPoll
	for {
		if o.StopIf != nil {
			stop, err := o.StopIf(ctx)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
		var open int64
		if err := pool.QueryRow(ctx, openSQL, st.XIP, st.Lo, st.Hi).Scan(&open); err != nil {
			return fmt.Errorf("barrier: status: %w", err)
		}
		if open == 0 {
			return nil
		}
		if waited := time.Since(st.Since); !st.Slow && o.SlowAfter > 0 && waited >= o.SlowAfter {
			st.Slow = true
			if o.OnSlow != nil {
				o.OnSlow(ctx, open, waited)
			}
		}
		activity.RecordHeartbeat(ctx, st)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		wait = min(wait*2, maxPoll)
	}
}

// takeSnapshot returns the wait set of the transactions in progress now: the
// snapshot's xip, its xmax as Lo, and as Hi a fence xid this transaction
// takes after it (see snapshotSQL). The transaction is rolled back; its only
// effect is the one xid it consumed.
func takeSnapshot(ctx context.Context, pool *pgxpool.Pool, afterSnapshot func(context.Context)) (State, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return State{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	st := State{Since: time.Now()}
	if err := tx.QueryRow(ctx, snapshotSQL).Scan(&st.XIP, &st.Lo); err != nil {
		return State{}, err
	}
	if afterSnapshot != nil {
		afterSnapshot(ctx)
	}
	if err := tx.QueryRow(ctx, fenceSQL).Scan(&st.Hi); err != nil {
		return State{}, err
	}
	return st, nil
}
