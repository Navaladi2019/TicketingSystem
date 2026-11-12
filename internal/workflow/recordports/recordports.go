// Package recordports is the worker's interp.RecordReader and
// interp.RecordWriter (P3-43). Both reach the database only through
// workerguard.InTenantTx, for the tenant the activity guard verified, and the
// writer writes through the records pipeline as source=workflow, so an
// update_record step is validated, audited, versioned and fed to the
// EventSink exactly as any other write (D-3-14).
package recordports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// verified returns the activity's verified tenant, and refuses a tenant the
// caller names that differs from it.
func verified(ctx context.Context, tenantID uuid.UUID) error {
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok || tenant != tenantID {
		return fmt.Errorf("%w: tenant is not the activity's verified tenant", interp.ErrForbidden)
	}
	return nil
}

// errNoRecord is a record the transaction's tenant does not have.
var errNoRecord = errors.New("recordports: no such record")

// recordSQL reads a record's entity key, version and payload. Only the
// records pipeline imports internal/records/store (D-3-6), so the worker's
// reads are this one statement; the tenant predicate is explicit as well as
// RLS's (TICK-480).
const recordSQL = `SELECT d.key, r.version, r.data
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
WHERE r.tenant_id = $1 AND r.id = $2`

// readRecord reads the record in tx: its entity key, version and payload,
// numbers as json.Number as the store decodes them.
func readRecord(ctx context.Context, tx pgx.Tx, tenantID, recordID uuid.UUID) (key string, version int64, data map[string]any, err error) {
	var raw []byte
	switch err := tx.QueryRow(ctx, recordSQL, tenantID, recordID).Scan(&key, &version, &raw); {
	case errors.Is(err, pgx.ErrNoRows):
		return "", 0, nil, errNoRecord
	case err != nil:
		return "", 0, nil, fmt.Errorf("recordports: read record: %w", err)
	}
	data = map[string]any{}
	if len(raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&data); err != nil {
			return "", 0, nil, fmt.Errorf("recordports: decode record: %w", err)
		}
		if data == nil {
			data = map[string]any{}
		}
	}
	return key, version, data, nil
}

// Reader is interp.RecordReader over the records table, in
// workerguard.InTenantTx.
type Reader struct {
	Pool *pgxpool.Pool
}

var _ interp.RecordReader = Reader{}

// Read returns the record's payload and its version. A record the tenant does
// not have, another tenant's included, is interp.ErrNotFound.
func (r Reader) Read(ctx context.Context, tenantID, recordID uuid.UUID) (map[string]any, int64, error) {
	if err := verified(ctx, tenantID); err != nil {
		return nil, 0, err
	}
	var (
		data    map[string]any
		version int64
	)
	err := workerguard.InTenantTx(ctx, r.Pool, func(tx pgx.Tx) (err error) {
		_, version, data, err = readRecord(ctx, tx, tenantID, recordID)
		return err
	})
	switch {
	case errors.Is(err, errNoRecord):
		return nil, 0, fmt.Errorf("%w: record", interp.ErrNotFound)
	case err != nil:
		return nil, 0, err
	}
	return data, version, nil
}

// Writer is interp.RecordWriter over the records pipeline.
type Writer struct {
	Pool *pgxpool.Pool
	// Pipeline's TxRunner must not draw on Pool (it opens a second
	// connection while the write's transaction is open): NewWriter gives it
	// a pool of its own (docs/budgets.md, "Worker pipeline pool").
	Pipeline *pipeline.Pipeline
}

// NewWriter is the worker's Writer: the records pipeline with the real
// EventSink (a workflow write fires triggers and signals like any other,
// D-3-14), whose TxRunner runs on pipePool, a pool of its own, never on pool.
func NewWriter(pool, pipePool *pgxpool.Pool) Writer {
	return Writer{Pool: pool, Pipeline: pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{},
		eventsink.Sink{}, pipeline.PoolRunner{Pool: pipePool}, pipeline.SystemClock{})}
}

var _ interp.RecordWriter = Writer{}

// Write applies set to the record as a JSON Merge Patch, once per
// idempotencyKey, as source=workflow. Causation comes only from the
// workflow_cause row cause names (workerguard.ResolveCause); the run's ref
// comes from the workflow id. A cause that is missing, another run's or
// older than the run's newest fails closed as interp.ErrForbidden; it never
// falls back to depth 0.
func (w Writer) Write(ctx context.Context, tenantID, recordID, _, cause uuid.UUID, idempotencyKey string, set map[string]any) error {
	if err := verified(ctx, tenantID); err != nil {
		return err
	}
	ref, err := workerguard.RunRef(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", interp.ErrForbidden, err)
	}
	actor := identity.Principal{TenantID: tenantID}
	err = workerguard.InTenantTx(ctx, w.Pool, func(tx pgx.Tx) error {
		causation, err := workerguard.ResolveCause(ctx, tx, cause)
		if errors.Is(err, workerguard.ErrStaleCause) {
			// The cause went stale: a newer delivery reached the run. If this
			// step's write already committed (its completion was lost and
			// Temporal retried), the earlier result stands and nothing is
			// written; otherwise the stale cause is refused as before.
			replayed, rerr := pipeline.Replayed(ctx, tx, actor, recordID, idempotencyKey)
			if rerr != nil {
				return rerr
			}
			if replayed {
				return nil
			}
		}
		if err != nil {
			return err
		}
		entityKey, version, _, err := readRecord(ctx, tx, tenantID, recordID)
		if err != nil {
			return err
		}
		_, err = w.Pipeline.Write(ctx, tx, pipeline.Command{
			Op:              pipeline.OpUpdate,
			EntityKey:       entityKey,
			RecordID:        recordID,
			ExpectedVersion: version,
			Data:            set,
			Actor:           actor,
			Source:          pipeline.SourceWorkflow,
			Causation:       causation,
			IdempotencyKey:  idempotencyKey,
			WorkflowRef:     ref,
		})
		return err
	})
	return mapError(err)
}

// mapError maps what the resolver and the pipeline refuse to the interp port
// errors. Anything else (a deadlock, a version conflict from a concurrent
// write, a database failure) is returned as is: retryable.
func mapError(err error) error {
	var (
		validation  *pipeline.ErrValidation
		transition  *pipeline.ErrIllegalTransition
		unique      *pipeline.ErrUniqueViolation
		idempotency *pipeline.ErrIdempotencyConflict
		notWritable *authz.ErrFieldNotWritable
	)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errNoRecord), errors.Is(err, pipeline.ErrNotFound), errors.Is(err, authz.ErrNotVisible):
		return fmt.Errorf("%w: record", interp.ErrNotFound)
	case errors.Is(err, pipeline.ErrRetryable), errors.Is(err, pipeline.ErrVersionConflict):
		return err
	case errors.Is(err, workerguard.ErrCauseNotFound), errors.Is(err, workerguard.ErrForeignCause),
		errors.Is(err, workerguard.ErrStaleCause), errors.Is(err, outbox.ErrCascadeTooDeep),
		errors.Is(err, authz.ErrForbidden), errors.As(err, &notWritable):
		return fmt.Errorf("%w: refused", interp.ErrForbidden)
	case errors.As(err, &transition):
		return fmt.Errorf("%w: transition", interp.ErrIllegalTransition)
	case errors.As(err, &validation), errors.As(err, &unique), errors.As(err, &idempotency),
		errors.Is(err, pipeline.ErrInvalidCommand):
		return fmt.Errorf("%w: invalid", interp.ErrValidation)
	}
	return fmt.Errorf("recordports: write record: %w", err)
}
