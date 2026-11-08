//go:build integration

package dbtest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
)

// P5-10b (TICK-450): ADR-004's audit_log, audit_chain_head and
// record_write_keys, and the nava_erasure role (ADR-001-A6.1). The tests
// name the ADR rule or CI case each one proves.

const (
	// insertEntry is a valid record entry through the R10 columns: tenant $1,
	// id $2, subject $3, version_after $4. It carries a diff, its salt and
	// its hash, so the row can be sealed and erased.
	insertEntry = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_system, source, diff, diff_salt, diff_sha256)
		VALUES ($1, $2, 'record', 'update', 'record', $3, $4, 'workflow', 'workflow',
		        '[{"op":"add","path":"/a","value":1}]', decode('00112233445566778899aabbccddeeff', 'hex'), sha256('d'))`
	// sealEntry is the R11 seal transition: tenant $1, id $2.
	sealEntry = `UPDATE audit_log SET seq = 1, prev_hash = sha256('p'), entry_hash = sha256('e'), sealed_at = now()
		WHERE tenant_id = $1 AND id = $2`
	// eraseEntry is the R11 erase transition: tenant $1, id $2, erasure_ref $3.
	eraseEntry = `UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $3
		WHERE tenant_id = $1 AND id = $2`
)

// auditWriters are the CI-A10 roles; nava_erasure is added where R23 makes
// it a writer.
func auditWriters() []string {
	return []string{"nava_app", "nava_worker", "nava_onboard", "nava_groupsync"}
}

// platformLogins are the manifest roles that may connect to the platform
// database and use schema public, the owner excluded. (pgbouncer_auth
// connects but cannot even resolve a name in public.)
func platformLogins(t *testing.T) []string {
	t.Helper()
	m := manifest(t)
	var out []string
	for _, name := range m.RoleNames() {
		r := m.Roles[name]
		if name != m.Owner() && slices.Contains(r.Database, "CONNECT") && slices.Contains(r.Schemas["public"], "USAGE") {
			out = append(out, name)
		}
	}
	return out
}

// appendEntry inserts insertEntry as role for tenant and returns its id.
func appendEntry(t *testing.T, conn *pgx.Conn, tenant string, version int) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := execIn(conn, tenant, insertEntry, tenant, id, pipelineRecord(tenant), version); err != nil {
		t.Fatalf("append entry as %s: %v", conn.Config().User, err)
	}
	return id
}

func pgErrDetail(err error) (code, constraint string) {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code, pe.ConstraintName
	}
	return "", ""
}

// R1, R28: the columns, the partitioning and the key shape.
func TestP510bAuditLogShape(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)

	if got := queryString(t, super, `SELECT p.partstrat::text || ':' || pg_get_partkeydef('audit_log'::regclass)
		FROM pg_partitioned_table p WHERE p.partrelid = 'audit_log'::regclass`); got != "r:RANGE (occurred_at)" {
		t.Errorf("partitioning = %q, want RANGE (occurred_at)", got)
	}
	if got := queryString(t, super, `SELECT string_agg(c.relname || '=' || pg_get_expr(c.relpartbound, c.oid), ',')
		FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid WHERE i.inhparent = 'audit_log'::regclass`); got != "audit_log_default=DEFAULT" {
		t.Errorf("partitions = %q, want the one DEFAULT partition", got)
	}

	cols := queryString(t, super, `SELECT string_agg(attname || ' ' || format_type(atttypid, atttypmod)
		|| CASE WHEN attnotnull THEN ' not null' ELSE '' END
		|| coalesce(' default ' || pg_get_expr(d.adbin, d.adrelid), ''), ', ' ORDER BY attnum)
		FROM pg_attribute a LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE attrelid = 'audit_log'::regclass AND attnum > 0 AND NOT attisdropped`)
	want := strings.Join([]string{
		"tenant_id uuid not null", "id uuid not null", "occurred_at timestamp with time zone not null default now()",
		"txid xid8 not null default pg_current_xact_id()", "domain text not null", "action text not null",
		"subject_type text not null", "subject_id uuid not null", "entity_definition_id uuid",
		"version_before bigint", "version_after bigint", "actor_principal_id uuid", "actor_system text",
		"source text not null", "source_ref text", "change_set_id uuid", "causation_id uuid",
		"causation_depth smallint not null default 0", "request_id text", "backfilled boolean not null default false",
		"content_sha256 bytea", "diff_sha256 bytea", "diff_truncated boolean not null default false", "diff json",
		"diff_salt bytea", "seq bigint", "prev_hash bytea", "entry_hash bytea", "sealed_at timestamp with time zone",
		"erased_at timestamp with time zone", "erasure_ref uuid",
	}, ", ")
	if cols != want {
		t.Errorf("audit_log columns:\n%s\nwant:\n%s", cols, want)
	}

	// The index set on the parent: R28 plus the PK and the R2 key, nothing more.
	idx := queryString(t, super, `SELECT string_agg(pg_get_indexdef(indexrelid), E'\n' ORDER BY indexrelid::regclass::text)
		FROM pg_index WHERE indrelid = 'audit_log'::regclass`)
	wantIdx := strings.Join([]string{
		"CREATE INDEX audit_log_change_set ON ONLY public.audit_log USING btree (tenant_id, change_set_id) WHERE (change_set_id IS NOT NULL)",
		"CREATE UNIQUE INDEX audit_log_commit_key ON ONLY public.audit_log USING btree (tenant_id, occurred_at, txid, subject_type, subject_id, version_after) NULLS NOT DISTINCT",
		"CREATE UNIQUE INDEX audit_log_pkey ON ONLY public.audit_log USING btree (tenant_id, occurred_at, id)",
		"CREATE INDEX audit_log_seq ON ONLY public.audit_log USING btree (tenant_id, seq) WHERE (seq IS NOT NULL)",
		"CREATE INDEX audit_log_subject ON ONLY public.audit_log USING btree (tenant_id, subject_type, subject_id, occurred_at DESC)",
		"CREATE INDEX audit_log_tenant_time ON ONLY public.audit_log USING btree (tenant_id, occurred_at DESC, id DESC)",
		"CREATE INDEX audit_log_unsealed ON ONLY public.audit_log USING btree (tenant_id, occurred_at, id) WHERE (sealed_at IS NULL)",
	}, "\n")
	if idx != wantIdx {
		t.Errorf("audit_log indexes:\n%s\nwant:\n%s", idx, wantIdx)
	}
	// The partition carries every parent index, each named for its parent.
	if got := queryString(t, super, `SELECT string_agg(c.relname || '<' || p.relname, ',' ORDER BY c.relname)
		FROM pg_index x JOIN pg_class c ON c.oid = x.indexrelid
		LEFT JOIN pg_inherits i ON i.inhrelid = c.oid LEFT JOIN pg_class p ON p.oid = i.inhparent
		WHERE x.indrelid = 'audit_log_default'::regclass`); got != strings.Join([]string{
		"audit_log_default_change_set<audit_log_change_set", "audit_log_default_commit_key<audit_log_commit_key",
		"audit_log_default_pkey<audit_log_pkey", "audit_log_default_seq<audit_log_seq",
		"audit_log_default_subject<audit_log_subject", "audit_log_default_tenant_time<audit_log_tenant_time",
		"audit_log_default_unsealed<audit_log_unsealed"}, ",") {
		t.Errorf("audit_log_default indexes = %s", got)
	}

	// D-5-3, TICK-626: the unique index name the store matches on, per
	// principal; the tenant-wide primary key is gone.
	if got := queryString(t, super, `SELECT pg_get_indexdef(indexrelid) || ' valid=' || indisvalid FROM pg_index
		WHERE indexrelid = 'record_write_keys_actor_key'::regclass`); got != "CREATE UNIQUE INDEX record_write_keys_actor_key ON public.record_write_keys USING btree (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT valid=true" {
		t.Errorf("record_write_keys_actor_key = %q", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_constraint
		WHERE conrelid = 'record_write_keys'::regclass AND conname = 'record_write_keys_pkey'`); got != "0" {
		t.Errorf("record_write_keys_pkey still exists")
	}
	if got := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid = 'audit_chain_head'::regclass AND contype = 'p'`); got != "PRIMARY KEY (tenant_id)" {
		t.Errorf("audit_chain_head primary key = %q", got)
	}
}

// R2: the subject_type CHECK equals internal/audit/subjects.go exactly, and
// the other closed vocabularies are the ADR's (with E1's cascade actions).
func TestP510bSubjectTypesMirrorRegistry(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	values := func(con string) []string {
		def := queryString(t, super, `SELECT pg_get_constraintdef(oid) FROM pg_constraint
			WHERE conrelid = 'audit_log'::regclass AND conname = $1`, con)
		var out []string
		for _, m := range regexp.MustCompile(`'([^']*)'::text`).FindAllStringSubmatch(def, -1) {
			out = append(out, m[1])
		}
		slices.Sort(out)
		return out
	}
	var registry []string
	for _, s := range audit.SubjectTypes() {
		registry = append(registry, string(s))
	}
	if got := values("audit_log_subject_type_check"); !slices.Equal(got, registry) {
		t.Errorf("subject_type CHECK = %v, internal/audit/subjects.go = %v", got, registry)
	}
	for con, want := range map[string][]string{
		"audit_log_domain_check": {"audit", "config", "record"},
		"audit_log_action_check": {"cascade_cycle", "cascade_too_deep", "create", "delete", "disable", "enable",
			"erase", "export", "promote", "publish", "reseed", "rollback", "state_change", "update"},
		"audit_log_source_check": {"ai", "change_set", "function", "onboarding", "plugin", "rule", "system", "user", "workflow"},
	} {
		if got := values(con); !slices.Equal(got, want) {
			t.Errorf("%s = %v, want %v", con, got, want)
		}
	}
}

// R1, R2, TICK-454: the row checks refuse what the canonical form or the
// chain could not hold.
func TestP510bAuditLogChecks(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	base := map[string]string{
		"domain": "'record'", "action": "'update'", "subject_type": "'record'", "subject_id": "gen_random_uuid()",
		"actor_system": "'workflow'", "source": "'workflow'",
	}
	insert := func(over map[string]string) error {
		cols := []string{"tenant_id", "id"}
		vals := []string{"$1", "gen_random_uuid()"}
		merged := map[string]string{}
		for k, v := range base {
			merged[k] = v
		}
		for k, v := range over {
			merged[k] = v
		}
		keys := slices.Sorted(func(yield func(string) bool) {
			for k := range merged {
				if !yield(k) {
					return
				}
			}
		})
		for _, k := range keys {
			if merged[k] == "" {
				continue
			}
			cols = append(cols, k)
			vals = append(vals, merged[k])
		}
		_, err := execIn(app, tenantA, "INSERT INTO audit_log ("+strings.Join(cols, ", ")+") VALUES ("+strings.Join(vals, ", ")+")", tenantA)
		return err
	}
	if err := insert(nil); err != nil {
		t.Fatalf("valid entry: %v", err)
	}
	cases := []struct {
		name, constraint string
		over             map[string]string
	}{
		{"unknown subject type", "audit_log_subject_type_check", map[string]string{"subject_type": "'tenant'"}},
		{"unknown action", "audit_log_action_check", map[string]string{"action": "'forge'"}},
		{"unknown source", "audit_log_source_check", map[string]string{"source": "'cron'"}},
		{"unknown domain", "audit_log_domain_check", map[string]string{"domain": "'other'"}},
		{"empty actor_system", "audit_log_text_nonempty_check", map[string]string{"actor_system": "''"}},
		{"empty source_ref", "audit_log_text_nonempty_check", map[string]string{"source_ref": "''"}},
		{"empty request_id", "audit_log_text_nonempty_check", map[string]string{"request_id": "''"}},
		{"no actor", "audit_log_actor_check", map[string]string{"actor_system": ""}},
		{"two actors", "audit_log_actor_check", map[string]string{"actor_principal_id": "$1"}},
		{"negative causation depth", "audit_log_causation_depth_check", map[string]string{"causation_depth": "-1"}},
		{"config entry without content hash", "audit_log_config_content_check",
			map[string]string{"domain": "'config'", "action": "'publish'", "subject_type": "'entity_definition'"}},
		{"short content hash", "audit_log_digest_length_check", map[string]string{"content_sha256": "'\\x00'::bytea"}},
		{"short salt", "audit_log_diff_salt_length_check",
			map[string]string{"diff": "'[]'", "diff_salt": "'\\x00'::bytea"}},
		{"diff without salt", "audit_log_diff_salt_pair_check", map[string]string{"diff": "'[]'"}},
		{"diff over 65,536 bytes", "audit_log_diff_size_check",
			map[string]string{"diff": "('[\"' || repeat('x', 65535) || '\"]')::json", "diff_salt": "decode(repeat('00', 16), 'hex')"}},
	}
	for _, c := range cases {
		err := insert(c.over)
		if code, con := pgErrDetail(err); code != "23514" || con != c.constraint {
			t.Errorf("%s: err = %v, want 23514 on %s", c.name, err, c.constraint)
		}
	}
	// A config delete needs no content hash; a diff at exactly the limit fits.
	if err := insert(map[string]string{"domain": "'config'", "action": "'delete'", "subject_type": "'entity_definition'"}); err != nil {
		t.Errorf("config delete without content hash: %v", err)
	}
	if err := insert(map[string]string{"diff": "('[\"' || repeat('x', 65532) || '\"]')::json", "diff_salt": "decode(repeat('00', 16), 'hex')"}); err != nil {
		t.Errorf("diff of exactly 65,536 bytes: %v", err)
	}
	// The foreign keys are composite: an actor or type of another tenant fails.
	for col, con := range map[string]string{"actor_principal_id": "audit_log_actor_principal_fkey", "entity_definition_id": "audit_log_entity_definition_fkey"} {
		over := map[string]string{col: "'" + tenantB + "'"}
		if col == "actor_principal_id" {
			over["actor_system"] = ""
		}
		if code, got := pgErrDetail(insert(over)); code != "23503" || got != con {
			t.Errorf("%s of tenant B: %s %s, want 23503 on %s", col, code, got, con)
		}
	}
}

// R2: at most one entry per (transaction, subject, version_after), with
// NULL versions equal; a new transaction is a new key.
func TestP510bCommitKey(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	app := asRole(t, "nava_app")
	subject := pipelineRecord(tenantA)
	for _, version := range []any{2, nil} {
		err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
			for range 2 {
				if _, err := tx.Exec(context.Background(), insertEntry, tenantA, uuid.NewString(), subject, version); err != nil {
					return err
				}
			}
			return nil
		})
		// The violation names the partition's copy of the key.
		if code, con := pgErrDetail(err); code != "23505" || con != "audit_log_default_commit_key" {
			t.Errorf("two entries for one subject and version %v in one transaction: %v, want 23505 on audit_log_default_commit_key", version, err)
		}
	}
	appendEntry(t, app, tenantA, 2)
	appendEntry(t, app, tenantA, 2)
	appendEntry(t, app, tenantA, 3)
}

// S1 finding F1 and its TICK-460 / Finding 6 catalogue check: audit_log and
// every leaf of pg_partition_tree carry the occurred_at override,
// n_distinct = 1 since Finding 5's re-measure (-0.0001 tied at 10k
// tenants). The query returns the relations that do not, and the test
// proves it would catch a partition created without the setting.
func TestP510bOccurredAtNDistinct(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	const missing = `SELECT coalesce(string_agg(t.relid::regclass::text, ',' ORDER BY t.relid::regclass::text), '')
		FROM pg_partition_tree('audit_log') t
		JOIN pg_attribute a ON a.attrelid = t.relid AND a.attname = 'occurred_at'
		WHERE NOT coalesce('n_distinct=1' = ANY (a.attoptions), false)`
	if got := queryString(t, super, missing); got != "" {
		t.Errorf("relations without n_distinct = 1 on occurred_at: %s", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_partition_tree('audit_log')`); got != "2" {
		t.Errorf("pg_partition_tree('audit_log') has %s relations, want the parent and the DEFAULT partition", got)
	}

	ctx := context.Background()
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `CREATE TABLE audit_log_2099 PARTITION OF audit_log
		FOR VALUES FROM ('2099-01-01') TO ('2100-01-01')`); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := tx.QueryRow(ctx, missing).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "audit_log_2099" {
		t.Errorf("a new partition without the setting: check reported %q, want audit_log_2099 (partitions do not inherit it)", got)
	}
}

// R9 and CI-1 on the DEFAULT partition: ENABLE + FORCE, no grant, and a
// direct statement is denied to every runtime role.
func TestP510bDefaultPartitionLocked(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	if got := queryString(t, super, `SELECT relrowsecurity::text || relforcerowsecurity::text || coalesce(relacl::text, '')
		FROM pg_class WHERE oid = 'audit_log_default'::regclass`); got != "truetrue" {
		t.Errorf("audit_log_default RLS enabled/forced and ACL = %s, want truetrue and no grant", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_policy WHERE polrelid = 'audit_log_default'::regclass`); got != "0" {
		t.Errorf("audit_log_default has %s policies, want 0", got)
	}
	for _, role := range platformLogins(t) {
		conn := asRole(t, role)
		for _, q := range []string{
			"SELECT count(*) FROM audit_log_default",
			"SELECT tenant_id FROM audit_log_default",
			"UPDATE audit_log_default SET seq = seq",
			"DELETE FROM audit_log_default",
			`INSERT INTO audit_log_default (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			 VALUES ($1, gen_random_uuid(), 'record', 'create', 'record', $1, 'system', 'system')`,
		} {
			var args []any
			if strings.Contains(q, "$1") {
				args = append(args, tenantA)
			}
			_, err := execIn(conn, tenantA, q, args...)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
}

// CI-A10 (INSERT, R10): for every writer role, an INSERT that names
// occurred_at, txid, backfilled, a seal or an erasure column is denied by
// the column grant, whatever the value.
func TestP510bProtectedInsertColumnsDenied(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	protected := map[string]string{
		"occurred_at": "now()", "txid": "pg_current_xact_id()", "backfilled": "false",
		"seq": "1", "prev_hash": "sha256('p')", "entry_hash": "sha256('e')", "sealed_at": "now()",
		"erased_at": "now()", "erasure_ref": "gen_random_uuid()",
	}
	for _, role := range append(auditWriters(), "nava_erasure") {
		conn := asRole(t, role)
		for col, val := range protected {
			q := `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, ` + col + `)
				VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, 'workflow', 'workflow', ` + val + `)`
			_, err := execIn(conn, tenantA, q, tenantA, pipelineRecord(tenantA))
			wantSQLState(t, role+" sets "+col, err, "42501")
		}
	}
	// The R10 grant itself: app, worker and erasure append. P5-11 generates
	// nava_onboard's grants, but its pinned policy admits only its own
	// provisioning tenant, so it cannot append for active tenant A (42501);
	// nava_groupsync writes no audited table and has no grant at all.
	for _, role := range []string{"nava_app", "nava_worker", "nava_erasure"} {
		appendEntry(t, asRole(t, role), tenantA, 1)
		exec(t, super, "DELETE FROM audit_log WHERE tenant_id = $1", tenantA)
	}
	for _, role := range []string{"nava_onboard", "nava_groupsync"} {
		_, err := execIn(asRole(t, role), tenantA, insertEntry, tenantA, uuid.NewString(), pipelineRecord(tenantA), 1)
		wantSQLState(t, role+" appends", err, "42501")
	}
}

// R10's trigger, independent of the grants: it asserts, for every role but
// nava_migrator, even the superuser that bypasses grants and RLS. The
// migrator may backfill (occurred_at in the past, backfilled = true) but not
// insert a sealed or erased row.
func TestP510bInsertAssertTrigger(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	const ins = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, %s)
		VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', gen_random_uuid(), 'workflow', 'workflow', %s)`
	bad := map[string]string{
		"occurred_at": "now() - interval '1 hour'", "txid": "'1'::xid8", "backfilled": "true",
		"seq, prev_hash, entry_hash, sealed_at": "1, sha256('p'), sha256('e'), now()",
		"erased_at, erasure_ref":                "now(), gen_random_uuid()",
	}
	for cols, vals := range bad {
		_, err := super.Exec(context.Background(), fmt.Sprintf(ins, cols, vals), tenantA)
		wantSQLState(t, "superuser sets "+cols, err, "42501")
	}

	// The migrator, through a policy of its own in a rolled-back transaction
	// (the P5-15 pattern), passes the first assert and not the second.
	mig := asRole(t, "nava_migrator")
	ctx := context.Background()
	run := func(cols, vals string) error {
		tx, err := mig.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		for _, q := range []string{
			"CREATE POLICY audit_log_test_migrator ON audit_log FOR INSERT TO nava_migrator WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)",
			"SELECT set_config('app.tenant_id', '" + tenantA + "', true)",
		} {
			if _, err := tx.Exec(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(ins, cols, vals), tenantA)
		return err
	}
	if err := run("occurred_at, backfilled", "now() - interval '1 year', true"); err != nil {
		t.Errorf("migrator backfill: %v", err)
	}
	wantSQLState(t, "migrator inserts a sealed row", run("seq, prev_hash, entry_hash, sealed_at", "1, sha256('p'), sha256('e'), now()"), "42501")
	wantSQLState(t, "migrator inserts an erased row", run("erased_at, erasure_ref", "now(), gen_random_uuid()"), "42501")
}

// CI-A10 (UPDATE, DELETE) and CI-A7 (the erase half), R11: exactly the
// two transitions. Seal is nava_worker's only, erase nava_erasure's only;
// a sealed row's content never changes; a second seal or erase fails;
// nobody deletes.
func TestP510bUpdateTransitions(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	app, worker, erasure := asRole(t, "nava_app"), asRole(t, "nava_worker"), asRole(t, "nava_erasure")
	id := appendEntry(t, app, tenantA, 1)
	ref := uuid.NewString()

	updates := map[string]string{
		"content":    "UPDATE audit_log SET action = 'delete' WHERE tenant_id = $1 AND id = $2",
		"tenant":     "UPDATE audit_log SET tenant_id = tenant_id WHERE tenant_id = $1 AND id = $2",
		"seal":       sealEntry,
		"seq only":   "UPDATE audit_log SET seq = 1 WHERE tenant_id = $1 AND id = $2",
		"erase":      eraseEntry,
		"diff only":  "UPDATE audit_log SET diff = NULL WHERE tenant_id = $1 AND id = $2",
		"truncated":  "UPDATE audit_log SET diff_truncated = true WHERE tenant_id = $1 AND id = $2",
		"occurred":   "UPDATE audit_log SET occurred_at = now() WHERE tenant_id = $1 AND id = $2",
		"backfilled": "UPDATE audit_log SET backfilled = true WHERE tenant_id = $1 AND id = $2",
	}
	run := func(conn *pgx.Conn, q string) (int64, error) {
		args := []any{tenantA, id}
		if strings.Contains(q, "$3") {
			args = append(args, ref)
		}
		return execIn(conn, tenantA, q, args...)
	}
	// Nothing but the role's own transition, for every writer.
	for _, role := range append(auditWriters(), "nava_erasure") {
		conn := asRole(t, role)
		for name, q := range updates {
			if (role == "nava_worker" && name == "seal") || (role == "nava_erasure" && name == "erase") {
				continue
			}
			_, err := run(conn, q)
			wantSQLState(t, role+" update "+name, err, "42501")
		}
		_, err := execIn(conn, tenantA, "DELETE FROM audit_log WHERE tenant_id = $1", tenantA)
		wantSQLState(t, role+" delete", err, "42501")
	}
	// The trigger, not only the grants: the superuser is refused too.
	for _, name := range []string{"content", "seal", "erase"} {
		_, err := super.Exec(context.Background(), strings.Replace(updates[name], "$3", "'"+ref+"'", 1), tenantA, id)
		wantSQLState(t, "superuser update "+name, err, "42501")
	}

	// Seal, once.
	if n, err := run(worker, sealEntry); err != nil || n != 1 {
		t.Fatalf("seal: %d rows, %v", n, err)
	}
	if _, err := run(worker, sealEntry); sqlState(err) != "42501" {
		t.Errorf("second seal: %v, want 42501", err)
	}
	// A sealed row's content cannot change, by any role.
	if _, err := super.Exec(context.Background(), updates["content"], tenantA, id); sqlState(err) != "42501" {
		t.Errorf("superuser rewrites a sealed row: %v, want 42501", err)
	}
	// Erase a sealed row, once; the seal and the envelope survive it.
	before := queryString(t, super, "SELECT concat_ws('|', seq, encode(entry_hash, 'hex'), encode(diff_sha256, 'hex'), diff_truncated) FROM audit_log WHERE id = $1", id)
	if n, err := run(erasure, eraseEntry); err != nil || n != 1 {
		t.Fatalf("erase: %d rows, %v", n, err)
	}
	if after := queryString(t, super, "SELECT concat_ws('|', seq, encode(entry_hash, 'hex'), encode(diff_sha256, 'hex'), diff_truncated) FROM audit_log WHERE id = $1", id); after != before {
		t.Errorf("erase changed the seal or envelope: %s -> %s", before, after)
	}
	if got := queryString(t, super, "SELECT (diff IS NULL AND diff_salt IS NULL AND erased_at IS NOT NULL AND erasure_ref = $2)::text FROM audit_log WHERE id = $1", id, ref); got != "true" {
		t.Errorf("erased row: %s, want diff and salt NULL, erased_at and erasure_ref set", got)
	}
	if _, err := run(erasure, eraseEntry); sqlState(err) != "42501" {
		t.Errorf("second erase: %v, want 42501", err)
	}
	// Erase applies to an unsealed row too, and never to a row with no diff.
	unsealed := appendEntry(t, app, tenantA, 2)
	if n, err := execIn(erasure, tenantA, eraseEntry, tenantA, unsealed, ref); err != nil || n != 1 {
		t.Errorf("erase an unsealed row: %d rows, %v", n, err)
	}
	noDiff := uuid.NewString()
	if _, err := execIn(app, tenantA, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
		VALUES ($1, $2, 'record', 'update', 'record', gen_random_uuid(), 'workflow', 'workflow')`, tenantA, noDiff); err != nil {
		t.Fatal(err)
	}
	_, err := execIn(erasure, tenantA, eraseEntry, tenantA, noDiff, ref)
	wantSQLState(t, "erase a row with no diff", err, "42501")
}

// CI-A7: nava_erasure reads only the R23 columns, never diff or diff_salt.
func TestP510bErasureCannotReadDiff(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	entry := appendEntry(t, asRole(t, "nava_app"), tenantA, 1)
	erasure := asRole(t, "nava_erasure")
	for _, q := range []string{"SELECT diff FROM audit_log", "SELECT diff_salt FROM audit_log", "SELECT * FROM audit_log",
		"SELECT count(*) FROM audit_log WHERE diff IS NOT NULL", "SELECT diff_sha256 FROM audit_log", "SELECT seq FROM audit_log"} {
		_, err := execIn(erasure, tenantA, q)
		wantSQLState(t, "nava_erasure: "+q, err, "42501")
	}
	var n int
	if err := inTenantTx(erasure, tenantA, func(tx pgx.Tx) error {
		// The test's own entry: seedTenants and seedPipeline also write the
		// seeded records' entries (P5-13c).
		return tx.QueryRow(context.Background(), `SELECT count(*) FROM (SELECT tenant_id, occurred_at, id, subject_type,
			subject_id, erased_at, erasure_ref FROM audit_log WHERE id = $1::uuid) s`, entry).Scan(&n)
	}); err != nil || n != 1 {
		t.Errorf("nava_erasure reads its R23 columns: %d rows, %v", n, err)
	}
	// R14, A6.1: nothing else.
	for _, table := range []string{"audit_chain_head", "record_write_keys", "records"} {
		_, err := execIn(erasure, tenantA, "SELECT count(*) FROM "+table)
		wantSQLState(t, "nava_erasure reads "+table, err, "42501")
	}
}

// CI-A4 floor: with no tenant context, or an empty one, every role that may
// read the three tables reaches zero rows; with tenant A it reaches only A's.
func TestP510bNoTenantNoRows(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	var entries []string // the test's own audit_log entries, one per tenant
	for _, tenant := range []string{tenantA, tenantB} {
		id := appendEntry(t, asRole(t, "nava_app"), tenant, 1)
		entries = append(entries, id)
		exec(t, super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", tenant)
		exec(t, super, `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id,
			number, version, schema_version, audit_entry_id) VALUES ($1, 'k', $1, 'create', $2, 'TCK0000001', 1, 1, $3)`,
			tenant, pipelineRecord(tenant), id)
	}
	probes := map[string]string{
		// Scoped to the test's own entries: seedTenants and seedPipeline
		// also write the seeded records' entries (P5-13c).
		"audit_log":         "SELECT count(*) FROM (SELECT tenant_id FROM audit_log WHERE tenant_id::text = ANY ($1::text[]) AND id::text = ANY ($2::text[])) s",
		"audit_chain_head":  "SELECT count(*) FROM audit_chain_head WHERE tenant_id::text = ANY ($1::text[])",
		"record_write_keys": "SELECT count(*) FROM record_write_keys WHERE tenant_id::text = ANY ($1::text[])",
	}
	readers := map[string][]string{
		"audit_log":         {"nava_app", "nava_worker", "nava_erasure", "nava_audit_reader"},
		"audit_chain_head":  {"nava_app", "nava_worker"},
		"record_write_keys": {"nava_app", "nava_worker"},
	}
	both := []string{tenantA, tenantB}
	for table, roles := range readers {
		for _, role := range roles {
			conn := asRole(t, role)
			for _, mode := range []string{"unset", "empty", "A"} {
				var n int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if mode != "unset" {
						v := map[string]string{"A": tenantA, "empty": ""}[mode]
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
							return err
						}
					}
					args := []any{both}
					if table == "audit_log" {
						args = append(args, entries)
					}
					return tx.QueryRow(context.Background(), probes[table], args...).Scan(&n)
				})
				want := map[string]int{"unset": 0, "empty": 0, "A": 1}[mode]
				if err != nil || n != want {
					t.Errorf("%s on %s, tenant %s: %d rows, %v; want %d", role, table, mode, n, err, want)
				}
			}
		}
	}
	// Writes with tenant A set cannot land in tenant B.
	_, err := execIn(asRole(t, "nava_app"), tenantA, insertEntry, tenantB, uuid.NewString(), pipelineRecord(tenantB), 9)
	wantSQLState(t, "nava_app appends for tenant B", err, "42501")
	_, err = execIn(asRole(t, "nava_worker"), tenantA, `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id,
		action, record_id, number, version, schema_version, audit_entry_id) VALUES ($1, 'x', $1, 'create', $2, 'n', 1, 1, gen_random_uuid())`,
		tenantB, pipelineRecord(tenantB))
	wantSQLState(t, "nava_worker writes a key for tenant B", err, "42501")
	if n, err := execIn(asRole(t, "nava_worker"), tenantA, "UPDATE audit_chain_head SET seq = seq + 1 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_worker bumps tenant B's head: %d rows, %v", n, err)
	}
}

// R14 [Sec] and CI-A10's head clause: column grants split the head between
// the sealer and the export lease; nobody inserts or deletes a head here.
func TestP510bChainHeadGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	exec(t, super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", tenantA)
	if got := queryString(t, super, `SELECT concat_ws('|', version, seq, encode(entry_hash, 'hex'), seal_interval_s)
		FROM audit_chain_head WHERE tenant_id = $1`, tenantA); got != "1|0|"+strings.Repeat("0", 64)+"|10" {
		t.Errorf("new head = %s, want version 1, seq 0, the genesis hash and a 10 s interval", got)
	}
	upd := func(role, set string) error {
		_, err := execIn(asRole(t, role), tenantA, "UPDATE audit_chain_head SET "+set+" WHERE tenant_id = $1", tenantA)
		return err
	}
	for _, set := range []string{"seq = 1", "entry_hash = sha256('x')", "version = 2", "seal_interval_s = 20", "tenant_id = tenant_id", "id = gen_random_uuid()"} {
		wantSQLState(t, "nava_app sets "+set, upd("nava_app", set), "42501")
	}
	for _, set := range []string{"export_lease_until = now()", "next_seal_outbox_id = gen_random_uuid()", "next_seal_at = now()"} {
		if err := upd("nava_app", set); err != nil {
			t.Errorf("nava_app sets %s: %v", set, err)
		}
	}
	for _, set := range []string{"export_lease_until = now()", "tenant_id = tenant_id", "id = gen_random_uuid()"} {
		wantSQLState(t, "nava_worker sets "+set, upd("nava_worker", set), "42501")
	}
	for _, set := range []string{"version = version + 1", "seq = 1", "entry_hash = sha256('x')", "next_seal_outbox_id = gen_random_uuid()",
		"next_seal_at = now()", "seal_interval_s = 20"} {
		if err := upd("nava_worker", set); err != nil {
			t.Errorf("nava_worker sets %s: %v", set, err)
		}
	}
	if code, con := pgErrDetail(upd("nava_worker", "seal_interval_s = 301")); code != "23514" || con != "audit_chain_head_seal_interval_check" {
		t.Errorf("interval past 300 s: %s %s", code, con)
	}
	for _, role := range platformLogins(t) {
		_, err := execIn(asRole(t, role), tenantA, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", tenantA)
		wantSQLState(t, role+" inserts a head", err, "42501")
		_, err = execIn(asRole(t, role), tenantA, "DELETE FROM audit_chain_head")
		wantSQLState(t, role+" deletes a head", err, "42501")
	}
}

// D-5-3: one key per tenant and principal (TICK-626), the duplicate named
// record_write_keys_actor_key, the same key free in another tenant, and no
// UPDATE or DELETE.
func TestP510bRecordWriteKeys(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	const ins = `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
		version, schema_version, audit_entry_id) VALUES ($1, $2, $1, $3, $4, 'TCK0000001', 1, 1, gen_random_uuid())`
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		key := "key-" + role
		for _, tenant := range []string{tenantA, tenantB} {
			if _, err := execIn(conn, tenant, ins, tenant, key, "create", pipelineRecord(tenant)); err != nil {
				t.Fatalf("%s claims %s in %s: %v", role, key, tenant, err)
			}
		}
		_, err := execIn(conn, tenantA, ins, tenantA, key, "update", pipelineRecord(tenantA))
		if code, con := pgErrDetail(err); code != "23505" || con != "record_write_keys_actor_key" {
			t.Errorf("%s claims %s twice: %v, want 23505 on record_write_keys_actor_key", role, key, err)
		}
		for _, q := range []string{"UPDATE record_write_keys SET version = 2", "DELETE FROM record_write_keys"} {
			_, err := execIn(conn, tenantA, q)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	app := asRole(t, "nava_app")
	for name, c := range map[string]struct{ key, action, constraint string }{
		"empty key":      {"", "create", "record_write_keys_idempotency_key_check"},
		"unknown action": {"k2", "delete", "record_write_keys_action_check"},
	} {
		_, err := execIn(app, tenantA, ins, tenantA, c.key, c.action, pipelineRecord(tenantA))
		if code, con := pgErrDetail(err); code != "23514" || con != c.constraint {
			t.Errorf("%s: %v, want 23514 on %s", name, err, c.constraint)
		}
	}
	// Composite foreign keys: tenant B's record (seedTenants' bare record,
	// id = tenant B) cannot back tenant A's key.
	_, err := execIn(app, tenantA, ins, tenantA, "k3", "create", tenantB)
	if code, con := pgErrDetail(err); code != "23503" || con != "record_write_keys_record_fkey" {
		t.Errorf("tenant B's record under tenant A's key: %v, want 23503 on record_write_keys_record_fkey", err)
	}
}

// CI-1 / CI-3 extended (A6.1) and the CI-A3 role assertions of the S1
// review (TICK-423, TICK-460): nava_erasure is in the manifest, logs in,
// belongs to nothing, has no members, owns nothing and is reached by no
// PUBLIC grant; only the migration login is nava_migrator; nava_migrator
// owns no SECURITY DEFINER function (R10 exempts current_user =
// 'nava_migrator', and a definer function would inherit that); and no role
// may SET session_replication_role, which would switch the R10 and R11
// triggers off.
func TestP510bErasureRoleAndMigratorAssertions(t *testing.T) {
	m := migratedCluster(t)
	if _, ok := m.Roles["nava_erasure"]; !ok {
		t.Fatal("nava_erasure is not in db/grants.yaml")
	}
	super := superNava(t)
	if got := queryString(t, super, `SELECT concat_ws(',', rolcanlogin::text, rolsuper::text, rolcreatedb::text, rolcreaterole::text,
		rolbypassrls::text, rolreplication::text)
		FROM pg_roles WHERE rolname = 'nava_erasure'`); got != "true,false,false,false,false,false" {
		t.Errorf("nava_erasure attributes = %s, want LOGIN only", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_auth_members
		WHERE member = 'nava_erasure'::regrole OR roleid = 'nava_erasure'::regrole`); got != "0" {
		t.Errorf("nava_erasure has %s memberships, want 0", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_auth_members WHERE roleid = 'nava_migrator'::regrole`); got != "0" {
		t.Errorf("%s roles are members of nava_migrator; only the migration login (nava_migrator itself) may act as it", got)
	}
	if got := queryString(t, super, `SELECT (
		(SELECT count(*) FROM pg_class WHERE relowner = 'nava_erasure'::regrole) +
		(SELECT count(*) FROM pg_proc WHERE proowner = 'nava_erasure'::regrole) +
		(SELECT count(*) FROM pg_namespace WHERE nspowner = 'nava_erasure'::regrole) +
		(SELECT count(*) FROM pg_type WHERE typowner = 'nava_erasure'::regrole) +
		(SELECT count(*) FROM pg_database WHERE datdba = 'nava_erasure'::regrole))::text`); got != "0" {
		t.Errorf("nava_erasure owns %s objects, want 0", got)
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(p.oid::regprocedure::text, ', '), '') FROM pg_proc p
		WHERE p.prosecdef AND p.proowner = 'nava_migrator'::regrole`); got != "" {
		t.Errorf("nava_migrator owns SECURITY DEFINER functions: %s", got)
	}
	if got := queryString(t, super, `SELECT count(*)::text FROM pg_parameter_acl WHERE parname = 'session_replication_role'`); got != "0" {
		t.Errorf("pg_parameter_acl has %s entries for session_replication_role, want 0", got)
	}
	// The R10 and R11 triggers are on, plain INVOKER and owned by the migrator.
	if got := queryString(t, super, `SELECT string_agg(t.tgname || ':' || t.tgenabled::text || ':' || p.prosecdef::text || ':' || pg_get_userbyid(p.proowner), ',' ORDER BY t.tgname)
		FROM pg_trigger t JOIN pg_proc p ON p.oid = t.tgfoid WHERE t.tgrelid = 'audit_log'::regclass AND NOT t.tgisinternal`); got != "audit_log_assert_insert:O:false:nava_migrator,audit_log_guard_update:O:false:nava_migrator" {
		t.Errorf("audit_log triggers = %s", got)
	}
	// Erasure can connect to the platform database and nothing else.
	for _, d := range serviceDatabases(m) {
		if _, err := connectErr(roleConfig(t, "nava_erasure", d)); err == nil {
			t.Errorf("nava_erasure connected to %s", d)
		}
	}
}
