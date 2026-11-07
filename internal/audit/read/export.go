package read

// The export's page reads (ADR-004 R19-R21, PLAN-005 C4, P5-40). They live in
// this package because only it holds the nava_audit_reader pool, the one
// connection that may read audit_log.diff and diff_salt. Every method here is
// one short transaction on that pool with a statement_timeout, never a
// long-held snapshot: sealed rows are immutable (R21), so a fixed head_seq
// bound gives a consistent file.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ExportFilter is the export's filter. A nil field does not filter. From is
// inclusive and To exclusive, on occurred_at.
type ExportFilter struct {
	From, To    *time.Time
	Domain      *string
	SubjectType *string
	SubjectID   *uuid.UUID
	ChangeSetID *uuid.UUID
	// AfterSeq keeps the sealed entries with a seq above it.
	AfterSeq *int64
}

// ExportRow is one sealed or unsealed row as the export reads it.
type ExportRow struct {
	TenantID   uuid.UUID
	Entry      Entry
	PrevHash   []byte
	EntryHash  []byte
	ErasureRef *uuid.UUID
	// Match is true when the row passes the filter, or is an audit/erase
	// entry for an erased entry in scope (R24). A row that does not match is
	// a stub: its Entry holds the envelope columns and no diff.
	Match bool
}

// ExportPageOptions carries the per-page limits.
type ExportPageOptions struct {
	// StatementTimeout is applied with SET LOCAL to each page; zero leaves the
	// role's default.
	StatementTimeout time.Duration
	// OnPage, when not nil, is called inside each page's transaction with the
	// transaction's start time and the statement_timeout in force. It is a
	// test hook that proves each page is its own transaction.
	OnPage func(txStart time.Time, statementTimeout string)
}

// exportMatch is the filter predicate over parameters $4..$11: from, to,
// domain, subject_type, subject_id, change_set_id, after_seq, erase refs.
const exportMatch = `((($4::timestamptz IS NULL OR occurred_at >= $4)
   AND ($5::timestamptz IS NULL OR occurred_at < $5)
   AND ($6::text IS NULL OR domain = $6)
   AND ($7::text IS NULL OR subject_type = $7)
   AND ($8::uuid IS NULL OR subject_id = $8)
   AND ($9::uuid IS NULL OR change_set_id = $9)
   AND ($10::bigint IS NULL OR seq > $10))
 OR (domain = 'audit' AND action = 'erase' AND subject_type = 'erasure_request' AND subject_id = ANY($11::uuid[])))`

// exportInner names the columns the reader may read. Never SELECT *: the role
// has no grant on txid, and a star would fail with 42501.
const exportInner = `tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated,
seq, sealed_at, erased_at, erasure_ref, prev_hash, entry_hash, diff, diff_salt`

const exportColumns = `tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated,
seq, sealed_at, erased_at, erasure_ref, prev_hash, entry_hash,
m AS matched,
CASE WHEN m THEN diff::text END AS diff_text,
CASE WHEN m THEN diff_salt END AS salt`

type exportScan struct {
	row ExportRow
}

func (r *Reader) pageTx(ctx context.Context, tenantID uuid.UUID, o ExportPageOptions, fn func(pgx.Tx) error) error {
	return r.inTx(ctx, tenantID, func(tx pgx.Tx) error {
		if o.StatementTimeout > 0 {
			// SET LOCAL ends with the page's transaction. The value is a
			// whole number of milliseconds, never request input.
			if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", o.StatementTimeout.Milliseconds())); err != nil {
				return fmt.Errorf("audit/read: export statement_timeout: %w", err)
			}
		}
		if o.OnPage != nil {
			var start time.Time
			var st string
			if err := tx.QueryRow(ctx, "SELECT transaction_timestamp(), current_setting('statement_timeout')").Scan(&start, &st); err != nil {
				return fmt.Errorf("audit/read: export page probe: %w", err)
			}
			o.OnPage(start, st)
		}
		return fn(tx)
	})
}

// ExportFirstSeq returns the first sealed seq (at or below head) whose
// occurred_at is inside the time window of f, whatever the other filters say
// (R20, ADR-004 TA-B5). found is false when no sealed entry is in the window.
//
// The query is the window-bounded form with an OFFSET 0 fence (spike S1
// finding F2): the inner select is bounded by the window, so the planner
// cannot turn min(seq) into a walk of the whole (tenant_id, seq) index.
func (r *Reader) ExportFirstSeq(ctx context.Context, tenantID uuid.UUID, f ExportFilter, head int64, o ExportPageOptions) (first int64, found bool, err error) {
	err = r.pageTx(ctx, tenantID, o, func(tx pgx.Tx) error {
		var s *int64
		if err := tx.QueryRow(ctx, `SELECT min(w.seq) FROM (
  SELECT seq FROM audit_log
  WHERE tenant_id = $1
    AND ($2::timestamptz IS NULL OR occurred_at >= $2)
    AND ($3::timestamptz IS NULL OR occurred_at < $3)
    AND seq IS NOT NULL AND seq <= $4
  OFFSET 0) w`, tenantID, f.From, f.To, head).Scan(&s); err != nil {
			return fmt.Errorf("audit/read: export first seq: %w", err)
		}
		if s != nil {
			first, found = *s, true
		}
		return nil
	})
	return first, found, err
}

// ExportErasureRefs returns the erasure_ref of every erased entry that
// matches f, so the page reads can include the matching audit/erase entries
// whatever the filter (R24).
func (r *Reader) ExportErasureRefs(ctx context.Context, tenantID uuid.UUID, f ExportFilter, o ExportPageOptions) ([]uuid.UUID, error) {
	var out []uuid.UUID
	err := r.pageTx(ctx, tenantID, o, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT DISTINCT erasure_ref FROM audit_log
WHERE tenant_id = $1 AND erased_at IS NOT NULL AND erasure_ref IS NOT NULL
  AND ($2::timestamptz IS NULL OR occurred_at >= $2)
  AND ($3::timestamptz IS NULL OR occurred_at < $3)
  AND ($4::text IS NULL OR domain = $4)
  AND ($5::text IS NULL OR subject_type = $5)
  AND ($6::uuid IS NULL OR subject_id = $6)
  AND ($7::uuid IS NULL OR change_set_id = $7)
  AND ($8::bigint IS NULL OR seq > $8)`,
			tenantID, f.From, f.To, f.Domain, f.SubjectType, f.SubjectID, f.ChangeSetID, f.AfterSeq)
		if err != nil {
			return fmt.Errorf("audit/read: export erasure refs: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return fmt.Errorf("audit/read: export erasure refs: %w", err)
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}

// ExportSeq returns the sealed row at seq, unmasked and diff-free: the anchor.
func (r *Reader) ExportSeq(ctx context.Context, tenantID uuid.UUID, seq int64, o ExportPageOptions) (ExportRow, bool, error) {
	var out ExportRow
	var found bool
	err := r.pageTx(ctx, tenantID, o, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+exportColumns+` FROM (
  SELECT `+exportInner+`, false AS m FROM audit_log WHERE tenant_id = $1 AND seq = $2) t`, tenantID, seq)
		if err != nil {
			return fmt.Errorf("audit/read: export anchor: %w", err)
		}
		got, err := scanExport(rows)
		if err != nil {
			return err
		}
		if len(got) > 0 {
			out, found = got[0], true
		}
		return nil
	})
	return out, found, err
}

// ExportRange returns up to limit sealed rows with after < seq <= head, in
// seq order. With onlyMatches false a row that does not match is returned as
// a stub (no diff); with it true only matching rows are read (stubs=false).
// Matching diffs are masked for the reader in the same transaction (R7).
func (r *Reader) ExportRange(ctx context.Context, tenantID uuid.UUID, f ExportFilter, erased []uuid.UUID, after, head int64, limit int, onlyMatches bool, o ExportPageOptions) ([]ExportRow, error) {
	var out []ExportRow
	err := r.pageTx(ctx, tenantID, o, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+exportColumns+` FROM (
  SELECT `+exportInner+`, `+exportMatch+` AS m FROM audit_log
  WHERE tenant_id = $1 AND seq > $2 AND seq <= $3
) t
WHERE ($12::boolean IS FALSE OR m)
ORDER BY seq
LIMIT $13`, tenantID, after, head, f.From, f.To, f.Domain, f.SubjectType, f.SubjectID, f.ChangeSetID, f.AfterSeq, nonNil(erased), onlyMatches, limit)
		if err != nil {
			return fmt.Errorf("audit/read: export range: %w", err)
		}
		got, err := scanExport(rows)
		if err != nil {
			return err
		}
		if err := r.maskExport(ctx, tx, got); err != nil {
			return err
		}
		out = got
		return nil
	})
	return out, err
}

// ExportUnsealed returns up to limit matching entries that are not part of
// the sealed range: unsealed ones, and ones sealed after head was read. They
// are ordered by (occurred_at, id) and are at or before start; after is the
// last position of the previous page.
func (r *Reader) ExportUnsealed(ctx context.Context, tenantID uuid.UUID, f ExportFilter, erased []uuid.UUID, head int64, start time.Time, after *Position, limit int, o ExportPageOptions) ([]ExportRow, error) {
	var (
		afterAt *time.Time
		afterID *uuid.UUID
	)
	if after != nil {
		afterAt, afterID = &after.At, &after.ID
	}
	var out []ExportRow
	err := r.pageTx(ctx, tenantID, o, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+exportColumns+` FROM (
  SELECT `+exportInner+`, true AS m FROM audit_log
  WHERE tenant_id = $1 AND (seq IS NULL OR seq > $2) AND occurred_at <= $3
    AND `+exportMatchUnsealed+`
    AND ($12::uuid IS NULL OR (occurred_at, id) > ($13::timestamptz, $12::uuid))
) t
ORDER BY occurred_at, id
LIMIT $14`, tenantID, head, start, f.From, f.To, f.Domain, f.SubjectType, f.SubjectID, f.ChangeSetID, f.AfterSeq, nonNil(erased), afterID, afterAt, limit)
		if err != nil {
			return fmt.Errorf("audit/read: export unsealed: %w", err)
		}
		got, err := scanExport(rows)
		if err != nil {
			return err
		}
		if err := r.maskExport(ctx, tx, got); err != nil {
			return err
		}
		out = got
		return nil
	})
	return out, err
}

// exportMatchUnsealed is exportMatch with the same parameter numbers; kept as
// its own name so a reader of ExportUnsealed sees the predicate is the same.
const exportMatchUnsealed = exportMatch

// nonNil renders ids as strings: the reader pool runs unnamed statements in
// text format (PgBouncer transaction mode), where pgx cannot encode a
// []uuid.UUID. A nil list is an empty array, never NULL.
func nonNil(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func scanExport(rows pgx.Rows) ([]ExportRow, error) {
	defer rows.Close()
	var out []ExportRow
	for rows.Next() {
		var (
			x    exportScan
			e    = &x.row.Entry
			diff *string
			ten  uuid.UUID
		)
		if err := rows.Scan(&ten, &e.ID, &e.OccurredAt, &e.Domain, &e.Action, &e.SubjectType, &e.SubjectID, &e.EntityDefinitionID,
			&e.VersionBefore, &e.VersionAfter, &e.ActorPrincipalID, &e.ActorSystem, &e.Source, &e.SourceRef, &e.ChangeSetID,
			&e.CausationID, &e.CausationDepth, &e.RequestID, &e.Backfilled, &e.ContentSHA256, &e.DiffSHA256, &e.DiffTruncated,
			&e.Seq, &e.SealedAt, &e.ErasedAt, &x.row.ErasureRef, &x.row.PrevHash, &x.row.EntryHash,
			&x.row.Match, &diff, &e.DiffSalt); err != nil {
			return nil, fmt.Errorf("audit/read: export scan: %w", err)
		}
		x.row.TenantID = ten
		if diff != nil {
			e.Diff = json.RawMessage(*diff)
		}
		out = append(out, x.row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit/read: export read: %w", err)
	}
	return out, nil
}

// maskExport masks the matching rows' diffs for the reader (R7): a redacted
// diff comes with no salt (CI-A14). A masker error fails the page; an entry
// is never emitted unmasked on error.
func (r *Reader) maskExport(ctx context.Context, tx pgx.Tx, rows []ExportRow) error {
	entries := make([]Entry, len(rows))
	for i := range rows {
		entries[i] = rows[i].Entry
	}
	if err := r.mask(ctx, tx, entries); err != nil {
		return err
	}
	for i := range rows {
		rows[i].Entry = entries[i]
	}
	return nil
}
