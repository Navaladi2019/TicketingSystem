package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
)

// ADR-004-A1 (TICK-558): Postgres, not convention, keeps the raw diff away
// from every runtime role but nava_audit_reader. Every query in this file
// either must be refused with 42501 or runs as nava_audit_reader, so the
// TICK-541 static guard (internal/audit/read/diffreaders_test.go) allows this
// one file.

// The A1 test ids (ADR-004-A1 "Tests (red first)").
const (
	caseDiffDenied   = "ADR-004-A1/diff-denied"
	caseReaderScoped = "ADR-004-A1/reader-scoped"
	caseReaderMasks  = "ADR-004-A1/reader-masks"
)

func auditReaderPaths() []Path {
	return []Path{
		{Name: "audit_log.diff and diff_salt: every runtime role but nava_audit_reader gets 42501 for every way of reading them (column, star, whole row, WHERE and ORDER BY oracles, partition, ONLY, COPY, CTE, LATERAL, RETURNING), through PgBouncer and nava_erasure directly; nava_audit_reader cannot write audit_log or read records",
			Case: caseDiffDenied, Role: auditEveryLogin, DB: auditDiffDenied},
		{Name: "nava_audit_reader: tenant A reads its own diff; tenant B set with tenant A's id, and no tenant, read 0 rows; txid is 42501",
			Case: caseReaderScoped, Role: audited.ReaderRole, DB: auditReaderScoped},
		{Name: "GET /api/v1/audit/entries on the reader pool masks an x-nava-secret property for an auditor with no mask permission: {\"redacted\":\"mask\"}, diff_redacted, no salt",
			Case: caseReaderMasks, Role: "nava_app, " + audited.ReaderRole, DB: func(ctx context.Context, cl Cluster) error {
				e, err := newETIso(ctx, cl)
				if err != nil {
					return err
				}
				defer e.stop()
				// The fixture entry names iso_et_sec, so it goes before the
				// type does, and the type's config entries after.
				clean := func() error {
					if err := auditCleanup(ctx, e.super); err != nil {
						return err
					}
					if err := e.cleanup(); err != nil {
						return err
					}
					return auditCleanup(ctx, e.super)
				}
				if err := clean(); err != nil {
					return err
				}
				err = auditReaderMasks(ctx, e)
				if cerr := clean(); err == nil {
					err = cerr
				}
				return err
			}},
	}
}

// diffDeniedRoles are the roles the diff-denied probes run as (ADR-004-A1):
// every runtime role but the reader. nava_erasure has no PgBouncer login, so
// Cluster.Pool connects it directly; through PgBouncer it would fail at login,
// not with 42501.
func diffDeniedRoles() []string {
	return []string{"nava_app", "nava_worker", "nava_relay", "nava_purge", "nava_onboard", "nava_groupsync", "nava_erasure"}
}

// diffProbe is one way of reading the diff or the salt. $1 is tenant A, $2
// the fixture entry; copy probes take no parameters.
type diffProbe struct {
	name string
	q    string
	copy bool
}

func diffProbes() []diffProbe {
	return []diffProbe{
		{name: "SELECT diff", q: "SELECT diff FROM audit_log WHERE tenant_id = $1 AND id = $2"},
		{name: "SELECT diff_salt", q: "SELECT diff_salt FROM audit_log WHERE tenant_id = $1 AND id = $2"},
		{name: "SELECT *", q: "SELECT * FROM audit_log WHERE tenant_id = $1 AND id = $2"},
		{name: "to_jsonb(a)", q: "SELECT to_jsonb(a) FROM audit_log a WHERE a.tenant_id = $1 AND a.id = $2"},
		{name: "a::text", q: "SELECT a::text FROM audit_log a WHERE a.tenant_id = $1 AND a.id = $2"},
		{name: "WHERE diff::text LIKE", q: "SELECT id FROM audit_log WHERE tenant_id = $1 AND id = $2 AND diff::text LIKE '%x%'"},
		{name: "ORDER BY diff_salt", q: "SELECT id FROM audit_log WHERE tenant_id = $1 AND id = $2 ORDER BY diff_salt"},
		{name: "FROM audit_log_default", q: "SELECT diff FROM audit_log_default WHERE tenant_id = $1 AND id = $2"},
		{name: "FROM ONLY audit_log", q: "SELECT diff FROM ONLY audit_log WHERE tenant_id = $1 AND id = $2"},
		{name: "COPY (SELECT diff ...)", q: "COPY (SELECT diff FROM audit_log) TO STDOUT", copy: true},
		{name: "COPY audit_log (diff)", q: "COPY audit_log (diff) TO STDOUT", copy: true},
		{name: "CTE SELECT *", q: "WITH c AS (SELECT * FROM audit_log WHERE tenant_id = $1 AND id = $2) SELECT * FROM c"},
		{name: "LATERAL json_array_elements(a.diff)", q: "SELECT e FROM audit_log a, LATERAL json_array_elements(a.diff) e WHERE a.tenant_id = $1 AND a.id = $2"},
		{name: "UPDATE ... RETURNING diff", q: "UPDATE audit_log SET seq = seq WHERE tenant_id = $1 AND id = $2 RETURNING diff"},
		{name: "INSERT ... RETURNING diff_salt", q: `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, source_ref)
			VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, 'system', 'system', 'iso-audit') RETURNING diff_salt`},
	}
}

// runProbe runs p as pool's role in tenant's transaction, rolled back.
func runProbe(ctx context.Context, pool *pgxpool.Pool, tenant string, p diffProbe, fx auditFixtureA) error {
	return inTx(ctx, pool, uuid.MustParse(tenant), func(tx pgx.Tx) error {
		if p.copy {
			_, err := tx.Conn().PgConn().CopyTo(ctx, io.Discard, p.q)
			return err
		}
		rows, err := tx.Query(ctx, p.q, tenant, fx.entry)
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	})
}

// auditDiffDenied is ADR-004-A1/diff-denied. With tenant A set, every probe
// against tenant A's own sealed entry, which holds a diff and a salt, is
// refused with 42501 for every role but the reader. Controls: nava_app reads
// the entry's other columns, and the reader reads its diff, so a refusal is
// not a missing row or a broken connection. Then the reader's own limits:
// INSERT, UPDATE and DELETE on audit_log, and SELECT on records, are 42501.
func auditDiffDenied(ctx context.Context, cl Cluster) error {
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
	logins, err := auditManifestLogins()
	if err != nil {
		return err
	}
	// Every platform login but the reader is probed: a role added to the
	// manifest later is probed too.
	for _, r := range logins {
		if r != audited.ReaderRole && !slices.Contains(diffDeniedRoles(), r) {
			return fmt.Errorf("control: platform login %s is not in the diff-denied roles %v", r, diffDeniedRoles())
		}
	}

	app, err := cl.Pool("nava_app")
	if err != nil {
		return err
	}
	defer app.Close()
	var n int
	if err := inTx(ctx, app, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM (SELECT tenant_id, id, diff_sha256 FROM audit_log WHERE tenant_id = $1 AND id = $2) s",
			TenantA, fx.entry).Scan(&n)
	}); err != nil || n != 1 {
		return fmt.Errorf("control: nava_app in tenant A reads its entry's other columns: %d rows, %s; want 1", n, errText(err))
	}
	reader, err := cl.Pool(audited.ReaderRole)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := inTx(ctx, reader, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM (SELECT diff, diff_salt FROM audit_log WHERE tenant_id = $1 AND id = $2 AND diff IS NOT NULL) s",
			TenantA, fx.entry).Scan(&n)
	}); err != nil || n != 1 {
		return fmt.Errorf("control: nava_audit_reader in tenant A reads the entry's diff: %d rows, %s; want 1", n, errText(err))
	}

	var bad []string
	for _, role := range diffDeniedRoles() {
		pool, err := cl.Pool(role)
		if err != nil {
			return err
		}
		for _, p := range diffProbes() {
			if err := runProbe(ctx, pool, TenantA, p, fx); sqlState(err) != "42501" {
				bad = append(bad, fmt.Sprintf("%s: %s: %s; want 42501", role, p.name, errText(err)))
			}
		}
		pool.Close()
	}
	for _, q := range []string{
		`INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, source_ref)
		 VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, 'system', 'system', 'iso-audit')`,
		"UPDATE audit_log SET seq = seq WHERE tenant_id = $1 AND id = $2",
		"DELETE FROM audit_log WHERE tenant_id = $1 AND id = $2",
		"SELECT count(*) FROM records WHERE tenant_id = $1 AND id <> $2",
		"SELECT txid FROM audit_log WHERE tenant_id = $1 AND id = $2",
	} {
		if _, err := execAs(ctx, reader, TenantA, q, TenantA, fx.entry); sqlState(err) != "42501" {
			bad = append(bad, fmt.Sprintf("%s: %s: %s; want 42501", audited.ReaderRole, strings.Join(strings.Fields(q)[:3], " "), errText(err)))
		}
	}
	if len(bad) > 0 {
		return errors.New(strings.Join(bad, "\n"))
	}
	return nil
}

// auditReaderScoped is ADR-004-A1/reader-scoped. Through read.Reader on the
// reader's own pool (as nava-core reads), tenant A's call returns tenant A's
// entry with its raw diff and salt, and tenant B's call for the same subject
// returns nothing. As raw SQL on the reader's login, the forged-tenant read
// (tenant B set, tenant A's tenant and id named) and the no-tenant read reach
// 0 rows, and txid is refused with 42501.
func auditReaderScoped(ctx context.Context, cl Cluster) error {
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
	pool, err := cl.AuditReader()
	if err != nil {
		return err
	}
	r := read.Reader{Pool: pool, Masker: read.StaticMasker{}}
	subject := uuid.MustParse(fx.subject)
	got, err := r.List(ctx, uuid.MustParse(TenantA), read.Filter{SubjectID: &subject}, nil, 100)
	if err != nil || len(got) != 1 || got[0].ID.String() != fx.entry || !strings.Contains(string(got[0].Diff), "a-only") || len(got[0].DiffSalt) == 0 {
		return fmt.Errorf("control: read.Reader on the reader pool in tenant A returns %d entries, %s; want tenant A's entry with its diff and salt", len(got), errText(err))
	}
	if got, err := r.List(ctx, uuid.MustParse(TenantB), read.Filter{SubjectID: &subject}, nil, 100); err != nil || len(got) != 0 {
		return fmt.Errorf("read.Reader in tenant B for tenant A's subject returns %d entries, %s; want none", len(got), errText(err))
	}

	login, err := cl.Pool(audited.ReaderRole)
	if err != nil {
		return err
	}
	defer login.Close()
	count := func(tenant uuid.UUID) (int, error) {
		var n int
		err := inTx(ctx, login, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM (SELECT diff, diff_salt FROM audit_log WHERE tenant_id = $1 AND id = $2) s",
				TenantA, fx.entry).Scan(&n)
		})
		return n, err
	}
	var bad []string
	if n, err := count(uuid.MustParse(TenantA)); err != nil || n != 1 {
		bad = append(bad, fmt.Sprintf("control: tenant A reads its own diff: %d rows, %s; want 1", n, errText(err)))
	}
	if n, err := count(uuid.MustParse(TenantB)); err != nil || n != 0 {
		bad = append(bad, fmt.Sprintf("tenant B set, tenant A's tenant and id named: %d rows, %s; want 0", n, errText(err)))
	}
	if n, err := count(uuid.Nil); err != nil || n != 0 {
		bad = append(bad, fmt.Sprintf("no tenant set: %d rows, %s; want 0", n, errText(err)))
	}
	if _, err := execAs(ctx, login, TenantA, "SELECT txid FROM audit_log WHERE tenant_id = $1 AND id = $2", TenantA, fx.entry); sqlState(err) != "42501" {
		bad = append(bad, fmt.Sprintf("SELECT txid: %s; want 42501", errText(err)))
	}
	if len(bad) > 0 {
		return errors.New(strings.Join(bad, "\n"))
	}
	return nil
}

// auditReaderMasks is ADR-004-A1/reader-masks. Tenant A's iso_et_sec type
// has the x-nava-secret property token. The record write path refuses a
// secret value (CI-A5), so the entry that sets it is written as the
// superuser, as a pre-S10 row would be. Tenant A's auditor (audit.read, no
// mask permission) reads it through GET /api/v1/audit/entries on the reader
// pool: the value is {"redacted":"mask"}, diff_redacted is true and there is
// no salt. Without nava_audit_reader in entity_definition_versions_select
// the masker sees no version and fails closed, a 500 (A1-4), so this case
// goes red. The fixture also asserts D-3-4: every definition, in every
// tenant, has at least one version row.
func auditReaderMasks(ctx context.Context, e *etIso) error {
	create := fmt.Sprintf(`{"key": "iso_et_sec", "name": "Iso sec", "number_prefix": "ISS", "definition": %s}`, auditSecDefinition("v1", ""))
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerA, create, nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A creating iso_et_sec = %d %s, want 201", code, body)
	}
	var def string
	if err := e.super.QueryRow(ctx, "SELECT id::text FROM entity_definitions WHERE tenant_id = $1 AND key = 'iso_et_sec'", TenantA).Scan(&def); err != nil {
		return fmt.Errorf("control: iso_et_sec's id: %w", err)
	}
	var orphans int
	if err := e.super.QueryRow(ctx, `SELECT count(*) FROM entity_definitions d WHERE NOT EXISTS (
		SELECT 1 FROM entity_definition_versions v WHERE v.tenant_id = d.tenant_id AND v.definition_id = d.id)`).Scan(&orphans); err != nil {
		return err
	}
	if orphans != 0 {
		return fmt.Errorf("D-3-4: %d entity definitions have no version row; the fail-closed masker (A1-4) assumes every one has", orphans)
	}
	secret := "iso-secret-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	subject := uuid.NewString()
	if _, err := e.super.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
		version_after, actor_system, source, source_ref, diff, diff_salt, diff_sha256)
		VALUES ($1, gen_random_uuid(), 'record', 'update', 'record', $2, $3, 2, 'system', 'system', 'iso-audit',
		        json_build_array(json_build_object('op', 'add', 'path', '/token', 'value', $4::text),
		                         json_build_object('op', 'add', 'path', '/note', 'value', 'plain')),
		        decode('00112233445566778899aabbccddeeff', 'hex'), sha256('d'))`, TenantA, subject, def, secret); err != nil {
		return fmt.Errorf("seed the secret entry: %w", err)
	}
	code, body := e.do(ctx, http.MethodGet, "/api/v1/audit/entries?subject_id="+subject, auditCallerAAuditor, "", nil)
	if code != http.StatusOK {
		return fmt.Errorf("tenant A's auditor reads the secret entry = %d %s; want 200 (a 500 is the masker finding no version: is nava_audit_reader in entity_definition_versions_select?)", code, body)
	}
	if strings.Contains(body, secret) {
		return fmt.Errorf("the response carries the secret value: %s", body)
	}
	var page struct {
		Items []struct {
			Diff         json.RawMessage `json:"diff"`
			DiffRedacted bool            `json:"diff_redacted"`
			DiffSalt     *string         `json:"diff_salt"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil || len(page.Items) != 1 {
		return fmt.Errorf("control: the page holds %d items, %s; want the one entry: %s", len(page.Items), errText(err), body)
	}
	it := page.Items[0]
	var ops []struct {
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(it.Diff, &ops); err != nil || len(ops) != 2 {
		return fmt.Errorf("the diff is not the two-op patch: %s: %s", errText(err), it.Diff)
	}
	if ops[0].Path != "/token" || string(ops[0].Value) != read.MaskedValue {
		return fmt.Errorf("/token's value = %s, want %s", ops[0].Value, read.MaskedValue)
	}
	if ops[1].Path != "/note" || string(ops[1].Value) != `"plain"` {
		return fmt.Errorf("control: /note's value = %s, want the stored \"plain\"", ops[1].Value)
	}
	if !it.DiffRedacted || it.DiffSalt != nil {
		return fmt.Errorf("diff_redacted = %v, diff_salt = %v; want true and no salt", it.DiffRedacted, it.DiffSalt)
	}
	return nil
}
