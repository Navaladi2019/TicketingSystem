package schemabackfill

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// Target is one record to materialize: its tenant, entity type and id, and
// the causation the rewrite's audit entry carries (D2): the job's causation
// audit entry as the origin at that entry's depth + 1, or empty (a minted
// origin at depth 0) for a job with none, such as a rollout job (D15).
type Target struct {
	Tenant    uuid.UUID
	EntityKey string
	RecordID  uuid.UUID
	Causation outbox.Causation
}

// Materializer writes one record's missing defaults in tx through the I4
// pipeline: pipeline.Write(OpMaterialize) with an empty patch, SourceSystem
// and actor_system schema-backfill (D3). This package may not import the
// records pipeline (D-3-6, TestRecordWritersOnlyInPipeline), so the worker
// passes recordports.Materializer.
//
// Materialize reports whether the record was rewritten: false for a record
// that already holds every default, or that no longer exists. Its errors are
// ErrConflict (a live write landed in between, or the transaction was
// aborted by a deadlock: retry the record later), a *Refusal (the pipeline
// refused the rewrite, with its stable code), or any other error, which is a
// database error. On any error the caller rolls tx back.
type Materializer interface {
	Materialize(ctx context.Context, tx pgx.Tx, t Target) (written bool, err error)
}

// ErrConflict means the rewrite met a live write: a version conflict, or a
// deadlock or serialization failure. Nothing was written; the record stays
// pending and is retried on a later pass (D7).
var ErrConflict = errors.New("schemabackfill: a live write landed in between, retry the record later")

// Refusal is a rewrite the pipeline refused. Code is its stable lower-snake
// code (pipeline.ErrorCode, such as validation_failed): never a value.
type Refusal struct {
	Code string
}

func (r *Refusal) Error() string {
	return "schemabackfill: the pipeline refused the rewrite: " + r.Code
}

// CodeDatabaseError is the failure code of a record whose rewrite, or the
// statements around it, failed with a database error (D7). It is never the
// SQLSTATE or the message.
const CodeDatabaseError = "db_error"
