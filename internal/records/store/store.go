// Package store is the data access of the record write pipeline (PLAN-003
// D-3-6): the records and record_write_keys statements, the read of S9's
// audit_log_v0 history, and the split of a record's payload between its fixed
// columns and data. The audit entry of a write is not written here: the
// pipeline writes it through audit.Append (ADR-004 decision 1, P5-13b), and
// audit_log_v0 is read-only. Only internal/records/pipeline
// writes through it; P3-13's static test keeps it that way.
//
// Every function takes the caller's transaction, opened by tenancy.InTenantTx,
// so row-level security scopes each statement to that transaction's tenant.
// Statements that name a row also carry an explicit tenant_id, the tenant the
// pipeline took from the transaction itself, never from client input.
package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

var (
	// ErrNotFound means the transaction's tenant has no such record or entity
	// type.
	ErrNotFound = errors.New("records/store: not found")
	// ErrStale means an update found no row at the expected version.
	ErrStale = errors.New("records/store: stale version")
	// ErrDuplicateKey means another write holds the idempotency key. The
	// statement aborted the caller's transaction.
	ErrDuplicateKey = errors.New("records/store: duplicate idempotency key")
)

// idempotencyIndex is the primary key of record_write_keys (D-5-3): one
// write per tenant and key.
const idempotencyIndex = "record_write_keys_pkey"

// Record is one records row of a pipeline record. Data never holds a property
// mapped to a fixed column (TICK-360); Payload merges them back.
type Record struct {
	TenantID           uuid.UUID
	ID                 uuid.UUID
	EntityDefinitionID uuid.UUID
	EntityKey          string
	Version            int64
	SchemaVersion      int
	Number             string
	State              string
	RequesterID        *uuid.UUID
	AssigneeID         *uuid.UUID
	AssignmentGroupID  *uuid.UUID
	CreatedBy          *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Data               map[string]any
	CausationOriginID  *uuid.UUID
	CausationDepth     int16
}

// DefinitionID returns the id of the tenant's entity type key, or ErrNotFound.
func DefinitionID(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("records/store: read entity type: %w", err)
	}
	return id, nil
}

const recordSelect = `
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0)
FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
WHERE r.tenant_id = $1 AND r.id = $2`

// Get returns the record, or ErrNotFound. A PLAN-001 bare row (no entity
// type) is not a pipeline record and reads as ErrNotFound.
func Get(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) (*Record, error) {
	return scan(tx.QueryRow(ctx, recordSelect, tenantID, id))
}

func scan(row pgx.Row) (*Record, error) {
	var (
		r    Record
		data []byte
	)
	err := row.Scan(&r.TenantID, &r.ID, &r.EntityDefinitionID, &r.EntityKey, &r.Version, &r.SchemaVersion, &r.Number, &r.State,
		&r.RequesterID, &r.AssigneeID, &r.AssignmentGroupID, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
		&data, &r.CausationOriginID, &r.CausationDepth)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("records/store: read record: %w", err)
	}
	if r.Data, err = DecodeObject(data); err != nil {
		return nil, fmt.Errorf("records/store: decode data: %w", err)
	}
	return &r, nil
}

// Insert writes a new record at r.Version.
func Insert(ctx context.Context, tx pgx.Tx, r *Record) error {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return fmt.Errorf("records/store: encode data: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12, $13, $14, $15)`,
		r.TenantID, r.ID, r.Version, r.EntityDefinitionID, r.SchemaVersion, r.Number, r.State,
		r.RequesterID, r.AssigneeID, r.AssignmentGroupID, r.CreatedBy, r.CreatedAt, jsonParam(data),
		r.CausationOriginID, r.CausationDepth); err != nil {
		return fmt.Errorf("records/store: insert record: %w", err)
	}
	return nil
}

// Update writes r's mutable columns if the stored version is expected, bumps
// the version by one, and returns the new version. It returns ErrStale when no
// row matched. This UPDATE is the only statement that bumps a pipeline
// record's version (D-3-6). The set-once columns (entity_definition_id,
// number, created_by, created_at) are not in it; nava_app has no grant on
// them.
func Update(ctx context.Context, tx pgx.Tx, r *Record, expected int64) (int64, error) {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return 0, fmt.Errorf("records/store: encode data: %w", err)
	}
	var v int64
	err = tx.QueryRow(ctx, `
UPDATE records SET schema_version = $3, state = $4, requester_id = $5, assignee_id = $6,
    assignment_group_id = $7, updated_at = $8, data = $9, causation_origin_id = $10,
    causation_depth = $11, version = version + 1
WHERE tenant_id = $1 AND id = $2 AND version = $12
RETURNING version`,
		r.TenantID, r.ID, r.SchemaVersion, r.State, r.RequesterID, r.AssigneeID,
		r.AssignmentGroupID, r.UpdatedAt, jsonParam(data), r.CausationOriginID, r.CausationDepth, expected).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrStale
	}
	if err != nil {
		return 0, fmt.Errorf("records/store: update record: %w", err)
	}
	return v, nil
}

// Snapshot is a record as an audit row's before or after holds it: the
// payload merged with its fixed columns, and what a replay returns.
type Snapshot struct {
	Number        string         `json:"number"`
	Version       int64          `json:"version"`
	SchemaVersion int            `json:"schema_version"`
	Data          map[string]any `json:"data"`
}

// PayloadJSON returns the payload of record id at version, built by
// PostgreSQL from the stored row and returned as jsonb text: data, with every
// property of fixed (property to column, metadata.Compiled.FixedColumns) set
// from its column and an unset uuid column leaving its property out, as
// Payload does. It returns ErrStale when the record is not at version.
//
// These bytes are the input to jsondiff.Diff for the record's audit entry
// (TICK-451 review, binding on P5-13b): the jsonb value as PostgreSQL holds
// it, never the request body or a re-marshalled Go value, so a number keeps
// the literal jsonb prints and diff_sha256 does not depend on how the client
// spelled it.
func PayloadJSON(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, version int64, fixed map[string]string) (json.RawMessage, error) {
	props := slices.Sorted(maps.Keys(fixed))
	args := []any{tenantID, id, version, props}
	expr := "(data - $4::text[])"
	for _, prop := range props {
		col := fixed[prop]
		switch col {
		case metadata.ColumnState, metadata.ColumnRequesterID, metadata.ColumnAssigneeID, metadata.ColumnAssignmentGroupID:
		default:
			return nil, fmt.Errorf("records/store: unknown fixed column %q", col)
		}
		args = append(args, prop)
		n := len(args)
		// col is one of the four constants above, never input.
		if col == metadata.ColumnState {
			expr += fmt.Sprintf(" || jsonb_build_object($%d::text, %s)", n, col)
			continue
		}
		expr += fmt.Sprintf(" || CASE WHEN %s IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($%d::text, %s) END", col, n, col)
	}
	var s string
	err := tx.QueryRow(ctx, "SELECT ("+expr+")::text FROM records WHERE tenant_id = $1 AND id = $2 AND version = $3", args...).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStale
	}
	if err != nil {
		return nil, fmt.Errorf("records/store: read payload as jsonb: %w", err)
	}
	return json.RawMessage(s), nil
}

// WriteKey is one record_write_keys row (D-5-3): the write that holds an
// idempotency key and the Result a replay returns.
type WriteKey struct {
	TenantID       uuid.UUID
	IdempotencyKey string
	DefinitionID   uuid.UUID
	// Action is "create" or "update".
	Action        string
	RecordID      uuid.UUID
	Number        string
	Version       int64
	SchemaVersion int
	// AuditEntryID is a soft pointer to the write's primary audit entry.
	AuditEntryID uuid.UUID
}

// InsertWriteKey claims k.IdempotencyKey for the write, in the write's
// transaction. A key another write holds is ErrDuplicateKey, also when that
// write has not committed yet (the insert waits for it); tx is aborted then.
func InsertWriteKey(ctx context.Context, tx pgx.Tx, k *WriteKey) error {
	_, err := tx.Exec(ctx, `
INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
    version, schema_version, audit_entry_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		k.TenantID, k.IdempotencyKey, k.DefinitionID, k.Action, k.RecordID, k.Number,
		k.Version, k.SchemaVersion, k.AuditEntryID)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == idempotencyIndex {
		return ErrDuplicateKey
	}
	if err != nil {
		return fmt.Errorf("records/store: insert write key: %w", err)
	}
	return nil
}

// WriteKeyByKey returns the write that holds the idempotency key, or
// ErrNotFound. It reads record_write_keys only, never the audit log.
func WriteKeyByKey(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string) (*WriteKey, error) {
	k := WriteKey{TenantID: tenantID, IdempotencyKey: key}
	err := tx.QueryRow(ctx, `
SELECT definition_id, action, record_id, number, version, schema_version, audit_entry_id
FROM record_write_keys
WHERE tenant_id = $1 AND idempotency_key = $2`, tenantID, key).
		Scan(&k.DefinitionID, &k.Action, &k.RecordID, &k.Number, &k.Version, &k.SchemaVersion, &k.AuditEntryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("records/store: read write key: %w", err)
	}
	return &k, nil
}

// jsonParam is a jsonb statement argument. Over QueryExecModeExec (nava-core's
// PgBouncer pool) the parameter has no declared type, and pgx sends []byte as
// bytea text, which jsonb refuses; a string is sent as text. nil stays NULL.
func jsonParam(b []byte) any {
	if b == nil {
		return nil
	}
	return string(b)
}

// DecodeObject decodes a JSON object with numbers as json.Number, the form
// metadata.Compiled.Validate takes. An empty input is an empty object.
func DecodeObject(b []byte) (map[string]any, error) {
	out := map[string]any{}
	if len(b) == 0 {
		return out, nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

// Payload returns the record's payload: a copy of Data with every property
// mapped to a fixed column set from its column (fixed maps property to
// column, metadata.Compiled.FixedColumns). An unset uuid column leaves its
// property out. The column wins over any stale copy in Data.
func (r *Record) Payload(fixed map[string]string) map[string]any {
	out := maps.Clone(r.Data)
	if out == nil {
		out = map[string]any{}
	}
	for prop, col := range fixed {
		delete(out, prop)
		var id *uuid.UUID
		switch col {
		case metadata.ColumnState:
			out[prop] = r.State
			continue
		case metadata.ColumnRequesterID:
			id = r.RequesterID
		case metadata.ColumnAssigneeID:
			id = r.AssigneeID
		case metadata.ColumnAssignmentGroupID:
			id = r.AssignmentGroupID
		}
		if id != nil {
			out[prop] = id.String()
		}
	}
	return out
}

// Split moves every property mapped to a fixed column out of payload and into
// r's columns, and sets r.Data to the rest; payload is not changed. The state
// property must hold a string and each uuid property, if present, a UUID
// string: the caller has validated payload against the schema, whose
// compiled mapping requires both (metadata.Compile).
func (r *Record) Split(payload map[string]any, fixed map[string]string) error {
	data := maps.Clone(payload)
	if data == nil {
		data = map[string]any{}
	}
	r.RequesterID, r.AssigneeID, r.AssignmentGroupID = nil, nil, nil
	for prop, col := range fixed {
		v, present := data[prop]
		delete(data, prop)
		if col == metadata.ColumnState {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("records/store: state property %q is not a string", prop)
			}
			r.State = s
			continue
		}
		if !present || v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("records/store: property %q is not a uuid string", prop)
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("records/store: property %q is not a uuid: %w", prop, err)
		}
		switch col {
		case metadata.ColumnRequesterID:
			r.RequesterID = &id
		case metadata.ColumnAssigneeID:
			r.AssigneeID = &id
		case metadata.ColumnAssignmentGroupID:
			r.AssignmentGroupID = &id
		default:
			return fmt.Errorf("records/store: unknown fixed column %q", col)
		}
	}
	r.Data = data
	return nil
}

// HistoryRow is one create or update row of a record's audit history.
type HistoryRow struct {
	ID            uuid.UUID
	At            time.Time
	ActorID       *uuid.UUID
	Source        string
	Action        string
	Before, After *Snapshot
}

// History returns the record's create and update audit rows, newest first
// (at, then id, both descending). When afterID is not uuid.Nil it returns only
// the rows older than the position (afterAt, afterID). It reads limit rows.
// The cascade rows (they change no record) are left out.
func History(ctx context.Context, tx pgx.Tx, tenantID, recordID uuid.UUID, afterAt time.Time, afterID uuid.UUID, limit int) ([]HistoryRow, error) {
	rows, err := tx.Query(ctx, `
SELECT id, at, actor_id, source, action, before, after
FROM audit_log_v0
WHERE tenant_id = $1 AND record_id = $2 AND action IN ('create', 'update')
  AND ($3::uuid IS NULL OR (at, id) < ($4::timestamptz, $3::uuid))
ORDER BY at DESC, id DESC
LIMIT $5`, tenantID, recordID, optionalID(afterID), afterAt, limit)
	if err != nil {
		return nil, fmt.Errorf("records/store: read history: %w", err)
	}
	defer rows.Close()
	var out []HistoryRow
	for rows.Next() {
		var (
			h             HistoryRow
			before, after []byte
		)
		if err := rows.Scan(&h.ID, &h.At, &h.ActorID, &h.Source, &h.Action, &before, &after); err != nil {
			return nil, fmt.Errorf("records/store: scan history: %w", err)
		}
		if h.Before, err = decodeSnapshot(before); err != nil {
			return nil, err
		}
		if h.After, err = decodeSnapshot(after); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("records/store: read history: %w", err)
	}
	return out, nil
}

func optionalID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// decodeSnapshot reads an audit row's before or after; NULL is nil. Numbers
// stay exact.
func decodeSnapshot(b []byte) (*Snapshot, error) {
	if b == nil {
		return nil, nil
	}
	var raw struct {
		Number        string          `json:"number"`
		Version       int64           `json:"version"`
		SchemaVersion int             `json:"schema_version"`
		Data          json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("records/store: decode audit snapshot: %w", err)
	}
	data, err := DecodeObject(raw.Data)
	if err != nil {
		return nil, fmt.Errorf("records/store: decode audit snapshot: %w", err)
	}
	return &Snapshot{Number: raw.Number, Version: raw.Version, SchemaVersion: raw.SchemaVersion, Data: data}, nil
}
