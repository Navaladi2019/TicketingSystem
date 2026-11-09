package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
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

// RowOf is what the ACL sees of rec for p: its fixed columns and p's
// pair-matched shares (authz.SharesFor), read in tx.
func RowOf(ctx context.Context, tx pgx.Tx, p identity.Principal, rec *Record) (authz.Row, error) {
	shares, err := authz.SharesFor(ctx, tx, p, rec.ID)
	if err != nil {
		return authz.Row{}, fmt.Errorf("pipeline: %w", err)
	}
	return *rowOf(rec, shares), nil
}

// DecodeObject decodes a JSON object with numbers kept exact (json.Number),
// the form Write takes in Command.Data.
func DecodeObject(b []byte) (map[string]any, error) { return store.DecodeObject(b) }

// ListQuery is the store's list query, for the read-only list route.
type ListQuery = store.ListQuery

// ListPage is one page of a list and its exact total.
type ListPage = store.ListPage

// List reads one page of the transaction's tenant's records. It reads only.
func List(ctx context.Context, tx pgx.Tx, q ListQuery) (*ListPage, error) {
	page, err := store.List(ctx, tx, q)
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return page, nil
}

// DefinitionID is the id of the tenant's entity type key, or ErrNotFound.
func DefinitionID(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string) (uuid.UUID, error) {
	id, err := store.DefinitionID(ctx, tx, tenantID, key)
	if errors.Is(err, store.ErrNotFound) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// TieBreaksOnUpdatedAt reports whether a list sorted by col orders by
// (col, updated_at, id), so its cursor carries the last row's updated_at.
func TieBreaksOnUpdatedAt(col string) bool { return store.TieBreaksOnUpdatedAt(col) }

// ErrStatementTimeout is the store's: a list with a data filter ran past its
// statement timeout.
var ErrStatementTimeout = store.ErrStatementTimeout
