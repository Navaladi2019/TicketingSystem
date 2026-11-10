package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// The allocator pool (TICK-657): none without tenant resolution; its own DSN
// when set, NAVA_DATABASE_URL (with a warning) when not; allocatorMaxConns
// wide unless the DSN sets pool_max_conns; and a bad DSN is refused without
// echoing it. pgxpool connects lazily, so nothing here needs a server.
func TestNewAllocatorPool(t *testing.T) {
	const app = "postgres://nava_app:hunter2@pgbouncer:5432/nava"
	const alloc = "postgres://nava_app:hunter2@pgbouncer:5432/nava_alloc"
	issuers := []string{"http://keycloak:8080/realms/nava"}
	open := func(t *testing.T, deps config.Dependencies) (string, string, int32, error) {
		t.Helper()
		var logs bytes.Buffer
		p, err := newAllocatorPool(t.Context(), slog.New(slog.NewTextHandler(&logs, nil)), newTel(t), config.Config{Deps: deps})
		if err != nil || p == nil {
			return "", logs.String(), 0, err
		}
		t.Cleanup(p.Close)
		return p.Config().ConnConfig.Database, logs.String(), p.Config().MaxConns, nil
	}

	for name, deps := range map[string]config.Dependencies{
		"no issuers":  {DatabaseURL: app, AllocatorDatabaseURL: alloc},
		"no database": {OIDCIssuers: issuers, AllocatorDatabaseURL: alloc},
	} {
		if db, _, _, err := open(t, deps); err != nil || db != "" {
			t.Errorf("%s: database %q, %v; want no pool", name, db, err)
		}
	}

	db, logs, maxConns, err := open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: alloc})
	if err != nil || db != "nava_alloc" || maxConns != allocatorMaxConns || logs != "" {
		t.Errorf("own DSN: database %q, max %d, logs %q, %v; want nava_alloc, %d, no warning", db, maxConns, logs, err, allocatorMaxConns)
	}

	db, logs, maxConns, err = open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app})
	if err != nil || db != "nava" || maxConns != allocatorMaxConns || !strings.Contains(logs, "NAVA_ALLOCATOR_DATABASE_URL is not set") {
		t.Errorf("fallback: database %q, max %d, logs %q, %v; want nava, %d and a warning", db, maxConns, logs, err, allocatorMaxConns)
	}
	if strings.Contains(logs, "hunter2") {
		t.Errorf("the warning quotes the password: %q", logs)
	}

	_, _, maxConns, err = open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: alloc + "?pool_max_conns=9"})
	if err != nil || maxConns != 9 {
		t.Errorf("pool_max_conns=9: max %d, %v; want 9", maxConns, err)
	}

	_, _, _, err = open(t, config.Dependencies{OIDCIssuers: issuers, DatabaseURL: app, AllocatorDatabaseURL: "postgres://nava_app:hunter2@pgbouncer:notaport/nava"})
	if err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("bad DSN: %v; want an error that does not quote it", err)
	}
}
