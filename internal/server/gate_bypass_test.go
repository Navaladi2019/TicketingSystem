package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// TICK-300 (adversarial): no spelling of a contract path reaches a contract
// handler without a token. The gate classifies on r.URL.Path and ServeMux
// routes on the escaped path, so each case here is a path the gate might not
// see as /api/v1 (or as /api/v1/operator/) while the router might still
// dispatch it. Every one must end before a handler runs: 401 from the gate,
// or the router's own 404 or redirect to the cleaned path, which the gate then
// sees. A 2xx or 4xx contract answer (200, 400, 403, 422) means the request
// reached a handler or parameter binding unauthenticated.
func TestNoPathSpellingBypassesTheAuthGate(t *testing.T) {
	t.Parallel()
	for name, h := range map[string]http.Handler{
		"nava-core":    operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{})),
		"nava-onboard": operatorHandler(t, server.WithOperator(stubOperators{}, &stubOnboarder{}), server.OperatorOnly()),
	} {
		probeSpellings(t, name, h)
	}
}

func probeSpellings(t *testing.T, name string, h http.Handler) {
	t.Helper()
	for _, c := range []struct{ method, path, rawPath string }{
		{http.MethodGet, "//api/v1/me", ""},
		{http.MethodGet, "/./api/v1/me", ""},
		{http.MethodGet, "/x/../api/v1/me", ""},
		{http.MethodGet, "/api/./v1/me", ""},
		{http.MethodGet, "/api//v1/me", ""},
		{http.MethodGet, "/API/v1/me", ""},
		{http.MethodGet, "/api/v1/me", "/api%2Fv1/me"},
		{http.MethodGet, "/api/v1/me", "/%61pi/v1/me"},
		{http.MethodPost, "//api/v1/operator/tenants", ""},
		{http.MethodPost, "/x/../api/v1/operator/tenants", ""},
		{http.MethodPost, "/api/v1/./operator/tenants", ""},
		{http.MethodPost, "/api/v1//operator/tenants", ""},
		{http.MethodPost, "/api/v1/operator/tenants", "/api/v1/operator%2Ftenants"},
		{http.MethodPost, "/api/v1/operator/tenants", "/api/v1/%6Fperator/tenants"},
	} {
		req := httptest.NewRequestWithContext(t.Context(), c.method, "/", nil)
		req.URL = &url.URL{Path: c.path, RawPath: c.rawPath}
		req.RequestURI = req.URL.RequestURI()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		switch rec.Code {
		case http.StatusUnauthorized, http.StatusNotFound, http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		default:
			t.Errorf("%s: %s %s (raw %q) without a token = %d %s, want 401, 404 or a redirect", name, c.method, c.path, c.rawPath, rec.Code, rec.Body.String())
		}
	}
}
