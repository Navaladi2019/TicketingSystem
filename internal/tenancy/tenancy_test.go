package tenancy_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TestNoTenant: uuid.Nil is refused before any transaction. The pool is nil,
// so any attempt to begin one would panic.
func TestNoTenant(t *testing.T) {
	t.Parallel()
	called := false
	err := tenancy.InTenantTx(t.Context(), nil, uuid.Nil, func(pgx.Tx) error {
		called = true
		return nil
	})
	if !errors.Is(err, tenancy.ErrNoTenant) {
		t.Errorf("err = %v, want ErrNoTenant", err)
	}
	if called {
		t.Error("fn ran without a tenant")
	}
}

// A pool that cannot connect fails at begin, wrapped, and fn never runs.
func TestBeginFailure(t *testing.T) {
	t.Parallel()
	// Port 1 on loopback refuses at once; pgxpool connects lazily, on Begin.
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=2&sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	called := false
	err = tenancy.InTenantTx(t.Context(), pool, uuid.New(), func(pgx.Tx) error {
		called = true
		return nil
	})
	if err == nil || !strings.HasPrefix(err.Error(), "tenancy: begin: ") {
		t.Fatalf("err = %v, want a wrapped begin error", err)
	}
	if errors.Is(err, tenancy.ErrNoTenant) {
		t.Errorf("err = %v, a connection failure is not ErrNoTenant", err)
	}
	if called {
		t.Error("fn ran without a transaction")
	}
}

type rowFunc func(dest ...any) error

func (f rowFunc) Scan(dest ...any) error { return f(dest...) }

// settingTx answers TenantOf's read with setting, or fails it with err.
type settingTx struct {
	pgx.Tx  // nil: any other method panics
	setting string
	err     error
}

func (x settingTx) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowFunc(func(dest ...any) error {
		if x.err != nil {
			return x.err
		}
		*dest[0].(*string) = x.setting
		return nil
	})
}

// TenantOf accepts only a non-nil tenant UUID: an empty, malformed or nil
// setting is no tenant, and a failed read is a wrapped error, not a tenant.
func TestTenantOfSetting(t *testing.T) {
	t.Parallel()
	tenantA := uuid.New()
	got, err := tenancy.TenantOf(t.Context(), settingTx{setting: tenantA.String()})
	if err != nil || got != tenantA {
		t.Fatalf("TenantOf(%s) = %s, %v", tenantA, got, err)
	}
	for name, s := range map[string]string{
		"empty":     "",
		"not uuid":  "tenant-b",
		"nil uuid":  uuid.Nil.String(),
		"truncated": tenantA.String()[:30],
	} {
		got, err := tenancy.TenantOf(t.Context(), settingTx{setting: s})
		if !errors.Is(err, tenancy.ErrNoTenant) || got != uuid.Nil {
			t.Errorf("%s: TenantOf = %s, %v; want ErrNoTenant", name, got, err)
		}
	}
	boom := errors.New("boom")
	got, err = tenancy.TenantOf(t.Context(), settingTx{err: boom})
	if !errors.Is(err, boom) || errors.Is(err, tenancy.ErrNoTenant) || got != uuid.Nil {
		t.Fatalf("read failure: %s, %v; want the wrapped cause", got, err)
	}
	if want := "tenancy: read tenant: boom"; err.Error() != want {
		t.Errorf("message %q, want %q", err, want)
	}
}

// BeginTenant recognises exactly the statement InTenantTx begins with, for
// any tenant, and nothing that only looks like it: another setting, a nil or
// malformed tenant, a non-canonical spelling, or anything around it.
func TestBeginTenant(t *testing.T) {
	t.Parallel()
	tenantA := uuid.New()
	begin := "BEGIN; SELECT set_config('app.tenant_id', '" + tenantA.String() + "', true)"
	if got, ok := tenancy.BeginTenant(begin); !ok || got != tenantA {
		t.Fatalf("BeginTenant(the tenancy BEGIN) = %s, %v; want %s", got, ok, tenantA)
	}
	for name, sql := range map[string]string{
		"plain begin":     "begin",
		"bound":           "SELECT set_config('app.tenant_id', $1, true)",
		"session setting": "BEGIN; SELECT set_config('app.tenant_id', '" + tenantA.String() + "', false)",
		"other setting":   "BEGIN; SELECT set_config('app.other', '" + tenantA.String() + "', true)",
		"nil tenant":      "BEGIN; SELECT set_config('app.tenant_id', '" + uuid.Nil.String() + "', true)",
		"not a uuid":      "BEGIN; SELECT set_config('app.tenant_id', 'tenant-b', true)",
		"upper case":      "BEGIN; SELECT set_config('app.tenant_id', '" + strings.ToUpper(tenantA.String()) + "', true)",
		"trailing":        begin + "; SELECT 1",
	} {
		if got, ok := tenancy.BeginTenant(sql); ok {
			t.Errorf("%s: BeginTenant(%q) = %s, want no match", name, sql, got)
		}
	}
}
