package cursor_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func principal(tenant, user uuid.UUID) context.Context {
	return identity.NewContext(context.Background(), identity.Principal{TenantID: tenant, PrincipalID: user})
}

func mustClaims(ctx context.Context, t *testing.T, route string, q url.Values) cursor.Claims {
	t.Helper()
	c, err := cursor.ClaimsFor(ctx, route, q)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func registry(t *testing.T) *problem.Registry {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestClaimsForSetsEveryClaimFromContext(t *testing.T) {
	tenant, user := uuid.New(), uuid.New()
	q := url.Values{"status": {"open"}}
	c := mustClaims(principal(tenant, user), t, "listThings", q)
	want := cursor.Claims{
		cursor.ClaimTenant:    tenant.String(),
		cursor.ClaimPrincipal: user.String(),
		cursor.ClaimQuery:     cursor.QueryHash(q),
		cursor.ClaimRoute:     "listThings",
	}
	if len(c) != len(want) {
		t.Fatalf("claims = %v", c)
	}
	for k, v := range want {
		if c[k] != v {
			t.Errorf("claim %s = %q, want %q", k, c[k], v)
		}
	}
}

func TestClaimsForIgnoresClientTenant(t *testing.T) {
	tenant, other := uuid.New(), uuid.New()
	q := url.Values{"tenant": {other.String()}, "tenant_id": {other.String()}}
	c := mustClaims(principal(tenant, uuid.New()), t, "r", q)
	if c[cursor.ClaimTenant] != tenant.String() {
		t.Fatalf("tenant claim = %q", c[cursor.ClaimTenant])
	}
}

func TestClaimsForStripsPagingBeforeQueryHash(t *testing.T) {
	ctx := principal(uuid.New(), uuid.New())
	first := url.Values{"status": {"open"}, "limit": {"10"}}
	next := url.Values{"status": {"open"}, "limit": {"50"}, "cursor": {"abc"}}
	a, b := mustClaims(ctx, t, "r", first), mustClaims(ctx, t, "r", next)
	if a[cursor.ClaimQuery] != b[cursor.ClaimQuery] {
		t.Fatal("query hash changed with cursor or limit")
	}
	c := mustClaims(ctx, t, "r", url.Values{"status": {"closed"}})
	if a[cursor.ClaimQuery] == c[cursor.ClaimQuery] {
		t.Fatal("query hash ignores a filter")
	}
	if len(next["cursor"]) != 1 || len(next["limit"]) != 1 {
		t.Fatal("caller's query was modified")
	}
}

func TestClaimsForRefusals(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx   context.Context
		route string
	}{
		"no principal":  {context.Background(), "r"},
		"nil tenant":    {principal(uuid.Nil, uuid.New()), "r"},
		"nil principal": {principal(uuid.New(), uuid.Nil), "r"},
		"empty route":   {principal(uuid.New(), uuid.New()), ""},
	} {
		if _, err := cursor.ClaimsFor(tc.ctx, tc.route, nil); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := cursor.ClaimsFor(context.Background(), "r", nil); !errors.Is(err, cursor.ErrNoPrincipal) {
		t.Fatalf("err = %v", err)
	}
}

func TestRouteClaimStopsCrossEndpointReplay(t *testing.T) {
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	ctx := principal(uuid.New(), uuid.New())
	tok, err := kr.Encode(mustClaims(ctx, t, "listA", nil), []byte("p"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kr.Decode(mustClaims(ctx, t, "listB", nil), tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("cursor from listA decoded on listB: %v", err)
	}
	if _, err := kr.Decode(mustClaims(ctx, t, "listA", nil), tok); err != nil {
		t.Fatal(err)
	}
}

func TestKeyringFromConfigRotation(t *testing.T) {
	k1 := config.CursorKey{ID: "k1", Secret: secret("a")}
	k2 := config.CursorKey{ID: "k2", Secret: secret("b")}
	claims := mustClaims(principal(uuid.New(), uuid.New()), t, "r", nil)

	old, err := cursor.NewKeyringFromConfig(config.Cursor{ActiveKeyID: "k1", Keys: []config.CursorKey{k1}})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := old.Encode(claims, []byte("p"))
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := cursor.NewKeyringFromConfig(config.Cursor{ActiveKeyID: "k2", Keys: []config.CursorKey{k1, k2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.Decode(claims, tok); err != nil {
		t.Fatalf("old cursor after rotation: %v", err)
	}
	fresh, err := rotated.Encode(claims, []byte("p"))
	if err != nil || !strings.HasPrefix(fresh, "k2.") {
		t.Fatalf("new cursor not signed by the active key: %q, %v", fresh, err)
	}

	dropped, err := cursor.NewKeyringFromConfig(config.Cursor{ActiveKeyID: "k2", Keys: []config.CursorKey{k2}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dropped.Decode(claims, tok); !errors.Is(err, cursor.ErrInvalid) {
		t.Fatalf("cursor of a dropped key decoded: %v", err)
	}

	if kr, err := cursor.NewKeyringFromConfig(config.Cursor{}); kr != nil || err != nil {
		t.Fatalf("unconfigured = %v, %v", kr, err)
	}
	if _, err := cursor.NewKeyringFromConfig(config.Cursor{ActiveKeyID: "k9", Keys: []config.CursorKey{k1}}); err == nil {
		t.Fatal("active key outside the keyring accepted")
	}
}

// A stand-in list handler built only from the helpers every list endpoint
// uses: tenant A's token with a cursor minted for tenant B is 400
// invalid_cursor. The first real list endpoint repeats this test when it lands.
func TestCrossTenantCursorIsInvalidCursor(t *testing.T) {
	reg := registry(t)
	kr := ring(t, "k1", cursor.Key{ID: "k1", Secret: secret("a")})
	user, tenantA, tenantB := uuid.New(), uuid.New(), uuid.New()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := cursor.ClaimsFor(r.Context(), "listThings", r.URL.Query())
		if err != nil {
			reg.Write(w, r, problem.InternalError)
			return
		}
		if _, err := kr.Decode(claims, r.URL.Query().Get("cursor")); err != nil {
			cursor.WriteInvalid(reg, w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mint := func(tenant uuid.UUID) string {
		tok, err := kr.Encode(mustClaims(principal(tenant, user), t, "listThings", nil), []byte("p"))
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	do := func(tok string) *httptest.ResponseRecorder {
		req, err := http.NewRequestWithContext(principal(tenantA, user), http.MethodGet, "/things?limit=5&cursor="+url.QueryEscape(tok), nil)
		if err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := do(mint(tenantA)); rec.Code != http.StatusOK {
		t.Fatalf("own cursor: %d %s", rec.Code, rec.Body)
	}
	rec := do(mint(tenantB))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"code":"invalid_cursor"`) {
		t.Fatalf("tenant B cursor: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), tenantB.String()) {
		t.Fatal("problem body leaks the other tenant")
	}
}
