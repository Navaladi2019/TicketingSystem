//go:build integration

// The store's statements against a real PostgreSQL, migrated with the real
// migrations and reached as nava_app under RLS, on a pool in the server's
// protocol mode (TICK-471). make test-integration
// (scripts/ci/test-integration.sh) sets NAVA_TEST_PG_URL to the shared
// cluster's superuser; this package drops and recreates the manifest's
// databases and roles there, so it runs with the other shared-cluster
// packages one at a time (-p 1).

package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func testPassword(role string) (string, bool) { return "test-pw-" + role, true }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	dsn := os.Getenv("NAVA_TEST_PG_URL")
	if dsn == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// serverPoolConfig parses dsn into the pool config nava-core builds for
// NAVA_DATABASE_URL (cmd/nava-core/identity.go): QueryExecModeExec, so
// statements stay unnamed behind PgBouncer's transaction mode, and a 5s
// connect timeout. In that mode no parameter has a declared type and pgx
// picks the wire format from the Go type alone: a []byte bound to a jsonb
// parameter goes as bytea and PostgreSQL refuses it (22P02). Keep it in step
// with nava-core's pool.
func serverPoolConfig(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	return cfg
}

type env struct {
	super *pgx.Conn // the platform database as superuser, for seeding and reading back
	app   *pgxpool.Pool
}

// migrated recreates the manifest's roles and databases, runs the real
// migrations, and opens a nava_app pool on the platform database in the
// server's protocol mode.
func migrated(t *testing.T) env {
	t.Helper()
	ctx := context.Background()
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	super, err := pgx.ConnectConfig(ctx, superConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range m.DatabaseNames() {
		mustExec(t, super, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range m.RoleNames() {
		mustExec(t, super, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	_ = super.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, superConfig(t), m, testPassword, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	owner := superConfig(t).Copy()
	owner.User, owner.Database = m.Owner(), m.PlatformDatabase
	owner.Password, _ = testPassword(m.Owner())
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	platformSuper := superConfig(t).Copy()
	platformSuper.Database = m.PlatformDatabase
	conn, err := pgx.ConnectConfig(ctx, platformSuper)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_app")
	u.User = url.UserPassword("nava_app", pw)
	u.Path = "/" + m.PlatformDatabase
	app, err := pgxpool.NewWithConfig(ctx, serverPoolConfig(t, u.String()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	return env{super: conn, app: app}
}

func mustExec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// assetFixed maps the asset type's properties to fixed columns.
func assetFixed() map[string]string {
	return map[string]string{"status": metadata.ColumnState, "owner": metadata.ColumnRequesterID}
}

func assetDefinition() metadata.Definition {
	return metadata.Definition{
		Key: "asset",
		Schema: json.RawMessage(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["name"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "status": {"type": "string", "enum": ["active", "retired"]},
    "serial": {"type": "string", "x-nava-unique": true},
    "tag": {"type": "number", "x-nava-unique": true},
    "owner": {"type": "string", "format": "uuid", "x-nava-unique": true}
  }
}`),
		NumberPrefix: "AST",
		FixedColumns: assetFixed(),
		States: metadata.StateModel{
			Initial:     "active",
			States:      []string{"active", "retired"},
			Transitions: []metadata.Transition{{From: "active", To: "retired"}},
		},
	}
}

// tx runs fn in one committed tenant transaction on the server-mode pool.
func tx(t *testing.T, e env, tenant uuid.UUID, fn func(context.Context, pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	if err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error { return fn(ctx, tx) }); err != nil {
		t.Fatal(err)
	}
}

// recordEntry appends rec's audit entry in tx, which the records commit check
// (ADR-004-E5, P5-13c) requires of every records write. The v0 audit
// statements left the store with P5-13b/P5-17.
func recordEntry(ctx context.Context, tx pgx.Tx, rec *store.Record, action audit.Action, before *int64, actor uuid.UUID) (uuid.UUID, error) {
	va, def := rec.Version, rec.EntityDefinitionID
	return audit.Append(ctx, tx, audit.Entry{Domain: audit.DomainRecord, Action: action, SubjectType: audit.SubjectRecord,
		SubjectID: rec.ID, EntityDefinitionID: &def, VersionBefore: before, VersionAfter: &va,
		Actor: audit.Actor{PrincipalID: &actor}, Source: audit.SourceUser})
}

// writeKey is rec's record_write_keys row for key.
func writeKey(rec *store.Record, key, action string, entry uuid.UUID) *store.WriteKey {
	return &store.WriteKey{TenantID: rec.TenantID, IdempotencyKey: key, DefinitionID: rec.EntityDefinitionID, Action: action,
		RecordID: rec.ID, Number: rec.Number, Version: rec.Version, SchemaVersion: rec.SchemaVersion, AuditEntryID: entry}
}

// TestStatementsOnTheServersPoolMode runs every store statement that takes a
// jsonb argument, and the reads that return one, on a nava_app pool in
// QueryExecModeExec (TICK-471). Before TICK-411 and TICK-358 each jsonb
// argument went as []byte and failed with 22P02 in this mode only; the
// pipeline's tests ran on pgx's default mode and missed it. Reverting
// jsonParam at any call site, UniqueTaken's included, fails this test.
func TestStatementsOnTheServersPoolMode(t *testing.T) {
	e := migrated(t)
	fixed := assetFixed()
	tenant, principal := uuid.New(), uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, tenant, principal, "kc-"+principal.String())
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Create(ctx, tx, tenant, entitytypes.NewInput{
			Name: "asset", PublishedBy: principal, Definition: assetDefinition(),
		})
		return err
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	key := "store-exec-mode-1"
	var rec store.Record
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		def, err := store.DefinitionID(ctx, tx, tenant, "asset")
		if err != nil {
			return err
		}
		n, err := store.NextNumber(ctx, tx, tenant, def, "")
		if err != nil {
			return err
		}
		rec = store.Record{TenantID: tenant, ID: uuid.New(), EntityDefinitionID: def, Version: 1, SchemaVersion: 1,
			Number: astNumber(t, n), CreatedBy: &principal, CreatedAt: now, UpdatedAt: now}
		if err := rec.Split(map[string]any{"name": "one", "status": "active", "serial": "SN-1",
			"tag": json.Number("7"), "owner": principal.String()}, fixed); err != nil {
			return err
		}
		if err := store.Insert(ctx, tx, &rec); err != nil {
			return err
		}
		entry, err := recordEntry(ctx, tx, &rec, audit.ActionCreate, nil, principal)
		if err != nil {
			return err
		}
		return store.InsertWriteKey(ctx, tx, writeKey(&rec, key, "create", entry))
	})

	// UniqueTaken binds the value as $4::jsonb, on data and on each fixed
	// column. Taken by the record, free for the record itself, and jsonb
	// equality (7 and 7.0 are the same number).
	other := uuid.New()
	for _, c := range []struct {
		field  string
		value  any
		except uuid.UUID
		want   bool
	}{
		{"serial", "SN-1", other, true},
		{"serial", "SN-2", other, false},
		{"serial", "SN-1", rec.ID, false},
		{"tag", json.Number("7.0"), other, true},
		{"tag", json.Number("8"), other, false},
		{"owner", principal.String(), other, true},
		{"owner", uuid.NewString(), other, false},
		{"status", "active", other, true},
		{"status", "retired", other, false},
	} {
		tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
			if err := store.LockUnique(ctx, tx, tenant, rec.EntityDefinitionID, c.field, "canonical"); err != nil {
				return err
			}
			got, err := store.UniqueTaken(ctx, tx, tenant, rec.EntityDefinitionID, c.except, c.field, c.value, fixed)
			if err != nil {
				return err
			}
			if got != c.want {
				t.Errorf("UniqueTaken(%s = %v, except the record: %v) = %v, want %v", c.field, c.value, c.except == rec.ID, got, c.want)
			}
			return nil
		})
	}

	// Update (data jsonb) and its audit entry, the payload read before and
	// after it, then a stale update.
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		before, err := store.PayloadJSON(ctx, tx, tenant, rec.ID, 1, fixed)
		if err != nil {
			return err
		}
		var b map[string]any
		if err := json.Unmarshal(before, &b); err != nil || b["name"] != "one" || b["status"] != "active" || b["owner"] != principal.String() {
			t.Errorf("PayloadJSON before = %s, %v", before, err)
		}
		rec.Data["name"] = "renamed"
		rec.UpdatedAt = now.Add(time.Second)
		v, err := store.Update(ctx, tx, &rec, 1)
		if err != nil {
			return err
		}
		if v != 2 {
			t.Errorf("update: version %d, want 2", v)
		}
		if _, err := store.Update(ctx, tx, &rec, 1); !errors.Is(err, store.ErrStale) {
			t.Errorf("stale update: %v, want ErrStale", err)
		}
		rec.Version = v
		after, err := store.PayloadJSON(ctx, tx, tenant, rec.ID, 2, fixed)
		if err != nil {
			return err
		}
		var a map[string]any
		if err := json.Unmarshal(after, &a); err != nil || a["name"] != "renamed" {
			t.Errorf("PayloadJSON after = %s, %v", after, err)
		}
		if _, err := store.PayloadJSON(ctx, tx, tenant, rec.ID, 1, fixed); !errors.Is(err, store.ErrStale) {
			t.Errorf("PayloadJSON at the old version: %v, want ErrStale", err)
		}
		v1 := int64(1)
		_, err = recordEntry(ctx, tx, &rec, audit.ActionUpdate, &v1, principal)
		return err
	})

	// The idempotency key is taken: ErrDuplicateKey, in a transaction of its own.
	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		return store.InsertWriteKey(context.Background(), tx, writeKey(&rec, key, "update", uuid.New()))
	})
	if !errors.Is(err, store.ErrDuplicateKey) {
		t.Errorf("second row with the key: %v, want ErrDuplicateKey", err)
	}

	// The reads decode what the writes stored.
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		got, err := store.Get(ctx, tx, tenant, rec.ID)
		if err != nil {
			return err
		}
		if got.Version != 2 || got.Data["name"] != "renamed" || got.Data["tag"] != json.Number("7") || got.State != "active" {
			t.Errorf("Get = version %d, data %v, state %q", got.Version, got.Data, got.State)
		}
		kw, err := store.WriteKeyByKey(ctx, tx, tenant, nil, key)
		if err != nil {
			return err
		}
		if kw.RecordID != rec.ID || kw.Action != "create" || kw.Version != 1 || kw.Number != rec.Number {
			t.Errorf("WriteKeyByKey = %+v", kw)
		}
		if _, err := store.WriteKeyByKey(ctx, tx, tenant, nil, "no-such-key"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("unknown key: %v, want ErrNotFound", err)
		}
		return nil
	})

	// Stored as jsonb objects, not as strings or bytea text.
	rows, err := e.super.Query(context.Background(), `
SELECT jsonb_typeof(data) FROM records WHERE tenant_id = $1`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	kinds, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 1 {
		t.Errorf("%d jsonb values, want 1 (the record's data)", len(kinds))
	}
	for _, k := range kinds {
		if k != "object" {
			t.Errorf("a jsonb value is a %s, want object", k)
		}
	}
}

// TestStatementsStayInTheTransactionsTenant: every store statement runs in
// a tenancy.InTenantTx transaction, and row-level security keeps it to that
// transaction's tenant even when the tenant_id argument names another one.
// Tenant B's transaction cannot read, update, number into, read the write key
// or the payload of tenant A's record, nor insert a row for tenant A, and tenant A's record is
// unchanged afterwards.
func TestStatementsStayInTheTransactionsTenant(t *testing.T) {
	e := migrated(t)
	fixed := assetFixed()
	tenantA, tenantB := uuid.New(), uuid.New()
	principals := map[uuid.UUID]uuid.UUID{tenantA: uuid.New(), tenantB: uuid.New()}
	for tenant, principal := range principals {
		mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
		mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
			VALUES ($1, $2, $3, true, now())`, tenant, principal, "kc-"+principal.String())
		tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
			_, err := entitytypes.Store{}.Create(ctx, tx, tenant, entitytypes.NewInput{
				Name: "asset", PublishedBy: principal, Definition: assetDefinition(),
			})
			return err
		})
	}

	// Tenant A's record, its create entry and its write key.
	now := time.Now().UTC().Truncate(time.Microsecond)
	key := "store-cross-tenant-1"
	pa := principals[tenantA]
	var rec store.Record
	tx(t, e, tenantA, func(ctx context.Context, tx pgx.Tx) error {
		def, err := store.DefinitionID(ctx, tx, tenantA, "asset")
		if err != nil {
			return err
		}
		n, err := store.NextNumber(ctx, tx, tenantA, def, "")
		if err != nil {
			return err
		}
		rec = store.Record{TenantID: tenantA, ID: uuid.New(), EntityDefinitionID: def, Version: 1, SchemaVersion: 1,
			Number: astNumber(t, n), CreatedBy: &pa, CreatedAt: now, UpdatedAt: now}
		if err := rec.Split(map[string]any{"name": "a-secret", "status": "active", "serial": "SN-A"}, fixed); err != nil {
			return err
		}
		if err := store.Insert(ctx, tx, &rec); err != nil {
			return err
		}
		entry, err := recordEntry(ctx, tx, &rec, audit.ActionCreate, nil, pa)
		if err != nil {
			return err
		}
		return store.InsertWriteKey(ctx, tx, writeKey(&rec, key, "create", entry))
	})

	// Tenant B's transaction, naming tenant A's id and record explicitly.
	tx(t, e, tenantB, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := store.DefinitionID(ctx, tx, tenantA, "asset"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("B reads A's entity type: %v, want ErrNotFound", err)
		}
		if _, err := store.Get(ctx, tx, tenantA, rec.ID); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("B reads A's record: %v, want ErrNotFound", err)
		}
		forged := rec
		forged.Data = map[string]any{"name": "overwritten"}
		if _, err := store.Update(ctx, tx, &forged, 1); !errors.Is(err, store.ErrStale) {
			t.Errorf("B updates A's record: %v, want ErrStale (no row)", err)
		}
		if _, err := store.WriteKeyByKey(ctx, tx, tenantA, nil, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("B reads A's idempotency key: %v, want ErrNotFound", err)
		}
		if p, err := store.PayloadJSON(ctx, tx, tenantA, rec.ID, 1, fixed); !errors.Is(err, store.ErrStale) {
			t.Errorf("B reads A's payload: %s, %v; want ErrStale (no row)", p, err)
		}
		taken, err := store.UniqueTaken(ctx, tx, tenantA, rec.EntityDefinitionID, uuid.Nil, "serial", "SN-A", fixed)
		if err != nil || taken {
			t.Errorf("B sees A's unique value: %v, %v; want not taken", taken, err)
		}
		return nil
	})

	// Writes for tenant A from tenant B's transaction are refused by RLS.
	for name, write := range map[string]func(context.Context, pgx.Tx) error{
		"insert record": func(ctx context.Context, tx pgx.Tx) error {
			r := rec
			r.ID = uuid.New()
			r.Number = "AST9999999"
			return store.Insert(ctx, tx, &r)
		},
		"insert write key": func(ctx context.Context, tx pgx.Tx) error {
			return store.InsertWriteKey(ctx, tx, writeKey(&rec, "forged-key", "update", uuid.New()))
		},
		"take a number": func(ctx context.Context, tx pgx.Tx) error {
			_, err := store.NextNumber(ctx, tx, tenantA, rec.EntityDefinitionID, "")
			return err
		},
	} {
		err := tenancy.InTenantTx(context.Background(), e.app, tenantB, func(tx pgx.Tx) error {
			return write(context.Background(), tx)
		})
		if err == nil {
			t.Errorf("%s for tenant A from tenant B's transaction succeeded", name)
		}
	}

	// Tenant A's rows are as A wrote them.
	tx(t, e, tenantA, func(ctx context.Context, tx pgx.Tx) error {
		got, err := store.Get(ctx, tx, tenantA, rec.ID)
		if err != nil {
			return err
		}
		if got.Version != 1 || got.Data["name"] != "a-secret" {
			t.Errorf("A's record after B's attempts: version %d, data %v", got.Version, got.Data)
		}
		kw, err := store.WriteKeyByKey(ctx, tx, tenantA, nil, key)
		if err != nil {
			return err
		}
		if kw.RecordID != rec.ID || kw.Action != "create" {
			t.Errorf("A's write key after B's attempts: %+v", kw)
		}
		if _, err := store.WriteKeyByKey(ctx, tx, tenantA, nil, "forged-key"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("B's forged key for A exists: %v", err)
		}
		return nil
	})
	var counters, records int
	if err := e.super.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM record_counters WHERE tenant_id = $1),
		(SELECT count(*) FROM records WHERE tenant_id = $1)`, tenantA).Scan(&counters, &records); err != nil {
		t.Fatal(err)
	}
	if counters != 1 || records != 1 {
		t.Errorf("tenant A has %d counter rows and %d records, want 1 and 1", counters, records)
	}
}

// astNumber is counter value n as the default format renders it under AST.
func astNumber(t *testing.T, n int64) string {
	t.Helper()
	s, err := numbering.Default().Render("AST", n)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
