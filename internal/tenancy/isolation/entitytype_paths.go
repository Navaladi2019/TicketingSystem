package isolation

import (
	"bytes"
	"context"
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

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
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
	h     http.Handler
	super *pgx.Conn
	keys  int
	stop  func()
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
		etCallerA: {TenantID: uuid.MustParse(TenantA), PrincipalID: uuid.MustParse(principalA1), Groups: []uuid.UUID{uuid.MustParse(adminGroupA)}},
		etCallerB: {TenantID: uuid.MustParse(TenantB), PrincipalID: uuid.MustParse(principalB1)},
	}
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, tokens, pgstore.New(pool),
		server.WithPool(pool), server.WithCursors(kr))
	if err != nil {
		_ = tel.Shutdown(context.Background())
		return nil, err
	}
	return &etIso{h: h, super: super, stop: func() { _ = tel.Shutdown(context.Background()) }}, nil
}

// cleanup deletes every row the cases write, as the superuser.
func (e *etIso) cleanup() error {
	ctx := context.Background()
	for _, q := range []string{
		`DELETE FROM entity_definition_versions WHERE definition_id IN (SELECT id FROM entity_definitions WHERE key LIKE 'iso\_et\_%')`,
		`DELETE FROM entity_definitions WHERE key LIKE 'iso\_et\_%'`,
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
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if method == http.MethodPost {
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
	return rec.Code, rec.Body.String()
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
	switch {
	case code == http.StatusCreated:
	case code == http.StatusBadRequest || code == http.StatusUnprocessableEntity:
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
	switch {
	case code == http.StatusCreated:
	case code == http.StatusBadRequest || code == http.StatusUnprocessableEntity:
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
