//go:build integration

// P5-31 (PLAN-005 C5, ADR-001 A6.3) against a real PostgreSQL as nava_app and
// nava_worker, under RLS. See outbox_integration_test.go for the setup.

package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// seedHead inserts, as the superuser, the tenant (if new) and its
// audit_chain_head at version, as onboarding (A6.2) will.
func seedHead(t *testing.T, conn *pgx.Conn, tenant uuid.UUID, version int64) {
	t.Helper()
	mustExec(t, conn, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	mustExec(t, conn, "INSERT INTO audit_chain_head (tenant_id, id, version) VALUES ($1, $2, $3)",
		tenant, uuid.Must(uuid.NewV7()), version)
}

// lockedSeal is the C5 caller: it locks the tenant's head FOR UPDATE, reads
// its version and enqueues the seal row at that version, in one transaction.
func lockedSeal(ctx context.Context, pool *pgxpool.Pool, tenant uuid.UUID) (uuid.UUID, int64, error) {
	var (
		id uuid.UUID
		v  int64
	)
	err := tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT version FROM audit_chain_head FOR UPDATE").Scan(&v); err != nil {
			return err
		}
		var err error
		id, err = outbox.EnqueueSeal(ctx, tx, v)
		return err
	})
	return id, v, err
}

func producers(e env) []struct {
	role string
	pool *pgxpool.Pool
} {
	return []struct {
		role string
		pool *pgxpool.Pool
	}{{"nava_app", e.app}, {"nava_worker", e.worker}}
}

// AC (C5): the seal row's columns, no workflow_ref, and dedupe on
// audit.seal:<headVersion>; a version the head does not have writes nothing.
func TestEnqueueSeal(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	for _, p := range producers(e) {
		t.Run(p.role, func(t *testing.T) {
			tenant := uuid.New()
			seedHead(t, e.super, tenant, 3)

			id, v, err := lockedSeal(ctx, p.pool, tenant)
			if err != nil {
				t.Fatal(err)
			}
			if id.Version() != 7 {
				t.Errorf("id %s is not a UUIDv7", id)
			}
			var (
				kind, dedupe, targetKey, payload string
				kindVersion, depth               int16
				targetSeq                        int64
				causation                        *uuid.UUID
			)
			if err := e.super.QueryRow(ctx, `SELECT kind, kind_version, dedupe_key, target_key, target_seq,
				causation_id, causation_depth, payload::text FROM outbox WHERE tenant_id = $1 AND id = $2`, tenant, id).
				Scan(&kind, &kindVersion, &dedupe, &targetKey, &targetSeq, &causation, &depth, &payload); err != nil {
				t.Fatal(err)
			}
			if kind != outbox.SealKind || kindVersion != 1 {
				t.Errorf("kind %q v%d, want %s v1", kind, kindVersion, outbox.SealKind)
			}
			if dedupe != "audit.seal:3" || v != 3 {
				t.Errorf("dedupe_key %q at version %d, want audit.seal:3", dedupe, v)
			}
			if targetKey != "audit.seal:"+id.String() || targetSeq != 3 {
				t.Errorf("target %q seq %d, want audit.seal:%s seq 3", targetKey, targetSeq, id)
			}
			if payload != "{}" {
				t.Errorf("payload %s, want {}", payload)
			}
			if causation == nil || *causation != id || depth != 0 {
				t.Errorf("causation %v depth %d, want the row's own id at depth 0", causation, depth)
			}
			if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 0 {
				t.Errorf("%d workflow_ref rows, want 0 (A6.3)", n)
			}

			// A retry at the same head version returns the first row.
			again, _, err := lockedSeal(ctx, p.pool, tenant)
			if err != nil || again != id {
				t.Errorf("retry = %s, %v; want %s", again, err, id)
			}
			// A version the head does not have writes nothing.
			err = tenancy.InTenantTx(ctx, p.pool, tenant, func(tx pgx.Tx) error {
				_, err := outbox.EnqueueSeal(ctx, tx, 4)
				return err
			})
			if !errors.Is(err, outbox.ErrVersionMismatch) {
				t.Errorf("stale version: %v, want ErrVersionMismatch", err)
			}
			if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); n != 1 {
				t.Errorf("%d outbox rows, want 1", n)
			}
		})
	}
}

// AC (C5): EnqueueSeal never writes without the head lock held in its
// transaction. Called without the caller's own FOR UPDATE, it takes the lock
// itself, so another transaction cannot lock the head until it ends.
func TestEnqueueSealHoldsHeadLock(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	seedHead(t, e.super, tenant, 1)

	enqueued := make(chan error, 1)
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
			_, err := outbox.EnqueueSeal(ctx, tx, 1)
			enqueued <- err
			<-release
			return err
		})
	}()
	if err := <-enqueued; err != nil {
		close(release)
		t.Fatal(err)
	}
	err := tenancy.InTenantTx(ctx, e.worker, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "SELECT 1 FROM audit_chain_head FOR UPDATE NOWAIT")
		return err
	})
	close(release)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("second lock while the seal transaction is open: %v, want lock_not_available (55P03)", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("seal transaction did not commit")
	}
}

// AC (C5, I1, I2): the tenant is the transaction's. Tenant A reaches no
// other tenant's head, a transaction with no tenant reaches none, and a seal
// row whose tenant_id is not the transaction's tenant is refused (Security,
// TICK-449 grooming).
func TestEnqueueSealTenantIsolation(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	for _, p := range producers(e) {
		t.Run(p.role, func(t *testing.T) {
			a, b := uuid.New(), uuid.New()
			mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", a)
			seedHead(t, e.super, b, 1)

			err := tenancy.InTenantTx(ctx, p.pool, a, func(tx pgx.Tx) error {
				_, err := outbox.EnqueueSeal(ctx, tx, 1)
				return err
			})
			if !errors.Is(err, outbox.ErrHeadNotFound) {
				t.Errorf("A with only B's head: %v, want ErrHeadNotFound", err)
			}

			tx, err := p.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := outbox.EnqueueSeal(ctx, tx, 1); !errors.Is(err, outbox.ErrHeadNotFound) {
				t.Errorf("no tenant: %v, want ErrHeadNotFound", err)
			}
			_ = tx.Rollback(ctx)

			// A seal row naming B, written in A's transaction, is refused.
			err = tenancy.InTenantTx(ctx, p.pool, a, func(tx pgx.Tx) error {
				id := uuid.Must(uuid.NewV7())
				_, err := tx.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, causation_id)
					VALUES ($1, $2, 'audit.seal', 'audit.seal:1', $3, 1, $1)`, id, b, outbox.SealKind+":"+id.String())
				return err
			})
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
				t.Errorf("seal row for B in A's transaction: %v, want an RLS violation (42501)", err)
			}
			if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id IN ($1, $2)", a, b); n != 0 {
				t.Errorf("%d outbox rows written, want 0", n)
			}
		})
	}
}
