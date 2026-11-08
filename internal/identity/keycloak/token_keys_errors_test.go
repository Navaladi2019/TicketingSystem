package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// tokenAdmin is a fake Keycloak whose token endpoint and admin route answer
// from the test, and which counts both.
type tokenAdmin struct {
	tokenHits, adminHits atomic.Int64
	token                func(n int64) (int, string) // n is the 1-based token request count
	admin                func(n int64) int           // n is the 1-based admin request count
}

func (k *tokenAdmin) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token") {
			code, body := k.token(k.tokenHits.Add(1))
			w.WriteHeader(code)
			fmt.Fprint(w, body)
			return
		}
		w.WriteHeader(k.admin(k.adminHits.Add(1)))
		fmt.Fprint(w, `[]`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func goodToken(int64) (int, string) { return 200, `{"access_token":"tok","expires_in":300}` }

func callAdmin(t *testing.T, url string) error {
	t.Helper()
	d, _ := directoryFor(t, url)
	_, err := d.api.do(t.Context(), d.runtime, http.MethodGet, "/users", "/users", nil, nil)
	return err
}

func TestAdminTokenEndpointFailuresStopTheCall(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		code    int
		body    string
		wantMsg string
	}{
		"unauthorized":      {401, `{}`, "status 401"},
		"server error":      {500, `{}`, "status 500"},
		"no access token":   {200, `{}`, "malformed"},
		"empty token":       {200, `{"access_token":"","expires_in":300}`, "malformed"},
		"not json":          {200, `nope`, "malformed"},
		"access token only": {200, `{"expires_in":1}`, "malformed"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			k := &tokenAdmin{token: func(int64) (int, string) { return tc.code, tc.body }, admin: func(int64) int { return 200 }}
			err := callAdmin(t, k.start(t))
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantMsg)
			}
			if tc.code != 200 {
				var se *statusError
				if !errors.As(err, &se) || se.status != tc.code {
					t.Fatalf("err = %v, want a *statusError with status %d", err, tc.code)
				}
			}
			// The admin API is never called without a token.
			if n := k.adminHits.Load(); n != 0 {
				t.Fatalf("%d admin requests were sent without a token", n)
			}
		})
	}
}

// A revoked cached token (admin 401) is dropped and the call retried once with
// a fresh token; a second 401 is the answer, not a loop.
func TestAdmin401RetriesOnceWithAFreshToken(t *testing.T) {
	t.Parallel()
	t.Run("recovers", func(t *testing.T) {
		t.Parallel()
		k := &tokenAdmin{token: goodToken, admin: func(n int64) int {
			if n == 1 {
				return 401
			}
			return 200
		}}
		if err := callAdmin(t, k.start(t)); err != nil {
			t.Fatal(err)
		}
		if k.tokenHits.Load() != 2 || k.adminHits.Load() != 2 {
			t.Fatalf("token fetches = %d, admin requests = %d, want 2 and 2", k.tokenHits.Load(), k.adminHits.Load())
		}
	})
	t.Run("two 401s end the call", func(t *testing.T) {
		t.Parallel()
		k := &tokenAdmin{token: goodToken, admin: func(int64) int { return 401 }}
		err := callAdmin(t, k.start(t))
		var se *statusError
		if !errors.As(err, &se) || se.status != 401 {
			t.Fatalf("err = %v, want a 401 *statusError", err)
		}
		if k.tokenHits.Load() != 2 || k.adminHits.Load() != 2 {
			t.Fatalf("token fetches = %d, admin requests = %d, want exactly 2 and 2 (no loop)", k.tokenHits.Load(), k.adminHits.Load())
		}
	})
	t.Run("the retry's token failure is returned", func(t *testing.T) {
		t.Parallel()
		k := &tokenAdmin{
			token: func(n int64) (int, string) {
				if n == 1 {
					return goodToken(n)
				}
				return 500, `{}`
			},
			admin: func(int64) int { return 401 },
		}
		err := callAdmin(t, k.start(t))
		var se *statusError
		if !errors.As(err, &se) || se.status != 500 {
			t.Fatalf("err = %v, want the token endpoint's 500", err)
		}
		if k.adminHits.Load() != 1 {
			t.Fatalf("%d admin requests, want 1: none after the failed re-auth", k.adminHits.Load())
		}
	})
}

func TestResolveRefusesWhenTheKeySetCannotBeFetched(t *testing.T) {
	t.Parallel()
	t.Run("initial fetch fails", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, []string{realmNava})
		tok := sign(t, h.idp.signer(realmNava), h.idp.tenantClaims(realmNava, h.clock.now()))
		h.idp.srv.Close() // the IdP is unreachable before the first key fetch
		_, err := h.a.verify(t.Context(), tok, DefaultAudience)
		if err == nil || !strings.Contains(err.Error(), "keys unavailable") {
			t.Fatalf("err = %v, want keys unavailable", err)
		}
	})
	t.Run("refresh after an unknown kid fails", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, []string{realmNava})
		s := h.idp.signer(realmNava)
		if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
			t.Fatal(err)
		}
		h.clock.advance(forceInterval)
		h.idp.srv.Close() // keys are cached, the forced refresh cannot reach the IdP
		stranger := signer{kid: "nobody", priv: s.priv}
		_, err := h.a.verify(t.Context(), sign(t, stranger, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience)
		if err == nil {
			t.Fatal("a token with an unknown kid was accepted while the refresh failed")
		}
		// The cached key still verifies the token the IdP signed earlier.
		if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
			t.Fatalf("a token with a cached kid must still resolve: %v", err)
		}
	})
	t.Run("a caller that gives up while waiting for the fetch gate", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t, []string{realmNava})
		s := h.idp.signer(realmNava)
		iss := h.idp.issuer(realmNava)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		gate := h.a.keys.issuers[iss].gate
		gate <- struct{}{} // another fetch is in flight and never finishes
		defer func() {
			select {
			case <-gate:
			default:
			}
		}()
		// No keys yet: the initial fetch waits for the gate and gives up.
		if _, err := h.a.verify(ctx, sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err == nil ||
			!strings.Contains(err.Error(), "keys unavailable") {
			t.Fatalf("initial: err = %v", err)
		}
		<-gate
		if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err != nil {
			t.Fatal(err)
		}
		gate <- struct{}{}
		// Keys cached, unknown kid: the refresh waits for the gate and gives up.
		h.clock.advance(forceInterval)
		ctx2, cancel2 := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel2()
		stranger := signer{kid: "nobody", priv: s.priv}
		if _, err := h.a.verify(ctx2, sign(t, stranger, h.idp.tenantClaims(realmNava, h.clock.now())), DefaultAudience); err == nil ||
			!strings.Contains(err.Error(), "keys unavailable") {
			t.Fatalf("refresh: err = %v", err)
		}
	})
}
