//go:build integration

// TICK-865 (ADR-004-E9, CI-A18 (g), (i)): a create refused with
// number_space_exhausted writes nothing to audit_log in Write. Its R2b entry
// is appended by the caller in a transaction of its own on the caller's
// pool, after the create's transaction has ended: never on the allocator
// pool, and never in the create's transaction. The HTTP caller
// (internal/server/records.go) is tested end to end in
// internal/server/records_exhausted_integration_test.go; this file pins the
// pipeline's half of the contract.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// closingRunner is the allocator-pool PoolRunner, closing its pool as soon
// as the closeAfter'th `gaps` allocation has returned (0: never): from then
// on nothing can run on the allocator pool (CI-A18 (i)).
type closingRunner struct {
	pipeline.PoolRunner
	closeAfter *int
}

func (r closingRunner) Allocate(ctx context.Context, tx pgx.Tx, def uuid.UUID, f numbering.Compiled) (pipeline.Allocation, error) {
	a, err := r.PoolRunner.Allocate(ctx, tx, def, f)
	if *r.closeAfter > 0 {
		if *r.closeAfter--; *r.closeAfter == 0 {
			r.Pool.Close()
		}
	}
	return a, err
}

// refuseAndAppend runs one refused create as records.go does: Write in the
// caller's tenant transaction on api, then, once that transaction has ended,
// audit.Append of the refusal's entry in a transaction of its own on api. It
// returns the create transaction's pg_current_xact_id() and the refusal.
func refuseAndAppend(t *testing.T, e env, p *pipeline.Pipeline, api *pgxpool.Pool, f tenantFixture) (string, *pipeline.ErrNumberSpaceExhausted) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var createXid string
	err := tenancy.InTenantTx(ctx, api, f.id, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&createXid); err != nil {
			return err
		}
		_, err := p.Write(ctx, tx, createCmd(f, map[string]any{"title": "refused"}))
		return err
	})
	var ex *pipeline.ErrNumberSpaceExhausted
	if !errors.As(err, &ex) {
		t.Fatalf("create: %v, want ErrNumberSpaceExhausted", err)
	}
	if n := api.Stat().AcquiredConns(); n != 0 {
		t.Fatalf("%d API connections held after the create ended, want 0", n)
	}
	// Write appended nothing: the entry exists only once the caller appends.
	if n := entries(t, e, f.id, string(audit.ActionNumberSpaceExhausted)); n != 0 {
		t.Fatalf("Write wrote %d number_space_exhausted entries, want 0", n)
	}
	// api has one connection: had the create's still been held, this would
	// wait until ctx's deadline and fail.
	if err := tenancy.InTenantTx(ctx, api, f.id, func(tx pgx.Tx) error {
		_, err := audit.Append(ctx, tx, ex.Entry)
		return err
	}); err != nil {
		t.Fatalf("append the refusal's entry: %v", err)
	}
	return createXid, ex
}

// CI-A18 (i) and (g): with the allocator pool closed after the allocation
// (`gaps`) or before the create (`gapless`), the refusal's row is still
// written, in a transaction that is not the create's, on a one-connection
// API pool whose create connection was released first. Two refusals of D
// append two entries with distinct txids and no ErrDuplicateEntry.
func TestExhaustedEntryIsAppendedOffTheAllocatorPoolAfterTheCreate(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			linkFormat(t, e, f, "{prefix}-{seq:1}", numbering.Refuse, "")
			alloc := allocPool(t, appDSN(t, e.platform))
			closeAfter := 0
			runner := closingRunner{PoolRunner: pipeline.PoolRunner{Pool: alloc, AllocTimeout: noLatencyBudget}, closeAfter: &closeAfter}
			fill := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
			for i := 1; i <= 9; i++ {
				if n := mustCreate(t, e, fill, f, map[string]any{"title": fmt.Sprint(i)}).Number; n != fmt.Sprintf("INC-%d", i) {
					t.Fatalf("create %d: number %s", i, n)
				}
			}
			before := counters(t, e, f)

			cfg := e.app.Config().Copy()
			cfg.MaxConns = 1
			api := newPool(t, cfg)
			// `gaps`: closed once the second refusal's allocation returns,
			// before its create ends; `gapless` never allocates on it.
			if mode == "gaps" {
				closeAfter = 2
			} else {
				alloc.Close()
			}
			p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})

			var txids []string
			for i := range 2 {
				createXid, ex := refuseAndAppend(t, e, p, api, f)
				wantCounter := map[string]int64{"gaps": int64(10 + i), "gapless": 10}[mode]
				if ex.Counter != wantCounter || ex.Epoch != 1 || ex.Width != 1 {
					t.Errorf("refusal %d: counter %d epoch %d width %d, want %d, 1, 1", i, ex.Counter, ex.Epoch, ex.Width, wantCounter)
				}
				var txid, status string
				if err := e.super.QueryRow(context.Background(), `SELECT txid::text, pg_xact_status(txid) FROM audit_log
  WHERE tenant_id = $1 AND action = 'number_space_exhausted' ORDER BY occurred_at DESC, id DESC LIMIT 1`, f.id).Scan(&txid, &status); err != nil {
					t.Fatal(err)
				}
				if txid == createXid || status != "committed" {
					t.Errorf("refusal %d: entry txid %s (%s), create txid %s; want another, committed transaction", i, txid, status, createXid)
				}
				txids = append(txids, txid)
			}
			if txids[0] == txids[1] {
				t.Errorf("two refusals share txid %s", txids[0])
			}
			if n := entries(t, e, f.id, string(audit.ActionNumberSpaceExhausted)); n != 2 {
				t.Errorf("%d entries, want 2", n)
			}
			if n := len(numbers(t, e, f.id, entityKey)); n != 9 {
				t.Errorf("%d records, want 9", n)
			}
			after := counters(t, e, f)
			if mode == "gapless" && fmtCounters(after) != fmtCounters(before) {
				t.Errorf("gapless counters %s, want them unchanged at %s", fmtCounters(after), fmtCounters(before))
			}
			if _, err := alloc.Acquire(context.Background()); err == nil {
				t.Error("the allocator pool is still open")
			}
		})
	}
}
