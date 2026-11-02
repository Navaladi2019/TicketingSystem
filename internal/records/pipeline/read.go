package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
)

// Record is a stored record as Get returns it. Only the pipeline imports the
// store (D-3-6), so the API reads records through these accessors.
type Record = store.Record

// Get reads a record of the transaction's tenant, or ErrNotFound. It reads
// only; every write goes through Write.
func Get(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) (*Record, error) {
	rec, err := store.Get(ctx, tx, tenantID, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return rec, nil
}

// DecodeObject decodes a JSON object with numbers kept exact (json.Number),
// the form Write takes in Command.Data.
func DecodeObject(b []byte) (map[string]any, error) { return store.DecodeObject(b) }

// Change is one create or update of a record, from its audit row.
type Change struct {
	ID      uuid.UUID
	At      time.Time
	ActorID *uuid.UUID
	Source  string
	Action  string
	// Before is nil on a create. Both are the record's payload (fixed columns
	// merged back), not the whole snapshot.
	Before, After map[string]any
}

// History returns up to limit changes of the record, newest first, older than
// the position (afterAt, afterID) when afterID is not uuid.Nil. It does not
// tell a missing record from one with no history: call Get first.
func History(ctx context.Context, tx pgx.Tx, tenantID, recordID uuid.UUID, afterAt time.Time, afterID uuid.UUID, limit int) ([]Change, error) {
	rows, err := store.History(ctx, tx, tenantID, recordID, afterAt, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	out := make([]Change, len(rows))
	for i, r := range rows {
		c := Change{ID: r.ID, At: r.At, ActorID: r.ActorID, Source: r.Source, Action: r.Action}
		if r.Before != nil {
			c.Before = r.Before.Data
		}
		if r.After != nil {
			c.After = r.After.Data
		}
		out[i] = c
	}
	return out, nil
}
