// Package read is the audit log's reader (ADR-004 R7, R8; PLAN-005 C3, P5-17):
// the tenant's one timeline, filtered and paged, and one record's history,
// both with every diff masked for the reader.
//
// The caller gates the read (authz.RequireAuditPermission for the timeline,
// the record ACL for a history) in its own nava_app transaction. The Reader
// then reads on its own Pool, logged in as nava_audit_reader, the one runtime
// role that may read audit_log.diff and diff_salt (ADR-004-A1): each read is
// one tenancy.InTenantTx with the masker in the same transaction. Row level
// security scopes every query to that tenant, and each query names the
// tenant as well so the planner uses the tenant-led indexes.
package read

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Entry is one audit_log entry as a reader sees it.
type Entry struct {
	ID                 uuid.UUID
	OccurredAt         time.Time
	Domain             string
	Action             string
	SubjectType        string
	SubjectID          uuid.UUID
	EntityDefinitionID *uuid.UUID
	VersionBefore      *int64
	VersionAfter       *int64
	ActorPrincipalID   *uuid.UUID
	ActorSystem        *string
	Source             string
	SourceRef          *string
	ChangeSetID        *uuid.UUID
	CausationID        *uuid.UUID
	CausationDepth     int16
	RequestID          *string
	Backfilled         bool
	ContentSHA256      []byte
	DiffSHA256         []byte
	DiffTruncated      bool
	Seq                *int64
	SealedAt           *time.Time
	ErasedAt           *time.Time

	// Diff is the entry's diff as this reader may see it: nil when the entry
	// has none (a config entry, or an erased one), the stored bytes when
	// nothing in it is masked, otherwise the masked form.
	Diff json.RawMessage
	// DiffRedacted is true when Diff differs from what is stored (R7). It is
	// computed per reader and never stored.
	DiffRedacted bool
	// DiffSalt is set only with an unredacted diff (R7, CI-A14): with the
	// stored bytes and the salt a reader can recompute DiffSHA256.
	DiffSalt []byte
}

// Position is the (occurred_at, id) of the last entry of a page. Entries are
// read newest first, so the next page holds the entries before it.
type Position struct {
	At time.Time
	ID uuid.UUID
}

// Filter narrows the timeline. A nil field does not filter.
type Filter struct {
	// From is inclusive, To exclusive, on occurred_at.
	From, To    *time.Time
	Domain      *string
	SubjectType *string
	SubjectID   *uuid.UUID
	// AfterSeq keeps the sealed entries with a seq above it.
	AfterSeq *int64
}

// DefaultMaxRecordEntries is the most audit entries RecordEntries replays for
// one record when Reader.MaxRecordEntries is not set. A history page replays
// the record's whole history (see RecordEntries), so the cap bounds what one
// page reads and replays (P5-17-fu2, TICK-542). See docs/budgets.md.
const DefaultMaxRecordEntries = 500

// ErrHistoryTooLong is returned by RecordEntries when the record has more
// entries than the cap. No entry is read or replayed in that case.
var ErrHistoryTooLong = errors.New("audit/read: record history is longer than the cap")

// HistoryTooLongError carries the numbers of an ErrHistoryTooLong for the
// caller's log and metric. It never carries record data.
type HistoryTooLongError struct {
	Entries, Max int64
}

func (e *HistoryTooLongError) Error() string {
	return fmt.Sprintf("%v: %d entries, cap %d", ErrHistoryTooLong, e.Entries, e.Max)
}

func (e *HistoryTooLongError) Unwrap() error { return ErrHistoryTooLong }

// Reader reads the log on Pool through a Masker.
type Reader struct {
	// Pool is the nava_audit_reader pool (OpenPool). A Reader without one
	// fails every read with ErrNoPool.
	Pool   *Pool
	Masker Masker
	// MaxRecordEntries caps the entries RecordEntries reads for one record;
	// zero means DefaultMaxRecordEntries.
	MaxRecordEntries int
	// OnRecordRows, when not nil, is called with the number of audit rows each
	// RecordEntries read returned. It is a test hook that counts rows read.
	OnRecordRows func(n int)
}

func (r *Reader) maxRecordEntries() int {
	if r.MaxRecordEntries > 0 {
		return r.MaxRecordEntries
	}
	return DefaultMaxRecordEntries
}

const columns = `id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated,
seq, sealed_at, erased_at, diff::text, diff_salt`

// List returns up to limit of tenantID's entries matching f, newest first
// (occurred_at, then id, both descending), older than after when after is not
// nil. Pages follow the primary key's order, so they neither overlap nor skip
// an entry that existed when the first page was read. It runs in its own
// transaction on the reader's pool, with the tenant set transaction-locally.
func (r *Reader) List(ctx context.Context, tenantID uuid.UUID, f Filter, after *Position, limit int) ([]Entry, error) {
	var out []Entry
	err := r.inTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		out, err = r.list(ctx, tx, tenantID, f, after, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RecordEntries returns every record-domain entry of one record that changed
// it (create, update, state_change), oldest first, masked for the reader. It
// reads the record's whole history, because a version's payload is the
// replay of the diffs before it (see Replay). That makes a page cost the whole
// history, so the history is capped: the entries are counted first, and a
// record with more than the cap gets *HistoryTooLongError (Is
// ErrHistoryTooLong) with nothing read. The caller has already checked, in
// its own nava_app transaction, that the record exists and that the reader
// may see it (ADR-004-A1 A1-7). It runs in its own transaction on the
// reader's pool.
func (r *Reader) RecordEntries(ctx context.Context, tenantID, recordID uuid.UUID) ([]Entry, error) {
	var out []Entry
	err := r.inTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		out, err = r.recordEntries(ctx, tx, tenantID, recordID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Reader) inTx(ctx context.Context, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	if r.Pool == nil || r.Pool.pool == nil {
		return ErrNoPool
	}
	return tenancy.InTenantTx(ctx, r.Pool.pool, tenantID, fn)
}

func (r *Reader) list(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, f Filter, after *Position, limit int) ([]Entry, error) {
	var (
		afterAt *time.Time
		afterID *uuid.UUID
	)
	if after != nil {
		afterAt, afterID = &after.At, &after.ID
	}
	rows, err := tx.Query(ctx, `SELECT `+columns+`
FROM audit_log
WHERE tenant_id = $1
  AND ($2::timestamptz IS NULL OR occurred_at >= $2)
  AND ($3::timestamptz IS NULL OR occurred_at < $3)
  AND ($4::text IS NULL OR domain = $4)
  AND ($5::text IS NULL OR subject_type = $5)
  AND ($6::uuid IS NULL OR subject_id = $6)
  AND ($7::bigint IS NULL OR seq > $7)
  AND ($8::uuid IS NULL OR (occurred_at, id) < ($9::timestamptz, $8::uuid))
ORDER BY occurred_at DESC, id DESC
LIMIT $10`, tenantID, f.From, f.To, f.Domain, f.SubjectType, f.SubjectID, f.AfterSeq, afterID, afterAt, limit)
	if err != nil {
		return nil, fmt.Errorf("audit/read: list: %w", err)
	}
	entries, err := scan(rows)
	if err != nil {
		return nil, err
	}
	if err := r.mask(ctx, tx, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *Reader) recordEntries(ctx context.Context, tx pgx.Tx, tenantID, recordID uuid.UUID) ([]Entry, error) {
	limit := r.maxRecordEntries()
	var n int64
	if err := tx.QueryRow(ctx, `SELECT count(*)
FROM audit_log
WHERE tenant_id = $1 AND domain = 'record' AND subject_type = 'record' AND subject_id = $2
  AND action IN ('create', 'update', 'state_change')`, tenantID, recordID).Scan(&n); err != nil {
		return nil, fmt.Errorf("audit/read: count record entries: %w", err)
	}
	if n > int64(limit) {
		return nil, &HistoryTooLongError{Entries: n, Max: int64(limit)}
	}
	// The read is bounded as well: if entries arrive between the count and
	// the read, it stops one row past the cap and is refused, never replayed
	// from a cut history.
	rows, err := tx.Query(ctx, `SELECT `+columns+`
FROM audit_log
WHERE tenant_id = $1 AND domain = 'record' AND subject_type = 'record' AND subject_id = $2
  AND action IN ('create', 'update', 'state_change')
ORDER BY occurred_at, id
LIMIT $3`, tenantID, recordID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("audit/read: record entries: %w", err)
	}
	entries, err := scan(rows)
	if err != nil {
		return nil, err
	}
	if r.OnRecordRows != nil {
		r.OnRecordRows(len(entries))
	}
	if len(entries) > limit {
		return nil, &HistoryTooLongError{Entries: int64(len(entries)), Max: int64(limit)}
	}
	if err := r.mask(ctx, tx, entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func scan(rows pgx.Rows) ([]Entry, error) {
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var (
			e    Entry
			diff *string
		)
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Domain, &e.Action, &e.SubjectType, &e.SubjectID, &e.EntityDefinitionID,
			&e.VersionBefore, &e.VersionAfter, &e.ActorPrincipalID, &e.ActorSystem, &e.Source, &e.SourceRef, &e.ChangeSetID,
			&e.CausationID, &e.CausationDepth, &e.RequestID, &e.Backfilled, &e.ContentSHA256, &e.DiffSHA256, &e.DiffTruncated,
			&e.Seq, &e.SealedAt, &e.ErasedAt, &diff, &e.DiffSalt); err != nil {
			return nil, fmt.Errorf("audit/read: scan: %w", err)
		}
		if diff != nil {
			e.Diff = json.RawMessage(*diff)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit/read: read: %w", err)
	}
	return out, nil
}

// mask applies the masker to every entry's diff (R7). The masked pointers are
// asked once per (domain, subject type, definition) of the page. A masker
// error fails the read: an entry is never returned unmasked on error.
func (r *Reader) mask(ctx context.Context, tx pgx.Tx, entries []Entry) error {
	type key struct {
		domain, subject string
		hasDef          bool
		def             uuid.UUID
	}
	cache := map[key][]patchwalk.Pointer{}
	for i := range entries {
		e := &entries[i]
		if e.Diff == nil {
			e.DiffSalt = nil
			continue
		}
		k := key{domain: e.Domain, subject: e.SubjectType}
		if e.EntityDefinitionID != nil {
			k.hasDef, k.def = true, *e.EntityDefinitionID
		}
		paths, ok := cache[k]
		if !ok {
			var err error
			paths, err = r.Masker.MaskedPaths(ctx, tx, Subject{Domain: e.Domain, SubjectType: e.SubjectType, EntityDefinitionID: e.EntityDefinitionID})
			if err != nil {
				return fmt.Errorf("audit/read: masker: %w", err)
			}
			cache[k] = paths
		}
		out, redacted, err := MaskDiff(e.Diff, paths)
		if err != nil {
			return fmt.Errorf("audit/read: mask: %w", err)
		}
		e.Diff, e.DiffRedacted = out, redacted
		if redacted {
			e.DiffSalt = nil // R7: the salt goes only with an unredacted diff
		}
	}
	return nil
}
