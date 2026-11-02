//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Lock order across writes (TICK-88, from the P3-11 review on PR #360): two
// transactions that each hold two pipeline writes, on R1 and R2 in opposite
// orders, deadlock. PostgreSQL aborts one with 40P01; Write returns it as
// ErrRetryable, never as a bare database error. The other commits, and
// running the aborted transaction again succeeds.
func TestTwoWritesPerTransactionDeadlockIsRetryable(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	r1 := mustCreate(t, e, p, f, map[string]any{"title": "one"}).RecordID
	r2 := mustCreate(t, e, p, f, map[string]any{"title": "two"}).RecordID

	// both runs first then second in one transaction. The update's expected
	// version is read in the transaction, so a retry after the other commits
	// still applies.
	both := func(first, second uuid.UUID, firstDone chan<- struct{}, goOn <-chan struct{}) error {
		return tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			step := func(id uuid.UUID) error {
				var v int64
				if err := tx.QueryRow(ctx, `SELECT version FROM records WHERE id = $1`, id).Scan(&v); err != nil {
					return err
				}
				_, err := p.Write(ctx, tx, updateCmd(f, id, v, map[string]any{"summary": "x"}))
				return err
			}
			if err := step(first); err != nil {
				return err
			}
			if firstDone != nil {
				close(firstDone)
				<-goOn
			}
			return step(second)
		})
	}

	aFirst, bFirst := make(chan struct{}), make(chan struct{})
	aGo, bGo := make(chan struct{}), make(chan struct{})
	errA, errB := make(chan error, 1), make(chan error, 1)
	go func() { errA <- both(r1, r2, aFirst, aGo) }()
	go func() { errB <- both(r2, r1, bFirst, bGo) }()
	<-aFirst
	<-bFirst
	close(aGo) // A now waits on R2, which B holds
	waitForLockWait(t, e)
	close(bGo) // B now waits on R1, which A holds: a deadlock

	results := []error{<-errA, <-errB}
	var retryable, committed int
	for _, err := range results {
		switch {
		case err == nil:
			committed++
		case errors.Is(err, pipeline.ErrRetryable):
			retryable++
		default:
			t.Errorf("err = %v, want nil or ErrRetryable", err)
		}
	}
	if retryable != 1 || committed != 1 {
		t.Fatalf("%d retryable, %d committed; want 1 and 1", retryable, committed)
	}

	// The aborted transaction runs again and commits.
	if err := both(r2, r1, nil, nil); err != nil {
		t.Fatalf("retry: %v", err)
	}
	for _, id := range []uuid.UUID{r1, r2} {
		if _, _, _, v, _ := storedData(t, e, id); v != 3 {
			t.Errorf("record %s at version %d, want 3 (one create, two committed updates)", id, v)
		}
	}
}
