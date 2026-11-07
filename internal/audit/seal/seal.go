// Package seal is the audit sealer's two activities (ADR-004 R13 to R16,
// PLAN-005 P5-34a):
//
//   - A1, Reschedule, runs first in its own transaction. Under the tenant's
//     audit_chain_head lock it enqueues the next run's start row and repoints
//     the head at it, unless this run's lineage is stale.
//   - A2, Seal, seals the tenant's unsealed audit_log entries in batches of
//     BatchSize, in (occurred_at, id) order, one transaction per batch, each
//     under the head lock. seq comes only from the locked head (R15a), and
//     the head's seq and entry_hash advance in the same transaction.
//
// The workflow that calls them, audit.seal, is in workflow.go (P5-34b). The
// tenant is never an argument: every transaction is opened by
// workerguard.InTenantTx with the tenant the activity guard verified (I2),
// and RLS limits every statement to it. Both run as nava_worker, the only
// role with the R11 seal transition and the head's sealer columns (R14).
//
// Neither returns record data: A1 returns the next run's outbox id, A2 a
// count, and the Temporal activities in activities.go return less still
// (workerguard accepts references only).
package seal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// BatchSize is the most entries one A2 transaction seals (R16).
const BatchSize = 1000

// The R16 interval bounds, in seconds; audit_chain_head_seal_interval_check
// holds the column to them.
const (
	MinIntervalSeconds = 10
	MaxIntervalSeconds = 300
)

var (
	// ErrHeadNotFound means the tenant has no audit_chain_head row (it is
	// seeded by onboarding, A6.2). Nothing is written.
	ErrHeadNotFound = errors.New("seal: audit chain head not found")
	// ErrUnsealable means an entry of the batch cannot be canonicalised
	// (chain.ErrInvalidEnvelope), so the whole batch is rolled back and no
	// seq is used. The message names the column, never a value.
	ErrUnsealable = errors.New("seal: entry cannot be canonicalised")
	// ErrSealRace means an entry of the batch was no longer unsealed when it
	// was updated: something sealed it without the head lock. The batch is
	// rolled back.
	ErrSealRace = errors.New("seal: entry sealed outside the head lock")
)

// NextInterval is the R16 rule: 10 s after a run that sealed anything,
// otherwise the current interval doubled, up to 300 s.
func NextInterval(current int32, sealedAny bool) int32 {
	if sealedAny {
		return MinIntervalSeconds
	}
	next := min(int64(current)*2, MaxIntervalSeconds)
	return int32(max(next, MinIntervalSeconds))
}

const (
	// The head lock (R15, R16). The predicate names the tenant as RLS does,
	// so a role that bypasses RLS still locks only this tenant's head.
	lockForRescheduleSQL = `SELECT next_seal_outbox_id, seal_interval_s FROM audit_chain_head
WHERE tenant_id = $1 FOR UPDATE`
	bumpVersionSQL = `UPDATE audit_chain_head SET version = version + 1 WHERE tenant_id = $1 RETURNING version`
	// next_seal_at uses the same now() and interval as the start row's
	// available_at (outbox.WithDelay), so the two are equal.
	repointSQL = `UPDATE audit_chain_head
SET next_seal_outbox_id = $2, next_seal_at = now() + make_interval(secs => $3)
WHERE tenant_id = $1`
)

// Reschedule is A1 for run o, the outbox id of the running workflow's start
// row. Under the head lock, if next_seal_outbox_id is o it bumps the head's
// version, enqueues start row o′ at that version with available_at = now() +
// seal_interval_s (outbox.EnqueueSeal, outbox.WithDelay), points
// next_seal_outbox_id and next_seal_at at o′, commits, and returns o′.
// Otherwise the lineage is stale (a reseed or an earlier A1 of this run moved
// the head on): it writes nothing and returns uuid.Nil. A retry after commit
// is therefore a no-op too.
func Reschedule(ctx context.Context, pool *pgxpool.Pool, o uuid.UUID) (uuid.UUID, error) {
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok {
		return uuid.Nil, tenancy.ErrNoTenant
	}
	var next uuid.UUID
	err := workerguard.InTenantTx(ctx, pool, func(tx pgx.Tx) error {
		var (
			current  *uuid.UUID
			interval int32
		)
		switch err := tx.QueryRow(ctx, lockForRescheduleSQL, tenant).Scan(&current, &interval); {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrHeadNotFound
		case err != nil:
			return fmt.Errorf("seal: lock head: %w", err)
		}
		if current == nil || *current != o {
			return nil // stale lineage
		}
		var version int64
		if err := tx.QueryRow(ctx, bumpVersionSQL, tenant).Scan(&version); err != nil {
			return fmt.Errorf("seal: bump head version: %w", err)
		}
		delay := time.Duration(interval) * time.Second
		id, err := outbox.EnqueueSeal(ctx, tx, version, outbox.WithDelay(delay))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, repointSQL, tenant, id, delay.Seconds()); err != nil {
			return fmt.Errorf("seal: repoint head: %w", err)
		}
		next = id
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return next, nil
}

const (
	lockForSealSQL = `SELECT seq, entry_hash, seal_interval_s FROM audit_chain_head WHERE tenant_id = $1 FOR UPDATE`
	// The envelope columns (chain.Envelope), oldest first, through the
	// audit_log_unsealed partial index. nava_worker cannot read diff or
	// diff_salt and does not need to: the envelope holds diff_sha256.
	unsealedSQL = `SELECT id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
    version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
    causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated
FROM audit_log
WHERE tenant_id = $1 AND sealed_at IS NULL
ORDER BY id
LIMIT $2`
	// The R11 seal transition, one statement for the batch. occurred_at is
	// the partition key, so each row is found in its partition. sealed_at IS
	// NULL makes a row sealed meanwhile count as not updated.
	sealRowsSQL = `UPDATE audit_log a
SET seq = u.seq, prev_hash = u.prev_hash, entry_hash = u.entry_hash, sealed_at = now()
FROM unnest($2::timestamptz[], $3::uuid[], $4::bigint[], $5::bytea[], $6::bytea[])
    AS u(occurred_at, id, seq, prev_hash, entry_hash)
WHERE a.tenant_id = $1 AND a.occurred_at = u.occurred_at AND a.id = u.id AND a.sealed_at IS NULL`
	advanceHeadSQL = `UPDATE audit_chain_head SET seq = $2, entry_hash = $3 WHERE tenant_id = $1`
	setIntervalSQL = `UPDATE audit_chain_head SET seal_interval_s = $2 WHERE tenant_id = $1`
)

// Option configures Seal.
type Option func(*sealOptions)

type sealOptions struct {
	maxBatches int
	progress   func(sealed int)
}

// WithMaxBatches caps Seal at n batches (n BatchSize entries); n <= 0 is no
// cap. The capped batch counts as the run's last: it sets the interval, and
// the next run continues the chain from the head (TICK-440 review, "bounded
// A2").
func WithMaxBatches(n int) Option { return func(o *sealOptions) { o.maxBatches = n } }

// WithProgress calls f, with the running total, after each committed batch
// that is not the run's last. The activity heartbeats there.
func WithProgress(f func(sealed int)) Option { return func(o *sealOptions) { o.progress = f } }

// Seal is A2: it seals the tenant's unsealed entries batch by batch until a
// batch finds fewer than BatchSize, or WithMaxBatches batches have run, and
// returns how many it sealed. Each batch is its own transaction under the
// head lock. The last one also sets seal_interval_s by NextInterval, for the
// next A1 (R16). On an error the failing batch is rolled back; batches
// committed before it stay sealed, and the interval is left as it was.
func Seal(ctx context.Context, pool *pgxpool.Pool, opts ...Option) (int, error) {
	var o sealOptions
	for _, opt := range opts {
		opt(&o)
	}
	total := 0
	for batch := 1; ; batch++ {
		final := o.maxBatches > 0 && batch >= o.maxBatches
		n, err := sealBatch(ctx, pool, total, final)
		total += n
		if err != nil {
			return total, err
		}
		if n < BatchSize || final {
			return total, nil
		}
		if o.progress != nil {
			o.progress(total)
		}
	}
}

// sealBatch is one A2 transaction. A short batch, or the final one under the
// cap, is the run's last: it also sets the interval, the run counting as
// sealing if sealedBefore or this batch sealed anything.
func sealBatch(ctx context.Context, pool *pgxpool.Pool, sealedBefore int, final bool) (int, error) {
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok {
		return 0, tenancy.ErrNoTenant
	}
	sealed := 0
	err := workerguard.InTenantTx(ctx, pool, func(tx pgx.Tx) error {
		var (
			seq      int64
			tip      []byte
			interval int32
		)
		switch err := tx.QueryRow(ctx, lockForSealSQL, tenant).Scan(&seq, &tip, &interval); {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrHeadNotFound
		case err != nil:
			return fmt.Errorf("seal: lock head: %w", err)
		}
		if len(tip) != len(chain.Genesis) {
			return fmt.Errorf("seal: head entry_hash is %d bytes, want 32", len(tip))
		}
		prev := [32]byte(tip)

		envs, err := unsealed(ctx, tx, tenant)
		if err != nil {
			return err
		}
		n := len(envs)
		var (
			occurred = make([]time.Time, n)
			ids      = make([]uuid.UUID, n)
			seqs     = make([]int64, n)
			prevs    = make([][]byte, n)
			hashes   = make([][]byte, n)
		)
		for i, env := range envs {
			sum, err := chain.EnvelopeSHA256(env)
			if err != nil {
				// chain names the column, never the value.
				return fmt.Errorf("%w: entry %s: %w", ErrUnsealable, env.ID, err)
			}
			h := chain.EntryHash(prev, sum)
			seq++
			occurred[i], ids[i], seqs[i] = env.OccurredAt, env.ID, seq
			prevs[i], hashes[i] = append([]byte(nil), prev[:]...), append([]byte(nil), h[:]...)
			prev = h
		}
		if n > 0 {
			tag, err := tx.Exec(ctx, sealRowsSQL, tenant, occurred, ids, seqs, prevs, hashes)
			if err != nil {
				return fmt.Errorf("seal: seal entries: %w", err)
			}
			if got := tag.RowsAffected(); got != int64(n) {
				return fmt.Errorf("%w: sealed %d of %d", ErrSealRace, got, n)
			}
			if _, err := tx.Exec(ctx, advanceHeadSQL, tenant, seq, prev[:]); err != nil {
				return fmt.Errorf("seal: advance head: %w", err)
			}
		}
		if n < BatchSize || final {
			next := NextInterval(interval, sealedBefore+n > 0)
			if next != interval {
				if _, err := tx.Exec(ctx, setIntervalSQL, tenant, next); err != nil {
					return fmt.Errorf("seal: set interval: %w", err)
				}
			}
		}
		sealed = n
		return nil
	})
	if err != nil {
		return 0, err
	}
	return sealed, nil
}

// unsealed reads the batch's envelopes. tenant_id is the locked head's.
func unsealed(ctx context.Context, tx pgx.Tx, tenant uuid.UUID) ([]chain.Envelope, error) {
	rows, err := tx.Query(ctx, unsealedSQL, tenant, BatchSize)
	if err != nil {
		return nil, fmt.Errorf("seal: read unsealed entries: %w", err)
	}
	defer rows.Close()
	envs := make([]chain.Envelope, 0, BatchSize)
	for rows.Next() {
		e := chain.Envelope{TenantID: tenant}
		var content, diff []byte
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Domain, &e.Action, &e.SubjectType, &e.SubjectID,
			&e.EntityDefinitionID, &e.VersionBefore, &e.VersionAfter, &e.ActorPrincipalID, &e.ActorSystem,
			&e.Source, &e.SourceRef, &e.ChangeSetID, &e.CausationID, &e.CausationDepth, &e.RequestID,
			&e.Backfilled, &content, &diff, &e.DiffTruncated); err != nil {
			return nil, fmt.Errorf("seal: scan unsealed entry: %w", err)
		}
		if e.ContentSHA256, err = hash32(content); err != nil {
			return nil, fmt.Errorf("%w: entry %s: content_sha256: %w", ErrUnsealable, e.ID, err)
		}
		if e.DiffSHA256, err = hash32(diff); err != nil {
			return nil, fmt.Errorf("%w: entry %s: diff_sha256: %w", ErrUnsealable, e.ID, err)
		}
		envs = append(envs, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("seal: read unsealed entries: %w", err)
	}
	return envs, nil
}

// hash32 maps a nullable bytea digest to the envelope's *[32]byte. The
// column's CHECK holds it to 32 bytes; anything else is unsealable.
func hash32(b []byte) (*[32]byte, error) {
	if b == nil {
		return nil, nil
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("%d bytes, want 32", len(b))
	}
	h := [32]byte(b)
	return &h, nil
}
