package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

type stubOperators struct{}

func (stubOperators) ResolveOperator(_ context.Context, raw string) (identity.Operator, error) {
	switch raw {
	case "op-token":
		return identity.Operator{PrincipalID: "op-1"}, nil
	case "op-down":
		return identity.Operator{}, errors.New("jwks down")
	}
	return identity.Operator{}, identity.ErrInvalidToken
}

type stubOnboarder struct {
	gotOp   identity.Operator
	gotKey  string
	gotHash []byte
	gotReq  onboard.Request
	err     error
	replay  bool
}

func (s *stubOnboarder) Onboard(_ context.Context, op identity.Operator, key string, hash []byte, req onboard.Request) (onboard.Result, error) {
	s.gotOp, s.gotKey, s.gotHash, s.gotReq = op, key, hash, req
	if s.err != nil {
		return onboard.Result{}, s.err
	}
	return onboard.Result{TenantID: onboard.TenantID(op.PrincipalID, key), Replayed: s.replay}, nil
}

const onboardBody = `{"admin_email":"a@acme.test","email_domains":["acme.test"],"identity_provider":{"provider_id":"oidc","config":{"clientId":"c"}},"tenant_id":"11111111-1111-1111-1111-111111111111"}`

func operatorHandler(t *testing.T, o ...server.Option) http.Handler {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	opts := o
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, stubResolver{}, nil, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func postOnboard(t *testing.T, h http.Handler, token, key, body, query string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/operator/tenants"+query, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func code(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var p struct{ Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	return p.Code
}

func TestOperatorOnboardSucceeds(t *testing.T) {
	ob := &stubOnboarder{}
	h := operatorHandler(t, server.WithOperator(stubOperators{}, ob))
	rec := postOnboard(t, h, "op-token", "k1", onboardBody, "", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var out struct{ TenantID, State string }
	_ = json.Unmarshal(rec.Body.Bytes(), &struct{}{})
	var raw map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	out.TenantID, out.State = raw["tenant_id"], raw["state"]
	if out.TenantID != onboard.TenantID("op-1", "k1").String() || out.State != "active" {
		t.Errorf("body = %s", rec.Body)
	}
	if ob.gotKey != "k1" || ob.gotOp.PrincipalID != "op-1" || len(ob.gotHash) != 32 || ob.gotReq.AdminEmail != "a@acme.test" {
		t.Errorf("onboarder saw %+v", ob)
	}
}

func TestOperatorOnboardReplayIs200(t *testing.T) {
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{replay: true}))
	if rec := postOnboard(t, h, "op-token", "k1", onboardBody, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

// A tenant id in the body, the query or X-Tenant-Id never reaches the tenant
// id: it is derived from the operator and the key.
func TestOperatorOnboardIgnoresSuppliedTenantID(t *testing.T) {
	ob := &stubOnboarder{}
	h := operatorHandler(t, server.WithOperator(stubOperators{}, ob))
	rec := postOnboard(t, h, "op-token", "k1", onboardBody, "?tenant_id=22222222-2222-2222-2222-222222222222",
		map[string]string{"X-Tenant-Id": "33333333-3333-3333-3333-333333333333"})
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), onboard.TenantID("op-1", "k1").String()) {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	plain := &stubOnboarder{}
	h2 := operatorHandler(t, server.WithOperator(stubOperators{}, plain))
	postOnboard(t, h2, "op-token", "k1", onboardBody, "", nil)
	if string(plain.gotHash) != string(ob.gotHash) {
		t.Error("a query parameter changed the request hash")
	}
}

func TestOperatorOnboardAuthFailures(t *testing.T) {
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{}))
	cases := []struct {
		name, token string
		status      int
		code        string
	}{
		{"no token", "", 401, "invalid_token"},
		{"garbage token", "nope", 401, "invalid_token"},
		{"tenant token", "token-a", 403, "forbidden"},
		{"resolver down", "op-down", 500, "internal_error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := postOnboard(t, h, c.token, "k", onboardBody, "", nil)
			if rec.Code != c.status || code(t, rec) != c.code {
				t.Errorf("status %d code %q: %s", rec.Code, code(t, rec), rec.Body)
			}
		})
	}
}

func TestOperatorOnboardErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{onboard.ErrKeyReuse, 409, "idempotency_key_reuse"},
		{onboard.ErrDomainClaimed, 409, "domain_claimed"},
		{onboard.ErrConflict, 409, "onboarding_conflict"},
		{&onboard.FieldError{Pointer: "/identity_provider/config"}, 422, "invalid_idp_config"},
		{errors.New("boom secret"), 500, "internal_error"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{err: c.err}))
			rec := postOnboard(t, h, "op-token", "k", onboardBody, "", nil)
			if rec.Code != c.status || code(t, rec) != c.code || strings.Contains(rec.Body.String(), "secret") {
				t.Errorf("status %d: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestOperatorOnboardValidation(t *testing.T) {
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{}))
	for name, tc := range map[string]struct{ key, body string }{
		"no key":      {"", onboardBody},
		"empty body":  {"k", ""},
		"missing idp": {"k", `{"admin_email":"a@acme.test","email_domains":["acme.test"]}`},
		"no domains":  {"k", `{"admin_email":"a@acme.test","email_domains":[],"identity_provider":{"provider_id":"oidc","config":{}}}`},
	} {
		t.Run(name, func(t *testing.T) {
			if rec := postOnboard(t, h, "op-token", tc.key, tc.body, "", nil); rec.Code != 400 {
				t.Errorf("status %d: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestOperatorRouteNotConfigured(t *testing.T) {
	h := operatorHandler(t)
	if rec := postOnboard(t, h, "op-token", "k", onboardBody, "", nil); rec.Code != 501 || code(t, rec) != "not_implemented" {
		t.Errorf("status %d: %s", rec.Code, rec.Body)
	}
}

// The nava-onboard process mounts the operator routes only.
func TestOperatorOnlyHandlerServesNothingElse(t *testing.T) {
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{}), server.OperatorOnly())
	if rec := postOnboard(t, h, "op-token", "k", onboardBody, "", nil); rec.Code != http.StatusCreated {
		t.Fatalf("operator route: %d", rec.Code)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer token-a")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /me on the onboarding process: %d, want 404", rec.Code)
	}
}

// TICK-271: a refused claim is 409 domain_claimed and the body names neither
// the owning tenant nor its organization, even when the service error carries
// them. The operator learns only that the domain is taken.
func TestOperatorOnboardDomainClaimedHidesOwner(t *testing.T) {
	const owner, org = "8f1d2c3b-0000-4000-8000-00000000000b", "org-of-tenant-b"
	err := fmt.Errorf("tenant %s organization %s: %w", owner, org, onboard.ErrDomainClaimed)
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{err: err}))
	rec := postOnboard(t, h, "op-token", "k", onboardBody, "", nil)
	if rec.Code != http.StatusConflict || code(t, rec) != "domain_claimed" {
		t.Fatalf("status %d code %q, want 409 domain_claimed", rec.Code, code(t, rec))
	}
	for _, leak := range []string{owner, org} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Errorf("409 body leaks %q: %s", leak, rec.Body)
		}
	}
}

// TICK-298 (T26 case 6): the operator boundary answers before the contract
// does. A tenant token is 403 and no token is 401 on every operator route,
// whatever headers or body the caller sends; a missing Idempotency-Key must
// not turn the refusal into a 400 that reaches parameter binding first.
func TestOperatorBoundaryPrecedesParameterBinding(t *testing.T) {
	h := operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{}))
	for _, c := range []struct {
		name, token string
		status      int
		code        string
	}{
		{"tenant token", "token-a", http.StatusForbidden, "forbidden"},
		{"no token", "", http.StatusUnauthorized, "invalid_token"},
	} {
		for name, in := range map[string]struct{ key, body string }{
			"full request": {"k", onboardBody},
			"no key":       {"", onboardBody},
			"no body":      {"k", ""},
			"nothing":      {"", ""},
		} {
			t.Run(c.name+"/"+name, func(t *testing.T) {
				rec := postOnboard(t, h, c.token, in.key, in.body, "", nil)
				if rec.Code != c.status || code(t, rec) != c.code {
					t.Errorf("status %d code %q, want %d %s: %s", rec.Code, code(t, rec), c.status, c.code, rec.Body)
				}
			})
		}
	}
}
