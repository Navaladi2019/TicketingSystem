package audit

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// readSalt fills a diff_salt. It is crypto/rand; the golden-vector test
// replaces it to reproduce a vector's fixed salt.
//
//nolint:gochecknoglobals // test seam for the salt source, read-only in production
var readSalt = func(b []byte) { _, _ = rand.Read(b) } // crypto/rand.Read never returns an error (Go 1.24+)

// insertSQL names the R10 columns only. occurred_at, txid and backfilled take
// their defaults, which the R10 trigger asserts; the seal and erasure columns
// stay NULL. tenant_id is the transaction's own tenant, read by
// tenancy.TenantOf, and the canonical WITH CHECK policy refuses any other.
// The diff is bound as text and cast, so the json column stores the exact
// bytes in every pgx exec mode (a []byte would go as bytea text under
// QueryExecModeExec).
const insertSQL = `INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)`

// Append writes e as one audit_log entry inside tx, the caller's transaction
// (ADR-004 decision 1, PLAN-005 C2), and returns its id. The entry commits or
// rolls back with the caller's change.
//
// In order, and before any INSERT:
//   - e is validated; a wrong entry returns ErrInvalidEntry before tx is used;
//   - the full diff is walked and checked for secrets (R5, R6); a secret
//     returns *ErrSecretInDiff, even when it sits in an op the 64 KB cut
//     would drop;
//   - an audit/erase diff must list 1 to MaxErasedIDs ids and is never cut
//     (ADR-004-E1);
//   - past MaxDiffBytes the diff keeps whole ops only and diff_truncated is
//     set (ADR-004-E1);
//   - a 16-byte salt is drawn and diff_sha256 is computed over the stored
//     bytes (R13);
//   - the tenant is read from tx's app.tenant_id (I2); with none set,
//     ErrInvalidEntry, and nothing is inserted.
//
// A second entry for the same subject and version_after in one transaction
// returns ErrDuplicateEntry. Append never sets occurred_at, txid,
// backfilled, or the seal or erasure columns (R10).
func Append(ctx context.Context, tx pgx.Tx, e Entry) (uuid.UUID, error) {
	id, args, err := prepare(ctx, tx, e)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, insertSQL, args...); err != nil {
		return uuid.Nil, insertError(err)
	}
	return id, nil
}

// QueueAppend is Append for a pgx.Batch the caller sends in tx (TICK-648):
// every check Append makes before its INSERT runs here, now, and a failure
// returns before anything is queued; the INSERT is queued on b. When the
// batch runs, a failed INSERT fails it with the error Append would have
// returned (ErrDuplicateEntry joined to the database error, or a wrapped
// one). It returns the entry's id.
func QueueAppend(ctx context.Context, tx pgx.Tx, b *pgx.Batch, e Entry) (uuid.UUID, error) {
	id, args, err := prepare(ctx, tx, e)
	if err != nil {
		return uuid.Nil, err
	}
	qq := b.Queue(insertSQL, args...)
	qq.Fn = func(br pgx.BatchResults) error {
		if _, err := br.Exec(); err != nil {
			return insertError(err)
		}
		return nil
	}
	return id, nil
}

// prepare makes every check of Append before the INSERT, in Append's order,
// and returns the entry's id and the INSERT's arguments.
func prepare(ctx context.Context, tx pgx.Tx, e Entry) (uuid.UUID, []any, error) {
	if err := e.validate(); err != nil {
		return uuid.Nil, nil, err
	}
	stored, truncated, err := prepareDiff(e)
	if err != nil {
		return uuid.Nil, nil, err
	}
	var salt []byte
	var diffSHA []byte
	if stored != nil {
		var s [16]byte
		readSalt(s[:])
		h := chain.DiffSHA256(s, stored)
		salt, diffSHA = s[:], h[:]
	}
	if e.ID == uuid.Nil {
		if e.ID, err = uuid.NewV7(); err != nil {
			return uuid.Nil, nil, fmt.Errorf("audit: mint id: %w", err)
		}
	}

	tenant, err := tenancy.TenantOf(ctx, tx)
	if errors.Is(err, tenancy.ErrNoTenant) {
		return uuid.Nil, nil, invalid("tenant_id", "the transaction has no tenant (I2)")
	}
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("audit: %w", err)
	}

	var content []byte
	if e.ContentSHA256 != nil {
		content = e.ContentSHA256[:]
	}
	var diffText *string
	if stored != nil {
		s := string(stored)
		diffText = &s
	}
	return e.ID, []any{
		tenant, e.ID, string(e.Domain), string(e.Action), string(e.SubjectType), e.SubjectID, e.EntityDefinitionID,
		e.VersionBefore, e.VersionAfter, e.Actor.PrincipalID, e.Actor.System, string(e.Source), e.SourceRef,
		e.ChangeSetID, e.CausationID, e.CausationDepth, e.RequestID, content, diffSHA,
		truncated, diffText, salt,
	}, nil
}

// insertError maps a failed INSERT.
func insertError(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" && strings.HasSuffix(pe.ConstraintName, "_commit_key") {
		return errors.Join(ErrDuplicateEntry, err)
	}
	return fmt.Errorf("audit: insert entry: %w", err)
}

// prepareDiff checks e's diff and returns the bytes to store: nil for no
// diff, else the diff or its whole-op cut, and whether it was cut.
func prepareDiff(e Entry) ([]byte, bool, error) {
	if e.Action == ActionErase {
		if err := checkEraseDiff(e.Diff); err != nil {
			return nil, false, err
		}
	}
	if e.Diff == nil {
		return nil, false, nil
	}
	// The full diff, before any cut: a secret in a dropped op still fails.
	if err := secrets.Check(e.Diff, e.SecretPaths, secrets.Domain(e.Domain)); err != nil {
		if errors.Is(err, patchwalk.ErrMalformed) || errors.Is(err, secrets.ErrInvalidInput) {
			return nil, false, fmt.Errorf("%w: diff: %w", ErrInvalidEntry, err)
		}
		return nil, false, err
	}
	if e.Action == ActionErase && len(e.Diff) > MaxDiffBytes {
		return nil, false, invalid("diff", "an erase entry is never truncated (ADR-004-E1)")
	}
	return truncateDiff(e.Diff)
}
