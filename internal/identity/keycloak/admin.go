package keycloak

// The hand-written Keycloak admin client behind Directory (TICK-94 item 12:
// no gocloak). Every call goes through the shared outbound client with a route
// template, so a span or an error names "/admin/realms/{realm}/users", never
// a path or query that holds an email or an id. An error built here carries
// the method, the route template and the status, never a response body: the
// body of a Keycloak 409 can quote the email.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const (
	maxAdminBody  = 8 << 20
	adminPageSize = 100
	tokenSkew     = 30 * time.Second
)

// ClientCredentials are the client-credentials grant of one service account.
// Both secrets come from the environment or a secret store, never a literal.
type ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// orgAccess says which /organizations routes an account may call. The zero
// value allows none, so an account built without a choice fails closed.
type orgAccess int

const (
	// The zero value allows no /organizations route.
	_ orgAccess = iota
	// orgMemberRead: only GET /organizations/members/{id}/organizations, the
	// one organization read group sync's view-organizations role is accepted
	// for (AR-034). It shows one user's organization ids.
	orgMemberRead
	// orgAll: every route, for the onboarding account.
	orgAll
)

// ErrOrgRouteDenied is returned, with nothing sent, for a request to an
// /organizations route the account may not call.
// It also covers a non-canonical path outside /organizations (see allows).
var ErrOrgRouteDenied = errors.New("keycloak admin: organizations route not allowed for this account")

// allows reports whether the account may send method to path (the real path
// below /admin/realms/{realm}, escaped). An account without orgAll may send
// only a canonical path: no "?", "#", "\\" or control byte anywhere, no empty
// segment, no ";" and no "%" (the one escaped
// {id} segment of the member-organizations route aside), and no "." or ".."
// segment. So a path that Keycloak or a proxy in front of it might normalise
// into /organizations (";x", "%6F", "//", "..") is refused before it is sent.
// "organizations" is compared case-insensitively, so /Organizations is an
// organization route too.
func (a *account) allows(method, path string) bool {
	if a.orgs == orgAll {
		return true
	}
	if !strings.HasPrefix(path, "/") {
		return false
	}
	if strings.ContainsAny(path, "?#\\") || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return false
	}
	segs := strings.Split(path[1:], "/")
	memberRoute := len(segs) == 4 && segs[0] == "organizations" && segs[1] == "members" &&
		segs[2] != "" && segs[3] == "organizations"
	for i, seg := range segs {
		if seg == "" || seg == "." || seg == ".." || strings.Contains(seg, ";") {
			return false
		}
		if strings.Contains(seg, "%") && (!memberRoute || i != 2) {
			return false
		}
	}
	if memberRoute {
		u, err := url.PathUnescape(segs[2])
		// A decoded id must not hold anything a second decode (a proxy, a
		// future Keycloak) could turn into a separator or traversal.
		if err != nil || u == "." || u == ".." || strings.ContainsAny(u, "/%\\;?#") ||
			strings.IndexFunc(u, unicode.IsControl) >= 0 {
			return false
		}
		return a.orgs == orgMemberRead && method == http.MethodGet
	}
	if strings.EqualFold(segs[0], "organizations") {
		return false
	}
	return true
}

// account is one service account and its cached token.
type account struct {
	creds ClientCredentials
	orgs  orgAccess
	mu    sync.Mutex
	token string
	exp   time.Time
}

// admin is the HTTP side: a base URL, a realm, an outbound client.
type admin struct {
	base  string
	realm string
	out   *telemetry.OutboundClient
	now   func() time.Time
}

// reply is one admin API response. Body is bounded.
type reply struct {
	status int
	body   []byte
	header http.Header
}

// statusError is a non-success reply. Its text never carries the body.
type statusError struct {
	method, route string
	status        int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("keycloak admin: %s %s: status %d", e.method, e.route, e.status)
}

// Is maps the statuses the directory acts on to the port's sentinels.
func (e *statusError) Is(target error) bool {
	switch target {
	case identity.ErrNotFound:
		return e.status == http.StatusNotFound
	case identity.ErrAlreadyExists:
		return e.status == http.StatusConflict
	}
	return false
}

// bearer returns a valid access token of acct, fetching one when the cached
// token is within tokenSkew of expiry.
func (a *admin) bearer(ctx context.Context, acct *account) (string, error) {
	acct.mu.Lock()
	defer acct.mu.Unlock()
	if acct.token != "" && a.now().Add(tokenSkew).Before(acct.exp) {
		return acct.token, nil
	}
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {acct.creds.ClientID},
		"client_secret": {acct.creds.ClientSecret},
	}
	const route = "/realms/{realm}/protocol/openid-connect/token"
	ctx = telemetry.WithRoute(ctx, route)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.base+"/realms/"+url.PathEscape(a.realm)+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.New("keycloak admin: token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r, err := a.send(req)
	if err != nil {
		return "", fmt.Errorf("keycloak admin: token: %w", err)
	}
	if r.status != http.StatusOK {
		return "", &statusError{http.MethodPost, route, r.status}
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if json.Unmarshal(r.body, &tok) != nil || tok.AccessToken == "" {
		return "", errors.New("keycloak admin: token: malformed response")
	}
	acct.token = tok.AccessToken
	acct.exp = a.now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return acct.token, nil
}

func (a *admin) send(req *http.Request) (reply, error) {
	resp, err := a.out.Do(req)
	if err != nil {
		return reply{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxAdminBody))
	if err != nil {
		return reply{}, errors.New("read response")
	}
	return reply{status: resp.StatusCode, body: b, header: resp.Header}, nil
}

// do calls the admin API as acct. route is the template; path is the real path
// below /admin/realms/{realm} (already escaped); query is form-encoded into
// the URL; body, when set, is sent as JSON. A status outside ok is a
// *statusError.
func (a *admin) do(ctx context.Context, acct *account, method, route, path string, query url.Values, body any, ok ...int) (reply, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return reply{}, errors.New("keycloak admin: encode body")
		}
	}
	if len(ok) == 0 {
		ok = []int{http.StatusOK}
	}
	// Before the token fetch and before anything is sent: the check reads the
	// path that is sent, so it holds however the path was built.
	if !acct.allows(method, path) {
		return reply{}, ErrOrgRouteDenied
	}
	full := "/admin/realms/{realm}" + route
	for attempt := 0; ; attempt++ {
		tok, err := a.bearer(ctx, acct)
		if err != nil {
			return reply{}, err
		}
		u := a.base + "/admin/realms/" + url.PathEscape(a.realm) + path
		if len(query) > 0 {
			u += "?" + query.Encode()
		}
		req, err := http.NewRequestWithContext(telemetry.WithRoute(ctx, full), method, u, bytes.NewReader(payload))
		if err != nil {
			return reply{}, errors.New("keycloak admin: build request")
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		r, err := a.send(req)
		if err != nil {
			return reply{}, fmt.Errorf("keycloak admin: %s %s: %w", method, full, err)
		}
		if r.status == http.StatusUnauthorized && attempt == 0 {
			// The cached token was revoked or the realm restarted: one retry
			// with a fresh token.
			acct.mu.Lock()
			acct.token = ""
			acct.mu.Unlock()
			continue
		}
		for _, s := range ok {
			if r.status == s {
				return r, nil
			}
		}
		return reply{}, &statusError{method, full, r.status}
	}
}

// getJSON GETs path and decodes the 200 body into v.
func (a *admin) getJSON(ctx context.Context, acct *account, route, path string, query url.Values, v any) error {
	r, err := a.do(ctx, acct, http.MethodGet, route, path, query, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(r.body, v); err != nil {
		return fmt.Errorf("keycloak admin: GET %s: malformed response", route)
	}
	return nil
}

// pages reads every page of a list endpoint.
func pages[T any](ctx context.Context, a *admin, acct *account, route, path string, query url.Values) ([]T, error) {
	var all []T
	for first := 0; ; first += adminPageSize {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		q.Set("first", fmt.Sprint(first))
		q.Set("max", fmt.Sprint(adminPageSize))
		var page []T
		if err := a.getJSON(ctx, acct, route, path, q, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < adminPageSize {
			return all, nil
		}
		if len(all) > maxDirectoryItems {
			return nil, errors.New("keycloak admin: list too large")
		}
	}
}

// idFromLocation reads the id after the last slash of a 201 Location header.
func idFromLocation(h http.Header) (string, bool) {
	loc := h.Get("Location")
	i := strings.LastIndex(loc, "/")
	if i < 0 || i == len(loc)-1 {
		return "", false
	}
	return loc[i+1:], true
}
