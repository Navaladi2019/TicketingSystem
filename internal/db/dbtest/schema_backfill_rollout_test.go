//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// P10-10 (TICK-925, folded into TICK-916; ADR-009 D15): the rollout
// migration queues one job and one outbox row per tenant and defaulted type,
// skips default + unique types with a notice, and never touches records.

const schemaBackfillRollout = "20261126100100_schema_backfill_rollout.sql"

const (
	// rolloutDefaulted: two top-level defaults, one property without.
	rolloutDefaulted = `{"type": "object", "properties": {"title": {"type": "string"},
		"priority": {"type": "string", "default": "low"}, "channel": {"type": "string", "default": "email"}}}`
	// rolloutUnique: a default on a unique property (D13 refuses it now).
	rolloutUnique = `{"type": "object", "properties": {"code": {"type": "string", "default": "X", "x-nava-unique": true},
		"note": {"type": "string", "default": "n"}}}`
)

// rolloutUp is the Up section of the rollout migration, as goose runs it:
// one transaction, every statement in order.
func rolloutUp(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(db.Migrations(), schemaBackfillRollout)
	if err != nil {
		t.Fatal(err)
	}
	up, _, ok := strings.Cut(string(b), "-- +goose Down")
	if !ok {
		t.Fatal("no Down section")
	}
	return up
}

// runRollout runs the rollout's Up as nava_migrator in one transaction and
// returns the NOTICE messages it raised.
func runRollout(t *testing.T) []string {
	t.Helper()
	var (
		mu      sync.Mutex
		notices []string
	)
	cfg := roleConfig(t, "nava_migrator", "nava")
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		mu.Lock()
		defer mu.Unlock()
		notices = append(notices, n.Message)
	}
	conn := connect(t, cfg)
	if err := pgx.BeginFunc(context.Background(), conn, func(tx pgx.Tx) error {
		_, err := tx.Exec(context.Background(), rolloutUp(t))
		return err
	}); err != nil {
		t.Fatalf("rollout: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(notices)
}

// seedRolloutType writes, as the superuser, tenant's entity type key at
// version 1 with schema, and returns its id.
func seedRolloutType(t *testing.T, super *pgx.Conn, tenant, key, prefix, schema string) string {
	t.Helper()
	exec(t, super, insertTypeNewID, tenant, key, prefix)
	id := queryString(t, super, "SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = $2", tenant, key)
	exec(t, super, insertVersion, tenant, newUUID(t, super), id, 1, schema, schemaSum(schema), "initial", tenant)
	return id
}

// Acceptance: two tenants, each with a defaulted type, a type with no default
// (seedEntityTypes' ticket) and a default + unique type, get exactly one job
// and one outbox row per defaulted type, none for the others, and one skip
// notice per default + unique type; every inserted row's tenant_id is its
// entity definition's (CI-3 (7)); a second run inserts nothing.
func TestP1010RolloutQueuesOneJobPerDefaultedType(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	defaulted, unique := map[string]string{}, map[string]string{}
	for _, tenant := range []string{tenantA, tenantB} {
		defaulted[tenant] = seedRolloutType(t, super, tenant, "incident", "INC", rolloutDefaulted)
		unique[tenant] = seedRolloutType(t, super, tenant, "asset", "AST", rolloutUnique)
	}
	recordsBefore := queryString(t, super, "SELECT count(*)::text || ':' || coalesce(max(version), 0)::text FROM records")
	otherOutbox := queryString(t, super, "SELECT count(*)::text FROM outbox WHERE kind <> 'schema.backfill'")

	notices := runRollout(t)

	var wantNotices []string
	for _, tenant := range []string{tenantA, tenantB} {
		wantNotices = append(wantNotices, "schema_backfill_skipped tenant="+tenant+" type="+unique[tenant])
	}
	wantNotices = append(wantNotices, "schema_backfill_rollout queued=2 skipped=2")
	if !slices.Equal(notices, wantNotices) {
		t.Errorf("notices:\n got %q\nwant %q", notices, wantNotices)
	}

	check := func(run string) {
		t.Helper()
		for _, tenant := range []string{tenantA, tenantB} {
			def := defaulted[tenant]
			if got := queryString(t, super, `SELECT coalesce(string_agg(entity_definition_id::text || ' v' || target_version || ' ' || state || ' ' ||
					array_to_string(default_keys, ',') || ' ' || (causation_audit_id IS NULL)::text, ';'), '')
				FROM schema_backfills WHERE tenant_id = $1`, tenant); got != def+" v1 pending channel,priority true" {
				t.Errorf("%s: tenant %s jobs = %q, want only %s's", run, tenant, got, def)
			}
			if got := queryString(t, super, `SELECT coalesce(string_agg(dedupe_key || ' ' || target_seq || ' ' || payload::text || ' ' ||
					(target_key = 'schema.backfill:' || id::text)::text || ' ' || (causation_id = id)::text || ' ' || causation_depth, ';'), '')
				FROM outbox WHERE tenant_id = $1 AND kind = 'schema.backfill'`, tenant); got !=
				"schema-backfill:"+def+":1 1 {\"entity_definition_id\": \""+def+"\"} true true 0" {
				t.Errorf("%s: tenant %s outbox rows = %q", run, tenant, got)
			}
		}
		// CI-3 (7): every row's tenant is its entity definition's.
		for _, q := range []string{
			`SELECT count(*)::text FROM schema_backfills j JOIN entity_definitions d ON d.id = j.entity_definition_id WHERE d.tenant_id <> j.tenant_id`,
			`SELECT count(*)::text FROM outbox o JOIN entity_definitions d ON d.id = (o.payload ->> 'entity_definition_id')::uuid
				WHERE o.kind = 'schema.backfill' AND d.tenant_id <> o.tenant_id`,
			`SELECT count(*)::text FROM outbox o WHERE o.kind = 'schema.backfill' AND NOT EXISTS (SELECT 1 FROM schema_backfills j
				WHERE j.tenant_id = o.tenant_id AND j.entity_definition_id = (o.payload ->> 'entity_definition_id')::uuid AND j.target_version = o.target_seq)`,
		} {
			if got := queryString(t, super, q); got != "0" {
				t.Errorf("%s: %s = %s, want 0", run, q, got)
			}
		}
		if got := queryString(t, super, "SELECT count(*)::text FROM outbox WHERE kind <> 'schema.backfill'"); got != otherOutbox {
			t.Errorf("%s: other outbox rows = %s, want %s untouched", run, got, otherOutbox)
		}
		if got := queryString(t, super, "SELECT count(*)::text || ':' || coalesce(max(version), 0)::text FROM records"); got != recordsBefore {
			t.Errorf("%s: records = %s, want %s untouched", run, got, recordsBefore)
		}
		if got := queryString(t, super, `SELECT count(*)::text FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)`); got != "0" {
			t.Errorf("%s: nava_migrator policies left = %s", run, got)
		}
	}
	check("first run")

	again := runRollout(t)
	if want := append(wantNotices[:2:2], "schema_backfill_rollout queued=0 skipped=2"); !slices.Equal(again, want) {
		t.Errorf("second run notices:\n got %q\nwant %q", again, want)
	}
	check("second run")
}

// Acceptance: the migration's SQL neither updates nor reads records.
func TestP1010RolloutNeverTouchesRecords(t *testing.T) {
	b, err := fs.ReadFile(db.Migrations(), schemaBackfillRollout)
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`(?i)\b(update|from|join|into|table)\s+(only\s+)?(public\.)?records\b`).Find(b); m != nil {
		t.Errorf("the rollout migration touches records: %q", m)
	}
}
