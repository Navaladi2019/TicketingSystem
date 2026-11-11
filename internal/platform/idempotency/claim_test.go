package idempotency_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
)

// TestResolveClaimWithoutScope: outside a ClaimInTx request (a test calling
// the handler directly, or a route that claims in a transaction of its own)
// ResolveClaim hands the batch's error back unchanged, a lock timeout
// included, and never touches the transaction (nil here).
func TestResolveClaimWithoutScope(t *testing.T) {
	t.Parallel()
	batch := errors.New("batch failed")
	lock := &pgconn.PgError{Code: "55P03"}
	for _, in := range []error{nil, batch, lock} {
		if got := idempotency.ResolveClaim(t.Context(), nil, in); got != in { //nolint:errorlint // identity is the point
			t.Errorf("ResolveClaim(%v) = %v, want it unchanged", in, got)
		}
	}
}

// TestQueueHoldWithoutScopeQueuesNothing: no scope, no claim and no hold.
func TestQueueHoldWithoutScopeQueuesNothing(t *testing.T) {
	t.Parallel()
	b := &pgx.Batch{}
	if err := idempotency.QueueHold(t.Context(), nil, b); err != nil {
		t.Fatal(err)
	}
	if b.Len() != 0 {
		t.Errorf("%d statements queued without a scope, want 0", b.Len())
	}
}
