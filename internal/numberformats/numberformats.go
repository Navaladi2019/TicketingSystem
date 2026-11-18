// Package numberformats is the number-format store: it creates a format,
// publishes its next immutable version and reads them back (ADR-005 §3.2,
// §4.2; TICK-672, P6-07). A publish stores the document as sent and the
// numbering.Compiled the compiler made of it, and writes one config audit
// entry in the same transaction (C12).
//
// Every function takes the caller's transaction, opened by
// tenancy.InTenantTx, so row-level security scopes each query to the tenant
// of that transaction. A format of another tenant is not visible there and
// reads as ErrNotFound.
package numberformats

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/jsondiff"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

var (
	// ErrNotFound means no such format or version in the tenant.
	ErrNotFound = errors.New("numberformats: not found")
	// ErrKeyExists means the tenant already has a format with that key.
	ErrKeyExists = errors.New("numberformats: key exists")
	// ErrNoPublisher means a write named no principal: published_by is
	// never taken from the request.
	ErrNoPublisher = errors.New("numberformats: write needs a principal")
)

// keyConstraint is the unique constraint of number_formats (db/migrations).
const keyConstraint = "number_formats_key_key"

// Store reads and writes number formats. It holds no state.
type Store struct{}

// Version is one published version of a format.
type Version struct {
	ID  uuid.UUID
	Key string
	// Version is this version's number; Current is the format's newest.
	Version int
	Current int
	// Document is the format as published: the stored jsonb, so key order
	// and spacing are PostgreSQL's.
	Document       json.RawMessage
	Compiled       json.RawMessage
	CompiledSHA256 string // hex
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
}

// Summary is one row of the list.
type Summary struct {
	ID             uuid.UUID
	Key            string
	CurrentVersion int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewInput is the input of Create.
type NewInput struct {
	Key       string
	Document  json.RawMessage
	Compiled  numbering.Compiled
	CreatedBy uuid.UUID
}

// PublishInput is the input of Publish. ExpectedVersion is the version the
// caller based the publish on (its If-Match).
type PublishInput struct {
	ID              uuid.UUID
	Document        json.RawMessage
	Compiled        numbering.Compiled
	CreatedBy       uuid.UUID
	ExpectedVersion []int
}

// ConflictError means the format is at version Current, not at one the
// caller named. Nothing was written.
type ConflictError struct{ Current int }

func (e *ConflictError) Error() string {
	return fmt.Sprintf("numberformats: format is at version %d", e.Current)
}

// Create inserts the format head and its version 1, and audits it.
func (Store) Create(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in NewInput) (*Version, error) {
	if in.CreatedBy == uuid.Nil {
		return nil, ErrNoPublisher
	}
	id := uuid.New()
	_, err := tx.Exec(ctx, `
INSERT INTO number_formats (tenant_id, id, key, current_version) VALUES ($1, $2, $3, 1)`,
		tenantID, id, in.Key)
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == keyConstraint {
		return nil, ErrKeyExists
	}
	if err != nil {
		return nil, fmt.Errorf("numberformats: insert format: %w", err)
	}
	if err := insertVersion(ctx, tx, tenantID, id, 1, in.Document, in.Compiled, in.CreatedBy); err != nil {
		return nil, err
	}
	if err := auditVersion(ctx, tx, id, 1, in.Compiled, in.CreatedBy); err != nil {
		return nil, err
	}
	return Store{}.GetVersion(ctx, tx, id, 1)
}

// Publish writes version N+1 of the format and makes it current. It locks
// the head row first, so two publishes take consecutive numbers and a stale
// ExpectedVersion is a *ConflictError.
func (Store) Publish(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, in PublishInput) (*Version, error) {
	if in.CreatedBy == uuid.Nil {
		return nil, ErrNoPublisher
	}
	var cur int
	err := tx.QueryRow(ctx, `SELECT current_version FROM number_formats WHERE id = $1 FOR UPDATE`, in.ID).Scan(&cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("numberformats: lock format: %w", err)
	}
	ok := false
	for _, v := range in.ExpectedVersion {
		ok = ok || v == cur
	}
	if !ok {
		return nil, &ConflictError{Current: cur}
	}
	next := cur + 1
	if err := insertVersion(ctx, tx, tenantID, in.ID, next, in.Document, in.Compiled, in.CreatedBy); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE number_formats SET current_version = $2, updated_at = now() WHERE id = $1`, in.ID, next); err != nil {
		return nil, fmt.Errorf("numberformats: move current version: %w", err)
	}
	if err := auditVersion(ctx, tx, in.ID, next, in.Compiled, in.CreatedBy); err != nil {
		return nil, err
	}
	return Store{}.GetVersion(ctx, tx, in.ID, next)
}

// Current returns the format's current version of the format id, or ErrNotFound.
func (Store) Current(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*Version, error) {
	return readVersion(ctx, tx, id, 0)
}

// GetVersion returns the given version of the format, or ErrNotFound.
func (Store) GetVersion(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int) (*Version, error) {
	return readVersion(ctx, tx, id, version)
}

// CurrentVersion returns the format's current version number, or ErrNotFound.
func (Store) CurrentVersion(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int, error) {
	var cur int
	err := tx.QueryRow(ctx, `SELECT current_version FROM number_formats WHERE id = $1`, id).Scan(&cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("numberformats: read current version: %w", err)
	}
	return cur, nil
}

// List returns up to limit formats ordered by key, after the key `after`.
func (Store) List(ctx context.Context, tx pgx.Tx, after string, limit int) ([]Summary, error) {
	rows, err := tx.Query(ctx, `
SELECT id, key, current_version, created_at, updated_at FROM number_formats
WHERE key > $1 ORDER BY key LIMIT $2`, after, limit)
	if err != nil {
		return nil, fmt.Errorf("numberformats: list: %w", err)
	}
	defer rows.Close()
	var out []Summary
	for rows.Next() {
		var s Summary
		if err := rows.Scan(&s.ID, &s.Key, &s.CurrentVersion, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("numberformats: scan list: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("numberformats: list: %w", err)
	}
	return out, nil
}

func readVersion(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int) (*Version, error) {
	var (
		v   Version
		sum []byte
	)
	// version 0 reads the current one: the join on current_version.
	err := tx.QueryRow(ctx, `
SELECT f.id, f.key, v.version, f.current_version, v.document::text, v.compiled::text, v.compiled_sha256, v.created_at, v.created_by
FROM number_formats f
JOIN number_format_versions v ON v.tenant_id = f.tenant_id AND v.format_id = f.id
WHERE f.id = $1 AND v.version = CASE WHEN $2 = 0 THEN f.current_version ELSE $2 END`, id, version).
		Scan(&v.ID, &v.Key, &v.Version, &v.Current, &v.Document, &v.Compiled, &sum, &v.CreatedAt, &v.CreatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("numberformats: read version: %w", err)
	}
	v.CompiledSHA256 = hex.EncodeToString(sum)
	return &v, nil
}

func insertVersion(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, version int, doc json.RawMessage, c numbering.Compiled, by uuid.UUID) error {
	compiled, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("numberformats: encode compiled: %w", err)
	}
	sum, err := hex.DecodeString(c.CompiledSHA256)
	if err != nil || len(sum) != sha256.Size {
		return fmt.Errorf("numberformats: compiled hash %q is not a SHA-256", c.CompiledSHA256)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7)`, tenantID, id, version, string(doc), string(compiled), sum, by); err != nil {
		return fmt.Errorf("numberformats: insert version: %w", err)
	}
	return nil
}

// auditVersion writes the config entry of the version just inserted: create
// for version 1, publish after. actor is the resolved principal and source
// is user, both set here and never read from the request (C12).
func auditVersion(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int, c numbering.Compiled, by uuid.UUID) error {
	after, err := documentJSONB(ctx, tx, id, version)
	if err != nil {
		return err
	}
	var before json.RawMessage
	action := audit.ActionCreate
	var vBefore *int64
	if version > 1 {
		if before, err = documentJSONB(ctx, tx, id, version-1); err != nil {
			return err
		}
		action = audit.ActionPublish
		vb := int64(version - 1)
		vBefore = &vb
	}
	diff, err := jsondiff.Diff(before, after)
	if err != nil {
		return fmt.Errorf("numberformats: diff document: %w", err)
	}
	sum, err := hex.DecodeString(c.CompiledSHA256)
	if err != nil || len(sum) != sha256.Size {
		return fmt.Errorf("numberformats: compiled hash is not a SHA-256")
	}
	var hash [sha256.Size]byte
	copy(hash[:], sum)
	va := int64(version)
	e := audit.Entry{
		Domain:        audit.DomainConfig,
		Action:        action,
		SubjectType:   audit.SubjectNumberFormat,
		SubjectID:     id,
		VersionBefore: vBefore,
		VersionAfter:  &va,
		Actor:         audit.Actor{PrincipalID: &by},
		Source:        audit.SourceUser,
		ContentSHA256: &hash,
		Diff:          diff,
	}
	if rid := requestid.FromContext(ctx); rid != "" {
		e.RequestID = &rid
	}
	if _, err := audit.Append(ctx, tx, e); err != nil {
		return fmt.Errorf("numberformats: audit version: %w", err)
	}
	return nil
}

// documentJSONB reads a stored document back as jsonb text, so the audit
// diff never depends on the request's formatting.
func documentJSONB(ctx context.Context, tx pgx.Tx, id uuid.UUID, version int) (json.RawMessage, error) {
	var s string
	err := tx.QueryRow(ctx, `SELECT document::text FROM number_format_versions WHERE format_id = $1 AND version = $2`, id, version).Scan(&s)
	if err != nil {
		return nil, fmt.Errorf("numberformats: read document: %w", err)
	}
	return json.RawMessage(s), nil
}
