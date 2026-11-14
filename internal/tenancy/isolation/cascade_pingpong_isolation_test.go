//go:build integration

package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// CascadePingPong runs CI-10 (g) on the suite's cluster. It lives in a test
// file: D-3-6 (internal/records TestRecordWritersOnlyInPipeline) lets only
// the named packages import the records pipeline outside tests, and the
// suite runs only under the integration tag (make isolation). The path that
// cascade_paths.go registers calls it through pingPonger.
func (c cluster) CascadePingPong(ctx context.Context) error { return cascadePingPong(ctx, c) }

// pingPongEntity is the entity type the check creates in tenant A (the
// seed's ticket type has a schema the pipeline does not compile).
const pingPongEntity = "ci10g_pingpong"

// pingPongPublisher is tenant A's seeded principal (testdata/seed.sql).
const pingPongPublisher = "0a000000-0000-4000-8000-0000000003a1"

// pingPongSchema is a schema the pipeline compiles: a title, a state
// (status, the fixed state column) and a free summary the writes change.
const pingPongSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["title"],
  "properties": {
    "title": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["open"]},
    "summary": {"type": "string"}
  }
}`

// pingPongDefinition is pingPongEntity's version 1.
func pingPongDefinition() metadata.Definition {
	return metadata.Definition{Key: pingPongEntity, NumberPrefix: "CTG", Schema: json.RawMessage(pingPongSchema),
		FixedColumns: map[string]string{"status": metadata.ColumnState},
		States:       metadata.StateModel{Initial: "open", States: []string{"open"}}}
}

// TestPingPongDefinitionCompiles: the CI-10 (g) row's entity type is one the
// pipeline compiles, checked without a database.
func TestPingPongDefinitionCompiles(t *testing.T) {
	if _, err := metadata.Compile(pingPongDefinition()); err != nil {
		t.Fatal(err)
	}
}

// createPingPongType creates pingPongEntity at version 1 in tenant A, as
// nava_app through entitytypes.Store, as an admin's publish does.
func createPingPongType(ctx context.Context, cl Cluster, tenant uuid.UUID) error {
	pool, err := cl.Pool("nava_app")
	if err != nil {
		return err
	}
	defer pool.Close()
	return tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		_, err := (entitytypes.Store{}).Create(ctx, tx, tenant, entitytypes.NewInput{
			Name: "CI-10 (g)", PublishedBy: uuid.MustParse(pingPongPublisher),
			Definition: pingPongDefinition(),
		})
		return err
	})
}

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
// R is a new tenant A record of a type the check creates, written by a system
// writer, so no seeded row changes; the two runs' workflow_ref rows are closed at the end. Each write
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
	if err := createPingPongType(ctx, cl, g.tenant); err != nil {
		return fmt.Errorf("create the %s type: %w", pingPongEntity, err)
	}

	created, err := g.write(ctx, pipeline.Command{Op: pipeline.OpCreate, EntityKey: pingPongEntity,
		Data: map[string]any{"title": "ci10g " + run}, Actor: actor, Source: pipeline.SourceSystem})
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
		// The refs must close even when ctx is already cancelled, so the
		// statement runs on the same uncancellable context as the transaction.
		cctx := context.WithoutCancel(ctx)
		_ = g.tx(cctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(cctx, `UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1 AND outbox_id = ANY($2)`,
				g.tenant, []uuid.UUID{x, y})
			return err
		})
	}()

	// A person's edit stands in as the depth-0 write that signals both runs.
	res, err := g.write(ctx, pipeline.Command{Op: pipeline.OpUpdate, EntityKey: pingPongEntity, RecordID: r,
		ExpectedVersion: created.Version, Data: map[string]any{"summary": "depth 0"}, Actor: actor, Source: pipeline.SourceSystem})
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
			ExpectedVersion: version, Data: map[string]any{"summary": fmt.Sprintf("depth %d", depth)}, Actor: actor,
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

	// The refused write commits nothing but its cascade_too_deep entry, which
	// carries the same causation; the 8 writes are the other entries.
	var writes, refused, stored int64
	if err := g.tx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE action <> 'cascade_too_deep'),
				count(*) FILTER (WHERE action = 'cascade_too_deep'),
				(SELECT version FROM records WHERE tenant_id = $1 AND id = $2)
			FROM audit_log WHERE tenant_id = $1 AND subject_id = $2 AND source = 'workflow' AND causation_id = $3`,
			g.tenant, r, origin).Scan(&writes, &refused, &stored)
	}); err != nil {
		return err
	}
	if writes != int64(limit) || refused != 1 || stored != version {
		return fmt.Errorf("record R has %d workflow writes and %d cascade_too_deep entries under the origin at version %d; want exactly %d and 1 at %d (the refused write changed nothing)",
			writes, refused, stored, limit, version)
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
