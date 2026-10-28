package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

//nolint:gochecknoglobals // fixed test ids, never written
var (
	tenantA    = uuid.MustParse("0a000000-0000-4000-8000-00000000000a")
	tenantB    = uuid.MustParse("0b000000-0000-4000-8000-00000000000b")
	principalA = uuid.MustParse("0a000000-0000-4000-8000-0000000003a1")
	groupA     = uuid.MustParse("0a000000-0000-4000-8000-0000000004a1")
)

// stubResolver resolves fixed tokens; any other token is invalid.
type stubResolver struct{}

func (stubResolver) Resolve(_ context.Context, raw string) (identity.Principal, error) {
	switch raw {
	case "token-a":
		return identity.Principal{TenantID: tenantA, PrincipalID: principalA, Groups: []uuid.UUID{groupA}, IdPAlias: "idp-a"}, nil
	case "token-unresolved":
		return identity.Principal{}, fmt.Errorf("%w: no tenant binding", identity.ErrTenantUnresolved)
	case "token-store-down":
		return identity.Principal{}, errors.New("pgstore: dial tcp 10.0.0.9:5432: secret-host-detail")
	}
	return identity.Principal{}, fmt.Errorf("%w: signature", identity.ErrInvalidToken)
}

// stubPaths reads the stored raw paths of tenant A's principal; any other
// caller is in no group.
type stubPaths struct{}

func (stubPaths) GroupPaths(_ context.Context, tenantID, principalID uuid.UUID) ([]identity.GroupPath, error) {
	if tenantID == tenantA && principalID == principalA {
		return []identity.GroupPath{{ID: groupA, Path: identity.TenantGroupPrefix(tenantA) + "service-desk"}}, nil
	}
	return nil, nil
}

// fakePaths returns fixed rows or an error.
type fakePaths struct {
	rows []identity.GroupPath
	err  error
}

func (f fakePaths) GroupPaths(context.Context, uuid.UUID, uuid.UUID) ([]identity.GroupPath, error) {
	return f.rows, f.err
}

func get(t *testing.T, h http.Handler, target string, hdr http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeMe(t *testing.T, rec *httptest.ResponseRecorder) api.Me {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/me = %d %s, want 200", rec.Code, rec.Body)
	}
	var me api.Me
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("body: %v: %s", err, rec.Body)
	}
	return me
}

func TestMeReturnsResolvedPrincipal(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	for _, scheme := range []string{"Bearer", "bearer", "BEARER"} {
		me := decodeMe(t, get(t, h, "/api/v1/me", http.Header{"Authorization": {scheme + " token-a"}}))
		want := api.Me{
			UserId: principalA, TenantId: tenantA, Groups: []uuid.UUID{groupA}, IdpAlias: "idp-a",
			GroupPaths: []api.GroupPath{{Id: groupA, Path: "/service-desk"}},
		}
		if fmt.Sprint(me) != fmt.Sprint(want) {
			t.Errorf("%s: me = %+v, want %+v", scheme, me, want)
		}
	}
}

// Every refusal is a registered 401 problem with WWW-Authenticate: Bearer;
// a resolver that cannot decide is a 500 whose body does not carry its error.
func TestMeRefusals(t *testing.T) {
	t.Parallel()
	h, _, reg := newAPIHandler(t)
	cases := []struct {
		name   string
		header http.Header
		status int
		code   problem.Code
	}{
		{"no Authorization header", nil, 401, problem.InvalidToken},
		{"empty header", http.Header{"Authorization": {""}}, 401, problem.InvalidToken},
		{"Basic scheme", http.Header{"Authorization": {"Basic dXNlcjpwdw=="}}, 401, problem.InvalidToken},
		{"Bearer with no token", http.Header{"Authorization": {"Bearer "}}, 401, problem.InvalidToken},
		{"token with a space", http.Header{"Authorization": {"Bearer token-a extra"}}, 401, problem.InvalidToken},
		{"two Authorization headers", http.Header{"Authorization": {"Bearer token-a", "Bearer token-a"}}, 401, problem.InvalidToken},
		{"invalid token", http.Header{"Authorization": {"Bearer forged"}}, 401, problem.InvalidToken},
		{"tenant unresolved", http.Header{"Authorization": {"Bearer token-unresolved"}}, 401, problem.TenantUnresolved},
		{"resolver cannot decide", http.Header{"Authorization": {"Bearer token-store-down"}}, 500, problem.InternalError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := get(t, h, "/api/v1/me", c.header)
			var b problem.Body
			if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
				t.Fatalf("body: %v: %s", err, rec.Body)
			}
			if _, ok := reg.Lookup(b.Code); !ok || rec.Code != c.status || b.Code != c.code ||
				rec.Header().Get("Content-Type") != problem.ContentType {
				t.Fatalf("= %d %s %+v, want %d %s", rec.Code, rec.Header().Get("Content-Type"), b, c.status, c.code)
			}
			if c.status == 401 && rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
			if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "token-") {
				t.Errorf("body echoes the token or the resolver's error: %s", rec.Body)
			}
		})
	}
}

// Forged tenant (CI-2): a token for tenant A with tenant B named in the query
// string, a header or baggage still resolves to tenant A.
func TestMeIgnoresForgedTenant(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	auth := "Bearer token-a"
	for name, c := range map[string]struct {
		target string
		header http.Header
	}{
		"query":   {"/api/v1/me?tenant_id=" + tenantB.String() + "&tenant=" + tenantB.String(), nil},
		"headers": {"/api/v1/me", http.Header{"X-Tenant-Id": {tenantB.String()}, "X-Nava-Tenant": {tenantB.String()}}},
		"baggage": {"/api/v1/me", http.Header{"Baggage": {"tenant_id=" + tenantB.String() + ",nava.tenant_id=" + tenantB.String()}}},
	} {
		hdr := http.Header{"Authorization": {auth}}
		for k, v := range c.header {
			hdr[k] = v
		}
		rec := get(t, h, c.target, hdr)
		if rec.Code != http.StatusOK {
			// The validator may refuse an undeclared parameter; a 4xx also
			// keeps tenant B out.
			if rec.Code < 400 || rec.Code >= 500 {
				t.Errorf("%s: %d %s, want 200 for tenant A or a 4xx", name, rec.Code, rec.Body)
			}
			continue
		}
		if me := decodeMe(t, rec); me.TenantId != tenantA {
			t.Errorf("%s: tenant = %s, want tenant A", name, me.TenantId)
		}
	}
}

// End to end through the real Keycloak adapter and a test realm: the store
// decides, and both TICK-69 carry-forward cases are 401 tenant_unresolved
// over HTTP. dbtest runs the same against PostgreSQL.
func TestMeThroughKeycloakAdapter(t *testing.T) {
	t.Parallel()
	realm := keycloaktest.New(t)
	store := &fixedStore{issuer: realm.Issuer()}
	h, _, _ := newAPIHandlerWith(t, realm.Resolver(t, store))
	org := map[string]string{"tenant-a": "org-a"}

	me := decodeMe(t, get(t, h, "/api/v1/me", http.Header{"Authorization": {"Bearer " + realm.Mint(t, keycloaktest.Token{Subject: "kc-a", Orgs: org})}}))
	if me.TenantId != tenantA || me.UserId != principalA || me.IdpAlias != "" {
		t.Fatalf("me = %+v", me)
	}
	for name, tok := range map[string]keycloaktest.Token{
		"federated_links NULL":          {Subject: "kc-links-null", Orgs: org},
		"no principals row":             {Subject: "kc-nobody", Orgs: org},
		"zero organizations":            {Subject: "kc-a"},
		"two organizations":             {Subject: "kc-a", Orgs: map[string]string{"tenant-a": "org-a", "tenant-b": "org-b"}},
		"unbound idp":                   {Subject: "kc-a", Orgs: org, IdP: "idp-evil"},
		"federated link to foreign IdP": {Subject: "kc-foreign-link", Orgs: org},
	} {
		rec := get(t, h, "/api/v1/me", http.Header{"Authorization": {"Bearer " + realm.Mint(t, tok)}})
		if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"code":"tenant_unresolved"`) {
			t.Errorf("%s: %d %s, want 401 tenant_unresolved", name, rec.Code, rec.Body)
		}
	}
}

// fixedStore binds org-a to tenant A (alias idp-a) and knows three subjects.
type fixedStore struct{ issuer string }

func (s *fixedStore) Binding(_ context.Context, issuer, orgID string) (identity.Binding, error) {
	if issuer == s.issuer && orgID == "org-a" {
		return identity.Binding{TenantID: tenantA, IdPAliases: []string{"idp-a"}, TenantState: identity.TenantStateActive}, nil
	}
	return identity.Binding{}, identity.ErrNoBinding
}

func (s *fixedStore) Principal(_ context.Context, tenant uuid.UUID, subject string) (identity.PrincipalRecord, error) {
	if tenant != tenantA {
		return identity.PrincipalRecord{}, identity.ErrNoPrincipal
	}
	switch subject {
	case "kc-a":
		return identity.PrincipalRecord{ID: principalA, Enabled: true, LinksKnown: true, Groups: []uuid.UUID{}}, nil
	case "kc-links-null":
		return identity.PrincipalRecord{ID: uuid.New(), Enabled: true}, nil
	case "kc-foreign-link":
		return identity.PrincipalRecord{ID: uuid.New(), Enabled: true, LinksKnown: true, FederatedIdPs: []string{"idp-b"}}, nil
	}
	return identity.PrincipalRecord{}, identity.ErrNoPrincipal
}

const bearerA = "Bearer token-a"

func meWith(t *testing.T, paths identity.GroupPathReader, log *slog.Logger) *httptest.ResponseRecorder {
	t.Helper()
	h, _, _ := newAPIHandlerFull(t, stubResolver{}, paths, log)
	return get(t, h, "/api/v1/me", http.Header{"Authorization": {bearerA}})
}

// AC3: no groups gives [], never null.
func TestMeNoGroupsGivesEmptyGroupPaths(t *testing.T) {
	t.Parallel()
	for name, rows := range map[string][]identity.GroupPath{"nil": nil, "empty": {}} {
		rec := meWith(t, fakePaths{rows: rows}, slog.New(slog.DiscardHandler))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"group_paths":[]`) {
			t.Errorf("%s: %d %s, want 200 with group_paths []", name, rec.Code, rec.Body)
		}
	}
}

// AC4: a path outside the caller's tenant prefix is dropped and logged once,
// by ids only; groups is unchanged and no path text reaches the body or logs.
func TestMeDropsPathWithoutTenantPrefix(t *testing.T) {
	t.Parallel()
	stray := uuid.MustParse("0a000000-0000-4000-8000-0000000004a2")
	const leak = "/t/" + "0b000000-0000-4000-8000-00000000000b" + "/b-only-marker"
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	rec := meWith(t, fakePaths{rows: []identity.GroupPath{
		{ID: groupA, Path: identity.TenantGroupPrefix(tenantA) + "service-desk/tier-2"},
		{ID: stray, Path: leak},
	}}, log)
	me := decodeMe(t, rec)
	if len(me.GroupPaths) != 1 || me.GroupPaths[0].Id != groupA || me.GroupPaths[0].Path != "/service-desk/tier-2" {
		t.Errorf("group_paths = %+v, want only the prefixed entry, stripped", me.GroupPaths)
	}
	if len(me.Groups) != 1 || me.Groups[0] != groupA {
		t.Errorf("groups = %v, want unchanged", me.Groups)
	}
	var errLines int
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line: %v: %s", err, line)
		}
		if rec["level"] == "ERROR" {
			errLines++
			if rec["tenant_id"] != tenantA.String() || rec["group_id"] != stray.String() {
				t.Errorf("ERROR line = %v, want tenant_id and group_id", rec)
			}
		}
	}
	if errLines != 1 {
		t.Errorf("ERROR lines = %d, want 1: %s", errLines, buf.String())
	}
	for _, text := range []string{"b-only-marker", "service-desk", leak} {
		if strings.Contains(buf.String(), text) {
			t.Errorf("log contains path text %q: %s", text, buf.String())
		}
	}
	if strings.Contains(rec.Body.String(), "b-only-marker") {
		t.Errorf("body contains the dropped path: %s", rec.Body)
	}
}

// A store failure is a 500 whose body does not carry the error.
func TestMeGroupPathsStoreErrorIs500(t *testing.T) {
	t.Parallel()
	rec := meWith(t, fakePaths{err: errors.New("pgstore: dial tcp 10.0.0.9:5432: secret-host-detail")}, slog.New(slog.DiscardHandler))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("= %d %s, want 500 without the error text", rec.Code, rec.Body)
	}
}
