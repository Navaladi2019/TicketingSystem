package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.yaml.in/yaml/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	platformdb "github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The S14 audit surface (ADR-004 CI-A4, CI-A5, CI-A10; PLAN-005 P5-19,
// TICK-436). The registry names each table: audit_log (through the parent),
// its DEFAULT partition, audit_chain_head, record_write_keys and
// tenant_auditor_group (whose four cases sit with the other identity tables
// in Paths). The SQL paths run as tenant A against tenant B, as every SQL
// path does; the DB paths take the ADR's direction, tenant B's admin,
// auditor and agent against tenant A's entries. Each path has a positive
// control, so a check that sees nothing cannot pass.

// Seeded audit rows (testdata/seed.sql).
const (
	auditEntryA1 = "0a000000-0000-4000-8000-0000000010a1"
	auditEntryB1 = "0b000000-0000-4000-8000-0000000010b1"
	auditEntryB2 = "0b000000-0000-4000-8000-0000000010b2"
	chainHeadB   = "0b000000-0000-4000-8000-0000000011b1"
	definitionA1 = "0a000000-0000-4000-8000-0000000006a1"
	principalA2  = "0a000000-0000-4000-8000-0000000003a2"
	// The auditors groups (TICK-434): a2 is in tenant A's, b1 in tenant B's.
	auditorGroupA = "0a000000-0000-4000-8000-0000000004a2"
	auditorGroupB = "0b000000-0000-4000-8000-0000000004b3"
)

// The audit callers of etIso (newETIso).
const (
	auditCallerAAuditor = "caller-a2-auditor"
	auditCallerAAgent   = "caller-a2-agent"
	auditCallerBAuditor = "caller-b1-auditor"
	auditCallerBAgent   = "caller-b1-agent"
)

// auditSealSeq is the seq a fixture entry is sealed with: far above any
// chain the suite seals, so after_seq can single it out.
const auditSealSeq = 900001

// auditEveryLogin is the report's Role for a DB path that runs as every
// platform login (auditManifestLogins).
const auditEveryLogin = "every login"

func auditPaths() []Path {
	sql := func(name, role string, expect Expect, query string, args ...any) Path {
		return Path{Name: name, Case: "CI-A4", Role: role, SQL: &SQLCall{Query: query, Args: args, Expect: expect}}
	}
	et := func(c, name string, fn func(context.Context, *etIso) error) Path {
		return Path{Name: name, Case: c, Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			clean := func() error {
				if err := e.cleanup(); err != nil {
					return err
				}
				return auditCleanup(ctx, e.super)
			}
			if err := clean(); err != nil { // rows a killed earlier run left
				return err
			}
			err = fn(ctx, e)
			if cerr := clean(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		// audit_log through the parent, by every key a caller could name.
		sql("audit_log: read tenant B's entry by id", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM audit_log WHERE id = $1 OR id = $2", auditEntryB1, auditEntryB2),
		sql("audit_log: the sealer reads tenant B's entry by id", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM audit_log WHERE id = $1", auditEntryB2),
		sql("audit_log: the erasure job reads tenant B's entry by its full key", "nava_erasure", OwnRowsOnly,
			"SELECT tenant_id FROM audit_log WHERE tenant_id = $1 AND id = $2", TenantB, auditEntryB1),
		// The diff and the salt are read only through read.Reader (TICK-541):
		// auditDirectSQLEveryRole asks the reader, unmasked, for tenant A's
		// entry from tenant B's transaction, tenant A's id forged.
		sql("audit_log: the sealer seals tenant B's entry", "nava_worker", Denied,
			`UPDATE audit_log SET seq = 1, prev_hash = sha256('p'), entry_hash = sha256('e'), sealed_at = now()
			 WHERE tenant_id = $1 AND id = $2`, TenantB, auditEntryB2),
		sql("audit_log: the erasure job erases tenant B's diffs", "nava_erasure", Denied,
			`UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $2
			 WHERE tenant_id = $1 AND diff IS NOT NULL`, TenantB, NewID),
		sql("audit_log: the erasure job appends for tenant B", "nava_erasure", Denied,
			`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			 VALUES ($1, $2, 'audit', 'erase', 'erasure_request', $2, 'erasure', 'system')`, TenantB, NewID),
		// audit_chain_head: one row per tenant, both seeded.
		sql("audit_chain_head: read tenant B's head", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM audit_chain_head WHERE tenant_id = $1 OR id = $2", TenantB, chainHeadB),
		sql("audit_chain_head: the sealer reads tenant B's head", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM audit_chain_head WHERE id = $1", chainHeadB),
		sql("audit_chain_head: take tenant B's export lease", "nava_app", Denied,
			"UPDATE audit_chain_head SET export_lease_until = now() + interval '1 hour' WHERE tenant_id = $1", TenantB),
		sql("audit_chain_head: advance tenant B's chain", "nava_worker", Denied,
			"UPDATE audit_chain_head SET version = version + 1, seq = seq + 1, entry_hash = sha256('x') WHERE tenant_id = $1", TenantB),
		// record_write_keys: shared-write-key is in both tenants.
		sql("record_write_keys: look a shared key up in every tenant", "nava_app", OwnRowsOnly,
			"SELECT tenant_id FROM record_write_keys WHERE idempotency_key = 'shared-write-key'"),
		sql("record_write_keys: read tenant B's key by its full key", "nava_worker", OwnRowsOnly,
			"SELECT tenant_id FROM record_write_keys WHERE tenant_id = $1 AND idempotency_key = 'other-write-key'", TenantB),
		sql("record_write_keys: claim a key for tenant B", "nava_app", Denied,
			`INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number, version,
			 schema_version, audit_entry_id) VALUES ($1, 'iso-new', $2, 'update', $3, 'TCK0000001', 3, 2, $4)`,
			TenantB, EntityDefinitionB1, RecordB2, NewID),
		// The composite foreign keys hold the tenant: tenant A cannot claim a
		// key on tenant B's record or type.
		sql("record_write_keys: claim a key on tenant B's record from tenant A", "nava_worker", Denied,
			`INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number, version,
			 schema_version, audit_entry_id) VALUES ($1, 'iso-new', $2, 'update', $3, 'TCK0000001', 3, 1, $4)`,
			TenantA, definitionA1, RecordB2, NewID),
		sql("record_write_keys: claim a key on tenant B's type from tenant A", "nava_app", Denied,
			`INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number, version,
			 schema_version, audit_entry_id) VALUES ($1, 'iso-new', $2, 'update', $3, 'TCK0000001', 3, 1, $4)`,
			TenantA, EntityDefinitionB1, RecordA2, NewID),

		// Role is only the report's label on a DB path (Path.Role): these run
		// as every platform login, or as both service pools.
		{Name: "audit_log_default: every runtime role is refused on the DEFAULT partition with 42501", Case: "CI-A4",
			Role: auditEveryLogin, DB: auditDefaultPartitionRefused},
		{Name: "audit_log: tenant B reads none of tenant A's entries by id, subject, change set, seq, diff or tenant, as every runtime role, and none with no tenant", Case: "CI-A4",
			Role: auditEveryLogin, DB: auditDirectSQLEveryRole},
		{Name: "audit_log: onboarding's pinned policies keep it to its own provisioning tenant", Case: "CI-A4",
			Role: "nava_onboard", DB: auditOnboardPinned},
		et("CI-A4", "audit entries: tenant B's admin, auditor and agent get, for tenant A's subject, change set, seq and page token, the unknown value's response", auditReaderOtherTenant),
		et("CI-A4", "record history: tenant B's admin, auditor and agent get the missing record's 404 for tenant A's record, and the history holds record-domain entries only", auditHistory),
		et("CI-A5", "secret canaries in records and S8 publishes are refused and never reach audit_log or entity_definition_versions bytes", auditCanaryBytes),
		{Name: "the nava-core and nava-worker pools connect as nava_app and nava_worker, and the P5-10b role matrix holds through them", Case: "CI-A10",
			Role: "nava_app, nava_worker", DB: auditServicePoolMatrix},
	}
}

// auditCleanup deletes every row the audit paths write past etIso.cleanup:
// fixture entries (source_ref iso-audit), the config entries of entity
// types that no longer exist (the iso_et_ types etIso.cleanup deleted), and
// the onboarding case's provisioning tenant.
func auditCleanup(ctx context.Context, super *pgx.Conn) error {
	for _, q := range []string{
		`DELETE FROM audit_log WHERE source_ref = 'iso-audit'`,
		`DELETE FROM audit_log WHERE subject_type = 'entity_definition' AND subject_id NOT IN (SELECT id FROM entity_definitions)`,
		`DELETE FROM tenants WHERE id = '0e000000-0000-4000-8000-00000000000e'`,
	} {
		if _, err := super.Exec(ctx, q); err != nil {
			return fmt.Errorf("audit cleanup: %w", err)
		}
	}
	return nil
}

// auditManifestLogins are the manifest roles that can log in to the platform
// database and use schema public, the owner excluded (as dbtest's
// platformLogins).
func auditManifestLogins() ([]string, error) {
	m, _, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return nil, err
	}
	var out []string
	for _, name := range m.RoleNames() {
		r := m.Roles[name]
		if name != m.Owner() && slices.Contains(r.Database, "CONNECT") && slices.Contains(r.Schemas["public"], "USAGE") {
			out = append(out, name)
		}
	}
	if len(out) < 5 {
		return nil, fmt.Errorf("control: only %d platform logins in the manifest: %v", len(out), out)
	}
	return out, nil
}

// inTx runs fn as pool's role in tenant's transaction (uuid.Nil: no
// tenant set) and rolls it back.
func inTx(ctx context.Context, pool *pgxpool.Pool, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	run := func(tx pgx.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errRollback
	}
	var err error
	if tenant == uuid.Nil {
		err = pgx.BeginFunc(ctx, pool, run)
	} else {
		err = tenancy.InTenantTx(ctx, pool, tenant, run)
	}
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

// commitTx runs fn in tenant's transaction and commits it.
func commitTx(ctx context.Context, pool *pgxpool.Pool, tenant string, fn func(pgx.Tx) error) error {
	return tenancy.InTenantTx(ctx, pool, uuid.MustParse(tenant), fn)
}

// execAs runs q as pool's role in tenant's transaction, rolled back, and
// returns the rows it reached.
func execAs(ctx context.Context, pool *pgxpool.Pool, tenant string, q string, args ...any) (int64, error) {
	var n int64
	err := inTx(ctx, pool, uuid.MustParse(tenant), func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, q, args...)
		n = tag.RowsAffected()
		return err
	})
	return n, err
}

func auditDefaultPartitionRefused(ctx context.Context, cl Cluster) error {
	super, err := cl.Super()
	if err != nil {
		return err
	}
	// Control: the partition holds the seeded entries, so a refusal is not
	// an empty table.
	var n int
	if err := super.QueryRow(ctx, "SELECT count(*) FROM audit_log_default WHERE tenant_id = ANY ($1::uuid[])",
		[]string{TenantA, TenantB}).Scan(&n); err != nil {
		return err
	}
	if n < 3 {
		return fmt.Errorf("control: audit_log_default holds %d seeded entries, want at least 3", n)
	}
	roles, err := auditManifestLogins()
	if err != nil {
		return err
	}
	probes := []string{
		"SELECT count(*) FROM audit_log_default",
		"SELECT tenant_id FROM audit_log_default WHERE tenant_id = $1",
		"UPDATE audit_log_default SET seq = seq WHERE tenant_id = $1",
		"DELETE FROM audit_log_default WHERE tenant_id = $1",
		`INSERT INTO audit_log_default (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
		 VALUES ($1, gen_random_uuid(), 'record', 'create', 'record', $1, 'system', 'system')`,
	}
	var bad []string
	for _, role := range roles {
		pool, err := cl.Pool(role)
		if err != nil {
			return err
		}
		for _, q := range probes {
			var args []any
			if strings.Contains(q, "$1") {
				args = append(args, TenantB)
			}
			_, err := execAs(ctx, pool, TenantA, q, args...)
			if sqlState(err) != "42501" {
				bad = append(bad, fmt.Sprintf("%s: %s: %v", role, strings.Fields(q)[0], err))
			}
		}
		pool.Close()
	}
	if len(bad) > 0 {
		return fmt.Errorf("the DEFAULT partition, queried directly, is not refused with 42501:\n%s", strings.Join(bad, "\n"))
	}
	return nil
}

// auditFixtureA is tenant A's sealed entry with a change set, a diff and a
// salt: the target of the direct-SQL probes.
type auditFixtureA struct {
	entry, subject, changeSet string
}

// seedAuditFixtureA inserts the fixture as the superuser and seals it as
// nava_worker, through the R11 seal transition.
func seedAuditFixtureA(ctx context.Context, cl Cluster, super *pgx.Conn, domain, subjectType, subject string) (auditFixtureA, error) {
	fx := auditFixtureA{entry: uuid.NewString(), subject: subject, changeSet: uuid.NewString()}
	if fx.subject == "" {
		fx.subject = uuid.NewString()
	}
	if _, err := super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		actor_system, source, source_ref, change_set_id, content_sha256, diff, diff_salt, diff_sha256)
		VALUES ($1, $2, $3, 'update', $4, $5, 'system', 'change_set', 'iso-audit', $6, sha256('c'),
		        '[{"op":"add","path":"/iso","value":"a-only"}]', decode('00112233445566778899aabbccddeeff', 'hex'), sha256('d'))`,
		TenantA, fx.entry, domain, subjectType, fx.subject, fx.changeSet); err != nil {
		return fx, fmt.Errorf("seed tenant A's entry: %w", err)
	}
	worker, err := cl.Pool("nava_worker")
	if err != nil {
		return fx, err
	}
	defer worker.Close()
	if err := commitTx(ctx, worker, TenantA, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE audit_log SET seq = $3, prev_hash = sha256('p'), entry_hash = sha256('e'), sealed_at = now()
			WHERE tenant_id = $1 AND id = $2`, TenantA, fx.entry, auditSealSeq)
		if err == nil && tag.RowsAffected() != 1 {
			err = fmt.Errorf("sealed %d rows, want 1", tag.RowsAffected())
		}
		return err
	}); err != nil {
		return fx, fmt.Errorf("control: seal tenant A's entry: %w", err)
	}
	return fx, nil
}

// auditProbe is one direct-SQL read of the fixture: the columns it needs
// and the query, $1 the value it names.
type auditProbe struct {
	name string
	cols []string
	q    string
	arg  func(auditFixtureA) any
}

func auditProbes() []auditProbe {
	return []auditProbe{
		{"by id", []string{"tenant_id", "id"}, "SELECT tenant_id FROM audit_log WHERE id = $1", func(f auditFixtureA) any { return f.entry }},
		{"by subject_id", []string{"tenant_id", "subject_id"}, "SELECT tenant_id FROM audit_log WHERE subject_id = $1", func(f auditFixtureA) any { return f.subject }},
		{"by change_set_id", []string{"tenant_id", "change_set_id"}, "SELECT tenant_id FROM audit_log WHERE change_set_id = $1", func(f auditFixtureA) any { return f.changeSet }},
		{"after seq", []string{"tenant_id", "seq"}, "SELECT tenant_id FROM audit_log WHERE seq >= $1", func(auditFixtureA) any { return auditSealSeq }},
		{"by tenant", []string{"tenant_id"}, "SELECT tenant_id FROM audit_log WHERE tenant_id = $1", func(auditFixtureA) any { return TenantA }},
	}
}

// auditReaderColumns are the audit_log columns read.Reader selects
// (internal/audit/read columns), diff and diff_salt included. A role with
// SELECT on all of them reads through the reader; one without gets 42501.
func auditReaderColumns() []string {
	return []string{"tenant_id", "id", "occurred_at", "domain", "action", "subject_type", "subject_id",
		"entity_definition_id", "version_before", "version_after", "actor_principal_id", "actor_system", "source",
		"source_ref", "change_set_id", "causation_id", "causation_depth", "request_id", "backfilled", "content_sha256",
		"diff_sha256", "diff_truncated", "seq", "sealed_at", "erased_at", "diff", "diff_salt"}
}

// auditReaderProbe reads, through read.Reader with no mask, the entries of
// fx's subject that the reader returns for tenant asked, in the transaction
// of tenant ctxTenant (uuid.Nil: no tenant set). The reader takes the tenant
// as an argument, so asked = tenant A from tenant B's transaction is the
// forged-tenant call: only row-level security stands between it and tenant
// A's raw diff and salt.
func auditReaderProbe(ctx context.Context, pool *pgxpool.Pool, ctxTenant, asked uuid.UUID, fx auditFixtureA) ([]read.Entry, error) {
	r := read.Reader{Masker: read.StaticMasker{}}
	subject := uuid.MustParse(fx.subject)
	var out []read.Entry
	err := inTx(ctx, pool, ctxTenant, func(tx pgx.Tx) error {
		var err error
		out, err = r.List(ctx, tx, asked, read.Filter{SubjectID: &subject}, nil, 100)
		return err
	})
	return out, err
}

// auditDirectSQLEveryRole: tenant B's transaction, as every platform login,
// names tenant A's entry by each of its keys. A role with SELECT on the
// probe's columns must succeed with zero rows: row-level security is the
// tenant boundary, and nava_app holds diff and diff_salt (TICK-435), so its
// probes must not be refused either. A role without them must get 42501.
// The same probes with no tenant reach zero rows. The diff and the salt are
// probed through read.Reader, unmasked (TICK-541: only the reader reads
// them): asked for tenant A, and for tenant B, it returns nothing of tenant
// A's entry. (An empty app.tenant_id
// is CI-2's generic check on every tenant table, audit_log's partitions
// included: only this package's tests may set it, navalint setconfig.)
// Control: nava_app in tenant A finds the entry through every probe, and the
// reader returns it with its diff and salt.
func auditDirectSQLEveryRole(ctx context.Context, cl Cluster) error {
	super, err := cl.Super()
	if err != nil {
		return err
	}
	if err := auditCleanup(ctx, super); err != nil {
		return err
	}
	defer func() { _ = auditCleanup(context.Background(), super) }()
	fx, err := seedAuditFixtureA(ctx, cl, super, "record", "record", "")
	if err != nil {
		return err
	}
	roles, err := auditManifestLogins()
	if err != nil {
		return err
	}
	granted := func(role string, cols []string) (bool, error) {
		var ok bool
		err := super.QueryRow(ctx, `SELECT bool_and(has_column_privilege($1, 'public.audit_log', c, 'SELECT')) FROM unnest($2::text[]) c`,
			role, cols).Scan(&ok)
		return ok, err
	}
	count := func(pool *pgxpool.Pool, tenant uuid.UUID, p auditProbe) (int, error) {
		var n int
		err := inTx(ctx, pool, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM ("+p.q+") s", p.arg(fx)).Scan(&n)
		})
		return n, err
	}

	app, err := cl.Pool("nava_app")
	if err != nil {
		return err
	}
	defer app.Close()
	for _, p := range auditProbes() {
		ok, err := granted("nava_app", p.cols)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("control: nava_app lacks SELECT on %v; the reader's grant (TICK-435) is gone", p.cols)
		}
		if n, err := count(app, uuid.MustParse(TenantA), p); err != nil || n < 1 {
			return fmt.Errorf("control: nava_app in tenant A %s: %d rows, %s; want tenant A's entry", p.name, n, errText(err))
		}
	}
	if ok, err := granted("nava_app", auditReaderColumns()); err != nil || !ok {
		return fmt.Errorf("control: nava_app lacks SELECT on the reader's columns (%s); the reader's grant (TICK-435) is gone", errText(err))
	}
	got, err := auditReaderProbe(ctx, app, uuid.MustParse(TenantA), uuid.MustParse(TenantA), fx)
	if err != nil || len(got) != 1 || got[0].ID.String() != fx.entry || !strings.Contains(string(got[0].Diff), "a-only") || len(got[0].DiffSalt) == 0 {
		return fmt.Errorf("control: read.Reader in tenant A returns %d entries, %s; want tenant A's entry with its diff and salt", len(got), errText(err))
	}

	var bad []string
	readers := 0
	for _, role := range roles {
		pool, err := cl.Pool(role)
		if err != nil {
			return err
		}
		ok, err := granted(role, auditReaderColumns())
		if err != nil {
			pool.Close()
			return err
		}
		for _, c := range []struct {
			name      string
			ctx, asks uuid.UUID
		}{
			{"tenant B, the reader asked for tenant A", uuid.MustParse(TenantB), uuid.MustParse(TenantA)},
			{"tenant B, the reader asked for tenant B", uuid.MustParse(TenantB), uuid.MustParse(TenantB)},
			{"no tenant, the reader asked for tenant A", uuid.Nil, uuid.MustParse(TenantA)},
		} {
			got, err := auditReaderProbe(ctx, pool, c.ctx, c.asks, fx)
			switch {
			case ok && (err != nil || len(got) != 0):
				bad = append(bad, fmt.Sprintf("%s, %s, diff and salt: %d entries, %v; want none", role, c.name, len(got), err))
			case !ok && sqlState(err) != "42501":
				bad = append(bad, fmt.Sprintf("%s, %s, diff and salt: %d entries, %v; want 42501 (no SELECT on the reader's columns)", role, c.name, len(got), err))
			}
		}
		for _, p := range auditProbes() {
			ok, err := granted(role, p.cols)
			if err != nil {
				pool.Close()
				return err
			}
			if ok && p.name == "by id" {
				readers++
			}
			for _, mode := range []struct {
				name   string
				tenant uuid.UUID
			}{{"tenant B", uuid.MustParse(TenantB)}, {"no tenant", uuid.Nil}} {
				n, err := count(pool, mode.tenant, p)
				switch {
				case ok && (err != nil || n != 0):
					bad = append(bad, fmt.Sprintf("%s, %s, %s: %d rows, %v; want 0 rows", role, mode.name, p.name, n, err))
				case !ok && sqlState(err) != "42501":
					bad = append(bad, fmt.Sprintf("%s, %s, %s: %d rows, %v; want 42501 (no SELECT on %v)", role, mode.name, p.name, n, err, p.cols))
				}
			}
		}
		pool.Close()
	}
	if readers < 3 {
		return fmt.Errorf("control: only %d roles read audit_log; want nava_app, nava_worker and nava_erasure at least", readers)
	}
	if len(bad) > 0 {
		return errors.New(strings.Join(bad, "\n"))
	}
	return nil
}

// auditProvisioningTenant is a tenant the onboarding case creates in state
// provisioning, and deletes after.
const auditProvisioningTenant = "0e000000-0000-4000-8000-00000000000e"

// auditOnboardPinned: nava_onboard's audit_log_onboard_select and
// audit_log_onboard_insert (catalog.PinnedTenantPolicies) admit its own
// tenant only while it is provisioning. From the provisioning tenant P it
// reads P's entry and nobody else's, and appends for P only; from active
// tenant A it reads and appends nothing, not even A's own.
func auditOnboardPinned(ctx context.Context, cl Cluster) error {
	super, err := cl.Super()
	if err != nil {
		return err
	}
	if err := auditCleanup(ctx, super); err != nil {
		return err
	}
	defer func() { _ = auditCleanup(context.Background(), super) }()
	if _, err := super.Exec(ctx, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", auditProvisioningTenant); err != nil {
		return err
	}
	subject := uuid.NewString() // the same subject in P, A and B
	for _, tenant := range []string{auditProvisioningTenant, TenantA, TenantB} {
		if _, err := super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
			actor_system, source, source_ref) VALUES ($1, gen_random_uuid(), 'record', 'create', 'record', $2, 'system', 'onboarding', 'iso-audit')`,
			tenant, subject); err != nil {
			return err
		}
	}
	onboard, err := cl.Pool("nava_onboard")
	if err != nil {
		return err
	}
	defer onboard.Close()
	read := func(tenant string) ([]string, error) {
		var out []string
		err := inTx(ctx, onboard, uuid.MustParse(tenant), func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, "SELECT tenant_id::text FROM audit_log WHERE subject_id = $1 ORDER BY 1", subject)
			if err != nil {
				return err
			}
			out, err = pgx.CollectRows(rows, pgx.RowTo[string])
			return err
		})
		return out, err
	}
	insert := func(context, target string) error {
		_, err := execAs(ctx, onboard, context, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
			actor_system, source, source_ref) VALUES ($1, gen_random_uuid(), 'record', 'create', 'record', $2, 'system', 'onboarding', 'iso-audit')`,
			target, subject)
		return err
	}
	if got, err := read(auditProvisioningTenant); err != nil || !slices.Equal(got, []string{auditProvisioningTenant}) {
		return fmt.Errorf("nava_onboard in provisioning tenant P reads %v, %s; want P's entry only", got, errText(err))
	}
	if got, err := read(TenantA); err != nil || len(got) != 0 {
		return fmt.Errorf("nava_onboard in active tenant A reads %v, %s; want nothing", got, errText(err))
	}
	if err := insert(auditProvisioningTenant, auditProvisioningTenant); err != nil {
		return fmt.Errorf("control: nava_onboard appends for its provisioning tenant: %w", err)
	}
	for _, c := range [][2]string{{auditProvisioningTenant, TenantB}, {auditProvisioningTenant, TenantA}, {TenantA, TenantA}, {TenantA, TenantB}} {
		if err := insert(c[0], c[1]); sqlState(err) != "42501" {
			return fmt.Errorf("nava_onboard in tenant %s appends for tenant %s: %s; want 42501", c[0], c[1], errText(err))
		}
	}
	return nil
}

// auditEntryPage is a page of GET /api/v1/audit/entries.
type auditEntryPage struct {
	Items []struct {
		ID        string `json:"id"`
		SubjectID string `json:"subject_id"`
	} `json:"items"`
	Next *string `json:"next_page_token"`
}

func (p auditEntryPage) ids() []string {
	out := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		out = append(out, it.ID)
	}
	return out
}

func (e *etIso) auditPage(ctx context.Context, token, query string) (int, string, auditEntryPage, error) {
	code, body := e.do(ctx, http.MethodGet, "/api/v1/audit/entries"+query, token, "", nil)
	var p auditEntryPage
	if code == http.StatusOK {
		if err := json.Unmarshal([]byte(body), &p); err != nil {
			return code, body, p, fmt.Errorf("GET /api/v1/audit/entries%s: %w: %s", query, err, body)
		}
	}
	return code, body, p, nil
}

// auditRecordFixture is tenant A's record of iso_et_aud with its create and
// update entries, and the sealed config-domain entry about the same record,
// in a change set.
type auditRecordFixture struct {
	record string
	extra  auditFixtureA
	// aOnly is every id that must never reach a tenant B response.
	aOnly []string
}

func auditRecordSetup(ctx context.Context, e *etIso) (*auditRecordFixture, error) {
	if err := e.create(ctx, "iso_et_aud", "ISA"); err != nil {
		return nil, err
	}
	code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_aud", etCallerA, `{"data": {"note": "mine"}}`, nil)
	if code != http.StatusCreated {
		return nil, fmt.Errorf("control: tenant A creating a record = %d %s, want 201", code, body)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.ID == "" {
		return nil, fmt.Errorf("control: tenant A's record has no id: %s: %s", errText(err), body)
	}
	patch := http.Header{"Content-Type": {"application/merge-patch+json"}, "If-Match": {`W/"1"`}}
	if code, body := e.do(ctx, http.MethodPatch, "/api/v1/records/iso_et_aud/"+r.ID, etCallerA, `{"data": {"note": "again"}}`, patch); code != http.StatusOK {
		return nil, fmt.Errorf("control: tenant A patching its record = %d %s, want 200", code, body)
	}
	extra, err := seedAuditFixtureA(ctx, e.cluster, e.super, "config", "record", r.ID)
	if err != nil {
		return nil, err
	}
	fx := &auditRecordFixture{record: r.ID, extra: extra}
	rows, err := e.super.Query(ctx, "SELECT id::text FROM audit_log WHERE tenant_id = $1", TenantA)
	if err != nil {
		return nil, err
	}
	if fx.aOnly, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		return nil, err
	}
	fx.aOnly = append(fx.aOnly, r.ID, extra.changeSet, TenantA)
	return fx, nil
}

// errText is err's message, or "no error", for a failure message that also
// fires when the call succeeded with the wrong result.
func errText(err error) string {
	if err == nil {
		return "no error"
	}
	return err.Error()
}

func leaks(body string, ids []string) string {
	for _, id := range ids {
		if strings.Contains(body, id) {
			return id
		}
	}
	return ""
}

// sameShape is nil when two responses are the same status and, but for the
// problem instance, the same body.
func sameShape(oc int, ob string, uc int, ub string) error {
	if oc != uc {
		return fmt.Errorf("status %d, the unknown value's %d", oc, uc)
	}
	if oc == http.StatusOK {
		if ob != ub {
			return fmt.Errorf("body differs from the unknown value's:\n%s\n%s", ob, ub)
		}
		return nil
	}
	ow, err := withoutInstance(ob)
	if err != nil {
		return err
	}
	uw, err := withoutInstance(ub)
	if err != nil {
		return err
	}
	if ow != uw {
		return fmt.Errorf("body differs from the unknown value's:\n%s\n%s", ob, ub)
	}
	return nil
}

func auditReaderOtherTenant(ctx context.Context, e *etIso) error {
	fx, err := auditRecordSetup(ctx, e)
	if err != nil {
		return err
	}
	// Controls, tenant A: its admin and auditor read the record's entries,
	// the sealed entry after the seq, and a second page; its agent is
	// refused. Then tenant B's admin names subject RecordA2, which the seed
	// gives an entry in both tenants: it must get its own entry only.
	for _, tok := range []string{etCallerA, auditCallerAAuditor} {
		code, body, p, err := e.auditPage(ctx, tok, "?subject_id="+fx.record)
		if err != nil || code != http.StatusOK || len(p.Items) < 3 {
			return fmt.Errorf("control: %s reads tenant A's record entries = %d, %s, %s; want 200 with the create, the update and the sealed entry", tok, code, errText(err), body)
		}
	}
	if code, body, p, err := e.auditPage(ctx, etCallerA, fmt.Sprintf("?after_seq=%d", auditSealSeq-1)); err != nil || code != http.StatusOK ||
		!slices.Equal(p.ids(), []string{fx.extra.entry}) {
		return fmt.Errorf("control: tenant A's admin after seq %d = %d, %s, %s; want its sealed entry only", auditSealSeq-1, code, errText(err), body)
	}
	code, body, first, err := e.auditPage(ctx, etCallerA, "?limit=1")
	if err != nil || code != http.StatusOK || first.Next == nil {
		return fmt.Errorf("control: tenant A's first page = %d, %s, %s; want 200 with a next_page_token", code, errText(err), body)
	}
	token := url.QueryEscape(*first.Next)
	if code, body, _, err := e.auditPage(ctx, etCallerA, "?limit=1&page_token="+token); err != nil || code != http.StatusOK {
		return fmt.Errorf("control: tenant A replaying its own page token = %d, %s, %s; want 200", code, errText(err), body)
	}
	if code, body := e.do(ctx, http.MethodGet, "/api/v1/audit/entries", auditCallerAAgent, "", nil); code != http.StatusForbidden {
		return fmt.Errorf("control: tenant A's agent reads the timeline = %d %s; want 403", code, body)
	}
	if code, body, p, err := e.auditPage(ctx, etCallerBAdmin, "?subject_id="+RecordA2); err != nil || code != http.StatusOK ||
		!slices.Equal(p.ids(), []string{auditEntryB1}) {
		return fmt.Errorf("tenant B's admin reads subject %s, which both tenants' seeded entries name = %d, %s, %s; want its own entry %s only", RecordA2, code, errText(err), body, auditEntryB1)
	}

	const unknown = "00000000-0000-4000-8000-0000000dead1"
	cases := []struct{ name, other, unknown string }{
		{"tenant A's subject_id", "?subject_id=" + fx.record, "?subject_id=" + unknown},
		{"tenant A's subject_id and subject_type", "?subject_type=record&subject_id=" + fx.record, "?subject_type=record&subject_id=" + unknown},
		{"tenant A's change_set_id", "?change_set_id=" + fx.extra.changeSet, "?change_set_id=" + unknown},
		{"tenant A's sealed seq", fmt.Sprintf("?after_seq=%d", auditSealSeq-1), "?after_seq=999999999999"},
		{"a page_token minted by tenant A", "?limit=1&page_token=" + token, "?limit=1&page_token=" + url.QueryEscape("AAAA"+strings.Repeat("A", len(*first.Next)-4))},
	}
	for _, caller := range []struct {
		tok  string
		want int // the status the unknown value's request must get
	}{{etCallerBAdmin, http.StatusOK}, {auditCallerBAuditor, http.StatusOK}, {auditCallerBAgent, http.StatusForbidden}} {
		for _, c := range cases {
			uc, ub := e.do(ctx, http.MethodGet, "/api/v1/audit/entries"+c.unknown, caller.tok, "", nil)
			oc, ob := e.do(ctx, http.MethodGet, "/api/v1/audit/entries"+c.other, caller.tok, "", nil)
			want := caller.want
			if c.name == "a page_token minted by tenant A" && want == http.StatusOK {
				want = http.StatusBadRequest
			}
			if uc != want {
				return fmt.Errorf("control: %s with %s's unknown value = %d %s; want %d", caller.tok, c.name, uc, ub, want)
			}
			if id := leaks(ob, fx.aOnly); id != "" {
				return fmt.Errorf("%s with %s: the response holds tenant A's %s: %s", caller.tok, c.name, id, ob)
			}
			if err := sameShape(oc, ob, uc, ub); err != nil {
				return fmt.Errorf("%s with %s: %w", caller.tok, c.name, err)
			}
		}
	}
	return nil
}

func auditHistory(ctx context.Context, e *etIso) error {
	fx, err := auditRecordSetup(ctx, e)
	if err != nil {
		return err
	}
	path := "/api/v1/records/iso_et_aud/" + fx.record + "/history"
	code, body := e.do(ctx, http.MethodGet, path, etCallerA, "", nil)
	var h struct {
		Items []json.RawMessage `json:"items"`
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &h) != nil {
		return fmt.Errorf("control: tenant A's admin reads its record's history = %d %s; want 200", code, body)
	}
	// The record has a create and an update; the config-domain entry about
	// the same record is not part of its history.
	if len(h.Items) != 2 || strings.Contains(body, fx.extra.entry) || strings.Contains(body, fx.extra.changeSet) {
		return fmt.Errorf("tenant A's history of its record holds %d items (want the create and the update) or the config-domain entry: %s", len(h.Items), body)
	}
	// Tenant B has a type of the same key, so the type lookup cannot be what
	// answers 404: only the tenant-scoped record lookup can.
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerBAdmin, etCreateBody("iso_et_aud", "ISB", ""), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant B creating its own iso_et_aud = %d %s, want 201", code, body)
	}
	const missing = "0194f2c0-0000-7000-8000-000000000000"
	for _, tok := range []string{etCallerBAdmin, auditCallerBAuditor, auditCallerBAgent} {
		oc, ob := e.do(ctx, http.MethodGet, path, tok, "", nil)
		uc, ub := e.do(ctx, http.MethodGet, "/api/v1/records/iso_et_aud/"+missing+"/history", tok, "", nil)
		if uc != http.StatusNotFound {
			return fmt.Errorf("control: %s reads a missing record's history = %d %s; want 404", tok, uc, ub)
		}
		if oc != http.StatusNotFound {
			return fmt.Errorf("%s reads tenant A's record's history = %d %s; want 404, not 403", tok, oc, ob)
		}
		if id := leaks(ob, fx.aOnly); id != "" {
			return fmt.Errorf("%s: the history 404 holds tenant A's %s: %s", tok, id, ob)
		}
		if err := sameShape(oc, ob, uc, ub); err != nil {
			return fmt.Errorf("%s reads tenant A's record's history: %w", tok, err)
		}
	}
	return nil
}

// auditSecSchema is iso_et_sec's schema: note, described by desc, and the
// x-nava-secret property token, with tokenExtra spliced into it.
func auditSecSchema(desc, tokenExtra string) string {
	return fmt.Sprintf(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,`+
		`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},`+
		`"note":{"type":"string","description":%q},"token":{"type":"string","x-nava-secret":true%s}}}`, desc, tokenExtra)
}

func auditSecDefinition(desc, tokenExtra string) string {
	return fmt.Sprintf(`{"schema": %s, %s}`, auditSecSchema(desc, tokenExtra), etDefTail)
}

// auditTableBytes is every byte of the heap and TOAST files of every
// audit_log partition (the DEFAULT one included) and of
// entity_definition_versions, after a CHECKPOINT, as the superuser.
func auditTableBytes(ctx context.Context, super *pgx.Conn) (log, versions []byte, err error) {
	if _, err := super.Exec(ctx, "CHECKPOINT"); err != nil {
		return nil, nil, err
	}
	read := func(q string) ([]byte, error) {
		rows, err := super.Query(ctx, q)
		if err != nil {
			return nil, err
		}
		paths, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return nil, err
		}
		if len(paths) < 2 {
			return nil, fmt.Errorf("control: %d relation files, want a heap and its TOAST at least", len(paths))
		}
		var all []byte
		for _, p := range paths {
			var b []byte
			if err := super.QueryRow(ctx, "SELECT coalesce(pg_read_binary_file($1, true), '')", p).Scan(&b); err != nil {
				return nil, fmt.Errorf("read %s: %w", p, err)
			}
			all = append(all, b...)
		}
		return all, nil
	}
	const files = `SELECT pg_relation_filepath(c.oid) FROM pg_class c WHERE c.oid IN (%[1]s)
		OR c.oid IN (SELECT reltoastrelid FROM pg_class WHERE oid IN (%[1]s) AND reltoastrelid <> 0)`
	if log, err = read(fmt.Sprintf(files, `SELECT relid FROM pg_partition_tree('audit_log') WHERE isleaf
		UNION SELECT 'audit_log_default'::regclass`)); err != nil {
		return nil, nil, err
	}
	versions, err = read(fmt.Sprintf(files, `SELECT 'entity_definition_versions'::regclass`))
	return log, versions, err
}

// auditCanaryBytes (CI-A5): tenant A's admin tries to store a canary through
// every route the S14 surface audits: a record create and a patch at the
// x-nava-secret property, a URL with userinfo in a record, and S8 publishes
// with the canary as the secret property's default, as a URL with userinfo
// and as an AWS key id in a description. Each is refused, and no canary is
// in the bytes of audit_log or entity_definition_versions, their TOAST
// included. Control: a record and a publish with non-secret markers from the
// same run succeed, and the scan finds both markers.
func auditCanaryBytes(ctx context.Context, e *etIso) error {
	run := strings.ReplaceAll(uuid.NewString(), "-", "")
	recordMarker, publishMarker := "iso-control-r-"+run, "iso-control-s-"+run
	create := fmt.Sprintf(`{"key": "iso_et_sec", "name": "Iso sec", "number_prefix": "ISS", "definition": %s}`, auditSecDefinition("v1", ""))
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerA, create, nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A creating iso_et_sec = %d %s, want 201", code, body)
	}
	code, body := e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_sec", etCallerA, fmt.Sprintf(`{"data": {"note": %q}}`, recordMarker), nil)
	if code != http.StatusCreated {
		return fmt.Errorf("control: tenant A's record with the marker = %d %s, want 201", code, body)
	}
	var r struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.ID == "" {
		return fmt.Errorf("control: the record has no id: %s: %s", errText(err), body)
	}
	publish := func(def string) (int, string) {
		return e.do(ctx, http.MethodPost, "/api/v1/entity-types/iso_et_sec/versions", etCallerA, fmt.Sprintf(`{"definition": %s}`, def), nil)
	}
	if code, body := publish(auditSecDefinition(publishMarker, "")); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A publishing the marker = %d %s, want 201", code, body)
	}

	canary := func(n int) string { return fmt.Sprintf("iso-canary-%d-%s", n, run) }
	aws := "AKIA" + strings.ToUpper(run[:16])
	patch := http.Header{"Content-Type": {"application/merge-patch+json"}, "If-Match": {`W/"1"`}}
	attempts := []struct {
		name, secret string
		do           func() (int, string)
	}{
		{"record create, the secret property", canary(1), func() (int, string) {
			return e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_sec", etCallerA, fmt.Sprintf(`{"data": {"token": %q}}`, canary(1)), nil)
		}},
		{"record patch, the secret property", canary(2), func() (int, string) {
			return e.do(ctx, http.MethodPatch, "/api/v1/records/iso_et_sec/"+r.ID, etCallerA, fmt.Sprintf(`{"data": {"token": %q}}`, canary(2)), patch)
		}},
		{"record create, a URL with userinfo", canary(3), func() (int, string) {
			return e.do(ctx, http.MethodPost, "/api/v1/records/iso_et_sec", etCallerA,
				fmt.Sprintf(`{"data": {"note": "https://iso:%s@example.test/"}}`, canary(3)), nil)
		}},
		{"S8 publish, the secret property's default", canary(4), func() (int, string) {
			return publish(auditSecDefinition("v3", fmt.Sprintf(`,"default":%q`, canary(4))))
		}},
		{"S8 publish, a URL with userinfo in a description", canary(5), func() (int, string) {
			return publish(auditSecDefinition(fmt.Sprintf("https://iso:%s@example.test/", canary(5)), ""))
		}},
		{"S8 publish, an AWS key id in a description", aws, func() (int, string) {
			return publish(auditSecDefinition("key "+aws, ""))
		}},
	}
	// A refusal that fails is reported with the byte scan, so a run with the
	// guard removed shows where the canary landed.
	var found []string
	for _, a := range attempts {
		code, body := a.do()
		if code < 400 || code >= 500 {
			found = append(found, fmt.Sprintf("%s = %d; want a 4xx refusal", a.name, code))
		}
		if strings.Contains(body, a.secret) {
			found = append(found, fmt.Sprintf("%s: the response carries the secret", a.name))
		}
	}
	var versions int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM entity_definition_versions v JOIN entity_definitions d
		ON d.tenant_id = v.tenant_id AND d.id = v.definition_id WHERE d.key = 'iso_et_sec'`).Scan(&versions); err != nil {
		return err
	}
	if versions != 2 {
		found = append(found, fmt.Sprintf("iso_et_sec has %d versions, want the create and the control publish only", versions))
	}

	if err := auditControlsUncompressed(ctx, e.super, publishMarker); err != nil {
		return err
	}
	logBytes, versionBytes, err := auditTableBytes(ctx, e.super)
	if err != nil {
		return err
	}
	for _, c := range []struct {
		what   string
		b      []byte
		marker string
	}{{"audit_log", logBytes, recordMarker}, {"audit_log", logBytes, publishMarker}, {"entity_definition_versions", versionBytes, publishMarker}} {
		if !bytes.Contains(c.b, []byte(c.marker)) {
			return fmt.Errorf("control: the marker %s is not in %s's bytes; the scan proves nothing", c.marker, c.what)
		}
	}
	for _, a := range attempts {
		for what, b := range map[string][]byte{"audit_log": logBytes, "entity_definition_versions": versionBytes} {
			if bytes.Contains(b, []byte(a.secret)) {
				found = append(found, fmt.Sprintf("%s: the canary is in %s's bytes", a.name, what))
			}
		}
	}
	if len(found) > 0 {
		slices.Sort(found)
		return errors.New(strings.Join(found, "\n"))
	}
	return nil
}

// auditControlsUncompressed (TICK-552): the scan looks for plain bytes, and
// PostgreSQL compresses a value (pglz or lz4, inline or in TOAST) once its
// tuple passes about 2 kB, after which a canary in it is no longer a plain
// byte string. The canary attempts have the controls' shape, so the control
// publish's entity_definition_versions.schema must be stored uncompressed. A
// fixture that grows past the threshold then fails here instead of hiding a
// leak.
//
// audit_log needs no query of its own, and may not have one: only
// internal/audit/read reads audit_log.diff (TICK-541). The markers reach
// audit_log in the diff only, so if a marker's diff is compressed, the marker
// is not in audit_log's plain bytes, and auditCanaryBytes's marker control
// already fails with "the scan proves nothing".
func auditControlsUncompressed(ctx context.Context, super *pgx.Conn, publishMarker string) error {
	var n int
	var compressed []string
	if err := super.QueryRow(ctx,
		`SELECT count(*), coalesce(array_agg(pg_column_compression(v.schema)) FILTER (WHERE pg_column_compression(v.schema) IS NOT NULL), '{}')
		 FROM entity_definition_versions v JOIN entity_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
		 WHERE d.tenant_id = $1 AND d.key = 'iso_et_sec' AND strpos(v.schema::text, $2) > 0`, TenantA, publishMarker).Scan(&n, &compressed); err != nil {
		return fmt.Errorf("control: compression of entity_definition_versions.schema: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("control: %d entity_definition_versions rows carry the marker %s, want 1; the scan proves nothing", n, publishMarker)
	}
	if len(compressed) > 0 {
		return fmt.Errorf("control: entity_definition_versions.schema with the marker %s is stored compressed (%s), so a canary in a row of its shape is not plain bytes; the scan proves nothing",
			publishMarker, strings.Join(compressed, ", "))
	}
	return nil
}

// auditServiceURL is service's NAVA_DATABASE_URL in compose.yaml, with
// ${VAR:-default} resolved to its default.
func auditServiceURL(service string) (string, error) {
	b, err := os.ReadFile("../../../compose.yaml")
	if err != nil {
		return "", err
	}
	var doc struct {
		Services map[string]struct {
			Environment map[string]any `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return "", err
	}
	v, ok := doc.Services[service].Environment["NAVA_DATABASE_URL"].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("compose.yaml: %s has no NAVA_DATABASE_URL", service)
	}
	return composeDefaults(service, v)
}

// composeDefaults resolves each ${VAR:-default} in v to its default. A
// variable with no default, or a ${ with no closing }, is an error, never a
// panic (TICK-552).
func composeDefaults(service, v string) (string, error) {
	for {
		i := strings.Index(v, "${")
		if i < 0 {
			return v, nil
		}
		j := strings.Index(v[i:], "}")
		if j < 0 {
			return "", fmt.Errorf("compose.yaml: %s: an unterminated ${ in NAVA_DATABASE_URL", service)
		}
		_, def, ok := strings.Cut(v[i+2:i+j], ":-")
		if !ok {
			return "", fmt.Errorf("compose.yaml: %s: a variable with no default in NAVA_DATABASE_URL", service)
		}
		v = v[:i] + def + v[i+j+1:]
	}
}

// auditServicePool opens the pool service opens: its compose
// NAVA_DATABASE_URL, parsed and built as its main does (nava-core in
// QueryExecModeExec, behind PgBouncer), through platform/db.NewPool. Only
// the address and the password are the suite's: its PgBouncer and its test
// credential, taken from the suite's own pool for the same role.
func auditServicePool(ctx context.Context, cl Cluster, tel *telemetry.Telemetry, service string) (*pgxpool.Pool, string, error) {
	raw, err := auditServiceURL(service)
	if err != nil {
		return nil, "", err
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		return nil, "", fmt.Errorf("compose.yaml: %s's NAVA_DATABASE_URL does not parse", service)
	}
	role := cfg.ConnConfig.User
	if cfg.ConnConfig.Host != "pgbouncer" {
		return nil, "", fmt.Errorf("compose.yaml: %s connects to %q, not pgbouncer", service, cfg.ConnConfig.Host)
	}
	ref, err := cl.PooledSize(role, 1)
	if err != nil {
		return nil, "", fmt.Errorf("compose.yaml: %s logs in as %s: %w", service, role, err)
	}
	rc := ref.Config().ConnConfig
	ref.Close()
	if cfg.ConnConfig.Database != rc.Database {
		return nil, "", fmt.Errorf("compose.yaml: %s uses database %q, not the platform's %q", service, cfg.ConnConfig.Database, rc.Database)
	}
	cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Password = rc.Host, rc.Port, rc.Password
	cfg.ConnConfig.TLSConfig, cfg.ConnConfig.Fallbacks = nil, nil
	if service == "nava-core" {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // cmd/nava-core/identity.go
	}
	cfg.MaxConns = 1
	pool, err := platformdb.NewPool(ctx, cfg, tel)
	return pool, role, err
}

// auditServicePoolMatrix (CI-A10, through the services): the pools that
// nava-core and nava-worker build from compose.yaml log in as nava_app and
// nava_worker, and the P5-10b matrix (internal/db/dbtest TestP510b*) holds
// through them: no protected INSERT column, no UPDATE but the worker's seal,
// no DELETE, and nava_app cannot write seq, entry_hash or version on the
// head. Controls: each pool appends, the worker seals once, nava_app takes
// the export lease and the worker advances the head. Everything but the two
// appended entries rolls back; those are deleted after.
func auditServicePoolMatrix(ctx context.Context, cl Cluster) error {
	super, err := cl.Super()
	if err != nil {
		return err
	}
	if err := auditCleanup(ctx, super); err != nil {
		return err
	}
	defer func() { _ = auditCleanup(context.Background(), super) }()
	tel, err := telemetry.New(ctx, config.Telemetry{})
	if err != nil {
		return err
	}
	defer func() { _ = tel.Shutdown(context.Background()) }()
	pools := map[string]*pgxpool.Pool{}
	for service, want := range map[string]string{"nava-core": "nava_app", "nava-worker": "nava_worker"} {
		pool, role, err := auditServicePool(ctx, cl, tel, service)
		if err != nil {
			return err
		}
		defer pool.Close()
		var cur, sess string
		if err := pool.QueryRow(ctx, "SELECT current_user::text, session_user::text").Scan(&cur, &sess); err != nil {
			return fmt.Errorf("%s's pool: %w", service, err)
		}
		if role != want || cur != want || sess != want {
			return fmt.Errorf("%s's pool logs in as %s (current_user %s, session_user %s); want %s", service, role, cur, sess, want)
		}
		pools[want] = pool
	}

	const insert = `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after,
		actor_system, source, source_ref, diff, diff_salt, diff_sha256)
		VALUES ($1, $2, 'record', 'update', 'record', $3, 1, 'workflow', 'workflow', 'iso-audit',
		        '[{"op":"add","path":"/a","value":1}]', decode('00112233445566778899aabbccddeeff', 'hex'), sha256('d'))`
	protected := map[string]string{
		"occurred_at": "now()", "txid": "pg_current_xact_id()", "backfilled": "false",
		"seq": "1", "prev_hash": "sha256('p')", "entry_hash": "sha256('e')", "sealed_at": "now()",
		"erased_at": "now()", "erasure_ref": "gen_random_uuid()",
	}
	updates := map[string]string{
		"content":    "UPDATE audit_log SET action = 'delete' WHERE tenant_id = $1 AND id = $2",
		"tenant":     "UPDATE audit_log SET tenant_id = tenant_id WHERE tenant_id = $1 AND id = $2",
		"seal":       "UPDATE audit_log SET seq = 1, prev_hash = sha256('p'), entry_hash = sha256('e'), sealed_at = now() WHERE tenant_id = $1 AND id = $2",
		"seq only":   "UPDATE audit_log SET seq = 1 WHERE tenant_id = $1 AND id = $2",
		"erase":      "UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = gen_random_uuid() WHERE tenant_id = $1 AND id = $2",
		"diff only":  "UPDATE audit_log SET diff = NULL WHERE tenant_id = $1 AND id = $2",
		"truncated":  "UPDATE audit_log SET diff_truncated = true WHERE tenant_id = $1 AND id = $2",
		"occurred":   "UPDATE audit_log SET occurred_at = now() WHERE tenant_id = $1 AND id = $2",
		"backfilled": "UPDATE audit_log SET backfilled = true WHERE tenant_id = $1 AND id = $2",
	}
	var bad []string
	refused := func(what string, err error) {
		if sqlState(err) != "42501" {
			bad = append(bad, fmt.Sprintf("%s: %v; want 42501", what, err))
		}
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		pool := pools[role]
		id := uuid.NewString()
		if err := commitTx(ctx, pool, TenantA, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, insert, TenantA, id, RecordA2)
			return err
		}); err != nil {
			return fmt.Errorf("control: %s appends through the R10 columns: %w", role, err)
		}
		for col, val := range protected {
			_, err := execAs(ctx, pool, TenantA, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
				actor_system, source, `+col+`) VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, 'workflow', 'workflow', `+val+`)`,
				TenantA, RecordA2)
			refused(role+" inserts "+col, err)
		}
		for name, q := range updates {
			if role == "nava_worker" && name == "seal" {
				continue
			}
			_, err := execAs(ctx, pool, TenantA, q, TenantA, id)
			refused(role+" updates "+name, err)
		}
		_, err := execAs(ctx, pool, TenantA, "DELETE FROM audit_log WHERE tenant_id = $1 AND id = $2", TenantA, id)
		refused(role+" deletes", err)
		if role == "nava_worker" {
			if err := commitTx(ctx, pool, TenantA, func(tx pgx.Tx) error {
				tag, err := tx.Exec(ctx, updates["seal"], TenantA, id)
				if err == nil && tag.RowsAffected() != 1 {
					err = fmt.Errorf("%d rows", tag.RowsAffected())
				}
				return err
			}); err != nil {
				return fmt.Errorf("control: nava_worker seals its entry: %w", err)
			}
			_, err := execAs(ctx, pool, TenantA, updates["seal"], TenantA, id)
			refused("nava_worker seals a sealed entry", err)
		}
	}
	head := func(role, set string) error {
		_, err := execAs(ctx, pools[role], TenantA, "UPDATE audit_chain_head SET "+set+" WHERE tenant_id = $1", TenantA)
		return err
	}
	for _, set := range []string{"seq = seq + 1", "entry_hash = sha256('x')", "version = version + 1", "seal_interval_s = 20", "tenant_id = tenant_id", "id = gen_random_uuid()"} {
		refused("nava_app sets "+set+" on the head", head("nava_app", set))
	}
	for _, set := range []string{"export_lease_until = now()", "tenant_id = tenant_id", "id = gen_random_uuid()"} {
		refused("nava_worker sets "+set+" on the head", head("nava_worker", set))
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		_, err := execAs(ctx, pools[role], TenantA, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, gen_random_uuid())", TenantA)
		refused(role+" inserts a head", err)
		_, err = execAs(ctx, pools[role], TenantA, "DELETE FROM audit_chain_head WHERE tenant_id = $1", TenantA)
		refused(role+" deletes a head", err)
	}
	if err := head("nava_app", "export_lease_until = now() + interval '1 minute'"); err != nil {
		return fmt.Errorf("control: nava_app takes tenant A's export lease: %w", err)
	}
	if err := head("nava_worker", "version = version + 1, seq = seq + 1, entry_hash = sha256('x')"); err != nil {
		return fmt.Errorf("control: nava_worker advances tenant A's head: %w", err)
	}
	if len(bad) > 0 {
		slices.Sort(bad)
		return errors.New(strings.Join(bad, "\n"))
	}
	return nil
}
