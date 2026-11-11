package tenancy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// allocBatch is the batch AllocTx sends for tenant, with one statement of
// the caller's.
func allocBatch(tenant any) *pgx.Batch {
	b := &pgx.Batch{}
	b.Queue("BEGIN")
	b.Queue("SELECT set_config('app.tenant_id', $1, true)", tenant)
	b.Queue("SET LOCAL synchronous_commit = off")
	b.Queue("SELECT 1")
	b.Queue("COMMIT")
	return b
}

// AllocTenant recognises only AllocTx's own batch, with a canonical tenant.
func TestAllocTenant(t *testing.T) {
	id := uuid.New()
	if got, ok := tenancy.AllocTenant(allocBatch(id.String())); !ok || got != id {
		t.Fatalf("AllocTx batch: %s, %v; want %s", got, ok, id)
	}
	short := allocBatch(id.String())
	short.QueuedQueries = short.QueuedQueries[:3]
	noCommit := allocBatch(id.String())
	noCommit.QueuedQueries[4].SQL = "ROLLBACK"
	other := allocBatch(id.String())
	other.QueuedQueries[2].SQL = "SELECT 2"
	for name, b := range map[string]*pgx.Batch{
		"nil tenant":    allocBatch(uuid.Nil.String()),
		"not a uuid":    allocBatch("x"),
		"not canonical": allocBatch("{" + id.String() + "}"),
		"not a string":  allocBatch(id),
		"too short":     short,
		"no commit":     noCommit,
		"other order":   other,
		"empty":         {},
	} {
		if got, ok := tenancy.AllocTenant(b); ok {
			t.Errorf("%s: recognised, tenant %s", name, got)
		}
	}
}

// AllocPoolConfig refuses any login but nava_app before anything connects,
// and installs the per-connection role check after any AfterConnect already
// set, which still runs first.
func TestAllocPoolConfig(t *testing.T) {
	if err := tenancy.AllocPoolConfig(nil); err == nil {
		t.Fatal("nil config accepted")
	}
	for _, user := range []string{"nava_owner", "nava_migrator", "postgres", "NAVA_APP", ""} {
		cfg, err := pgxpool.ParseConfig("postgres://" + user + "@localhost:1/nava")
		if err != nil {
			t.Fatal(err)
		}
		cfg.ConnConfig.User = user
		if err := tenancy.AllocPoolConfig(cfg); !errors.Is(err, tenancy.ErrAllocatorRole) {
			t.Errorf("user %q: %v, want ErrAllocatorRole", user, err)
		}
	}
	cfg, err := pgxpool.ParseConfig("postgres://nava_app@localhost:1/nava")
	if err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	cfg.AfterConnect = func(context.Context, *pgx.Conn) error { return boom }
	if err := tenancy.AllocPoolConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AfterConnect(t.Context(), nil); !errors.Is(err, boom) {
		t.Fatalf("AfterConnect = %v, want the earlier hook's error first", err)
	}
}

// IsAllocatorRoleCheck matches the role check's statement only.
func TestIsAllocatorRoleCheck(t *testing.T) {
	if tenancy.IsAllocatorRoleCheck("SELECT 1") || tenancy.IsAllocatorRoleCheck("") {
		t.Fatal("matched another statement")
	}
}
