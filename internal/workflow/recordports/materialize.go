package recordports

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
)

// Materializer is schemabackfill.Materializer over the records pipeline
// (ADR-009 D3): pipeline.Write(OpMaterialize) with an empty patch,
// SourceSystem and actor_system schema-backfill. The schema backfill may not
// import the pipeline (D-3-6), so the worker builds this one and hands it
// over, as it does the interpreter's Writer.
type Materializer struct {
	Pipeline *pipeline.Pipeline
}

var _ schemabackfill.Materializer = Materializer{}

// NewMaterializer is the worker's Materializer. OpMaterialize tells no
// EventSink (D6), so the sink is the no-op one; the pipeline's own TxRunner
// runs on pipePool, a pool of its own, never on the pool the job's
// transactions hold (docs/budgets.md, "Worker pipeline pool").
func NewMaterializer(pipePool *pgxpool.Pool) Materializer {
	return Materializer{Pipeline: pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{},
		pipeline.NoopSink{}, pipeline.PoolRunner{Pool: pipePool}, pipeline.SystemClock{})}
}

// Materialize rewrites t's record in tx and maps the pipeline's outcome to
// the port's: written; a version conflict or a deadlock is
// schemabackfill.ErrConflict (the id stays pending); a record that no longer
// exists is not written (nothing to do); a typed refusal is a
// *schemabackfill.Refusal with pipeline.ErrorCode's code; anything else is
// returned as is, a database error.
func (m Materializer) Materialize(ctx context.Context, tx pgx.Tx, t schemabackfill.Target) (bool, error) {
	res, err := m.Pipeline.Write(ctx, tx, pipeline.Command{
		Op:          pipeline.OpMaterialize,
		EntityKey:   t.EntityKey,
		RecordID:    t.RecordID,
		Actor:       identity.Principal{TenantID: t.Tenant},
		Source:      pipeline.SourceSystem,
		ActorSystem: pipeline.ActorSchemaBackfill,
		Causation:   t.Causation,
	})
	switch code := pipeline.ErrorCode(err); code {
	case "":
		return res.Written, nil
	case pipeline.CodeVersionConflict, pipeline.CodeRetryable:
		return false, fmt.Errorf("%w: %s", schemabackfill.ErrConflict, code)
	case pipeline.CodeNotFound:
		return false, nil
	case pipeline.CodeDatabaseError:
		return false, err
	default:
		return false, &schemabackfill.Refusal{Code: code}
	}
}
