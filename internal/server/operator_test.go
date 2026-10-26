package server_test

import (
	"context"
	"encoding/json"
	"errors"
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

func operatorHandler(t *testing.T, o server.Option) http.Handler {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	opts := []server.Option{}
	if o != nil {
		opts = append(opts, o)
	}
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, stubResolver{}, opts...)
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
		{&onboard.FieldError{Pointer: "/idp/config"}, 400, "validation_failed"},
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
	h := operatorHandler(t, nil)
	if rec := postOnboard(t, h, "op-token", "k", onboardBody, "", nil); rec.Code != 501 || code(t, rec) != "not_implemented" {
		t.Errorf("status %d: %s", rec.Code, rec.Body)
	}
}
