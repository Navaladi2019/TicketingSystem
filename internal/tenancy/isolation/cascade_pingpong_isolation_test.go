//go:build integration

package isolation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// cascadePingPong lives in a test file: D-3-6 (internal/records
// TestRecordWritersOnlyInPipeline) lets only the named packages import the
// records pipeline outside tests. The suite runs only under the integration
// tag (make isolation), so this file sets the CI-10 (g) path's check, which
// cascade_paths.go registers, before any path runs.
func init() { pingPongCheck = cascadePingPong }

// pingPongEntity is tenant A's seeded pipeline type (testdata/seed.sql).
const pingPongEntity = "ticket"

// cascadePingPong is ADR-001 CI-10 (g) (A5-7, CTO ruling on TICK-88,
// 2026-11-01; named PASS row per PLAN-003 D-3-16 and P3-44, TICK-713): runs X
// and Y both wait on record_changed on R, and each writes R when signalled.
// Everything runs as nava_worker under RLS in tenant A, through the real
// pipeline.Write and the real EventSink, as the worker's RecordWriter does:
// each write takes its causation only from the workflow_cause row of the
// signal that resumed it (the worker reads that row the same way,
// workerguard.ResolveCause), one deeper, and names its own run, so the sink
// signals the other run and never the writer. The write at depth
// CascadeDepthLimit+1 (9) must be refused with outbox.ErrCascadeTooDeep, and
// R must have exactly CascadeDepthLimit (8) workflow writes under the origin.
//
// R is a new tenant A record, created by a system writer, so no seeded row
// changes; the two runs' workflow_ref rows are closed at the end. Each write
// commits in its own transaction, as on the worker: the refused write's
// cascade_too_deep audit entry is written on the pipeline's own pool.
func cascadePingPong(ctx context.Context, cl Cluster) error {
	pool, err := cl.PoolSize("nava_worker", 2)
	if err != nil {
		return err
	}
	defer pool.Close()
	pipePool, err := cl.PoolSize("nava_worker", 2)
	if err != nil {
		return err
	}
	defer pipePool.Close()
	g := pingPong{pool: pool, tenant: uuid.MustParse(TenantA),
		p: pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, eventsink.Sink{},
			pipeline.PoolRunner{Pool: pipePool}, pipeline.SystemClock{})}
	actor := identity.Principal{TenantID: g.tenant}
	run := uuid.NewString()[:8]

	created, err := g.write(ctx, pipeline.Command{Op: pipeline.OpCreate, EntityKey: pingPongEntity,
		Data: map[string]any{"note": "ci10g " + run}, Actor: actor, Source: pipeline.SourceSystem})
	if err != nil {
		return fmt.Errorf("create R: %w", err)
	}
	r := created.RecordID
	x, y := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	if err := g.tx(ctx, func(tx pgx.Tx) error {
		for _, ref := range []uuid.UUID{x, y} {
			if _, err := tx.Exec(ctx, `INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, record_changed_signal_id)
				VALUES ($1, $2, $3, $4, $5)`, g.tenant, ref, r, outbox.InterpreterKind, uuid.New()); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("open the two waiting runs: %w", err)
	}
	defer func() {
		_ = g.tx(context.WithoutCancel(ctx), func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1 AND outbox_id = ANY($2)`,
				g.tenant, []uuid.UUID{x, y})
			return err
		})
	}()

	// A person's edit stands in as the depth-0 write that signals both runs.
	res, err := g.write(ctx, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: pingPongEntity, RecordID: r,
		ExpectedVersion: created.Version, Data: map[string]any{"note": "depth 0"}, Actor: actor, Source: pipeline.SourceSystem})
	if err != nil {
		return fmt.Errorf("the depth-0 write: %w", err)
	}
	version := res.Version
	var origin uuid.UUID
	if err := g.tx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT causation_origin_id FROM records WHERE tenant_id = $1 AND id = $2`, g.tenant, r).Scan(&origin)
	}); err != nil {
		return err
	}

	limit := int(config.CascadeDepthLimit)
	writer, other := x, y
	for depth := 1; ; depth++ {
		in, err := g.causes(ctx, writer)
		if err != nil {
			return err
		}
		if len(in) == 0 {
			return fmt.Errorf("depth %d: the writer was never signalled", depth)
		}
		c := in[len(in)-1]
		if c.origin != origin || int(c.depth) != depth-1 {
			return fmt.Errorf("depth %d: the resuming signal's stored causation (%s, %d), want (%s, %d)", depth, c.origin, c.depth, origin, depth-1)
		}
		otherBefore, err := g.causes(ctx, other)
		if err != nil {
			return err
		}
		res, err := g.write(ctx, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: pingPongEntity, RecordID: r,
			ExpectedVersion: version, Data: map[string]any{"note": fmt.Sprintf("depth %d", depth)}, Actor: actor,
			Source: pipeline.SourceWorkflow, Causation: outbox.Causation{OriginID: c.origin, Depth: c.depth + 1},
			IdempotencyKey: fmt.Sprintf("iso-ci10g:%s:%d", run, depth), WorkflowRef: writer})
		if depth == limit+1 {
			if err == nil {
				return fmt.Errorf("the write at depth %d committed, want ErrCascadeTooDeep", depth)
			}
			if !errors.Is(err, outbox.ErrCascadeTooDeep) {
				return fmt.Errorf("the write at depth %d: %w, want ErrCascadeTooDeep", depth, err)
			}
			break
		}
		if err != nil {
			return fmt.Errorf("the write at depth %d: %w", depth, err)
		}
		version = res.Version
		after, err := g.causes(ctx, writer)
		if err != nil {
			return err
		}
		if len(after) != len(in) {
			return fmt.Errorf("depth %d: the writer was signalled about its own write (%d -> %d signals)", depth, len(in), len(after))
		}
		got, err := g.causes(ctx, other)
		if err != nil {
			return err
		}
		if len(got) != len(otherBefore)+1 || got[len(got)-1].origin != origin || int(got[len(got)-1].depth) != depth {
			return fmt.Errorf("depth %d: the other run got %d new signals (want 1, at depth %d under %s)", depth, len(got)-len(otherBefore), depth, origin)
		}
		writer, other = other, writer
	}

	var writes, stored int64
	if err := g.tx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_id = $2
				AND source = 'workflow' AND causation_id = $3),
			(SELECT version FROM records WHERE tenant_id = $1 AND id = $2)`, g.tenant, r, origin).Scan(&writes, &stored)
	}); err != nil {
		return err
	}
	if writes != int64(limit) || stored != version {
		return fmt.Errorf("record R has %d workflow writes under the origin at version %d; want exactly %d at %d (the refused write changed nothing)",
			writes, stored, limit, version)
	}
	return nil
}

// pingPong is cascadePingPong's tenant A writer.
type pingPong struct {
	pool   *pgxpool.Pool
	tenant uuid.UUID
	p      *pipeline.Pipeline
}

func (g pingPong) tx(ctx context.Context, fn func(pgx.Tx) error) error {
	return tenancy.InTenantTx(ctx, g.pool, g.tenant, fn)
}

// write runs one pipeline.Write in its own committed tenant transaction.
func (g pingPong) write(ctx context.Context, cmd pipeline.Command) (pipeline.Result, error) {
	var res pipeline.Result
	err := g.tx(ctx, func(tx pgx.Tx) error {
		var err error
		res, err = g.p.Write(ctx, tx, cmd)
		return err
	})
	return res, err
}

// storedCause is a workflow_cause row: the causation the sink copied from an
// interpreter-bound outbox row (D-3-29).
type storedCause struct {
	origin uuid.UUID
	depth  int16
}

// causes is every stored delivery causation to run ref, oldest first (the
// outbox ids are UUIDv7).
func (g pingPong) causes(ctx context.Context, ref uuid.UUID) ([]storedCause, error) {
	var out []storedCause
	err := g.tx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT causation_origin_id, causation_depth FROM workflow_cause
			WHERE tenant_id = $1 AND ref_outbox_id = $2 ORDER BY outbox_id`, g.tenant, ref)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (storedCause, error) {
			var c storedCause
			return c, r.Scan(&c.origin, &c.depth)
		})
		return err
	})
	return out, err
}
