// Package entitytypes is the definition store: it reads and creates entity
// types and their published, immutable versions (PLAN-003 D-3-2, D-3-4;
// TICK-343). It owns DefinitionReader, which the record write pipeline
// (P3-10) uses, and the writes the API handlers use. P3-05b (TICK-365)
// adds Publish, which writes version N+1.
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
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var (
	// ErrNotFound means no such entity type or version in the tenant.
	ErrNotFound = errors.New("entitytypes: not found")
	// ErrKeyExists means the tenant already has an entity type with that key.
	ErrKeyExists = errors.New("entitytypes: key exists")
	// ErrPrefixTaken means another entity type of the tenant has that prefix.
	ErrPrefixTaken = errors.New("entitytypes: number prefix taken")
	// ErrNoPublisher means a publish named no principal. published_by is
	// nullable in the table for onboarding's seed (P3-06), so this is the guard.
	// SetNumberingMode returns it too: its config entry needs an actor.
	ErrNoPublisher = errors.New("entitytypes: publish needs a principal")
	// ErrInvalidNumberingMode means a numbering mode other than gaps or
	// gapless (entity_definitions_numbering_mode_check).
	ErrInvalidNumberingMode = errors.New("entitytypes: numbering mode must be gaps or gapless")
)

// BreakingError is returned by Publish when the new definition is a breaking
// change of the current version (D-3-5). Paths are JSON pointers into the
// definition (see metadata.Classify).
type BreakingError struct{ Paths []string }

func (e *BreakingError) Error() string {
	return "entitytypes: breaking change at " + strings.Join(e.Paths, ", ")
}

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
	NumberingMode  metadata.NumberingMode
	CurrentVersion int
	UpdatedAt      time.Time
}

// Version is one published version, with the schema as the stored bytes.
type Version struct {
	Key          string
	Name         string
	NumberPrefix string
	// NumberingMode is the type's, not the version's (D-3-9): every version
	// read reads the type's current mode.
	NumberingMode metadata.NumberingMode
	Version       int
	// Schema is the exact bytes that were published.
	Schema       json.RawMessage
	SchemaSHA256 [sha256.Size]byte
	FixedColumns map[string]string
	States       metadata.StateModel
	Compat       string
	Current      bool
	PublishedAt  time.Time
	// NumberFormat is the pinned number format, nil for the default numbering.
	NumberFormat *FormatRef
}

const versionSelect = `
SELECT d.key, d.name, d.number_prefix, d.numbering_mode, d.current_version, v.version, v.schema::text,
       v.schema_sha256, v.fixed_columns, v.states, v.compat, v.published_at,
       v.number_format_id, v.number_format_version
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id`

func scanVersion(row pgx.Row) (*Version, error) {
	var (
		v      Version
		schema string
		sum    []byte
		cur    int
		fmtID  *uuid.UUID
		fmtVer *int
	)
	err := row.Scan(&v.Key, &v.Name, &v.NumberPrefix, &v.NumberingMode, &cur, &v.Version, &schema, &sum, &v.FixedColumns, &v.States, &v.Compat, &v.PublishedAt, &fmtID, &fmtVer)
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
	if fmtID != nil && fmtVer != nil {
		v.NumberFormat = &FormatRef{ID: *fmtID, Version: *fmtVer}
	}
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
SELECT key, name, number_prefix, numbering_mode, current_version, updated_at
FROM entity_definitions WHERE key > $1 ORDER BY key LIMIT $2`, after, limit)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: list: %w", err)
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.Key, &s.Name, &s.NumberPrefix, &s.NumberingMode, &s.CurrentVersion, &s.UpdatedAt); err != nil {
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
	// Definition carries Key, NumberPrefix, NumberingMode (empty is gaps)
	// and the exact Schema bytes.
	Definition metadata.Definition
}

// Create inserts the entity type and its version 1 and returns the stored
// version. It returns ErrKeyExists or ErrPrefixTaken; the key is checked
// first, so a request that clashes on both reports the key. The caller's
// transaction is unusable after an error.
func (s Store) Create(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in NewInput) (*Version, error) {
	defer tenancy.Forget(tx, currentKey{in.Definition.Key})
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
	fixed, err := encodeFixedColumns(d.FixedColumns)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode fixed columns: %w", err)
	}
	states, err := marshalJSON(d.States)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode states: %w", err)
	}
	if err := CheckSchemaSecrets(d.Schema); err != nil {
		return nil, err
	}
	if err := CheckSchemaSecretRequired(d.Schema); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(d.Schema)
	defID, verID := uuid.New(), uuid.New()
	// A race with a concurrent create surfaces as a unique violation.
	if _, err := tx.Exec(ctx, `
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, numbering_mode, current_version)
VALUES ($1, $2, $3, $4, $5, $6, 1)`, tenantID, defID, d.Key, in.Name, d.NumberPrefix, numberingMode(d.NumberingMode)); err != nil {
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
	if err := auditVersion(ctx, tx, defID, 1, sum, in.PublishedBy); err != nil {
		return nil, err
	}
	return s.GetVersion(ctx, tx, d.Key, 1)
}

// SeedInput is one entity type onboarding seeds at version 1. Both ids are
// supplied (UUIDv5 over the tenant and key), so a retry names the same rows.
type SeedInput struct {
	DefinitionID uuid.UUID
	VersionID    uuid.UUID
	Name         string
	// Definition is compiled by the caller (metadata.Compile) before Seed.
	Definition metadata.Definition
}

// Seed inserts the entity type and its version 1 with the ids in in, each as
// INSERT ... ON CONFLICT DO NOTHING, and reports whether it inserted anything.
// It needs INSERT only (nava_onboard has no SELECT on either table, so it reads
// nothing back) and writes published_by NULL: the tenant is still provisioning
// and has no principal. A retry after a crash finishes what is missing and
// never duplicates; on a second run it inserts nothing. When it inserts, it
// also writes the version's config create entry (auditSeed), which the
// commit check on both tables requires from migration B (P5-32) on. The
// caller's transaction must run as the tenant (tenancy.InTenantTx).
func (Store) Seed(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in SeedInput) (bool, error) {
	defer tenancy.Forget(tx, currentKey{in.Definition.Key})
	d := in.Definition
	fixed, err := encodeFixedColumns(d.FixedColumns)
	if err != nil {
		return false, fmt.Errorf("entitytypes: encode fixed columns: %w", err)
	}
	states, err := marshalJSON(d.States)
	if err != nil {
		return false, fmt.Errorf("entitytypes: encode states: %w", err)
	}
	sum := sha256.Sum256(d.Schema)
	def, err := tx.Exec(ctx, `
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, numbering_mode, current_version)
VALUES ($1, $2, $3, $4, $5, $6, 1)
ON CONFLICT DO NOTHING`, tenantID, in.DefinitionID, d.Key, in.Name, d.NumberPrefix, numberingMode(d.NumberingMode))
	if err != nil {
		return false, fmt.Errorf("entitytypes: seed definition: %w", err)
	}
	ver, err := tx.Exec(ctx, `
INSERT INTO entity_definition_versions
  (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
VALUES ($1, $2, $3, 1, $4::json, $5, $6::jsonb, $7::jsonb, 'initial', NULL)
ON CONFLICT DO NOTHING`, tenantID, in.VersionID, in.DefinitionID, string(d.Schema), sum[:], string(fixed), string(states))
	if err != nil {
		return false, fmt.Errorf("entitytypes: seed version: %w", err)
	}
	if def.RowsAffected()+ver.RowsAffected() == 0 {
		return false, nil
	}
	if err := auditSeed(ctx, tx, in.DefinitionID, sum); err != nil {
		return false, err
	}
	return true, nil
}

// PublishInput is the next version of an existing entity type. Definition
// carries Schema, FixedColumns and States; its Key, NumberPrefix and
// NumberingMode are ignored, since the stored type owns them.
type PublishInput struct {
	Key         string
	PublishedBy uuid.UUID
	Definition  metadata.Definition
	// NumberFormat changes the pinned number format; its zero value keeps the
	// previous version's (ADR-005 §3.2).
	NumberFormat NumberFormatChange
	// ExpectedVersion, when not empty, holds the versions the caller based the
	// publish on (its If-Match); the type's current version must be one of
	// them, else *ConflictError.
	ExpectedVersion []int
}

// Publish writes version N+1 of the entity type and makes it current, and
// returns it. It runs in the caller's transaction and locks the definition
// row, so concurrent publishes queue and number 2..N with no gap or
// duplicate. It returns ErrNoPublisher, ErrNotFound, *metadata.ErrInvalidDefinition
// when the definition does not compile, and *BreakingError when the change is
// Breaking; nothing is written then. The caller's transaction is unusable
// after an error.
func (s Store) Publish(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in PublishInput) (*Version, error) {
	defer tenancy.Forget(tx, currentKey{in.Key})
	if in.PublishedBy == uuid.Nil {
		return nil, ErrNoPublisher
	}
	var (
		defID   uuid.UUID
		prefix  string
		mode    metadata.NumberingMode
		current int
	)
	err := tx.QueryRow(ctx, `
SELECT id, number_prefix, numbering_mode, current_version FROM entity_definitions WHERE key = $1 FOR UPDATE`,
		in.Key).Scan(&defID, &prefix, &mode, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("entitytypes: lock definition: %w", err)
	}
	if len(in.ExpectedVersion) > 0 && !slices.Contains(in.ExpectedVersion, current) {
		return nil, &ConflictError{Current: current}
	}
	prevVersion, err := s.GetVersion(ctx, tx, in.Key, current)
	if err != nil {
		return nil, err
	}
	prev, err := compile(prevVersion)
	if err != nil {
		return nil, err
	}
	d := in.Definition
	d.Key, d.NumberPrefix, d.NumberingMode = in.Key, prefix, mode
	if err := CheckSchemaSecrets(d.Schema); err != nil {
		return nil, err
	}
	if err := CheckSchemaSecretRequired(d.Schema); err != nil {
		return nil, err
	}
	next, err := metadata.Compile(d)
	if err != nil {
		return nil, err
	}
	class, paths, err := metadata.Classify(prev, next)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: classify: %w", err)
	}
	if class == metadata.Breaking {
		return nil, &BreakingError{Paths: paths}
	}
	fixed, err := encodeFixedColumns(d.FixedColumns)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode fixed columns: %w", err)
	}
	states, err := marshalJSON(d.States)
	if err != nil {
		return nil, fmt.Errorf("entitytypes: encode states: %w", err)
	}
	// The next number follows the highest version, which the row lock makes
	// stable until this transaction ends.
	var number int
	if err := tx.QueryRow(ctx, `
SELECT coalesce(max(version), 0) + 1 FROM entity_definition_versions WHERE definition_id = $1`, defID).Scan(&number); err != nil {
		return nil, fmt.Errorf("entitytypes: next version: %w", err)
	}
	// The epoch is taken after the row lock above, in this transaction (C10).
	lk, err := resolveLink(ctx, tx, tenantID, defID, prefix, current, in.NumberFormat)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(d.Schema)
	if _, err := tx.Exec(ctx, `
INSERT INTO entity_definition_versions
  (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by,
   number_format_id, number_format_version, number_compiled)
VALUES ($1, $2, $3, $4, $5::json, $6, $7::jsonb, $8::jsonb, $9, $10, $11, $12, $13::jsonb)`,
		tenantID, uuid.New(), defID, number, string(d.Schema), sum[:], string(fixed), string(states), class.String(), in.PublishedBy,
		lk.id, lk.version, lk.compiled); err != nil {
		return nil, mapFormatFK(fmt.Errorf("entitytypes: insert version: %w", err))
	}
	if _, err := tx.Exec(ctx, `
UPDATE entity_definitions SET current_version = $1, updated_at = now() WHERE id = $2`, number, defID); err != nil {
		return nil, fmt.Errorf("entitytypes: set current version: %w", err)
	}
	if err := auditVersion(ctx, tx, defID, number, sum, in.PublishedBy); err != nil {
		return nil, err
	}
	return s.GetVersion(ctx, tx, in.Key, number)
}

// SetNumberingModeInput changes an entity type's numbering mode (D-3-9).
type SetNumberingModeInput struct {
	Key  string
	Mode metadata.NumberingMode
	// By is the principal making the change; its config entry names them.
	By uuid.UUID
}

// SetNumberingMode sets the entity type's numbering mode (PLAN-003 D-3-9,
// P3-36g) and reports whether it changed. The type's counter row carries on
// either way, so no record is renumbered. A mode equal to the stored one
// writes nothing. A change writes the config update entry the commit check
// requires (audited-tables.yaml lists numbering_mode), with the mode before
// and after as its diff. It runs in the caller's InTenantTx, so a key of
// another tenant is ErrNotFound, and locks the definition row, so it queues
// behind a concurrent Publish or SetNumberingMode of the same type. It
// returns ErrNoPublisher, ErrInvalidNumberingMode or ErrNotFound with nothing
// written. The caller's transaction is unusable after any other error.
func (Store) SetNumberingMode(ctx context.Context, tx pgx.Tx, in SetNumberingModeInput) (bool, error) {
	defer tenancy.Forget(tx, currentKey{in.Key})
	if in.By == uuid.Nil {
		return false, ErrNoPublisher
	}
	if !in.Mode.Valid() {
		return false, ErrInvalidNumberingMode
	}
	var (
		defID uuid.UUID
		prev  metadata.NumberingMode
	)
	err := tx.QueryRow(ctx, `
SELECT id, numbering_mode FROM entity_definitions WHERE key = $1 FOR UPDATE`, in.Key).Scan(&defID, &prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("entitytypes: lock definition: %w", err)
	}
	if prev == in.Mode {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
UPDATE entity_definitions SET numbering_mode = $1, updated_at = now() WHERE id = $2`, string(in.Mode), defID); err != nil {
		return false, fmt.Errorf("entitytypes: set numbering mode: %w", err)
	}
	if err := auditNumberingMode(ctx, tx, defID, prev, in.Mode, in.By); err != nil {
		return false, err
	}
	return true, nil
}

// numberingMode is the mode a new type is stored with: m, or gaps when the
// definition leaves it empty. An invalid m is refused by
// entity_definitions_numbering_mode_check.
func numberingMode(m metadata.NumberingMode) string {
	if m == "" {
		return string(metadata.NumberingGaps)
	}
	return string(m)
}

// marshalJSON is json.Marshal; tests replace it to reach the encode-failure
// branches, which a map of strings and a states slice cannot trigger.
var marshalJSON = json.Marshal //nolint:gochecknoglobals // test seam, replaced only by tests

// encodeFixedColumns encodes a definition's fixed columns as a JSON object.
// A nil map (the request omitted fixed_columns, or sent null) is {}, never
// null: entity_definition_versions_fixed_columns_check accepts only an
// object (TICK-378).
func encodeFixedColumns(m map[string]string) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return marshalJSON(m)
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

// Current implements DefinitionReader. A definition QueueCurrent prefetched
// in tx is returned without a round trip.
func (s Store) Current(ctx context.Context, tx pgx.Tx, entityKey string) (*metadata.Compiled, int, error) {
	if v, ok := tenancy.Recall(tx, currentKey{entityKey}); ok {
		c := v.(current) //nolint:forcetypeassert // only QueueCurrent stores under currentKey
		return c.def, c.version, c.err
	}
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

// currentKey is the tx memo key of an entity type's prefetched current
// definition; current is what is stored under it.
type currentKey struct{ key string }

type current struct {
	def     *metadata.Compiled
	version int
	err     error // what Current returns: ErrNotFound, or a stored row that does not scan or compile
}

// QueueCurrent queues Current's read on b, a batch the caller sends in tx
// (TICK-648). When the batch runs, the compiled definition, or ErrNotFound,
// goes in tx's memo (tenancy.Remember), and every Current of entityKey in the
// rest of tx answers from it with no round trip. Create, Seed and Publish
// drop it, so a transaction that changes the type reads it again. A stored
// version that no longer scans or compiles is not a batch failure: Current
// returns that error, as it would have unbatched. A failed statement fails the
// batch.
func (Store) QueueCurrent(tx pgx.Tx, b *pgx.Batch, entityKey string) {
	b.Queue(versionSelect+` WHERE d.key = $1 AND v.version = d.current_version`, entityKey).
		QueryRow(func(row pgx.Row) error {
			v, err := scanVersion(row)
			var pe *pgconn.PgError
			if errors.As(err, &pe) {
				return err // the statement failed: so does the batch
			}
			if err != nil {
				// ErrNotFound, or a row that does not scan: Current returns it.
				tenancy.Remember(tx, currentKey{entityKey}, current{err: err})
				return nil
			}
			c, err := compile(v)
			tenancy.Remember(tx, currentKey{entityKey}, current{def: c, version: v.Version, err: err})
			return nil
		})
}

// compile compiles a stored version. A stored row that no longer compiles is
// a defect of the store, not of the caller, and is not ErrNotFound.
func compile(v *Version) (*metadata.Compiled, error) {
	c, err := metadata.Compile(metadata.Definition{
		Key: v.Key, Schema: v.Schema, NumberPrefix: v.NumberPrefix, NumberingMode: v.NumberingMode,
		FixedColumns: v.FixedColumns, States: v.States,
	})
	if err != nil {
		return nil, fmt.Errorf("entitytypes: stored %s v%d does not compile: %v", v.Key, v.Version, err) //nolint:errorlint // %v on purpose: must not match *metadata.ErrInvalidDefinition (a caller 422)
	}
	return c, nil
}
