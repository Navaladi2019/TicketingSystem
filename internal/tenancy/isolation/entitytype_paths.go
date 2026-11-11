package isolation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// entityTypePaths are the entity-type routes (P3-07, TICK-345; routes from
// TICK-343 and TICK-365): the real nava-core handler with its pool and
// cursor keyring, over the suite's database as nava_app through PgBouncer.
// Token validation is CI-5's; here a token resolves straight to a seeded
// principal, so each case starts at the tenant the token proved. Tenant A's
// admin (a1, in tenant A's admins group) aims at tenant B's types; tenant B
// owns the key incident, which tenant A does not have, and version 2 of
// ticket, where tenant A's ticket has only version 1. A case deletes every
// row it writes (keys iso_et_*, Idempotency-Keys iso-et-*) before it returns.
func entityTypePaths() []Path {
	db := func(c, name string, fn func(context.Context, *etIso) error) Path {
		return Path{Name: "entity types: " + name, Case: c, Role: "nava_app", DB: func(ctx context.Context, cl Cluster) error {
			e, err := newETIso(ctx, cl)
			if err != nil {
				return err
			}
			defer e.stop()
			if err := e.cleanup(); err != nil { // rows a killed earlier run left
				return err
			}
			err = fn(ctx, e)
			if cerr := e.cleanup(); err == nil {
				err = cerr
			}
			return err
		}}
	}
	return []Path{
		db("CI-8", "control: tenant B reads its own seeded ticket, current and version 2, with 200 and its own body", etReadOwnSeeded),
		db("CI-8", "tenant A reading tenant B's key gets the unknown key's 404, byte for byte", etReadOtherTenantsKey),
		db("CI-8", "tenant A publishing to tenant B's key gets the unknown key's 404, byte for byte, and B's type is unchanged", etPublishOtherTenantsKey),
		db("CI-2", "forged tenant in the body of POST /api/v1/entity-types", etForgedCreate),
		db("CI-2", "forged tenant in the body of POST /api/v1/entity-types/{key}/versions", etForgedPublish),
		db("CI-2", "tenant A's list holds none of tenant B's types, with tenant B named in a header and the query", etListOwnOnly),
		db("CI-8", "a list cursor issued to tenant B is refused for tenant A with 400", etCursorReplay),
	}
}

// Seeded principals and groups (testdata/seed.sql).
const (
	principalA1  = "0a000000-0000-4000-8000-0000000003a1"
	adminGroupA  = "0a000000-0000-4000-8000-0000000004a1"
	principalB1  = "0b000000-0000-4000-8000-0000000003b1"
	incidentDefB = "0b000000-0000-4000-8000-0000000006b2"
	etCallerA    = "caller-a1-admin"
	etCallerB    = "caller-b1"
	// etCallerBAdmin is tenant B's admin (testdata/seed.sql: principal b2 in the
	// tenant admin group b2), who can create entity types in tenant B.
	etCallerBAdmin = "caller-b2-admin"
	principalB2    = "0b000000-0000-4000-8000-0000000003b2"
	adminGroupB    = "0b000000-0000-4000-8000-0000000004b2"
)

// etSchema and etDefTail form a definition that compiles: create and publish
// bodies are built from them.
const (
	etSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,` +
		`"required":["state"],"properties":{"state":{"type":"string","enum":["new","closed"],"default":"new"},"note":{"type":"string","description":"%s"}}}`
	etDefTail = `"fixed_columns": {"state": "state"}, "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}`
)

func etDefinition(note string) string {
	return fmt.Sprintf(`{"schema": %s, %s}`, fmt.Sprintf(etSchema, note), etDefTail)
}

// etCreateBody is a create body; extra is spliced in as more members.
func etCreateBody(key, prefix, extra string) string {
	return fmt.Sprintf(`{%s"key": %q, "name": "Iso %s", "number_prefix": %q, "definition": %s}`, extra, key, key, prefix, etDefinition("v1"))
}

func etPublishBody(note, extra string) string {
	return fmt.Sprintf(`{%s"definition": %s}`, extra, etDefinition(note))
}

// etCallers resolves the two bearer values of these cases.
type etCallers map[string]identity.Principal

func (m etCallers) Resolve(_ context.Context, raw string) (identity.Principal, error) {
	if p, ok := m[raw]; ok {
		return p, nil
	}
	return identity.Principal{}, fmt.Errorf("%w: unknown token", identity.ErrInvalidToken)
}

type etIso struct {
	h      http.Handler
	pool   *pgxpool.Pool // nava_app
	tokens etCallers
	super  *pgx.Conn
	keys   int
	stop   func()
	// cluster is the suite's database, for cases that also drive it as
	// another role (the audit paths seal as nava_worker).
	cluster Cluster
}

func newETIso(ctx context.Context, c Cluster) (*etIso, error) {
	pool, err := c.Pool("nava_app")
	if err != nil {
		return nil, err
	}
	super, err := c.Super()
	if err != nil {
		return nil, err
	}
	tel, err := telemetry.New(ctx, config.Telemetry{})
	if err != nil {
		return nil, err
	}
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("e"), 32)})
	if err != nil {
		return nil, err
	}
	tokens := etCallers{
		etCallerA:      {TenantID: uuid.MustParse(TenantA), PrincipalID: uuid.MustParse(principalA1), Groups: []uuid.UUID{uuid.MustParse(adminGroupA)}},
		etCallerB:      {TenantID: uuid.MustParse(TenantB), PrincipalID: uuid.MustParse(principalB1)},
		etCallerBAdmin: {TenantID: uuid.MustParse(TenantB), PrincipalID: uuid.MustParse(principalB2), Groups: []uuid.UUID{uuid.MustParse(adminGroupB)}},
		// The audit readers (P5-19, TICK-436): each tenant's auditor (a member
		// of its auditors group) and an agent (in a group that is neither).
		auditCallerAAuditor: {TenantID: uuid.MustParse(TenantA), PrincipalID: uuid.MustParse(principalA2), Groups: []uuid.UUID{uuid.MustParse(auditorGroupA)}},
		auditCallerAAgent:   {TenantID: uuid.MustParse(TenantA), PrincipalID: uuid.MustParse(principalA2)},
		auditCallerBAuditor: {TenantID: uuid.MustParse(TenantB), PrincipalID: uuid.MustParse(principalB1), Groups: []uuid.UUID{uuid.MustParse(auditorGroupB)}},
		auditCallerBAgent:   {TenantID: uuid.MustParse(TenantB), PrincipalID: uuid.MustParse(principalB1), Groups: []uuid.UUID{uuid.MustParse(GroupB1)}},
	}
	// The audit routes read on the reader's own pool, as nava-core does
	// (ADR-004-A1 A1-5).
	reader, err := c.AuditReader()
	if err != nil {
		_ = tel.Shutdown(context.Background())
		return nil, err
	}
	// A create takes its number on the allocator pool (ADR-005 L2s): on
	// pool, one connection the create holds, it would starve.
	alloc, err := c.AllocatorPool()
	if err != nil {
		_ = tel.Shutdown(context.Background())
		return nil, err
	}
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, tokens, pgstore.New(pool),
		server.WithPool(pool), server.WithAllocatorPool(alloc), server.WithAuditReader(reader), server.WithCursors(kr))
	if err != nil {
		_ = tel.Shutdown(context.Background())
		return nil, err
	}
	return &etIso{h: h, pool: pool, tokens: tokens, super: super, stop: func() { _ = tel.Shutdown(context.Background()) }, cluster: c}, nil
}

// cleanup deletes every row the cases write, as the superuser.
func (e *etIso) cleanup() error {
	ctx := context.Background()
	for _, q := range []string{
		`DELETE FROM audit_log WHERE subject_type = 'record' AND subject_id IN (SELECT r.id FROM records r JOIN entity_definitions d
			ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id WHERE d.key LIKE 'iso_et_%')`,
		// The delete writes each record's entry, as the records commit-check
		// trigger requires (P5-13c); the next statement removes those
		// entries before their definitions go.
		`WITH r AS (DELETE FROM records WHERE entity_definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')
			RETURNING tenant_id, id, version, entity_definition_id)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id, version_after, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'record', 'delete', 'record', id, entity_definition_id, version, 'system', 'system' FROM r`,
		`DELETE FROM audit_log WHERE subject_type = 'record' AND action = 'delete' AND entity_definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`,
		`DELETE FROM record_counters WHERE entity_definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`,
		// acl_versions has the commit check too (P3-35): the delete writes one
		// acl_policy delete entry per type, and the next statement removes
		// them with the entries acltest.Publish and the PUT route wrote.
		`WITH a AS (DELETE FROM acl_versions WHERE entity_definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')
			RETURNING tenant_id, entity_definition_id)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'config', 'delete', 'acl_policy', entity_definition_id, 'system', 'system'
			FROM (SELECT DISTINCT tenant_id, entity_definition_id FROM a) d`,
		`DELETE FROM audit_log WHERE subject_type = 'acl_policy' AND subject_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`,
		// The entity-definition deletes write one config delete entry per
		// type, as their commit-check triggers require (P5-32, migration B).
		`WITH v AS (DELETE FROM entity_definition_versions WHERE definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')
			RETURNING tenant_id, definition_id)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'config', 'delete', 'entity_definition', definition_id, 'system', 'system'
			FROM (SELECT DISTINCT tenant_id, definition_id FROM v) d`,
		`WITH d AS (DELETE FROM entity_definitions WHERE key LIKE 'iso\_et\_%' RETURNING tenant_id, id)
			INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source)
			SELECT tenant_id, gen_random_uuid(), 'config', 'delete', 'entity_definition', id, 'system', 'system' FROM d`,
		`DELETE FROM idempotency_keys WHERE key LIKE 'iso-et-%'`,
	} {
		if _, err := e.super.Exec(ctx, q); err != nil {
			return fmt.Errorf("cleanup: %w", err)
		}
	}
	return nil
}

// do sends a request with token; a POST gets a fresh Idempotency-Key.
func (e *etIso) do(ctx context.Context, method, path, token, body string, header http.Header) (int, string) {
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" && header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == http.MethodPost || method == http.MethodPatch {
		e.keys++
		req.Header.Set("Idempotency-Key", fmt.Sprintf("iso-et-%d", e.keys))
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if method == http.MethodPost && path == "/api/v1/entity-types" && rec.Code == http.StatusCreated {
		if err := e.allowTenantCallers(ctx, token, body); err != nil {
			return http.StatusInternalServerError, "harness: publish the test ACL: " + err.Error()
		}
	}
	return rec.Code, rec.Body.String()
}

// allowTenantCallers publishes, for an entity type token's tenant just
// created, an ACL that lets every caller of that tenant read and write every
// record: record authorization fails closed (P3-31), and these cases test
// the tenant wall, not the ACL. It writes as nava_app in the tenant's own
// transaction; cleanup deletes the row with its type.
func (e *etIso) allowTenantCallers(ctx context.Context, token, body string) error {
	var in struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		return err
	}
	tenant := e.tokens[token].TenantID
	var users []uuid.UUID
	for _, p := range e.tokens {
		if p.TenantID == tenant && !slices.Contains(users, p.PrincipalID) {
			users = append(users, p.PrincipalID)
		}
	}
	return tenancy.InTenantTx(ctx, e.pool, tenant, func(tx pgx.Tx) error {
		_, err := acltest.Publish(ctx, tx, in.Key, acltest.AllowUsers(users...))
		return err
	})
}

// create makes key in tenant A, the positive control that the route is live
// and the token is an admin.
func (e *etIso) create(ctx context.Context, key, prefix string) error {
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerA, etCreateBody(key, prefix, ""), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A creating %s = %d %s, want 201", key, code, body)
	}
	return nil
}

// tenantsOf returns the tenant of every entity_definitions row with key and
// of every version under them, read as the superuser past RLS, so a row filed
// under another tenant cannot hide.
func (e *etIso) tenantsOf(ctx context.Context, key string) (defs, versions []string, err error) {
	read := func(q string) ([]string, error) {
		rows, err := e.super.Query(ctx, q, key)
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, pgx.RowTo[string])
	}
	if defs, err = read(`SELECT tenant_id::text FROM entity_definitions WHERE key = $1`); err != nil {
		return nil, nil, err
	}
	versions, err = read(`SELECT v.tenant_id::text FROM entity_definition_versions v
		JOIN entity_definitions d ON d.id = v.definition_id WHERE d.key = $1`)
	return defs, versions, err
}

// sameNotFound requires both requests to give 404 with identical bytes.
func (e *etIso) sameNotFound(ctx context.Context, method, body, other, unknown string) error {
	oc, ob := e.do(ctx, method, other, etCallerA, body, nil)
	uc, ub := e.do(ctx, method, unknown, etCallerA, body, nil)
	switch {
	case uc != http.StatusNotFound:
		return fmt.Errorf("control: %s %s = %d %s, want 404", method, unknown, uc, ub)
	case oc != http.StatusNotFound:
		return fmt.Errorf("%s %s (tenant B's) = %d %s, want 404", method, other, oc, ob)
	case ob != ub:
		return fmt.Errorf("%s %s (tenant B's) 404 differs from the unknown key's:\n%s\n%s", method, other, ob, ub)
	}
	return nil
}

func etReadOtherTenantsKey(ctx context.Context, e *etIso) error {
	if err := e.create(ctx, "iso_et_read", "ISR"); err != nil {
		return err
	}
	if code, body := e.do(ctx, http.MethodGet, "/api/v1/entity-types/iso_et_read", etCallerA, "", nil); code != http.StatusOK {
		return fmt.Errorf("control: tenant A reading its own type = %d %s, want 200", code, body)
	}
	for _, c := range [][2]string{
		{"/api/v1/entity-types/incident", "/api/v1/entity-types/iso_et_none"},
		{"/api/v1/entity-types/incident/versions/1", "/api/v1/entity-types/iso_et_none/versions/1"},
		// Tenant A has ticket v1 only; tenant B's ticket has v2.
		{"/api/v1/entity-types/ticket/versions/2", "/api/v1/entity-types/iso_et_read/versions/2"},
	} {
		if err := e.sameNotFound(ctx, http.MethodGet, "", c[0], c[1]); err != nil {
			return err
		}
	}
	return nil
}

// etSeededB is what tenant B's seeded ticket version 2 must read back as
// (testdata/seed.sql). fixed_columns and states must be JSON objects: the
// store scans them into map[string]string and metadata.StateModel.
type etSeededB struct {
	Key          string            `json:"key"`
	Name         string            `json:"name"`
	NumberPrefix string            `json:"number_prefix"`
	Version      int               `json:"version"`
	Current      bool              `json:"current"`
	Schema       json.RawMessage   `json:"schema"`
	SchemaSHA256 string            `json:"schema_sha256"`
	FixedColumns map[string]string `json:"fixed_columns"`
	States       struct {
		Initial string   `json:"initial"`
		States  []string `json:"states"`
	} `json:"states"`
	Compat string `json:"compat"`
}

// etReadOwnSeeded is the positive control for the seeded entity types
// (TICK-378): tenant B reads its own ticket, by its current version and by
// version 2, and gets 200 with its own body. Without it a seed the store
// cannot scan turns every read into a 500 while the 404 cases above, which
// never reach a seeded row of the caller, still pass.
func etReadOwnSeeded(ctx context.Context, e *etIso) error {
	const schemaB2 = `{"type": "object", "title": "B v2"}`
	sum := sha256.Sum256([]byte(schemaB2))
	var bodies []string
	for _, path := range []string{"/api/v1/entity-types/ticket", "/api/v1/entity-types/ticket/versions/2"} {
		code, body := e.do(ctx, http.MethodGet, path, etCallerB, "", nil)
		if code != http.StatusOK {
			return fmt.Errorf("tenant B reading its own %s = %d %s, want 200", path, code, body)
		}
		var got etSeededB
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			return fmt.Errorf("tenant B reading its own %s: %w: %s", path, err, body)
		}
		switch {
		case got.Key != "ticket" || got.Name != "Ticket B" || got.NumberPrefix != "TCK":
			return fmt.Errorf("tenant B reading its own %s got key %q name %q prefix %q, want ticket, Ticket B, TCK: %s",
				path, got.Key, got.Name, got.NumberPrefix, body)
		case got.Version != 2 || !got.Current || got.Compat != "compatible":
			return fmt.Errorf("tenant B reading its own %s got version %d current %t compat %q, want 2, true, compatible: %s",
				path, got.Version, got.Current, got.Compat, body)
		case string(got.Schema) != schemaB2:
			return fmt.Errorf("tenant B reading its own %s got schema %s, want %s verbatim", path, got.Schema, schemaB2)
		case got.SchemaSHA256 != hex.EncodeToString(sum[:]):
			return fmt.Errorf("tenant B reading its own %s got schema_sha256 %s, want %x", path, got.SchemaSHA256, sum)
		case got.FixedColumns == nil || got.States.Initial != "new" || !slices.Equal(got.States.States, []string{"new"}):
			return fmt.Errorf("tenant B reading its own %s got fixed_columns %v states %+v, want {} and the seeded state model: %s",
				path, got.FixedColumns, got.States, body)
		}
		bodies = append(bodies, body)
	}
	if bodies[0] != bodies[1] {
		return fmt.Errorf("tenant B's current ticket and its version 2 differ:\n%s\n%s", bodies[0], bodies[1])
	}
	return nil
}

func etPublishOtherTenantsKey(ctx context.Context, e *etIso) error {
	if err := e.create(ctx, "iso_et_pub", "ISP"); err != nil {
		return err
	}
	if code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types/iso_et_pub/versions", etCallerA, etPublishBody("v2", ""), nil); code != http.StatusCreated {
		return fmt.Errorf("control: tenant A publishing its own type = %d %s, want 201", code, body)
	}
	if err := e.sameNotFound(ctx, http.MethodPost, etPublishBody("v2", ""),
		"/api/v1/entity-types/incident/versions", "/api/v1/entity-types/iso_et_none/versions"); err != nil {
		return err
	}
	var cur, n int
	if err := e.super.QueryRow(ctx, `SELECT d.current_version, (SELECT count(*) FROM entity_definition_versions v WHERE v.definition_id = d.id)
		FROM entity_definitions d WHERE d.id = $1`, incidentDefB).Scan(&cur, &n); err != nil {
		return err
	}
	if cur != 1 || n != 1 {
		return fmt.Errorf("tenant B's incident has current_version %d and %d versions after tenant A's publish, want 1 and 1", cur, n)
	}
	return nil
}

// onlyTenantA requires every row to be tenant A's, and want rows of each kind.
func onlyTenantA(what string, tenants []string, want int) error {
	if slices.ContainsFunc(tenants, func(t string) bool { return t != TenantA }) {
		return fmt.Errorf("%s: rows filed under %v, want only tenant A (the handler trusted the body's tenant_id)", what, tenants)
	}
	if len(tenants) != want {
		return fmt.Errorf("%s: %d rows, want %d", what, len(tenants), want)
	}
	return nil
}

func etForgedCreate(ctx context.Context, e *etIso) error {
	forged := fmt.Sprintf(`"tenant_id": %q, `, TenantB)
	code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types", etCallerA, etCreateBody("iso_et_forged", "ISF", forged),
		http.Header{"X-Tenant-Id": {TenantB}})
	want := 1
	switch code {
	case http.StatusCreated:
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		want = 0 // the body was refused: nothing may be written anywhere
	default: // 401, 403 or 404 would mean the case never reached the write
		return fmt.Errorf("create with a forged tenant = %d %s, want 201, or 400 or 422 for the unknown member", code, body)
	}
	defs, versions, err := e.tenantsOf(ctx, "iso_et_forged")
	if err != nil {
		return err
	}
	if err := onlyTenantA("entity_definitions", defs, want); err != nil {
		return err
	}
	return onlyTenantA("entity_definition_versions", versions, want)
}

func etForgedPublish(ctx context.Context, e *etIso) error {
	if err := e.create(ctx, "iso_et_forged_pub", "ISG"); err != nil {
		return err
	}
	forged := fmt.Sprintf(`"tenant_id": %q, `, TenantB)
	code, body := e.do(ctx, http.MethodPost, "/api/v1/entity-types/iso_et_forged_pub/versions", etCallerA, etPublishBody("v2", forged),
		http.Header{"X-Tenant-Id": {TenantB}})
	want := 2
	switch code {
	case http.StatusCreated:
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		want = 1
	default:
		return fmt.Errorf("publish with a forged tenant = %d %s, want 201, or 400 or 422 for the unknown member", code, body)
	}
	defs, versions, err := e.tenantsOf(ctx, "iso_et_forged_pub")
	if err != nil {
		return err
	}
	if err := onlyTenantA("entity_definitions", defs, 1); err != nil {
		return err
	}
	return onlyTenantA("entity_definition_versions", versions, want)
}

type etList struct {
	Items []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func etDecodeList(body string) (etList, error) {
	var l etList
	err := json.Unmarshal([]byte(body), &l)
	return l, err
}

func etListOwnOnly(ctx context.Context, e *etIso) error {
	rows, err := e.super.Query(ctx, `SELECT key FROM entity_definitions WHERE tenant_id = $1 ORDER BY key`, TenantA)
	if err != nil {
		return err
	}
	own, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for i, c := range []struct {
		path   string
		header http.Header
	}{
		{"/api/v1/entity-types", http.Header{"X-Tenant-Id": {TenantB}, "X-Nava-Tenant": {TenantB}}},
		{"/api/v1/entity-types?tenant_id=" + TenantB, nil},
	} {
		code, body := e.do(ctx, http.MethodGet, c.path, etCallerA, "", c.header)
		if i == 1 && code == http.StatusBadRequest {
			continue // refusing the unknown parameter keeps tenant B out too
		}
		if code != http.StatusOK {
			return fmt.Errorf("GET %s = %d %s, want 200", c.path, code, body)
		}
		l, err := etDecodeList(body)
		if err != nil {
			return fmt.Errorf("GET %s: %w: %s", c.path, err, body)
		}
		var keys []string
		for _, it := range l.Items {
			keys = append(keys, it.Key)
			if strings.Contains(it.Name, " B") {
				return fmt.Errorf("GET %s returned tenant B's %q", c.path, it.Name)
			}
		}
		if !slices.Equal(keys, own) {
			return fmt.Errorf("GET %s listed %v, want tenant A's %v", c.path, keys, own)
		}
	}
	return nil
}

func etCursorReplay(ctx context.Context, e *etIso) error {
	code, body := e.do(ctx, http.MethodGet, "/api/v1/entity-types?limit=1", etCallerB, "", nil)
	if code != http.StatusOK {
		return fmt.Errorf("control: tenant B's first page = %d %s, want 200", code, body)
	}
	l, err := etDecodeList(body)
	if err != nil {
		return fmt.Errorf("control: tenant B's first page: %w: %s", err, body)
	}
	if l.NextCursor == nil {
		return fmt.Errorf("control: tenant B's first page has no next_cursor: %s", body)
	}
	next := "/api/v1/entity-types?limit=1&cursor=" + url.QueryEscape(*l.NextCursor)
	if code, body := e.do(ctx, http.MethodGet, next, etCallerB, "", nil); code != http.StatusOK {
		return fmt.Errorf("control: tenant B replaying its own cursor = %d %s, want 200", code, body)
	}
	code, body = e.do(ctx, http.MethodGet, next, etCallerA, "", nil)
	if code != http.StatusBadRequest || !strings.Contains(body, `"invalid_cursor"`) {
		return fmt.Errorf("tenant A replaying tenant B's cursor = %d %s, want 400 invalid_cursor", code, body)
	}
	return nil
}
