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
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
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
	m, err := grants.Parse(db.Grants())
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
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
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
		n, err := store.NextNumber(ctx, tx, tenant, def)
		if err != nil {
			return err
		}
		rec = store.Record{TenantID: tenant, ID: uuid.New(), EntityDefinitionID: def, Version: 1, SchemaVersion: 1,
			Number: store.FormatNumber("AST", n), CreatedBy: &principal, CreatedAt: now, UpdatedAt: now}
		if err := rec.Split(map[string]any{"name": "one", "status": "active", "serial": "SN-1",
			"tag": json.Number("7"), "owner": principal.String()}, fixed); err != nil {
			return err
		}
		if err := store.Insert(ctx, tx, &rec); err != nil {
			return err
		}
		return store.InsertAudit(ctx, tx, &store.Audit{TenantID: tenant, ID: uuid.New(), RecordID: rec.ID,
			Action: "create", ActorID: &principal, Source: "user", IdempotencyKey: &key, At: now,
			After: &store.Snapshot{Number: rec.Number, Version: 1, SchemaVersion: 1, Data: rec.Payload(fixed)}})
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

	// Update (data jsonb) and its audit row (before and after), then a stale
	// update and a cascade row with a detail.
	tx(t, e, tenant, func(ctx context.Context, tx pgx.Tx) error {
		before := store.Snapshot{Number: rec.Number, Version: 1, SchemaVersion: 1, Data: rec.Payload(fixed)}
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
		if err := store.InsertAudit(ctx, tx, &store.Audit{TenantID: tenant, ID: uuid.New(), RecordID: rec.ID,
			Action: "update", ActorID: &principal, Source: "user", At: rec.UpdatedAt, Before: &before,
			After: &store.Snapshot{Number: rec.Number, Version: 2, SchemaVersion: 1, Data: rec.Payload(fixed)}}); err != nil {
			return err
		}
		return store.InsertAudit(ctx, tx, &store.Audit{TenantID: tenant, ID: uuid.New(), Action: "cascade_rejected",
			Source: "system", At: rec.UpdatedAt, Detail: map[string]string{"reason": "depth"}})
	})

	// The idempotency key is taken: ErrDuplicateKey, in a transaction of its own.
	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		return store.InsertAudit(context.Background(), tx, &store.Audit{TenantID: tenant, ID: uuid.New(), RecordID: rec.ID,
			Action: "update", Source: "system", IdempotencyKey: &key, At: now,
			After: &store.Snapshot{Number: rec.Number, Version: 2, SchemaVersion: 1, Data: map[string]any{}}})
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
		kw, err := store.AuditByKey(ctx, tx, tenant, key)
		if err != nil {
			return err
		}
		if kw.RecordID != rec.ID || kw.Action != "create" || kw.After.Data["name"] != "one" {
			t.Errorf("AuditByKey = %+v", kw)
		}
		h, err := store.History(ctx, tx, tenant, rec.ID, time.Time{}, uuid.Nil, 10)
		if err != nil {
			return err
		}
		if len(h) != 2 || h[0].Action != "update" || h[0].Before == nil || h[0].Before.Data["name"] != "one" ||
			h[0].After.Data["name"] != "renamed" || h[1].Action != "create" {
			t.Errorf("History = %+v, want the update then the create", h)
		}
		return nil
	})

	// Stored as jsonb objects, not as strings or bytea text.
	rows, err := e.super.Query(context.Background(), `
SELECT jsonb_typeof(data) FROM records WHERE tenant_id = $1
UNION ALL SELECT jsonb_typeof(after) FROM audit_log WHERE tenant_id = $1
UNION ALL SELECT jsonb_typeof(before) FROM audit_log WHERE tenant_id = $1 AND before IS NOT NULL`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	kinds, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(kinds) != 5 {
		t.Errorf("%d jsonb values, want 5 (data, three afters, one before)", len(kinds))
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
// Tenant B's transaction cannot read, update, number into, audit or history
// tenant A's record, nor insert a row for tenant A, and tenant A's record is
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

	// Tenant A's record and its keyed create row.
	now := time.Now().UTC().Truncate(time.Microsecond)
	key := "store-cross-tenant-1"
	pa := principals[tenantA]
	var rec store.Record
	tx(t, e, tenantA, func(ctx context.Context, tx pgx.Tx) error {
		def, err := store.DefinitionID(ctx, tx, tenantA, "asset")
		if err != nil {
			return err
		}
		n, err := store.NextNumber(ctx, tx, tenantA, def)
		if err != nil {
			return err
		}
		rec = store.Record{TenantID: tenantA, ID: uuid.New(), EntityDefinitionID: def, Version: 1, SchemaVersion: 1,
			Number: store.FormatNumber("AST", n), CreatedBy: &pa, CreatedAt: now, UpdatedAt: now}
		if err := rec.Split(map[string]any{"name": "a-secret", "status": "active", "serial": "SN-A"}, fixed); err != nil {
			return err
		}
		if err := store.Insert(ctx, tx, &rec); err != nil {
			return err
		}
		return store.InsertAudit(ctx, tx, &store.Audit{TenantID: tenantA, ID: uuid.New(), RecordID: rec.ID,
			Action: "create", ActorID: &pa, Source: "user", IdempotencyKey: &key, At: now,
			After: &store.Snapshot{Number: rec.Number, Version: 1, SchemaVersion: 1, Data: rec.Payload(fixed)}})
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
		if _, err := store.AuditByKey(ctx, tx, tenantA, key); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("B reads A's idempotency key: %v, want ErrNotFound", err)
		}
		h, err := store.History(ctx, tx, tenantA, rec.ID, time.Time{}, uuid.Nil, 10)
		if err != nil || len(h) != 0 {
			t.Errorf("B reads A's history: %v, %v; want none", h, err)
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
		"insert audit": func(ctx context.Context, tx pgx.Tx) error {
			return store.InsertAudit(ctx, tx, &store.Audit{TenantID: tenantA, ID: uuid.New(), RecordID: rec.ID,
				Action: "update", Source: "system", At: now, After: &store.Snapshot{Data: map[string]any{}}})
		},
		"take a number": func(ctx context.Context, tx pgx.Tx) error {
			_, err := store.NextNumber(ctx, tx, tenantA, rec.EntityDefinitionID)
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
		h, err := store.History(ctx, tx, tenantA, rec.ID, time.Time{}, uuid.Nil, 10)
		if err != nil {
			return err
		}
		if len(h) != 1 || h[0].Action != "create" {
			t.Errorf("A's history after B's attempts: %+v, want the create only", h)
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
