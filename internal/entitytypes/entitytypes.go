// Package entitytypes is the definition store: it reads and creates entity
// types and their published, immutable versions (PLAN-003 D-3-2, D-3-4;
// TICK-343). It owns DefinitionReader, which the record write pipeline
// (P3-10) uses, and the writes the API handlers use. P3-05b (TICK-365)
// reuses it to publish version N+1.
//
// Every function takes the caller's transaction, opened by tenancy.InTenantTx,
// so row-level security scopes each query to the tenant of that transaction.
// A key of another tenant is not visible there and reads as ErrNotFound.
package entitytypes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

var (
	// ErrNotFound means no such entity type or version in the tenant.
	ErrNotFound = errors.New("entitytypes: not found")
	// ErrKeyExists means the tenant already has an entity type with that key.
	ErrKeyExists = errors.New("entitytypes: key exists")
	// ErrPrefixTaken means another entity type of the tenant has that prefix.
	ErrPrefixTaken = errors.New("entitytypes: number prefix taken")
)

// Unique constraints of entity_definitions (db/migrations).
const (
	keyConstraint    = "entity_definitions_key_key"
	prefixConstraint = "entity_definitions_number_prefix_key"
)

// DefinitionReader reads compiled definitions inside the caller's InTenantTx.
// The record write pipeline (P3-10) consumes it.
type DefinitionReader interface {
	// Version returns version of the entity type, or ErrNotFound.
	Version(ctx context.Context, tx pgx.Tx, entityKey string, version int) (*metadata.Compiled, error)
	// Current returns the current version and its number, or ErrNotFound.
	Current(ctx context.Context, tx pgx.Tx, entityKey string) (*metadata.Compiled, int, error)
}

// Store reads and writes entity types. It holds no state.
type Store struct{}

var _ DefinitionReader = Store{}

// Summary is one row of the entity-type list.
type Summary struct {
	Key            string
	Name           string
	NumberPrefix   string
	CurrentVersion int
	UpdatedAt      time.Time
}

// Version is one published version, with the schema as the stored bytes.
type Version struct {
	Key          string
	Name         string
	NumberPrefix string
	Version      int
	// Schema is the exact bytes that were published.
	Schema       json.RawMessage
	SchemaSHA256 [sha256.Size]byte
	FixedColumns map[string]string
	States       metadata.StateModel
	Compat       string
	Current      bool
	PublishedAt  time.Time
}

const versionSelect = `
SELECT d.key, d.name, d.number_prefix, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id`

func scanVersion(row pgx.Row) (*Version, error) {
	var (
		v      Version
		schema string
		sum    []byte
		cur    int
	)
	err := row.Scan(&v.Key, &v.Name, &v.NumberPrefix, &cur, &v.Version, &schema, &sum, &v.FixedColumns, &v.States, &v.Compat, &v.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("entitytypes: read version: %w", err)
	}
	if len(sum) != sha256.Size {
		return nil, errors.New("entitytypes: stored schema_sha256 has the wrong length")
	}
	v.Schema = json.RawMessage(schema)
	copy(v.SchemaSHA256[:], sum)
	v.Current = cur == v.Version
	return &v, nil
}

// GetVersion returns one version with its stored schema bytes.
func (Store) GetVersion(ctx context.Context, tx pgx.Tx, key string, version int) (*Version, error) {
	if version < 1 {
		return nil, ErrNotFound
	}
	return scanVersion(tx.QueryRow(ctx, versionSelect+` WHERE d.key = $1 AND v.version = $2`, key, version))
}

// GetCurrent returns the current version of the entity type.
func (Store) GetCurrent(ctx context.Context, tx pgx.Tx, key string) (*Version, error) {
	return scanVersion(tx.QueryRow(ctx, versionSelect+` WHERE d.key = $1 AND v.version = d.current_version`, key))
}

// List returns up to limit entity types with key greater than after, by key.
func (Store) List(ctx context.Context, tx pgx.Tx, after string, limit int) ([]Summary, error) {
	rows, err := tx.Query(ctx, `
SELECT key, name, number_prefix, current_version, updated_at
FROM entity_definitions WHERE key > $1 ORDER BY key LIMIT $2`, after, limit)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: list: %w", err)
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.Key, &s.Name, &s.NumberPrefix, &s.CurrentVersion, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("entitytypes: list: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("entitytypes: list: %w", err)
	}
	return out, nil
}

// NewInput is a new entity type, already validated and compiled by the caller.
type NewInput struct {
	Name        string
	PublishedBy uuid.UUID
	// Definition carries Key, NumberPrefix and the exact Schema bytes.
	Definition metadata.Definition
}

// Create inserts the entity type and its version 1 and returns the stored
// version. It returns ErrKeyExists or ErrPrefixTaken; the key is checked
// first, so a request that clashes on both reports the key. The caller's
// transaction is unusable after an error.
func (s Store) Create(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in NewInput) (*Version, error) {
	d := in.Definition
	var one int
	switch err := tx.QueryRow(ctx, `SELECT 1 FROM entity_definitions WHERE key = $1`, d.Key).Scan(&one); {
	case err == nil:
		return nil, ErrKeyExists
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("entitytypes: check key: %w", err)
	}
	switch err := tx.QueryRow(ctx, `SELECT 1 FROM entity_definitions WHERE number_prefix = $1`, d.NumberPrefix).Scan(&one); {
	case err == nil:
		return nil, ErrPrefixTaken
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("entitytypes: check prefix: %w", err)
	}
	fixed, err := json.Marshal(d.FixedColumns)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode fixed columns: %w", err)
	}
	states, err := json.Marshal(d.States)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode states: %w", err)
	}
	sum := sha256.Sum256(d.Schema)
	defID, verID := uuid.New(), uuid.New()
	// A race with a concurrent create surfaces as a unique violation.
	if _, err := tx.Exec(ctx, `
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
VALUES ($1, $2, $3, $4, $5, 1)`, tenantID, defID, d.Key, in.Name, d.NumberPrefix); err != nil {
		return nil, mapUnique(err)
	}
	// The schema goes in as text cast to json, so the stored bytes are the
	// request bytes; the table CHECK ties schema_sha256 to them.
	if _, err := tx.Exec(ctx, `
INSERT INTO entity_definition_versions
  (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
VALUES ($1, $2, $3, 1, $4::json, $5, $6::jsonb, $7::jsonb, 'initial', $8)`,
		tenantID, verID, defID, string(d.Schema), sum[:], string(fixed), string(states), in.PublishedBy); err != nil {
		return nil, fmt.Errorf("entitytypes: insert version: %w", err)
	}
	return s.GetVersion(ctx, tx, d.Key, 1)
}

func mapUnique(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		switch pe.ConstraintName {
		case keyConstraint:
			return ErrKeyExists
		case prefixConstraint:
			return ErrPrefixTaken
		}
	}
	return fmt.Errorf("entitytypes: insert entity type: %w", err)
}

// Version implements DefinitionReader.
func (s Store) Version(ctx context.Context, tx pgx.Tx, entityKey string, version int) (*metadata.Compiled, error) {
	v, err := s.GetVersion(ctx, tx, entityKey, version)
	if err != nil {
		return nil, err
	}
	return compile(v)
}

// Current implements DefinitionReader.
func (s Store) Current(ctx context.Context, tx pgx.Tx, entityKey string) (*metadata.Compiled, int, error) {
	v, err := s.GetCurrent(ctx, tx, entityKey)
	if err != nil {
		return nil, 0, err
	}
	c, err := compile(v)
	if err != nil {
		return nil, 0, err
	}
	return c, v.Version, nil
}

// compile compiles a stored version. A stored row that no longer compiles is
// a defect of the store, not of the caller, and is not ErrNotFound.
func compile(v *Version) (*metadata.Compiled, error) {
	c, err := metadata.Compile(metadata.Definition{
		Key: v.Key, Schema: v.Schema, NumberPrefix: v.NumberPrefix, FixedColumns: v.FixedColumns, States: v.States,
	})
	if err != nil {
		return nil, fmt.Errorf("entitytypes: stored %s v%d does not compile: %w", v.Key, v.Version, err)
	}
	return c, nil
}
