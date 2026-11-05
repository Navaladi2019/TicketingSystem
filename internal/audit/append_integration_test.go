//go:build integration

// audit.Append against a real PostgreSQL, migrated with the real migrations
// and reached as the real nava_app role under RLS (P5-12b, TICK-430).
// make test-integration (scripts/ci/test-integration.sh) sets
// NAVA_TEST_PG_URL to the shared cluster's superuser; this package drops and
// recreates the manifest's databases and roles there, so it runs with the
// other shared-cluster packages one at a time (-p 1).

package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

type itEnv struct {
	super    *pgx.Conn // the platform database as superuser: seeding and reading back
	app      *pgxpool.Pool
	appExec  *pgxpool.Pool // nava_app in QueryExecModeExec, as behind PgBouncer
	platform string
}

func (e itEnv) appConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.User = "nava_app"
	cfg.Password, _ = testPassword("nava_app")
	cfg.Database = e.platform
	return cfg
}

//nolint:gochecknoglobals // one migrated cluster per package run
var (
	envOnce sync.Once
	envErr  error
	shared  itEnv
)

// migrated recreates the manifest's roles and databases and runs the real
// migrations once per package run, then opens nava_app pools.
func migrated(t *testing.T) itEnv {
	t.Helper()
	envOnce.Do(func() { shared, envErr = setup(superConfig(t)) })
	if envErr != nil {
		t.Fatal(envErr)
	}
	return shared
}

func setup(super *pgx.ConnConfig) (itEnv, error) {
	ctx := context.Background()
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
	e := itEnv{platform: m.PlatformDatabase}
	if e.super, err = pgx.ConnectConfig(ctx, platformSuper); err != nil {
		return itEnv{}, err
	}
	pool := func(mode pgx.QueryExecMode) (*pgxpool.Pool, error) {
		cfg, err := pgxpool.ParseConfig("")
		if err != nil {
			return nil, err
		}
		cfg.ConnConfig = super.Copy()
		cfg.ConnConfig.User = "nava_app"
		cfg.ConnConfig.Password, _ = testPassword("nava_app")
		cfg.ConnConfig.Database = m.PlatformDatabase
		cfg.ConnConfig.DefaultQueryExecMode = mode
		return pgxpool.NewWithConfig(ctx, cfg)
	}
	if e.app, err = pool(pgx.QueryExecModeCacheStatement); err != nil {
		return itEnv{}, err
	}
	if e.appExec, err = pool(pgx.QueryExecModeExec); err != nil {
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

func count(t *testing.T, conn *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// seedTenant inserts, as the superuser, an active tenant with the principal
// and (when not nil) the entity definition the audit_log foreign keys need.
// A tenant has at most one definition here (key and prefix are per tenant).
func seedTenant(t *testing.T, e itEnv, tenant, principal uuid.UUID, definition *uuid.UUID) {
	t.Helper()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT (id) DO NOTHING", tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now()) ON CONFLICT (tenant_id, id) DO NOTHING`, tenant, principal, "kc-"+principal.String())
	if definition != nil {
		mustExec(t, e.super, `INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
			VALUES ($1, $2, 'incident', 'Incident', 'INC', 1) ON CONFLICT (tenant_id, id) DO NOTHING`, tenant, *definition)
	}
}

func newTenant(t *testing.T, e itEnv) uuid.UUID {
	t.Helper()
	tenant := uuid.New()
	seedTenant(t, e, tenant, testPrincipal(), nil)
	return tenant
}

// storedRow is an audit_log row read back as the superuser.
type storedRow struct {
	env      chain.Envelope
	diff     *string
	salt     []byte
	occurred time.Time
}

func readRow(t *testing.T, e itEnv, id uuid.UUID) storedRow {
	t.Helper()
	var (
		r                   storedRow
		domain, action, st  string
		source              string
		content, diffSHA    []byte
		actorSys, srcRef    *string
		reqID               *string
		backfilled, trunc   bool
		depth               int16
		txid                *string
		seq                 *int64
		sealedAt, erasedAt  *time.Time
		prevHash, entryHash []byte
	)
	err := e.super.QueryRow(context.Background(), `SELECT tenant_id, id, occurred_at, domain, action, subject_type,
		subject_id, entity_definition_id, version_before, version_after, actor_principal_id, actor_system, source,
		source_ref, change_set_id, causation_id, causation_depth, request_id, backfilled, content_sha256,
		diff_sha256, diff_truncated, diff::text, diff_salt, txid::text, seq, prev_hash, entry_hash, sealed_at, erased_at
		FROM audit_log WHERE id = $1`, id).Scan(
		&r.env.TenantID, &r.env.ID, &r.occurred, &domain, &action, &st,
		&r.env.SubjectID, &r.env.EntityDefinitionID, &r.env.VersionBefore, &r.env.VersionAfter,
		&r.env.ActorPrincipalID, &actorSys, &source, &srcRef, &r.env.ChangeSetID, &r.env.CausationID,
		&depth, &reqID, &backfilled, &content, &diffSHA, &trunc, &r.diff, &r.salt,
		&txid, &seq, &prevHash, &entryHash, &sealedAt, &erasedAt)
	if err != nil {
		t.Fatalf("read entry %s: %v", id, err)
	}
	if txid == nil || seq != nil || prevHash != nil || entryHash != nil || sealedAt != nil || erasedAt != nil || backfilled {
		t.Errorf("entry %s: R10 columns set by the writer", id)
	}
	r.env.OccurredAt = r.occurred
	r.env.Domain, r.env.Action, r.env.SubjectType, r.env.Source = domain, action, st, source
	r.env.ActorSystem, r.env.SourceRef, r.env.RequestID = actorSys, srcRef, reqID
	r.env.CausationDepth, r.env.Backfilled, r.env.DiffTruncated = depth, backfilled, trunc
	if content != nil {
		h := [32]byte(content)
		r.env.ContentSHA256 = &h
	}
	if diffSHA != nil {
		h := [32]byte(diffSHA)
		r.env.DiffSHA256 = &h
	}
	return r
}

func loadVectors(t *testing.T) vectors.File {
	t.Helper()
	b, err := os.ReadFile("chain/testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var f vectors.File
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// entryFromEnvelope is the Entry a writer would pass to produce env.
func entryFromEnvelope(env chain.Envelope) Entry {
	return Entry{
		ID: env.ID, Domain: Domain(env.Domain), Action: Action(env.Action),
		SubjectType: SubjectType(env.SubjectType), SubjectID: env.SubjectID,
		EntityDefinitionID: env.EntityDefinitionID, VersionBefore: env.VersionBefore, VersionAfter: env.VersionAfter,
		Actor:  Actor{PrincipalID: env.ActorPrincipalID, System: env.ActorSystem},
		Source: Source(env.Source), SourceRef: env.SourceRef, ChangeSetID: env.ChangeSetID,
		CausationID: env.CausationID, CausationDepth: env.CausationDepth, RequestID: env.RequestID,
		ContentSHA256: env.ContentSHA256,
	}
}

// The writer test against P5-02's vectors.json (PLAN-005 D-5-7): every chain
// vector a writer can produce is written through Append, read back, and its
// stored envelope hashes to the vector's envelope_sha256, with the stored
// diff and salt byte-equal. occurred_at is the one envelope column no writer
// controls (R10: the database stamps now()), so the vector's value is put in
// its place before hashing; every other column is the stored one. Run in
// both pgx exec modes, so the json diff survives the PgBouncer path too.
func TestAppendGoldenVectors(t *testing.T) {
	e := migrated(t)
	f := loadVectors(t)
	pools := map[string]*pgxpool.Pool{"cache-statement": e.app, "exec": e.appExec}
	for mode, pool := range pools {
		for _, v := range f.Chain {
			if v.Envelope.Backfilled || v.ErasedAt != nil {
				continue // written by the migrator (R25) or changed by erasure (R11), never by Append
			}
			t.Run(mode+"/"+v.Name, func(t *testing.T) {
				env := v.Envelope
				principal := testPrincipal()
				if env.ActorPrincipalID != nil {
					principal = *env.ActorPrincipalID
				}
				seedTenant(t, e, env.TenantID, principal, env.EntityDefinitionID)
				mustExec(t, e.super, "DELETE FROM audit_log WHERE tenant_id = $1 AND id = $2", env.TenantID, env.ID)

				in := entryFromEnvelope(env)
				if v.Diff != nil {
					in.Diff = []byte(*v.Diff)
					salt, err := chain.ParseSalt16(*v.DiffSalt)
					if err != nil {
						t.Fatal(err)
					}
					defer func(f func([]byte)) { readSalt = f }(readSalt)
					readSalt = func(b []byte) { copy(b, salt[:]) }
				}
				if env.DiffTruncated {
					// The original diff: the stored ops plus one that cannot fit.
					in.Diff = append(bytes.TrimSuffix(in.Diff, []byte("]")),
						[]byte(","+op("/attachments/2", MaxDiffBytes)+"]")...)
				}
				err := tenancy.InTenantTx(context.Background(), pool, env.TenantID, func(tx pgx.Tx) error {
					_, err := Append(context.Background(), tx, in)
					return err
				})
				if err != nil {
					t.Fatalf("Append: %v", err)
				}

				got := readRow(t, e, env.ID)
				if (got.diff == nil) != (v.Diff == nil) || (got.diff != nil && *got.diff != *v.Diff) {
					t.Errorf("stored diff differs from the vector's bytes")
				}
				if v.DiffSalt != nil && hex.EncodeToString(got.salt) != *v.DiffSalt {
					t.Errorf("stored salt %x, want %s", got.salt, *v.DiffSalt)
				}
				got.env.OccurredAt = env.OccurredAt
				sum, err := chain.EnvelopeSHA256(got.env)
				if err != nil {
					t.Fatal(err)
				}
				if hex.EncodeToString(sum[:]) != v.EnvelopeSHA256 {
					t.Errorf("stored envelope hashes to %x, want the vector's %s", sum, v.EnvelopeSHA256)
				}
			})
		}
	}
}

func recordEntry(subject uuid.UUID, version int64, diff string) Entry {
	return Entry{
		Domain: DomainRecord, Action: ActionUpdate, SubjectType: SubjectRecord, SubjectID: subject,
		VersionAfter: &version, Actor: Actor{PrincipalID: ptr(testPrincipal())}, Source: SourceUser,
		Diff: []byte(diff),
	}
}

var errRollback = errors.New("roll back")

// CI-A1: the entry is in the caller's transaction. A rolled-back
// transaction leaves no entry; a committed one leaves exactly one.
func TestAppendSameTransaction(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	subject := uuid.New()
	var rolledBack uuid.UUID
	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		var err error
		rolledBack, err = Append(context.Background(), tx, recordEntry(subject, 1, `[{"op":"add","path":"/a","value":1}]`))
		if err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("err = %v", err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenant); n != 0 {
		t.Errorf("rolled back: %d entries, want 0", n)
	}
	if rolledBack == uuid.Nil {
		t.Errorf("Append inside the rolled-back transaction returned no id")
	}

	err = tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		_, err := Append(context.Background(), tx, recordEntry(subject, 1, `[{"op":"add","path":"/a","value":1}]`))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenant); n != 1 {
		t.Errorf("committed: %d entries, want 1", n)
	}
}

// insertTracer records every statement sent on a connection.
type insertTracer struct {
	mu  sync.Mutex
	sql []string
}

func (r *insertTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	r.mu.Lock()
	r.sql = append(r.sql, d.SQL)
	r.mu.Unlock()
	return ctx
}

func (r *insertTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func (r *insertTracer) inserts() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, s := range r.sql {
		if strings.Contains(strings.ToUpper(s), "INSERT") {
			n++
		}
	}
	return n
}

// Security (TICK-449): the tenant is the transaction's. Called for tenant B,
// Append writes a B row and cannot produce an A row; with no tenant set it
// fails before any INSERT is sent.
func TestAppendTenantFromTransaction(t *testing.T) {
	e := migrated(t)
	tenantA, tenantB := newTenant(t, e), newTenant(t, e)
	subject := uuid.New()

	var id uuid.UUID
	err := tenancy.InTenantTx(context.Background(), e.app, tenantB, func(tx pgx.Tx) error {
		var err error
		id, err = Append(context.Background(), tx, recordEntry(subject, 1, `[]`))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readRow(t, e, id).env.TenantID; got != tenantB {
		t.Errorf("entry written for %s, want tenant B %s", got, tenantB)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenantA); n != 0 {
		t.Errorf("tenant A has %d entries, want 0", n)
	}

	// No tenant: a bare transaction on a traced connection.
	cfg := e.appConfig(t)
	tr := &insertTracer{}
	cfg.Tracer = tr
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	_, err = Append(context.Background(), tx, recordEntry(uuid.New(), 1, `[]`))
	if !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("no tenant: err = %v, want ErrInvalidEntry", err)
	}
	if n := tr.inserts(); n != 0 {
		t.Errorf("no tenant: %d INSERTs sent, want 0", n)
	}
}

// relationBytes is every byte of the audit_log partitions' heap and TOAST
// files, after a CHECKPOINT has flushed them.
func relationBytes(t *testing.T, e itEnv) []byte {
	t.Helper()
	mustExec(t, e.super, "CHECKPOINT")
	rows, err := e.super.Query(context.Background(), `
		SELECT pg_relation_filepath(c.oid) FROM pg_class c
		WHERE c.oid IN (SELECT relid FROM pg_partition_tree('audit_log') WHERE isleaf)
		   OR c.oid IN (SELECT reltoastrelid FROM pg_class WHERE oid IN
		                (SELECT relid FROM pg_partition_tree('audit_log') WHERE isleaf))`)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := pgx.CollectRows(rows, pgx.RowTo[*string])
	if err != nil {
		t.Fatal(err)
	}
	var all []byte
	for _, p := range paths {
		if p == nil {
			continue
		}
		var b []byte
		if err := e.super.QueryRow(context.Background(), "SELECT pg_read_binary_file($1, true)", *p).Scan(&b); err != nil {
			t.Fatalf("read %s: %v", *p, err)
		}
		all = append(all, b...)
	}
	return all
}

// CI-A5: a canary at a secret path is rejected with audit_secret_in_diff,
// nothing is written, and the canary is nowhere in the table's bytes. The
// control marker, written in an ordinary entry, proves the scan sees diffs.
func TestAppendSecretNeverStored(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	marker := "control-" + uuid.NewString()
	canaryValue := "canary-" + uuid.NewString()

	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		_, err := Append(context.Background(), tx, recordEntry(uuid.New(), 1, `[{"op":"add","path":"/note","value":"`+marker+`"}]`))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	err = tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		en := recordEntry(uuid.New(), 1, `[{"op":"add","path":"/note","value":"ok"},{"op":"add","path":"/creds","value":{"api_key":"`+canaryValue+`"}}]`)
		en.SecretPaths = []patchwalk.Pointer{"/creds/api_key"}
		_, err := Append(context.Background(), tx, en)
		return err
	})
	var se *ErrSecretInDiff
	if !errors.As(err, &se) || se.Code() != "audit_secret_in_diff" || se.Path != "/creds/api_key" {
		t.Fatalf("err = %v, want audit_secret_in_diff at /creds/api_key", err)
	}
	if strings.Contains(err.Error(), canaryValue) {
		t.Errorf("the error carries the secret")
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenant); n != 1 {
		t.Errorf("%d entries, want only the control", n)
	}
	b := relationBytes(t, e)
	if !bytes.Contains(b, []byte(marker)) {
		t.Fatalf("the control marker is not in the table bytes; the scan proves nothing")
	}
	if bytes.Contains(b, []byte(canaryValue)) {
		t.Errorf("the canary is in the table bytes")
	}
}

// Security (TICK-449) and CI-A17(g): a diff over 64 KB is stored as the
// byte-exact join of its leading whole ops with diff_truncated = true, and
// diff_sha256 covers the stored bytes; a first op over the limit stores "[]".
func TestAppendTruncatesOverLimit(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	a, c := op("/a", 100), `{ "op" : "remove", "path" : "/c" }`
	cases := map[string]struct{ diff, stored string }{
		"keeps the ops that fit": {"[" + a + "," + c + "," + op("/b", MaxDiffBytes) + "]", "[" + a + "," + c + "]"},
		"first op too big":       {"[" + op("/b", MaxDiffBytes) + "," + a + "]", "[]"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var id uuid.UUID
			err := tenancy.InTenantTx(context.Background(), e.appExec, tenant, func(tx pgx.Tx) error {
				var err error
				id, err = Append(context.Background(), tx, recordEntry(uuid.New(), 1, tc.diff))
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			got := readRow(t, e, id)
			if !got.env.DiffTruncated || got.diff == nil || *got.diff != tc.stored {
				t.Fatalf("stored %v, truncated %v", got.diff, got.env.DiffTruncated)
			}
			want := sha256.Sum256(append(append([]byte{}, got.salt...), *got.diff...))
			if got.env.DiffSHA256 == nil || *got.env.DiffSHA256 != want {
				t.Errorf("diff_sha256 is not SHA-256(salt ‖ stored bytes)")
			}
		})
	}
}

// R2 commit key: a second entry for the same subject and version in one
// transaction is ErrDuplicateEntry.
func TestAppendDuplicate(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	subject := uuid.New()
	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		if _, err := Append(context.Background(), tx, recordEntry(subject, 3, `[]`)); err != nil {
			return err
		}
		_, err := Append(context.Background(), tx, recordEntry(subject, 3, `[]`))
		return err
	})
	if !errors.Is(err, ErrDuplicateEntry) {
		t.Fatalf("err = %v, want ErrDuplicateEntry", err)
	}
}

// CI-A17(g): an audit/erase entry with 1,501 ids returns ErrInvalidEntry and
// the transaction rolls back; 1,500 ids are written whole.
func TestAppendEraseCap(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	erase := func(n int) error {
		return tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
			en := Entry{
				Domain: DomainAudit, Action: ActionErase, SubjectType: SubjectErasureRequest, SubjectID: uuid.New(),
				Actor: Actor{System: ptr("erasure")}, Source: SourceSystem, Diff: erasedIDs(n),
			}
			_, err := Append(context.Background(), tx, en)
			return err
		})
	}
	if err := erase(MaxErasedIDs + 1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("1,501 ids: err = %v, want ErrInvalidEntry", err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1", tenant); n != 0 {
		t.Errorf("1,501 ids: %d entries written", n)
	}
	if err := erase(MaxErasedIDs); err != nil {
		t.Fatalf("1,500 ids: %v", err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND NOT diff_truncated", tenant); n != 1 {
		t.Errorf("1,500 ids: %d untruncated entries, want 1", n)
	}
}
