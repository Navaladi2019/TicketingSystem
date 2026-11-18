//go:build integration

// TICK-657 on TICK-659's numbering_mode (PLAN-003 D-3-9, ADR-005 §2.1.1):
// `gaps` types take their number with tenancy.AllocTx (L2s), `gapless` types
// last inside the create transaction (L1). Both upsert the same counter row,
// so types of either mode in one tenant do not interfere, and a type whose
// mode changes while creates run never puts one number on two records.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// gaplessDefinition is d numbered `gapless`.
func gaplessDefinition(d metadata.Definition) metadata.Definition {
	d.NumberingMode = metadata.NumberingGapless
	return d
}

// rolledBack runs a create in a transaction that then rolls back, and
// returns the number the create had.
func rolledBack(t *testing.T, e env, p *pipeline.Pipeline, f tenantFixture, cmd pipeline.Command) string {
	t.Helper()
	ctx := context.Background()
	rollback := errors.New("roll back")
	var res pipeline.Result
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		var err error
		if res, err = p.Write(ctx, tx, cmd); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rolled-back create: %v", err)
	}
	return res.Number
}

// setMode sets the type's numbering mode in its own committed transaction.
func setMode(t *testing.T, e env, f tenantFixture, key string, mode metadata.NumberingMode) {
	t.Helper()
	if err := tenancy.InTenantTx(context.Background(), e.app, f.id, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.SetNumberingMode(context.Background(), tx,
			entitytypes.SetNumberingModeInput{Key: key, Mode: mode, By: f.principal.PrincipalID})
		return err
	}); err != nil {
		t.Errorf("set %s to %s: %v", key, mode, err)
	}
}

// AC: a `gaps` type and a `gapless` type of one tenant number side by side:
// a rollback leaves a gap in the first and gives the number back in the
// second, and switching a type's mode carries on from its counter.
func TestNumberingModesDoNotInterfere(t *testing.T) {
	e := migrated(t)
	f := seedTenantWith(t, e, definition(schemaV1), gaplessDefinition(assetDefinition()))
	p := newPipeline(e, nil)
	ctx := context.Background()
	inc := func(title string) pipeline.Command { return createCmd(f, map[string]any{"title": title}) }
	ast := func(name string) pipeline.Command { return assetCmd(f, map[string]any{"name": name}) }
	create := func(cmd pipeline.Command) {
		t.Helper()
		if _, err := write(ctx, e, p, f.id, cmd); err != nil {
			t.Fatal(err)
		}
	}

	create(inc("one"))
	create(ast("one"))
	if n := rolledBack(t, e, p, f, inc("lost")); n != "INC0000002" {
		t.Fatalf("gaps rollback took %s, want INC0000002", n)
	}
	if n := rolledBack(t, e, p, f, ast("returned")); n != "AST0000002" {
		t.Fatalf("gapless rollback took %s, want AST0000002", n)
	}
	create(inc("two"))
	create(ast("two"))
	if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, []string{"INC0000001", "INC0000003"}) {
		t.Fatalf("gaps type numbers %v, want INC0000001, INC0000003", got)
	}
	if got := numbers(t, e, f.id, "asset"); !slices.Equal(got, wantNumbers(t, "AST", 1, 2)) {
		t.Fatalf("gapless type numbers %v, want AST0000001..2", got)
	}

	// Switch both: each carries on from its own counter.
	setMode(t, e, f, entityKey, metadata.NumberingGapless)
	setMode(t, e, f, "asset", metadata.NumberingGaps)
	if n := rolledBack(t, e, p, f, inc("returned")); n != "INC0000004" {
		t.Fatalf("incident, now gapless, rolled back %s, want INC0000004", n)
	}
	if n := rolledBack(t, e, p, f, ast("lost")); n != "AST0000003" {
		t.Fatalf("asset, now gaps, rolled back %s, want AST0000003", n)
	}
	create(inc("three"))
	create(ast("three"))
	if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, []string{"INC0000001", "INC0000003", "INC0000004"}) {
		t.Fatalf("incident numbers %v, want 1, 3, 4", got)
	}
	if got := numbers(t, e, f.id, "asset"); !slices.Equal(got, []string{"AST0000001", "AST0000002", "AST0000004"}) {
		t.Fatalf("asset numbers %v, want 1, 2, 4", got)
	}
}

// AC (ADR-005 §2.1.1, pinned): 8 concurrent creators on one type while its
// mode flips gaps -> gapless -> gaps several times, with every fifth create
// rolled back after it has its number. No two records share a number, and
// no create fails but the forced ones.
func TestModeSwitchNeverReusesANumber(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	modeSwitchNeverReusesANumber(t, e, f)
}

// modeSwitchNeverReusesANumber is the §2.1.1 pinned test on the incident
// type of f, whatever its number format.
func modeSwitchNeverReusesANumber(t *testing.T, e env, f tenantFixture) {
	t.Helper()
	p := newPipeline(e, nil)
	const (
		workers = 8
		each    = 30
		flips   = 6
	)
	forced := errors.New("forced failure")
	var (
		wg        sync.WaitGroup
		committed atomic.Int64
		mu        sync.Mutex
		failures  []error
	)
	for w := range workers {
		wg.Go(func() {
			for i := range each {
				fail := (w*each+i)%5 == 4
				err := tenancy.InTenantTx(context.Background(), e.app, f.id, func(tx pgx.Tx) error {
					cmd := createCmd(f, map[string]any{"title": fmt.Sprintf("%d-%d", w, i)})
					if _, err := p.Write(context.Background(), tx, cmd); err != nil {
						return err
					}
					if fail {
						return forced
					}
					return nil
				})
				switch {
				case err == nil:
					committed.Add(1)
				case !errors.Is(err, forced):
					mu.Lock()
					failures = append(failures, err)
					mu.Unlock()
				}
			}
		})
	}
	modes := []metadata.NumberingMode{metadata.NumberingGapless, metadata.NumberingGaps}
	for i := range flips {
		time.Sleep(40 * time.Millisecond)
		setMode(t, e, f, entityKey, modes[i%2])
	}
	wg.Wait()

	if len(failures) > 0 {
		t.Fatalf("%d creates failed beyond the forced ones, first: %v", len(failures), failures[0])
	}
	got := numbers(t, e, f.id, entityKey)
	if int64(len(got)) != committed.Load() {
		t.Fatalf("%d records, %d committed creates", len(got), committed.Load())
	}
	if uniq := slices.Compact(slices.Clone(got)); len(uniq) != len(got) {
		t.Fatalf("a number is on two records: %d records, %d distinct numbers", len(got), len(uniq))
	}
	next := counterNext(t, e, f.id, entityKey)
	if next <= len(got) {
		t.Fatalf("counter next %d, want past the %d committed records", next, len(got))
	}
	t.Logf("%d committed, %d forced to fail, counter next %d, over %d mode flips", len(got), workers*each-len(got), next, flips)
}
