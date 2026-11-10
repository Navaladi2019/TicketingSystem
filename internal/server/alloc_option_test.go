package server_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// The record routes refuse to start without an allocator pool (ADR-005 L2s,
// TICK-657): the number allocator never falls back onto the request pool,
// where creates holding every connection would starve their own numbers.
func TestRecordRoutesNeedAnAllocatorPool(t *testing.T) {
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	// Pools that never connect: NewHandler does not reach the database.
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{}, stubPaths{}, server.WithPool(pool))
	if err == nil || h != nil || !strings.Contains(err.Error(), "WithAllocatorPool") {
		t.Fatalf("NewHandler with WithPool only = %v, %v; want an error naming WithAllocatorPool", h, err)
	}
	alloc, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(alloc.Close)
	if _, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{}, stubPaths{},
		server.WithPool(pool), server.WithAllocatorPool(alloc)); err != nil {
		t.Fatalf("NewHandler with both pools: %v", err)
	}
	// No pool at all: no record routes, nothing to refuse.
	if _, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{}, stubPaths{}); err != nil {
		t.Fatalf("NewHandler without pools: %v", err)
	}
}
