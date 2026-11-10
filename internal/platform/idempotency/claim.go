package idempotency

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The claim of a key inside the handler's own transaction, for the routes
// registered with ClaimInTx (TICK-688; see the package doc).

// insertClaim inserts the key's in_flight row, or nothing when a row with the
// key exists. When another transaction holds an uncommitted row with the
// key, it waits for that transaction to end: it then inserts nothing if that
// one committed, and inserts the row if it rolled back.
const insertClaim = `
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() + $5::double precision * interval '1 second',
        now() + $6::double precision * interval '1 second')
ON CONFLICT (tenant_id, principal_id, key) DO NOTHING`

// pgLockNotAvailable is SQLSTATE 55P03: a lock wait passed lock_timeout.
const pgLockNotAvailable = "55P03"

// ErrSettled means ResolveClaim settled the request without the handler's
// write, because the key was already taken: the middleware answers with the
// stored response, 409 idempotency_key_reuse or 409 idempotency_in_flight,
// in place of whatever the handler writes. The handler's transaction must
// roll back (return the error from the InTenantTx function); it wrote
// nothing.
var ErrSettled = errors.New("idempotency: the request was settled by its key")

// queueClaim queues the claim of a ClaimInTx route on b, under ClaimWait.
func queueClaim(sc *scope, tx pgx.Tx, b *pgx.Batch) {
	req := sc.req
	tenancy.QueueUnderLockTimeout(b, ClaimWait, insertClaim,
		req.p.TenantID, req.p.PrincipalID, req.key, req.hash, Lease.Seconds(), TTL.Seconds()).
		Exec(func(ct pgconn.CommandTag) error {
			sc.claimSent = true
			if ct.RowsAffected() == 1 {
				sc.claimed = true
				// The row is this transaction's and uncommitted, so nobody
				// else can complete it: CompleteInTx may store at COMMIT.
				sc.held.Store(&tx)
			}
			return nil
		})
}

// ResolveClaim settles the claim QueueHold queued for a ClaimInTx route, once
// its batch has run; batchErr is what the batch's Close returned. Call it
// right after that batch, before validating, authorising or writing.
//
// It returns nil when the request holds the key and must run; ErrSettled
// when the key was already taken (a replay, a reuse, or a wait past
// ClaimWait); or another error, which the handler returns as it would any.
// Without a ClaimInTx scope in ctx it returns batchErr unchanged.
func ResolveClaim(ctx context.Context, tx pgx.Tx, batchErr error) error {
	sc, ok := ctx.Value(scopeKey{}).(*scope)
	if !ok || !sc.inTx {
		return batchErr
	}
	if batchErr != nil {
		if !sc.claimSent && isLockTimeout(batchErr) {
			// The claim waited ClaimWait on another transaction's
			// uncommitted row with the key, which neither committed nor
			// rolled back. This transaction is aborted and wrote nothing.
			return sc.settle(&outcome{kind: outcomeInFlight})
		}
		return batchErr
	}
	switch {
	case sc.claimed:
		return nil
	case !sc.claimSent:
		return errors.New("idempotency: the key's claim was not queued (QueueHold) in the batch")
	}
	// A row with the key was committed before the claim, or while it waited.
	// This new statement sees it (READ COMMITTED).
	var (
		hash   []byte
		state  string
		status *int16
		body   []byte
		hdrs   []byte
		stale  bool
	)
	req := sc.req
	if err := tx.QueryRow(ctx, selectRow, req.p.TenantID, req.p.PrincipalID, req.key).
		Scan(&hash, &state, &status, &body, &hdrs, &stale); err != nil {
		return fmt.Errorf("idempotency: select: %w", err)
	}
	switch {
	case !bytes.Equal(hash, req.hash):
		return sc.settle(&outcome{kind: outcomeReuse})
	case state == "completed" && status != nil:
		headers, err := decodeHeaders(hdrs)
		if err != nil {
			return err
		}
		return sc.settle(&outcome{kind: outcomeReplay, status: int(*status), body: body, headers: headers})
	case !stale:
		// An in_flight row committed by a binary that claims in a
		// transaction of its own, still within its lease.
		return sc.settle(&outcome{kind: outcomeInFlight})
	}
	return retake(ctx, sc, tx)
}

// retake takes over, inside the handler's transaction, a stale in_flight row
// that a binary claiming in a transaction of its own committed and
// abandoned. The UPDATE locks the row until the transaction ends, so
// CompleteInTx may store at COMMIT. Its wait for the row is bounded by
// ClaimWait too.
func retake(ctx context.Context, sc *scope, tx pgx.Tx) error {
	req := sc.req
	b := &pgx.Batch{}
	var taken bool
	tenancy.QueueUnderLockTimeout(b, ClaimWait, takeOver,
		req.p.TenantID, req.p.PrincipalID, req.key, req.hash, Lease.Seconds()).
		Exec(func(ct pgconn.CommandTag) error {
			taken = ct.RowsAffected() == 1
			return nil
		})
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		if isLockTimeout(err) {
			return sc.settle(&outcome{kind: outcomeInFlight})
		}
		return fmt.Errorf("idempotency: take over: %w", err)
	}
	if !taken {
		// Another request took it over or completed it first.
		return sc.settle(&outcome{kind: outcomeInFlight})
	}
	sc.claimed, sc.retaken = true, true
	sc.held.Store(&tx)
	return nil
}

// settle records o as the request's outcome and returns ErrSettled.
func (sc *scope) settle(o *outcome) error {
	sc.settled.Store(o)
	return ErrSettled
}

func isLockTimeout(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == pgLockNotAvailable
}
