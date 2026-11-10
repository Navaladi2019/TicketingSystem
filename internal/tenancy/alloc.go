package tenancy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AllocatorRole is the only role the record-number allocator pool may log in
// as (ADR-005 A3): the owner, the migrator and any BYPASSRLS role would
// allocate outside row-level security.
const AllocatorRole = "nava_app"

// ErrAllocatorRole means the allocator pool would log in as a role other
// than AllocatorRole, or as a superuser or BYPASSRLS role.
var ErrAllocatorRole = errors.New("tenancy: the allocator pool must log in as nava_app, without SUPERUSER or BYPASSRLS")

// The statements AllocTx sends around the caller's: they open the
// transaction, scope it to the tenant (transaction-local) and make its commit
// asynchronous (transaction-local, ADR-005 A4), then commit it.
// synchronous_commit appears nowhere else in the platform's Go code; the
// navalint setconfig analyzer reports it outside this package.
const (
	allocBeginSQL      = "BEGIN"
	allocTenantSQL     = "SELECT set_config('app.tenant_id', $1, true)"
	allocDurabilitySQL = "SET LOCAL synchronous_commit = off"
	allocCommitSQL     = "COMMIT"
)

// AllocInfo is what AllocTx reports besides its error.
type AllocInfo struct {
	// Tenant is the tenant the allocation transaction ran under:
	// TenantOf(createTx).
	Tenant uuid.UUID
	// Acquire is how long the allocator pool took to hand out a connection.
	Acquire time.Duration
}

// AllocTx runs a record-number allocation (ADR-005 L2s, rules A1, A2, A4) in
// a transaction of its own on pool, the dedicated allocator pool, and commits
// it before returning. The tenant is TenantOf(createTx), the create's own
// transaction, never a parameter; queue gets it and queues the allocation's
// statements. Everything goes in one pipelined send:
//
//	BEGIN; set_config('app.tenant_id', tenant, true);
//	SET LOCAL synchronous_commit = off; <queue's statements>; COMMIT
//
// SET LOCAL ends with the transaction, so the next transaction on the
// connection (through PgBouncer, maybe another client's) runs at the server
// default. The commit does not wait for its WAL flush: the record that uses
// the number commits synchronously later, and that flush covers the counter
// commit too (ADR-005 §2.2, AR-TICK-653-03).
//
// On any error the connection is not idle, and the pool destroys it on
// release, so neither the tenant nor the setting can leak. The caller bounds
// the call with ctx (ADR-005 A5).
func AllocTx(ctx context.Context, pool *pgxpool.Pool, createTx pgx.Tx, queue func(b *pgx.Batch, tenant uuid.UUID)) (AllocInfo, error) {
	tenant, err := TenantOf(ctx, createTx)
	if err != nil {
		return AllocInfo{}, err
	}
	info := AllocInfo{Tenant: tenant}
	start := time.Now()
	conn, err := pool.Acquire(ctx)
	info.Acquire = time.Since(start)
	if err != nil {
		return info, fmt.Errorf("tenancy: allocate: acquire: %w", err)
	}
	// Release destroys a connection that is not idle and out of any
	// transaction (TxStatus 'I').
	defer conn.Release()
	b := &pgx.Batch{}
	b.Queue(allocBeginSQL)
	b.Queue(allocTenantSQL, tenant.String())
	b.Queue(allocDurabilitySQL)
	queue(b, tenant)
	b.Queue(allocCommitSQL)
	if err := conn.SendBatch(ctx, b).Close(); err != nil {
		return info, fmt.Errorf("tenancy: allocate: %w", err)
	}
	if st := conn.Conn().PgConn().TxStatus(); st != 'I' {
		return info, fmt.Errorf("tenancy: allocate: the transaction did not commit (status %q)", st)
	}
	return info, nil
}

// AllocTenant reports whether b is an AllocTx batch, and the tenant it sets.
// It is for tracers that check the GUC ordering from the driver's side
// (tenancytest.Guard), so the statements' text stays owned by this package.
func AllocTenant(b *pgx.Batch) (uuid.UUID, bool) {
	q := b.QueuedQueries
	if len(q) < 4 || q[0].SQL != allocBeginSQL || q[1].SQL != allocTenantSQL || q[2].SQL != allocDurabilitySQL ||
		len(q[1].Arguments) != 1 || q[len(q)-1].SQL != allocCommitSQL {
		return uuid.Nil, false
	}
	s, ok := q[1].Arguments[0].(string)
	if !ok {
		return uuid.Nil, false
	}
	tenant, err := uuid.Parse(s)
	if err != nil || tenant == uuid.Nil || tenant.String() != s {
		return uuid.Nil, false
	}
	return tenant, true
}

// AllocPoolConfig makes cfg an allocator pool config (ADR-005 A3). It
// refuses, before anything connects, a login other than AllocatorRole, and
// checks every new connection's role in PostgreSQL before the pool hands it
// out (CheckAllocatorRole), so a role renamed or given BYPASSRLS later is
// refused too. Call it before creating the pool.
func AllocPoolConfig(cfg *pgxpool.Config) error {
	if cfg == nil {
		return errors.New("tenancy: nil allocator pool config")
	}
	if user := cfg.ConnConfig.User; user != AllocatorRole {
		return fmt.Errorf("%w: it logs in as %q", ErrAllocatorRole, user)
	}
	prev := cfg.AfterConnect
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		if prev != nil {
			if err := prev(ctx, c); err != nil {
				return err
			}
		}
		return CheckAllocatorRole(ctx, c)
	}
	return nil
}

// RowQuerier is the one method CheckAllocatorRole needs: a *pgx.Conn, a
// pool or a transaction.
type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// allocRoleSQL reads the session role's name and whether it bypasses RLS.
// It reads a catalog view only, under no tenant.
const allocRoleSQL = `SELECT r.rolname, r.rolsuper, r.rolbypassrls FROM pg_roles r WHERE r.rolname = current_user`

// IsAllocatorRoleCheck reports whether sql is CheckAllocatorRole's
// statement: the one statement an allocator connection sends with no tenant
// (once, as it opens). For tracers such as tenancytest.Guard.
func IsAllocatorRoleCheck(sql string) bool { return sql == allocRoleSQL }

// CheckAllocatorRole returns ErrAllocatorRole unless the session's role is
// AllocatorRole with neither SUPERUSER nor BYPASSRLS (ADR-005 A3).
func CheckAllocatorRole(ctx context.Context, q RowQuerier) error {
	var (
		user               string
		super, bypassesRLS bool
	)
	if err := q.QueryRow(ctx, allocRoleSQL).Scan(&user, &super, &bypassesRLS); err != nil {
		return fmt.Errorf("tenancy: read the allocator pool's role: %w", err)
	}
	if user != AllocatorRole || super || bypassesRLS {
		return fmt.Errorf("%w: connected as %q (superuser %v, bypassrls %v)", ErrAllocatorRole, user, super, bypassesRLS)
	}
	return nil
}
