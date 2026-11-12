package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The allocator pool (TICK-657): none without tenant resolution; its own DSN,
// required when there is tenant resolution (no fallback to the API pool's
// DSN, which shares its PgBouncer server pool); allocatorMaxConns
// wide unless the DSN sets pool_max_conns; and a bad DSN is refused without
// echoing it. pgxpool connects lazily, so nothing here needs a server.
func TestNewAllocatorPool(t *testing.T) {
	const app = "postgres://nava_app:hunter2@pgbouncer:5432/nava"
	const alloc = "postgres://nava_app:hunter2@pgbouncer:5432/nava_alloc"
	issuers := []string{"http://keycloak:8080/realms/nava"}
	var minConns int32
	open := func(t *testing.T, deps config.Dependencies) (string, int32, error) {
		t.Helper()
		p, err := newAllocatorPool(t.Context(), newTel(t), config.Config{Deps: deps})
		if err != nil || p == nil {
			return "", 0, err
		}
		t.Cleanup(p.Close)
		minConns = p.Config().MinConns
		return p.Config().ConnConfig.Database, p.Config().MaxConns, nil
	}

	for name, deps := range map[string]config.Dependencies{
		"no issuers":  {DatabaseURL: app, AllocatorDatabaseURL: alloc},
		"no database": {OIDCIssuers: issuers, AllocatorDatabaseURL: alloc},
	} {
		if db, _, err := open(t, deps); err != nil || db != "" {
			t.Errorf("%s: database %q, %v; want no pool", name, db, err)
		}
	}

	db, maxConns, err := open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: alloc})
	if err != nil || db != "nava_alloc" || maxConns != allocatorMaxConns {
		t.Errorf("own DSN: database %q, max %d, %v; want nava_alloc, %d", db, maxConns, err, allocatorMaxConns)
	}
	// TICK-778: one connection stays open while idle.
	if minConns != allocatorMinConns || allocatorMinConns < 1 {
		t.Errorf("own DSN: min %d; want %d (at least 1)", minConns, allocatorMinConns)
	}
	if _, _, err := open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: alloc + "?pool_min_conns=2"}); err != nil || minConns != 2 {
		t.Errorf("pool_min_conns=2: min %d, %v; want 2", minConns, err)
	}

	// No fallback to NAVA_DATABASE_URL: startup fails, naming the variable.
	if _, _, err := open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app}); err == nil ||
		!strings.Contains(err.Error(), "NAVA_ALLOCATOR_DATABASE_URL: required") || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("no allocator DSN: %v; want NAVA_ALLOCATOR_DATABASE_URL required, without the password", err)
	}

	_, maxConns, err = open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: alloc + "?pool_max_conns=9"})
	if err != nil || maxConns != 9 {
		t.Errorf("pool_max_conns=9: max %d, %v; want 9", maxConns, err)
	}

	// ADR-005 A3: startup refuses an allocator pool that logs in as anything
	// but nava_app, from its own DSN or the fallback.
	for _, deps := range []config.Dependencies{
		{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: "postgres://nava_owner:hunter2@pgbouncer:5432/nava_alloc"},
		{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: "postgres://nava_migrator:hunter2@pgbouncer:5432/nava_alloc"},
	} {
		if _, _, err := open(t, deps); !errors.Is(err, tenancy.ErrAllocatorRole) || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("non-nava_app login: %v; want tenancy.ErrAllocatorRole, without the password", err)
		}
	}

	_, _, err = open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: "postgres://nava_app:hunter2@pgbouncer:notaport/nava"})
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("bad DSN: %v; want an error that does not quote it", err)
	}
}

// Without an allocator pool (no record routes) the postgres check is pg's
// own; with one, a failing ping of it fails the check (TICK-778,
// TestRunNotReadyUntilTheAllocatorPoolIsWarm drives it through /readyz).
func TestWithAllocatorWarm(t *testing.T) {
	ok := health.Check{Name: "postgres", Critical: true, Probe: func(context.Context) error { return nil }}
	if c := withAllocatorWarm(ok, nil); c.Name != "postgres" || !c.Critical || c.Probe(t.Context()) != nil {
		t.Errorf("no allocator pool: %+v; want pg unchanged", c)
	}
	cfg, err := pgxpool.ParseConfig("postgres://nava_app:hunter2@127.0.0.1:1/nava_alloc?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	cold, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cold.Close)
	c := withAllocatorWarm(ok, cold)
	if err := c.Probe(t.Context()); err == nil || !strings.Contains(err.Error(), "allocator pool") || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("cold allocator pool: %v; want an allocator pool error without the password", err)
	}
	if c.Name != "postgres" || !c.Critical {
		t.Errorf("check %q critical %v; want postgres, critical", c.Name, c.Critical)
	}
	boom := errors.New("api database down")
	down := withAllocatorWarm(health.Check{Name: "postgres", Critical: true, Probe: func(context.Context) error { return boom }}, cold)
	if err := down.Probe(t.Context()); !errors.Is(err, boom) {
		t.Errorf("api database down: %v; want its error first", err)
	}
}
