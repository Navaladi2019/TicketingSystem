// Package store is the data access of the record write pipeline (PLAN-003
// D-3-6): the records and audit_log statements, and the split of a record's
// payload between its fixed columns and data. Only internal/records/pipeline
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
	// ErrDuplicateKey means another write holds the audit row's idempotency
	// key. The statement aborted the caller's transaction.
	ErrDuplicateKey = errors.New("records/store: duplicate idempotency key")
)

// idempotencyIndex is the partial unique index on audit_log (P3-09).
const idempotencyIndex = "audit_log_idempotency_key"

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

// Audit is one audit_log row.
type Audit struct {
	TenantID uuid.UUID
	ID       uuid.UUID
	// RecordID is uuid.Nil (NULL) only when there is no record to name: a
	// rejected create.
	RecordID      uuid.UUID
	Action        string
	ActorID       *uuid.UUID
	Source        string
	Before, After *Snapshot
	// Detail is written to after on a row that changed no record (the
	// cascade rows: its reason, and the suppressed definition). It holds
	// references and codes only, never record data. Set it or After, not
	// both.
	Detail            map[string]string
	CausationOriginID *uuid.UUID
	CausationDepth    int16
	// IdempotencyKey is set on the primary row of a write only (D-3-13).
	IdempotencyKey *string
	At             time.Time
}

// InsertAudit writes a. A second row with the same idempotency key is
// ErrDuplicateKey; tx is aborted then.
func InsertAudit(ctx context.Context, tx pgx.Tx, a *Audit) error {
	before, err := encodeSnapshot(a.Before)
	if err != nil {
		return err
	}
	after, err := encodeSnapshot(a.After)
	if err != nil {
		return err
	}
	if a.Detail != nil {
		if a.After != nil {
			return errors.New("records/store: an audit row has a snapshot or a detail, not both")
		}
		if after, err = json.Marshal(a.Detail); err != nil {
			return fmt.Errorf("records/store: encode audit detail: %w", err)
		}
	}
	var record *uuid.UUID
	if a.RecordID != uuid.Nil {
		record = &a.RecordID
	}
	_, err = tx.Exec(ctx, `
INSERT INTO audit_log (tenant_id, id, record_id, action, actor_id, source, before, after,
    causation_origin_id, causation_depth, idempotency_key, at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		a.TenantID, a.ID, record, a.Action, a.ActorID, a.Source, jsonParam(before), jsonParam(after),
		a.CausationOriginID, a.CausationDepth, a.IdempotencyKey, a.At)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == idempotencyIndex {
		return ErrDuplicateKey
	}
	if err != nil {
		return fmt.Errorf("records/store: insert audit: %w", err)
	}
	return nil
}

// KeyedWrite is the write that holds an idempotency key: its primary audit
// row and the entity type of its record.
type KeyedWrite struct {
	RecordID  uuid.UUID
	EntityKey string
	// Action is the audit row's action: "create" or "update".
	Action string
	After  *Snapshot
}

// AuditByKey returns the write that holds the idempotency key, or
// ErrNotFound.
func AuditByKey(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, key string) (*KeyedWrite, error) {
	var (
		record    *uuid.UUID
		entityKey *string
		action    string
		after     []byte
	)
	err := tx.QueryRow(ctx, `
SELECT a.record_id, d.key, a.action, a.after
FROM audit_log a
LEFT JOIN records r ON r.tenant_id = a.tenant_id AND r.id = a.record_id
LEFT JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
WHERE a.tenant_id = $1 AND a.idempotency_key = $2 AND a.idempotency_key IS NOT NULL`, tenantID, key).
		Scan(&record, &entityKey, &action, &after)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("records/store: read audit by key: %w", err)
	}
	if record == nil || entityKey == nil || after == nil {
		return nil, errors.New("records/store: keyed audit row names no pipeline record")
	}
	var s Snapshot
	if err := json.Unmarshal(after, &s); err != nil {
		return nil, fmt.Errorf("records/store: decode audit snapshot: %w", err)
	}
	return &KeyedWrite{RecordID: *record, EntityKey: *entityKey, Action: action, After: &s}, nil
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

func encodeSnapshot(s *Snapshot) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("records/store: encode audit snapshot: %w", err)
	}
	return b, nil
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
