package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
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
	open := func(t *testing.T, deps config.Dependencies) (string, int32, error) {
		t.Helper()
		p, err := newAllocatorPool(t.Context(), newTel(t), config.Config{Deps: deps})
		if err != nil || p == nil {
			return "", 0, err
		}
		t.Cleanup(p.Close)
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
