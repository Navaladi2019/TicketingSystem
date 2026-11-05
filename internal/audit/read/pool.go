package read

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// PoolMaxConns is the reader pool's size (ADR-004-A1 A1-5). PgBouncer caps
// the role at the same number of server connections (A1-8, pool_size=5).
const PoolMaxConns = 5

// Pool is the audit reader's own database pool (ADR-004-A1 A1-5). It logs in
// as nava_audit_reader, the one runtime role that may read audit_log.diff and
// diff_salt, so only this package holds it: the type exposes no
// *pgxpool.Pool, and every read on it runs in the tenant's transaction next
// to the masker (Reader.List, Reader.RecordEntries).
type Pool struct {
	pool *pgxpool.Pool
}

// PoolOption configures OpenPool.
type PoolOption func(*poolOptions)

type poolOptions struct {
	tel *telemetry.Telemetry
}

// WithTelemetry traces and measures the pool through tel, as every other
// platform pool is (platform/db.NewPool: no SQL text, no parameters).
func WithTelemetry(tel *telemetry.Telemetry) PoolOption {
	return func(o *poolOptions) { o.tel = tel }
}

// errBadDSN never quotes the DSN: it holds the reader's password.
var errBadDSN = errors.New("audit/read: the audit reader DSN is not a valid PostgreSQL connection string")

// OpenPool returns the reader's pool for dsn (NAVA_AUDIT_READER_DATABASE_URL),
// with at most PoolMaxConns connections and unnamed statements, so it works
// behind PgBouncer's transaction mode. It does not connect: the first read
// does. No role switch: the DSN's user is the role.
func OpenPool(ctx context.Context, dsn string, opts ...PoolOption) (*Pool, error) {
	var o poolOptions
	for _, opt := range opts {
		opt(&o)
	}
	if dsn == "" {
		return nil, errors.New("audit/read: no audit reader DSN")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errBadDSN
	}
	cfg.MaxConns = PoolMaxConns
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	if cfg.ConnConfig.ConnectTimeout == 0 {
		cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	}
	var p *pgxpool.Pool
	if o.tel != nil {
		p, err = platformdb.NewPool(ctx, cfg, o.tel)
	} else {
		p, err = pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			err = errors.New("audit/read: cannot create the audit reader pool")
		}
	}
	if err != nil {
		return nil, err
	}
	return &Pool{pool: p}, nil
}

// Close closes the pool's connections. A nil Pool is a no-op.
func (p *Pool) Close() {
	if p != nil && p.pool != nil {
		p.pool.Close()
	}
}

// ErrNoPool means a Reader was used without a Pool: a wiring bug.
var ErrNoPool = errors.New("audit/read: the reader has no pool")
