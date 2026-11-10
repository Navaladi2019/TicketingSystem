package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// allocatorMaxConns is the allocator pool's default width. An allocation is
// one short transaction (BEGIN with the tenant, the record_counters upsert,
// COMMIT), and allocations of one type queue on its counter row whatever the
// width, so the pool bounds only how many tenants and types allocate at
// once: about allocatorMaxConns / (3 round trips + a commit) allocations a
// second. A caller beyond it waits on this pool only, never on the API pool.
// pool_max_conns in NAVA_ALLOCATOR_DATABASE_URL overrides it; PgBouncer's
// nava_alloc pool_size (deploy/compose/pgbouncer/pgbouncer.ini) must match.
const allocatorMaxConns = 4

// newAllocatorPool opens the record pipeline's allocator pool (TICK-657): the
// nava_app pool a create takes its number on, in a transaction of its own
// while the create's transaction holds an API pool connection (D-3-9 as
// amended 2026-11-10). It is a separate pool so an allocation never waits for
// a connection an open create holds, which with every API connection in a
// create would never come. It connects with NAVA_ALLOCATOR_DATABASE_URL, or
// NAVA_DATABASE_URL when that is unset. Nil when tenant resolution is not
// configured (as newResolver): there are then no record routes.
func newAllocatorPool(ctx context.Context, log *slog.Logger, tel *telemetry.Telemetry, cfg config.Config) (*pgxpool.Pool, error) {
	if len(cfg.Deps.OIDCIssuers) == 0 || cfg.Deps.DatabaseURL == "" {
		return nil, nil
	}
	dsn := cfg.Deps.AllocatorDatabaseURL
	if dsn == "" {
		// Behind PgBouncer the two pools then share one server pool, and a
		// create holding the last server connection waits for its number
		// until PgBouncer's query_wait_timeout.
		log.WarnContext(ctx, "NAVA_ALLOCATOR_DATABASE_URL is not set: the allocator pool connects with NAVA_DATABASE_URL")
		dsn = cfg.Deps.DatabaseURL
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// The error quotes the connection string, which is a credential.
		return nil, errors.New("config: the allocator pool's connection string is not a valid PostgreSQL connection string")
	}
	// As the API pool (newResolver): unnamed statements for PgBouncer's
	// transaction mode, and a 5 s connect timeout.
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if !strings.Contains(dsn, "pool_max_conns") {
		poolCfg.MaxConns = allocatorMaxConns
	}
	// ADR-005 A3: nava_app only. Another login fails startup here; every
	// connection is also checked for SUPERUSER and BYPASSRLS as it opens.
	if err := tenancy.AllocPoolConfig(poolCfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return db.NewPool(ctx, poolCfg, tel)
}
