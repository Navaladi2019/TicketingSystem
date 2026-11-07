//go:build integration

// P5-34a (ADR-004 R13 to R16) against a real PostgreSQL, migrated with the
// real migrations and reached as the real nava_worker role under RLS.
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package seal_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

type itEnv struct {
	super  *pgx.Conn // the platform database as superuser: seeding and reading back
	worker *pgxpool.Pool
}

//nolint:gochecknoglobals // one migrated cluster per package run
var (
	envOnce sync.Once
	envErr  error
	shared  itEnv
)

func migrated(t *testing.T) itEnv {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	envOnce.Do(func() { shared, envErr = setup(url) })
	if envErr != nil {
		t.Fatal(envErr)
	}
	return shared
}

func setup(url string) (itEnv, error) {
	ctx := context.Background()
	super, err := pgx.ParseConfig(url)
	if err != nil {
		return itEnv{}, err
	}
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return itEnv{}, err
	}
	conn, err := pgx.ConnectConfig(ctx, super)
	if err != nil {
		return itEnv{}, err
	}
	for _, d := range m.DatabaseNames() {
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)"); err != nil {
			return itEnv{}, err
		}
	}
	for _, r := range m.RoleNames() {
		if _, err := conn.Exec(ctx, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize()); err != nil {
			return itEnv{}, err
		}
	}
	_ = conn.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, super, m, testPassword, log); err != nil {
		return itEnv{}, err
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		return itEnv{}, err
	}
	owner := super.Copy()
	owner.User = m.Owner()
	owner.Password, _ = testPassword(m.Owner())
	owner.Database = m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		return itEnv{}, err
	}
	platformSuper := super.Copy()
	platformSuper.Database = m.PlatformDatabase
	var e itEnv
	if e.super, err = pgx.ConnectConfig(ctx, platformSuper); err != nil {
		return itEnv{}, err
	}
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		return itEnv{}, err
	}
	cfg.ConnConfig = super.Copy()
	cfg.ConnConfig.User = "nava_worker"
	cfg.ConnConfig.Password, _ = testPassword("nava_worker")
	cfg.ConnConfig.Database = m.PlatformDatabase
	cfg.MaxConns = 8
	if e.worker, err = pgxpool.NewWithConfig(ctx, cfg); err != nil {
		return itEnv{}, err
	}
	return e, nil
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func count(t *testing.T, conn *pgx.Conn, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// asTenant is the context the activity guard hands an activity of tenant.
func asTenant(tenant uuid.UUID) context.Context {
	return workerguard.ContextWithTenant(context.Background(), tenant)
}

// newTenant seeds, as the superuser, an active tenant and its head (as
// onboarding, A6.2, will), and returns the tenant.
func newTenant(t *testing.T, e itEnv) uuid.UUID {
	t.Helper()
	tenant := uuid.New()
	seedHead(t, e, tenant)
	return tenant
}

func seedHead(t *testing.T, e itEnv, tenant uuid.UUID) {
	t.Helper()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	mustExec(t, e.super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2)", tenant, uuid.Must(uuid.NewV7()))
}

// writeEntries inserts n unsealed entries for tenant as nava_worker, through
// the normal R10 path, in one transaction (so in id order), and returns their
// ids in that order.
func writeEntries(t *testing.T, e itEnv, tenant uuid.UUID, n int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV7())
	}
	err := tenancy.InTenantTx(context.Background(), e.worker, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `INSERT INTO audit_log
			(tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			SELECT $1, u, 'audit', 'export', 'audit_export', gen_random_uuid(), 'system', 'system'
			FROM unnest($2::uuid[]) AS u`, tenant, ids)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

// writeEntriesBackwards writes n entries in txs transactions, so they carry
// txs distinct occurred_at values (now() is the transaction's start), with
// ids running backwards against time: each transaction writes ids older than
// the one before it. Sealing in id order alone puts seq against
// (occurred_at, id) order, which assertChain catches.
func writeEntriesBackwards(t *testing.T, e itEnv, tenant uuid.UUID, n, txs int) {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV7())
	}
	per := (n + txs - 1) / txs
	for end := n; end > 0; end -= per {
		chunk := ids[max(0, end-per):end]
		if err := tenancy.InTenantTx(context.Background(), e.worker, tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(context.Background(), `INSERT INTO audit_log
				(tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
				SELECT $1, u, 'audit', 'export', 'audit_export', gen_random_uuid(), 'system', 'system'
				FROM unnest($2::uuid[]) AS u`, tenant, chunk)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	if got := count(t, e.super, `SELECT count(DISTINCT occurred_at) FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NULL`, tenant); got != int64(txs) {
		t.Fatalf("%d distinct occurred_at values, want %d: the fixture needs one per transaction", got, txs)
	}
}

type head struct {
	version   int64
	seq       int64
	entryHash []byte
	next      *uuid.UUID
	nextAt    *time.Time
	interval  int32
}

func readHead(t *testing.T, e itEnv, tenant uuid.UUID) head {
	t.Helper()
	var h head
	if err := e.super.QueryRow(context.Background(), `SELECT version, seq, entry_hash, next_seal_outbox_id,
		next_seal_at, seal_interval_s FROM audit_chain_head WHERE tenant_id = $1`, tenant).
		Scan(&h.version, &h.seq, &h.entryHash, &h.next, &h.nextAt, &h.interval); err != nil {
		t.Fatal(err)
	}
	return h
}

// assertChain checks the tenant's sealed entries as the verifier would: seq
// 1..n without a gap or duplicate, each prev_hash the entry_hash before it
// (genesis first), each entry_hash recomputed from the stored envelope, seq
// in (occurred_at, id) order (the order A2 seals in; Postgres review of
// PR #486), and the head at the last one. It returns n.
func assertChain(t *testing.T, e itEnv, tenant uuid.UUID) int64 {
	t.Helper()
	ctx := context.Background()
	rows, err := e.super.Query(ctx, `SELECT seq, prev_hash, entry_hash, `+envelopeColumns+`
		FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NOT NULL ORDER BY seq, id`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	prev := chain.Genesis
	var n int64
	for rows.Next() {
		var (
			seq            int64
			prevHash, hash []byte
		)
		env := scanEnvelope(t, rows, tenant, &seq, &prevHash, &hash)
		n++
		if seq != n {
			t.Fatalf("seq %d at position %d: a duplicate or a gap (R15)", seq, n)
		}
		if [32]byte(prevHash) != prev {
			t.Fatalf("seq %d: prev_hash is not the entry_hash of seq %d", seq, seq-1)
		}
		sum, err := chain.EnvelopeSHA256(env)
		if err != nil {
			t.Fatal(err)
		}
		if want := chain.EntryHash(prev, sum); [32]byte(hash) != want {
			t.Fatalf("seq %d: entry_hash %x, want %x", seq, hash, want)
		}
		prev = [32]byte(hash)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if bad := count(t, e.super, `SELECT count(*) FROM (SELECT seq, row_number() OVER (ORDER BY occurred_at, id) AS rn
		FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NOT NULL) s WHERE seq <> rn`, tenant); bad != 0 {
		t.Fatalf("%d sealed entries whose seq is not their position in (occurred_at, id) order", bad)
	}
	h := readHead(t, e, tenant)
	if h.seq != n || [32]byte(h.entryHash) != prev {
		t.Fatalf("head at seq %d, want %d with the last entry_hash", h.seq, n)
	}
	return n
}

const envelopeColumns = `id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref, change_set_id,
	causation_id, causation_depth, request_id, backfilled, content_sha256, diff_sha256, diff_truncated`

// scanEnvelope scans lead, then envelopeColumns.
func scanEnvelope(t *testing.T, rows pgx.Rows, tenant uuid.UUID, lead ...any) chain.Envelope {
	t.Helper()
	env := chain.Envelope{TenantID: tenant}
	var content, diff []byte
	dest := append(lead, &env.ID, &env.OccurredAt, &env.Domain, &env.Action, &env.SubjectType, &env.SubjectID,
		&env.EntityDefinitionID, &env.VersionBefore, &env.VersionAfter, &env.ActorPrincipalID, &env.ActorSystem,
		&env.Source, &env.SourceRef, &env.ChangeSetID, &env.CausationID, &env.CausationDepth, &env.RequestID,
		&env.Backfilled, &content, &diff, &env.DiffTruncated)
	if err := rows.Scan(dest...); err != nil {
		t.Fatal(err)
	}
	if content != nil {
		h := [32]byte(content)
		env.ContentSHA256 = &h
	}
	if diff != nil {
		h := [32]byte(diff)
		env.DiffSHA256 = &h
	}
	return env
}

// AC R15 (CI-A16): concurrent A2 runs for one tenant serialise on the head
// and never duplicate a seq. Four sealers race over 2,500 entries (three
// batches' worth); every entry is sealed once, seq is 1..2500 with no
// duplicate or gap, and the chain links end to end.
func TestConcurrentSealersNeverDuplicateSeq(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	const n = 2500
	writeEntries(t, e, tenant, n)

	const sealers = 4
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		sealed int
		errs   []error
	)
	start := make(chan struct{})
	for range sealers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := seal.Seal(asTenant(tenant), e.worker)
			mu.Lock()
			defer mu.Unlock()
			sealed += got
			if err != nil {
				errs = append(errs, err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if sealed != n {
		t.Errorf("the sealers report %d sealed in total, want %d (an entry sealed twice or not at all)", sealed, n)
	}
	if got := assertChain(t, e, tenant); got != n {
		t.Errorf("%d sealed entries, want %d", got, n)
	}
	if dup := count(t, e.super, `SELECT count(*) FROM (SELECT seq FROM audit_log WHERE tenant_id = $1
		GROUP BY seq HAVING count(*) > 1) d`, tenant); dup != 0 {
		t.Errorf("%d duplicated seq values", dup)
	}
}

// AC R15: an A2 transaction waits for the head lock another transaction
// holds, and assigns seq only after it from the head it then reads. The
// holder is a sealer that has already advanced the head's seq (as a
// concurrent A2 batch would) but not committed.
func TestSealWaitsForHeadLock(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := newTenant(t, e)
	writeEntries(t, e, tenant, 3)

	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT 1 FROM audit_chain_head WHERE tenant_id = $1 FOR UPDATE", tenant); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := seal.Seal(asTenant(tenant), e.worker)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("Seal finished while another transaction held the head lock: %v", err)
	case <-time.After(500 * time.Millisecond):
	}
	if got := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NOT NULL", tenant); got != 0 {
		t.Fatalf("%d entries sealed while the head lock was held", got)
	}
	// The holder moves the head on and commits; the waiting sealer must
	// continue from it, not from the head it would have read before.
	moved := [32]byte{1}
	if _, err := tx.Exec(ctx, "UPDATE audit_chain_head SET seq = 7, entry_hash = $2 WHERE tenant_id = $1", tenant, moved[:]); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Seal did not finish after the head lock was released")
	}
	var (
		first int64
		prev  []byte
	)
	if err := e.super.QueryRow(ctx, `SELECT seq, prev_hash FROM audit_log WHERE tenant_id = $1
		ORDER BY seq LIMIT 1`, tenant).Scan(&first, &prev); err != nil {
		t.Fatal(err)
	}
	if first != 8 || [32]byte(prev) != moved {
		t.Errorf("first seq %d after the lock holder's 7, want 8 linked to its entry_hash", first)
	}
	if h := readHead(t, e, tenant); h.seq != 10 {
		t.Errorf("head seq %d, want 10", h.seq)
	}
}

// AC: an entry that cannot be canonicalised fails its whole A2 batch, and no
// seq is used: nothing in the batch is sealed and the head does not move.
// version_after past 2^53 - 1 is the one value the database stores and the
// canonical form refuses (I-JSON). The good entry is older, so it is first
// in the batch.
func TestUnsealableEntryFailsItsBatch(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	writeEntries(t, e, tenant, 1)
	err := tenancy.InTenantTx(context.Background(), e.worker, tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), `INSERT INTO audit_log
			(tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
			VALUES ($1, $2, 'record', 'update', 'record', gen_random_uuid(), $3, 'system', 'system')`,
			tenant, uuid.Must(uuid.NewV7()), int64(chain.MaxSafeInteger)+1)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	before := readHead(t, e, tenant)

	n, err := seal.Seal(asTenant(tenant), e.worker)
	if !errors.Is(err, seal.ErrUnsealable) || !errors.Is(err, chain.ErrInvalidEnvelope) {
		t.Fatalf("Seal = %d, %v; want ErrUnsealable wrapping chain.ErrInvalidEnvelope", n, err)
	}
	if n != 0 {
		t.Errorf("Seal reports %d sealed, want 0", n)
	}
	if got := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND seq IS NOT NULL", tenant); got != 0 {
		t.Errorf("%d entries have a seq after the failed batch, want 0", got)
	}
	after := readHead(t, e, tenant)
	if after.seq != before.seq || string(after.entryHash) != string(before.entryHash) || after.interval != before.interval {
		t.Errorf("the head moved: %+v -> %+v", before, after)
	}
}

// AC: A1 of a stale lineage is a no-op; A1 of the current lineage bumps the
// version, enqueues o′ at it with available_at = now() + seal_interval_s, and
// repoints the head; a retry of that A1 after commit is then stale too.
func TestReschedule(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	o := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, `UPDATE audit_chain_head SET next_seal_outbox_id = $2, seal_interval_s = 40, version = 3
		WHERE tenant_id = $1`, tenant, o)
	ctx := asTenant(tenant)

	// A stale lineage: another run, or none.
	for _, stale := range []uuid.UUID{uuid.Must(uuid.NewV7()), uuid.Nil} {
		next, err := seal.Reschedule(ctx, e.worker, stale)
		if err != nil || next != uuid.Nil {
			t.Fatalf("stale Reschedule = %s, %v; want a no-op", next, err)
		}
	}
	if h := readHead(t, e, tenant); h.version != 3 || h.next == nil || *h.next != o || h.nextAt != nil {
		t.Fatalf("a stale A1 changed the head: %+v", h)
	}
	if got := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); got != 0 {
		t.Fatalf("a stale A1 wrote %d outbox rows", got)
	}

	next, err := seal.Reschedule(ctx, e.worker, o)
	if err != nil || next == uuid.Nil {
		t.Fatalf("Reschedule(o) = %s, %v", next, err)
	}
	h := readHead(t, e, tenant)
	if h.version != 4 || h.next == nil || *h.next != next {
		t.Errorf("head version %d next %v, want 4 and %s", h.version, h.next, next)
	}
	var (
		kind, dedupe, targetKey string
		targetSeq               int64
		availableAt             time.Time
		gapUS                   int64
	)
	if err := e.super.QueryRow(context.Background(), `SELECT kind, dedupe_key, target_key, target_seq, available_at,
		(extract(epoch FROM available_at - created_at) * 1000000)::bigint
		FROM outbox WHERE tenant_id = $1 AND id = $2`, tenant, next).
		Scan(&kind, &dedupe, &targetKey, &targetSeq, &availableAt, &gapUS); err != nil {
		t.Fatal(err)
	}
	if kind != outbox.SealKind || dedupe != outbox.SealDedupeKey(4) || targetKey != outbox.SealKind+":"+next.String() || targetSeq != 4 {
		t.Errorf("o′ is %s %s %s seq %d, want a seal start at version 4", kind, dedupe, targetKey, targetSeq)
	}
	// R16: available_at = now() + seal_interval_s, on the row itself.
	if gapUS != 40_000_000 {
		t.Errorf("available_at - created_at = %d µs, want 40 s", gapUS)
	}
	if h.nextAt == nil || !h.nextAt.Equal(availableAt) {
		t.Errorf("next_seal_at %v, want o′'s available_at %s", h.nextAt, availableAt)
	}

	// The same run's A1 retried after commit finds the head moved on.
	again, err := seal.Reschedule(ctx, e.worker, o)
	if err != nil || again != uuid.Nil {
		t.Errorf("retried Reschedule(o) = %s, %v; want a no-op", again, err)
	}
	if got := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); got != 1 {
		t.Errorf("%d outbox rows, want 1", got)
	}
	if h := readHead(t, e, tenant); h.version != 4 {
		t.Errorf("head version %d after the retry, want 4", h.version)
	}
}

// AC: two A1s of the same run racing commit one o′ between them: the head
// lock serialises them and the second sees the lineage moved on.
func TestConcurrentRescheduleOneSuccessor(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	o := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "UPDATE audit_chain_head SET next_seal_outbox_id = $2 WHERE tenant_id = $1", tenant, o)

	const racers = 4
	results := make(chan uuid.UUID, racers)
	errs := make(chan error, racers)
	var wg sync.WaitGroup
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			next, err := seal.Reschedule(asTenant(tenant), e.worker, o)
			if err != nil {
				errs <- err
			}
			results <- next
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	winners := 0
	for next := range results {
		if next != uuid.Nil {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("%d A1s rescheduled, want exactly 1", winners)
	}
	if got := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", tenant); got != 1 {
		t.Errorf("%d outbox rows, want 1", got)
	}
	if h := readHead(t, e, tenant); h.version != 2 {
		t.Errorf("head version %d, want 2", h.version)
	}
}

// R16 interval: the run's last batch sets seal_interval_s to 10 s after a
// run that sealed anything, otherwise doubles it up to 300 s. A run that
// sealed only in an earlier batch counts as sealing.
func TestSealSetsInterval(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	ctx := asTenant(tenant)
	set := func(s int32) {
		mustExec(t, e.super, "UPDATE audit_chain_head SET seal_interval_s = $2 WHERE tenant_id = $1", tenant, s)
	}
	for _, c := range []struct {
		name    string
		from    int32
		entries int
		want    int32
	}{
		{"idle doubles", 10, 0, 20},
		{"idle doubles again", 20, 0, 40},
		{"idle caps at 300", 160, 0, 300},
		{"idle stays at 300", 300, 0, 300},
		{"sealing resets", 300, 3, 10},
		// Exactly one full batch: the run's last batch finds nothing, but
		// the run sealed, so it is 10 s, not doubled.
		{"full batch then empty", 80, seal.BatchSize, 10},
	} {
		t.Run(c.name, func(t *testing.T) {
			set(c.from)
			writeEntries(t, e, tenant, c.entries)
			n, err := seal.Seal(ctx, e.worker)
			if err != nil || n != c.entries {
				t.Fatalf("Seal = %d, %v; want %d", n, err, c.entries)
			}
			if h := readHead(t, e, tenant); h.interval != c.want {
				t.Errorf("seal_interval_s %d, want %d", h.interval, c.want)
			}
		})
	}
	assertChain(t, e, tenant)
}

// I1, I2: A2 and A1 reach only the guard's tenant. B's entries stay unsealed
// and B's head unmoved when A seals; a tenant with no head is
// ErrHeadNotFound.
func TestSealTenantIsolation(t *testing.T) {
	e := migrated(t)
	a, b := newTenant(t, e), newTenant(t, e)
	writeEntries(t, e, a, 2)
	writeEntries(t, e, b, 3)
	bo := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "UPDATE audit_chain_head SET next_seal_outbox_id = $2 WHERE tenant_id = $1", b, bo)

	if n, err := seal.Seal(asTenant(a), e.worker); err != nil || n != 2 {
		t.Fatalf("Seal(A) = %d, %v; want 2", n, err)
	}
	if got := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NOT NULL", b); got != 0 {
		t.Errorf("A's sealer sealed %d of B's entries", got)
	}
	if h := readHead(t, e, b); h.seq != 0 {
		t.Errorf("B's head moved to seq %d", h.seq)
	}
	// B's run id, presented under A's tenant, does not match A's head.
	if next, err := seal.Reschedule(asTenant(a), e.worker, bo); err != nil || next != uuid.Nil {
		t.Errorf("Reschedule(A, B's run) = %s, %v; want a no-op", next, err)
	}
	if h := readHead(t, e, b); h.version != 1 {
		t.Errorf("B's head version %d, want 1", h.version)
	}

	none := uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", none)
	if _, err := seal.Seal(asTenant(none), e.worker); !errors.Is(err, seal.ErrHeadNotFound) {
		t.Errorf("Seal with no head: %v, want ErrHeadNotFound", err)
	}
	if _, err := seal.Reschedule(asTenant(none), e.worker, bo); !errors.Is(err, seal.ErrHeadNotFound) {
		t.Errorf("Reschedule with no head: %v, want ErrHeadNotFound", err)
	}
}
