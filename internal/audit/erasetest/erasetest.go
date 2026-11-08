// Package erasetest is the test-only erase helper of PLAN-005 D-5-10 (P5-42,
// TICK-447). It runs the ADR-004 R23 erase step exactly as the Epic 30
// erasure job will: logged in as nava_erasure, inside tenancy.InTenantTx, in
// one transaction that applies the R11 erase transition and appends the
// audit/erase entry naming the erased ids.
//
// It is not the erasure job. No production binary may link it: an
// architecture test fails when any cmd/* package depends on it.
//
// The helper never names the tenant in its SQL. Row-level security on
// audit_log is the only thing that keeps an erase to the transaction's own
// tenant, so the adversarial tests that point it at another tenant's ids
// prove the policy, not a predicate written here.
package erasetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Role is the only login Erase runs as (ADR-001-A6.1, ADR-004 R23).
const Role = "nava_erasure"

// ActorSystem is the system actor of the audit/erase entries Erase writes.
const ActorSystem = "erasure"

// eraseSQL is the R11 erase transition. It reads no diff and no salt (R23:
// nava_erasure's SELECT grant has neither) and has no RETURNING, so it reads
// nothing past the M1 columns. erased_at IS NULL keeps a second erase from
// reaching an erased row; the BEFORE UPDATE trigger refuses one that does.
const eraseSQL = `UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $1
WHERE id = ANY($2::uuid[]) AND erased_at IS NULL`

var (
	// ErrWrongRole means the pool is not logged in as nava_erasure.
	ErrWrongRole = errors.New("erasetest: the pool is not logged in as " + Role)
	// ErrBadRequest means no ids, more than audit.MaxErasedIDs, or a nil ref.
	ErrBadRequest = errors.New("erasetest: an erase names a ref and 1 to audit.MaxErasedIDs entry ids")
	// ErrNotErasable means the transaction could not erase every id it was given
	// (an id of another tenant, an unknown id, or one already erased). The
	// transaction is rolled back and no audit/erase entry is written.
	ErrNotErasable = errors.New("erasetest: not every entry id is an unerased entry of this tenant")
)

// Erase erases ids in tenant under ref, as R23 describes, and returns the id
// of the audit/erase entry. pool must be logged in as nava_erasure.
func Erase(ctx context.Context, pool *pgxpool.Pool, tenant, ref uuid.UUID, ids []uuid.UUID) (uuid.UUID, error) {
	var entry uuid.UUID
	err := tenancy.InTenantTx(ctx, pool, tenant, func(tx pgx.Tx) error {
		var who string
		if err := tx.QueryRow(ctx, "SELECT current_user").Scan(&who); err != nil {
			return fmt.Errorf("erasetest: current_user: %w", err)
		}
		if who != Role {
			return fmt.Errorf("%w (it is %s)", ErrWrongRole, who)
		}
		var err error
		entry, err = Step(ctx, tx, ref, ids)
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return entry, nil
}

// Step is the R23 step inside the caller's tenant transaction, whatever its
// role: the tests run it as nava_app and nava_worker to prove the database,
// not this package, refuses them. It erases ids, refuses with ErrNotErasable
// before writing anything else when not every id was erased, then appends
// the audit/erase entry through audit.Append. On any error the caller rolls
// tx back.
func Step(ctx context.Context, tx pgx.Tx, ref uuid.UUID, ids []uuid.UUID) (uuid.UUID, error) {
	list, err := normalise(ref, ids)
	if err != nil {
		return uuid.Nil, err
	}
	tag, err := tx.Exec(ctx, eraseSQL, ref, list)
	if err != nil {
		return uuid.Nil, fmt.Errorf("erasetest: erase transition: %w", err)
	}
	if tag.RowsAffected() != int64(len(list)) {
		return uuid.Nil, fmt.Errorf("%w: erased %d of %d", ErrNotErasable, tag.RowsAffected(), len(list))
	}
	diff, err := EraseDiff(list)
	if err != nil {
		return uuid.Nil, err
	}
	actor := ActorSystem
	id, err := audit.Append(ctx, tx, audit.Entry{
		Domain:      audit.DomainAudit,
		Action:      audit.ActionErase,
		SubjectType: audit.SubjectErasureRequest,
		SubjectID:   ref,
		Actor:       audit.Actor{System: &actor},
		Source:      audit.SourceSystem,
		Diff:        diff,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("erasetest: audit/erase entry: %w", err)
	}
	return id, nil
}

// EraseDiff is the audit/erase diff for ids (audit-export-v1.md section 4
// step 7): one add of /erased_ids listing them as lowercase UUIDs.
func EraseDiff(ids []uuid.UUID) ([]byte, error) {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	b, err := json.Marshal([]map[string]any{{"op": "add", "path": "/erased_ids", "value": s}})
	if err != nil {
		return nil, fmt.Errorf("erasetest: encode diff: %w", err)
	}
	return b, nil
}

// normalise checks the request and returns the ids sorted and without
// duplicates, so the row count can be compared with the list.
func normalise(ref uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if ref == uuid.Nil || len(ids) == 0 || len(ids) > audit.MaxErasedIDs {
		return nil, ErrBadRequest
	}
	list := slices.Clone(ids)
	slices.SortFunc(list, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	list = slices.Compact(list)
	if slices.Contains(list, uuid.Nil) {
		return nil, ErrBadRequest
	}
	return list, nil
}
