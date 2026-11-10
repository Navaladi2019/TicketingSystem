//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// P3-36f (TICK-659, D-3-9): the definition the create path loads carries the
// type's numbering mode, read in the same statement (no extra round trip),
// and SetNumberingMode changes it within the caller's tenant only.

// numberingTenant creates an active tenant with one principal and returns both.
func numberingTenant(t *testing.T, e env) (tenant, principal uuid.UUID) {
	t.Helper()
	tenant, principal = uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, tenant, principal, "kc-"+principal.String())
	return tenant, principal
}

// tenancyTx is tx returning fn's error instead of failing the test.
func tenancyTx(e env, tenant uuid.UUID, fn func(context.Context, pgx.Tx) error) error {
	ctx := context.Background()
	return tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error { return fn(ctx, tx) })
}

func createType(t *testing.T, e env, tenant, principal uuid.UUID, key, prefix string, mode metadata.NumberingMode) {
	t.Helper()
	d := assetDefinition()
	d.Key, d.NumberPrefix, d.NumberingMode = key, prefix, mode
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Create(ctx, tx, tenant, entitytypes.NewInput{Name: key, PublishedBy: principal, Definition: d})
		return err
	})
}

// currentMode reads key's mode as the create path does: unbatched Current,
// and Current after QueueCurrent prefetched it in a batch.
func currentMode(t *testing.T, e env, tenant uuid.UUID, key string) (unbatched, batched metadata.NumberingMode) {
	t.Helper()
	s := entitytypes.Store{}
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		c, _, err := s.Current(ctx, tx, key)
		if err != nil {
			return err
		}
		unbatched = c.NumberingMode()
		b := &pgx.Batch{}
		s.QueueCurrent(tx, b, key)
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
		c, _, err = s.Current(ctx, tx, key)
		if err != nil {
			return err
		}
		batched = c.NumberingMode()
		return nil
	})
	return unbatched, batched
}

func modeEntries(t *testing.T, e env, tenant uuid.UUID) int {
	t.Helper()
	var n int
	if err := e.super.QueryRow(context.Background(), `SELECT count(*) FROM audit_log
		WHERE tenant_id = $1 AND domain = 'config' AND action = 'update' AND subject_type = 'entity_definition'`, tenant).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestP336fNumberingModeRead(t *testing.T) {
	e := migrated(t)
	tenant, principal := numberingTenant(t, e)
	createType(t, e, tenant, principal, "asset", "AST", "")
	createType(t, e, tenant, principal, "widget", "WDG", metadata.NumberingGapless)

	for key, want := range map[string]metadata.NumberingMode{"asset": metadata.NumberingGaps, "widget": metadata.NumberingGapless} {
		if u, b := currentMode(t, e, tenant, key); u != want || b != want {
			t.Errorf("%s: Current mode %q, batched %q; want %q", key, u, b, want)
		}
	}
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		v, err := entitytypes.Store{}.GetVersion(ctx, tx, "widget", 1)
		if err != nil {
			return err
		}
		if v.NumberingMode != metadata.NumberingGapless {
			t.Errorf("GetVersion(widget, 1).NumberingMode = %q, want gapless", v.NumberingMode)
		}
		return nil
	})
}

func TestP336fSetNumberingMode(t *testing.T) {
	e := migrated(t)
	tenant, principal := numberingTenant(t, e)
	createType(t, e, tenant, principal, "asset", "AST", "")
	s := entitytypes.Store{}

	set := func(mode metadata.NumberingMode, by uuid.UUID) (bool, error) {
		var changed bool
		err := tenancyTx(e, tenant, func(ctx context.Context, tx pgx.Tx) error {
			var err error
			changed, err = s.SetNumberingMode(ctx, tx, entitytypes.SetNumberingModeInput{Key: "asset", Mode: mode, By: by})
			return err
		})
		return changed, err
	}

	if changed, err := set(metadata.NumberingGapless, principal); err != nil || !changed {
		t.Fatalf("gaps -> gapless: changed %v, %v; want true, nil", changed, err)
	}
	if u, b := currentMode(t, e, tenant, "asset"); u != metadata.NumberingGapless || b != metadata.NumberingGapless {
		t.Errorf("after set: %q, %q; want gapless", u, b)
	}
	if n := modeEntries(t, e, tenant); n != 1 {
		t.Errorf("%d config update entries, want 1", n)
	}
	var diff string
	if err := e.super.QueryRow(context.Background(), `SELECT diff::text FROM audit_log
		WHERE tenant_id = $1 AND domain = 'config' AND action = 'update'`, tenant).Scan(&diff); err != nil {
		t.Fatal(err)
	}
	if diff == "" || diff == "null" {
		t.Errorf("config update entry has no diff")
	}

	if changed, err := set(metadata.NumberingGapless, principal); err != nil || changed {
		t.Errorf("gapless -> gapless: changed %v, %v; want false, nil", changed, err)
	}
	if n := modeEntries(t, e, tenant); n != 1 {
		t.Errorf("a no-op set wrote an entry: %d, want 1", n)
	}
	if _, err := set("sometimes", principal); !errors.Is(err, entitytypes.ErrInvalidNumberingMode) {
		t.Errorf("invalid mode: %v, want ErrInvalidNumberingMode", err)
	}
	if _, err := set(metadata.NumberingGaps, uuid.Nil); !errors.Is(err, entitytypes.ErrNoPublisher) {
		t.Errorf("no principal: %v, want ErrNoPublisher", err)
	}

	// In one transaction: a prefetched definition is dropped by the set, so
	// the create path's next Current reads the new mode.
	if err := tenancyTx(e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		b := &pgx.Batch{}
		s.QueueCurrent(tx, b, "asset")
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
		if _, err := s.SetNumberingMode(ctx, tx, entitytypes.SetNumberingModeInput{Key: "asset", Mode: metadata.NumberingGaps, By: principal}); err != nil {
			return err
		}
		c, _, err := s.Current(ctx, tx, "asset")
		if err != nil {
			return err
		}
		if c.NumberingMode() != metadata.NumberingGaps {
			t.Errorf("Current after a set in the same transaction: %q, want gaps", c.NumberingMode())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Acceptance: tenant A can neither read nor set tenant B's numbering mode.
func TestP336fNumberingModeCrossTenant(t *testing.T) {
	e := migrated(t)
	tenantA, principalA := numberingTenant(t, e)
	tenantB, principalB := numberingTenant(t, e)
	createType(t, e, tenantA, principalA, "asset", "AST", "")
	createType(t, e, tenantB, principalB, "asset", "AST", metadata.NumberingGapless)
	createType(t, e, tenantB, principalB, "widget", "WDG", metadata.NumberingGapless)
	s := entitytypes.Store{}

	// The same key reads each tenant's own row.
	if u, b := currentMode(t, e, tenantA, "asset"); u != metadata.NumberingGaps || b != metadata.NumberingGaps {
		t.Errorf("tenant A's asset: %q, %q; want gaps (tenant B's is gapless)", u, b)
	}
	// A key only tenant B has is not found in tenant A, read or set.
	if err := tenancyTx(e, tenantA, func(ctx context.Context, tx pgx.Tx) error {
		if _, _, err := s.Current(ctx, tx, "widget"); !errors.Is(err, entitytypes.ErrNotFound) {
			t.Errorf("tenant A reads tenant B's widget: %v, want ErrNotFound", err)
		}
		_, err := s.SetNumberingMode(ctx, tx, entitytypes.SetNumberingModeInput{Key: "widget", Mode: metadata.NumberingGaps, By: principalA})
		if !errors.Is(err, entitytypes.ErrNotFound) {
			t.Errorf("tenant A sets tenant B's widget: %v, want ErrNotFound", err)
		}
		// Setting its own asset touches only its own row.
		_, err = s.SetNumberingMode(ctx, tx, entitytypes.SetNumberingModeInput{Key: "asset", Mode: metadata.NumberingGapless, By: principalA})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := e.super.QueryRow(context.Background(), `SELECT string_agg(key || '=' || numbering_mode, ',' ORDER BY key)
		FROM entity_definitions WHERE tenant_id = $1`, tenantB).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "asset=gapless,widget=gapless" {
		t.Errorf("tenant B's modes after tenant A's writes: %s, want unchanged", got)
	}
	if n := modeEntries(t, e, tenantB); n != 0 {
		t.Errorf("tenant B has %d config update entries, want 0", n)
	}
	if u, _ := currentMode(t, e, tenantA, "asset"); u != metadata.NumberingGapless {
		t.Errorf("tenant A's own asset: %q, want gapless", u)
	}
}
