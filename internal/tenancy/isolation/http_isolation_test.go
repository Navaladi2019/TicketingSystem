//go:build integration

package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// The HTTP runner (T17): the real nava-core API handler, its resolver the
// Keycloak adapter trusting a test realm and reading tenant bindings and
// principals as nava_app through PgBouncer. The token is for tenant A: the
// realm's organization is bound to tenant A, and its subject is tenant A's
// seeded principal kc-user-shared (a Keycloak id tenant B also has).
const (
	isoOrgID   = HTTPOrgA
	isoSubject = "kc-user-shared"
)

type apiRunner struct {
	t        *testing.T
	h        http.Handler
	token    string
	realm    *keycloaktest.Realm
	resolver *keycloak.Adapter
	// untrusted is a second realm the API does not trust (CI-5.03).
	untrusted *keycloaktest.Realm
}

var _ Server = (*apiRunner)(nil)

func newAPIRunner(t *testing.T) *apiRunner {
	t.Helper()
	realm := keycloaktest.New(t)

	// Bind the test realm's organization to tenant A for this test only.
	super := suite.superConn(t)
	if _, err := super.Exec(t.Context(),
		"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
		TenantA, realm.Issuer(), isoOrgID, []string{HTTPIdPA}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = super.Exec(context.Background(), "DELETE FROM tenant_identity_binding WHERE issuer = $1", realm.Issuer())
	})

	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig = suite.roleConfig("nava_app", true)
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec // as nava-core, behind PgBouncer
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	resolver := realm.Resolver(t, pgstore.New(pool), OperatorSubject)
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, resolver, pgstore.New(pool),
		server.WithOperator(resolver, refusingOnboarder{}))
	if err != nil {
		t.Fatal(err)
	}
	r := &apiRunner{t: t, h: h, realm: realm, resolver: resolver, untrusted: keycloaktest.New(t)}
	r.token = r.Mint(Token{Subject: isoSubject})

	// Without this baseline every path could pass on a 401: the token must
	// really resolve to tenant A.
	rec := r.do(t, &HTTPCall{Method: http.MethodGet, Path: "/api/v1/me"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"tenant_id":"`+TenantA+`"`) {
		t.Fatalf("baseline GET /api/v1/me with tenant A's token = %d %s; want 200 for tenant A", rec.Code, rec.Body)
	}
	return r
}

// refusingOnboarder stands in for the onboarding service (T26 case 6): the
// operator routes are on, so a tenant token is refused by the operator
// authentication before any handler runs. Reaching it is a failure.
type refusingOnboarder struct{}

func (refusingOnboarder) Onboard(context.Context, identity.Operator, string, []byte, onboard.Request) (onboard.Result, error) {
	return onboard.Result{}, errors.New("the onboarder was reached")
}

// Mint implements Server.
func (r *apiRunner) Mint(tok Token) string {
	kt := keycloaktest.Token{Subject: tok.Subject, Audiences: tok.Audiences, RealmRoles: tok.RealmRoles}
	if !tok.NoOrg {
		kt.Orgs = map[string]string{"tenant-a": isoOrgID}
	}
	return r.realm.Mint(r.t, kt)
}

// Do implements Server.
func (r *apiRunner) Do(ctx context.Context, method, path, token, body string, header http.Header) (int, string) {
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// ResolveOperator implements Server.
func (r *apiRunner) ResolveOperator(ctx context.Context, token string) error {
	_, err := r.resolver.ResolveOperator(ctx, token)
	return err
}

func (r *apiRunner) do(t *testing.T, c *HTTPCall) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), c.Method, c.Path, strings.NewReader(c.Body))
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if c.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	rec := httptest.NewRecorder()
	r.h.ServeHTTP(rec, req)
	return rec
}

// run passes a 200 whose body names tenant A and never tenant B. A call to a
// route that does not exist yet (NoRoute) passes only on 404 or 405, which
// proves just that tenant B was not reached. Anything else fails: a 401 means
// the token was not honoured, and any other 4xx would hide a regression that
// starts refusing the request.
func (r *apiRunner) run(t *testing.T, c *HTTPCall) (status, detail string) {
	t.Helper()
	if c.Token != nil {
		return r.runShape(t, c)
	}
	rec := r.do(t, c)
	body := rec.Body.String()
	switch {
	case rec.Code == http.StatusUnauthorized:
		return statusFail, "401: the token for tenant A was not honoured, so this proves nothing: " + body
	case c.NoRoute != "":
		if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
			return statusPass, fmt.Sprintf("%d (no route until %s), tenant B not reached", rec.Code, c.NoRoute)
		}
		return statusFail, fmt.Sprintf("%d, want 404 or 405 until %s: %s", rec.Code, c.NoRoute, body)
	case rec.Code != http.StatusOK:
		return statusFail, fmt.Sprintf("%d, want 200 for tenant A: %s", rec.Code, body)
	case strings.Contains(body, TenantB):
		return statusFail, fmt.Sprintf("%d with tenant B in the body: %s", rec.Code, body)
	}
	var v struct {
		TenantID   string `json:"tenant_id"`
		GroupPaths []struct {
			ID string `json:"id"`
		} `json:"group_paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil || v.TenantID != TenantA {
		return statusFail, fmt.Sprintf("%d, tenant_id %q, want tenant A: %s", rec.Code, v.TenantID, body)
	}
	// ADR-002 AC2: /me lists exactly the caller's memberships, never another
	// tenant's marker group and never a tenant A group the caller is not in.
	if strings.HasPrefix(c.Path, "/api/v1/me") {
		if strings.Contains(body, "b-only-marker") {
			return statusFail, fmt.Sprintf("%d with tenant B's group marker in the body: %s", rec.Code, body)
		}
		got := make([]string, 0, len(v.GroupPaths))
		for _, g := range v.GroupPaths {
			got = append(got, g.ID)
		}
		if !slices.Equal(got, []string{groupA1}) {
			return statusFail, fmt.Sprintf("%d, group_paths ids %v, want exactly the caller's membership [%s]: %s", rec.Code, got, groupA1, body)
		}
	}
	return statusPass, fmt.Sprintf("%d, tenant A only", rec.Code)
}

// runShape runs a CI-5 Token path: the API must answer 401, and the resolver
// must refuse the same token with the path's sentinel and reason, so a 401
// for another reason (a missing seed row, an unbound organization) fails.
func (r *apiRunner) runShape(t *testing.T, c *HTTPCall) (status, detail string) {
	t.Helper()
	s := c.Token
	if s.WantErr == nil || s.WantReason == "" {
		return statusFail, "a Token path needs WantErr and WantReason"
	}
	issuer := r.realm
	if s.Untrusted {
		issuer = r.untrusted
	}
	token := issuer.Mint(t, keycloaktest.Token{Subject: s.Subject, Orgs: s.Orgs, IdP: s.IdP})
	code, body := r.Do(t.Context(), c.Method, c.Path, token, c.Body, nil)
	if code != http.StatusUnauthorized {
		return statusFail, fmt.Sprintf("%d, want 401 for this token: %s", code, body)
	}
	p, err := r.resolver.Resolve(t.Context(), token)
	if !errors.Is(err, s.WantErr) || !strings.Contains(err.Error(), s.WantReason) {
		return statusFail, fmt.Sprintf("401, but the resolver gave %+v, %v; want %v (%s)", p, err, s.WantErr, s.WantReason)
	}
	return statusPass, fmt.Sprintf("401, refused: %s", s.WantReason)
}
