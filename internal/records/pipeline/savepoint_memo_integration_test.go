//go:build integration

// TICK-651: a record created inside a savepoint is in the transaction's memo
// only while the savepoint's work is: rolling the savepoint back drops it, so
// a later Get in the same transaction misses; releasing it keeps it.

package pipeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

func TestGetAfterSavepointCreate(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	for _, c := range []struct {
		name     string
		rollback bool
	}{
		{"rolled back misses", true},
		{"released hits", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			inTx(t, e, f.id, func(tx pgx.Tx) error {
				sp, err := tx.Begin(ctx)
				if err != nil {
					return err
				}
				res, err := p.Write(ctx, sp, createCmd(f, map[string]any{"title": "in a savepoint"}))
				if err != nil {
					_ = sp.Rollback(ctx)
					return err
				}
				if c.rollback {
					err = sp.Rollback(ctx)
				} else {
					err = sp.Commit(ctx)
				}
				if err != nil {
					return err
				}
				rec, err := pipeline.Get(ctx, tx, f.id, res.RecordID)
				switch {
				case c.rollback && !errors.Is(err, pipeline.ErrNotFound):
					t.Errorf("Get after rollback = %v, %v; want ErrNotFound", rec, err)
				case !c.rollback && err != nil:
					t.Errorf("Get after release: %v", err)
				case !c.rollback && rec.ID != res.RecordID:
					t.Errorf("Get after release = record %s, want %s", rec.ID, res.RecordID)
				}
				return nil
			})
		})
	}
}
