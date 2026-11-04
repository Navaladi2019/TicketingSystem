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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
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
	// WithPool turns the records routes and the Idempotency-Key store on, so
	// a StoredUnderA call reaches the real create (P3-18).
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, resolver, pgstore.New(pool),
		server.WithOperator(resolver, refusingOnboarder{}), server.WithPool(pool))
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
// route that does not exist yet (NoRoute) passes only on 404 or 405 with no
// tenant B id in the body, which proves just that tenant B was not reached.
// Anything else fails: a 401 means the token was not honoured, and any other
// 4xx would hide a regression that starts refusing the request. A write is
// checked by runStoredUnderA.
func (r *apiRunner) run(t *testing.T, c *HTTPCall) (status, detail string) {
	t.Helper()
	if c.Token != nil {
		return r.runShape(t, c)
	}
	if c.Expect == StoredUnderA {
		return r.runStoredUnderA(t, c)
	}
	rec := r.do(t, c)
	body := rec.Body.String()
	switch {
	case rec.Code == http.StatusUnauthorized:
		return statusFail, "401: the token for tenant A was not honoured, so this proves nothing: " + body
	case c.NoRoute != "":
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			return statusFail, fmt.Sprintf("%d, want 404 or 405 until %s: %s", rec.Code, c.NoRoute, body)
		}
		if strings.Contains(body, TenantB) {
			return statusFail, fmt.Sprintf("%d with tenant B in the body: %s", rec.Code, body)
		}
		return statusPass, fmt.Sprintf("%d (no route until %s), tenant B not reached", rec.Code, c.NoRoute)
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

// rowCounts is one tenant's records and record-domain audit_log entry counts
// (the pipeline writes audit_log since P5-13b, TICK-452).
type rowCounts struct{ records, audit int }

// countRows reads tenant's counts on the superuser connection, past
// row-level security, so a row misfiled under tenant cannot hide.
func countRows(ctx context.Context, super *pgx.Conn, tenant string) (rowCounts, error) {
	var n rowCounts
	err := super.QueryRow(ctx, `SELECT (SELECT count(*) FROM records WHERE tenant_id = $1),
		(SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record')`, tenant).Scan(&n.records, &n.audit)
	return n, err
}

// recordIDs reads tenant's record ids on the superuser connection, past
// row-level security.
func recordIDs(ctx context.Context, super *pgx.Conn, tenant string) (map[string]struct{}, error) {
	rows, err := super.Query(ctx, `SELECT id::text FROM records WHERE tenant_id = $1`, tenant)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set, nil
}

// canonicalID returns raw as a canonical uuid (lowercase, hyphenated), the
// form id::text gives, and false when raw is no uuid.
func canonicalID(raw string) (string, bool) {
	u, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return u.String(), true
}

// TestCanonicalID (TICK-473 review): every form of a seed id that Postgres
// casts to the same uuid canonicalizes to the snapshot's key, so a 201 naming
// "…A2" or the unhyphenated id is caught as already held.
func TestCanonicalID(t *testing.T) {
	const seed = "0a000000-0000-4000-8000-0000000000a2"
	for _, raw := range []string{seed, "0A000000-0000-4000-8000-0000000000A2", "0a0000000000400080000000000000a2",
		"{0a000000-0000-4000-8000-0000000000a2}", "urn:uuid:0a000000-0000-4000-8000-0000000000a2"} {
		if got, ok := canonicalID(raw); !ok || got != seed {
			t.Errorf("canonicalID(%q) = %q, %v; want %q, true", raw, got, ok, seed)
		}
	}
	for _, raw := range []string{"", "a2", "0a000000-0000-4000-8000-0000000000a2x"} {
		if got, ok := canonicalID(raw); ok {
			t.Errorf("canonicalID(%q) = %q, true; want false", raw, got)
		}
	}
}

// runStoredUnderA runs a StoredUnderA write (P3-18, TICK-366), under the CTO's
// D-P10 conditions on TICK-366 (C1 to C5 below). It passes only:
//
//   - "refused 404": the 404 that the same request to c.Unknown (a key no
//     tenant has) gets, byte for byte after withoutInstance, with nothing
//     written in either tenant (Postgres review P1); or
//   - "stored under A, row <id>": a 201 whose id names a record filed under
//     tenant A that tenant A did not hold before the call (TICK-473), and
//     tenant A gained exactly that one record. A 200 can only be
//     a replay, which fails anyway (C4).
//
// Every read is on the superuser connection, past row-level security: a read
// as tenant A cannot see a row misfiled under tenant B. The call's own
// Idempotency-Key row is deleted before it returns. Its record rows are
// deleted by id only once that id has been shown to be the one new tenant A
// record, absent from the snapshot taken before the call, so a 201 naming a
// seed row can never delete it (C2, TICK-473).
func (r *apiRunner) runStoredUnderA(t *testing.T, c *HTTPCall) (status, detail string) {
	t.Helper()
	ctx := t.Context()
	// C4: each case has its own key (TestRegistryIsWellFormed checks they are
	// unique); without one the call is a 400 that proves nothing.
	key := c.Header.Get("Idempotency-Key")
	if key == "" {
		return statusFail, "a StoredUnderA call needs its own Idempotency-Key"
	}
	super := suite.superConn(t)
	defer func() {
		if _, err := super.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1`, key); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	// C1: tenant B's snapshot is taken before the call and compared after it
	// on every outcome, refusals included.
	beforeA, err := countRows(ctx, super, TenantA)
	if err != nil {
		return statusFail, err.Error()
	}
	beforeB, err := countRows(ctx, super, TenantB)
	if err != nil {
		return statusFail, err.Error()
	}
	// TICK-473: tenant A's record ids before the call. A 201 that names one
	// of them is not this call's row, even when tenant A also gained one, so
	// the cleanup can never delete a seed row.
	seedA, err := recordIDs(ctx, super, TenantA)
	if err != nil {
		return statusFail, err.Error()
	}
	rec := r.do(t, c)
	body := rec.Body.String()
	created := rec.Code == http.StatusCreated // C2: 201 only

	afterA, err := countRows(ctx, super, TenantA)
	if err != nil {
		return statusFail, err.Error()
	}
	afterB, err := countRows(ctx, super, TenantB)
	if err != nil {
		return statusFail, err.Error()
	}
	// C2: the stored row is the one the response names, by id, never the
	// latest row. The +1 on tenant A is a guard on top of that lookup: only
	// when both hold is the id this call's own row, safe to delete.
	var row struct {
		ID string `json:"id"`
	}
	// The lookup has no tenant predicate, so it finds the row wherever it
	// landed; an id can repeat across tenants (the seed's IDOR fixtures do),
	// so it must name exactly one row.
	var stored []string // the tenants rows with that id are filed under
	var existed bool    // the id was tenant A's before the call
	if created {
		if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
			return statusFail, fmt.Sprintf("201 with no record id the runner can resolve: %s", body)
		}
		// The seed snapshot holds Postgres's canonical id::text, but Postgres
		// also reads an uppercase or unhyphenated id as the same uuid, so the
		// lookup, the query and the cleanup all use the canonical form; else
		// "…A2" would miss seedA and the cleanup would delete seed "…a2".
		id, ok := canonicalID(row.ID)
		if !ok {
			return statusFail, fmt.Sprintf("201 with no record id the runner can resolve: %s", body)
		}
		row.ID = id
		rows, err := super.Query(ctx, `SELECT tenant_id::text FROM records WHERE id = $1 ORDER BY tenant_id`, row.ID)
		if err != nil {
			return statusFail, err.Error()
		}
		if stored, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
			return statusFail, err.Error()
		}
		_, existed = seedA[row.ID]
		if !existed && slices.Equal(stored, []string{TenantA}) && afterA.records == beforeA.records+1 {
			defer func() {
				// TODO(TODO-S9-01): P3-42: this leaves the record's outbox and
				// cascade_start rows behind. Once the EventSink is wired,
				// delete cascade_start first, then outbox, then these.
				for _, q := range []string{`DELETE FROM audit_log WHERE tenant_id = $1 AND subject_id = $2`, `DELETE FROM records WHERE tenant_id = $1 AND id = $2`} {
					if _, err := super.Exec(context.Background(), q, TenantA, row.ID); err != nil {
						t.Errorf("cleanup: %v", err)
					}
				}
			}()
		}
	}

	switch {
	case afterB != beforeB: // C1
		return statusFail, fmt.Sprintf("%d, and tenant B's records and audit rows went from %+v to %+v: the write reached tenant B: %s",
			rec.Code, beforeB, afterB, body)
	case strings.Contains(body, TenantB): // C3, refusals included
		return statusFail, fmt.Sprintf("%d with tenant B in the body: %s", rec.Code, body)
	case rec.Header().Get(idempotency.ReplayedHeader) == "true": // C4
		return statusFail, fmt.Sprintf("%d replayed from the key store: the handler never read this call's forged field: %s", rec.Code, body)
	case rec.Code == http.StatusUnauthorized: // C3
		return statusFail, "401: the token for tenant A was not honoured, so this proves nothing: " + body
	case created: // C2, C5
		if existed {
			return statusFail, fmt.Sprintf("201 naming record %s, which tenant A already held before the call: not this call's row, so it is left in place: %s",
				row.ID, body)
		}
		if !slices.Equal(stored, []string{TenantA}) {
			return statusFail, fmt.Sprintf("201, record %s is filed under tenants %v, want exactly one row, in tenant A", row.ID, stored)
		}
		if afterA.records != beforeA.records+1 {
			return statusFail, fmt.Sprintf("201 naming record %s, but tenant A's records went from %d to %d, want one more: not this call's row, so it is left in place",
				row.ID, beforeA.records, afterA.records)
		}
		return statusPass, fmt.Sprintf("stored under A, row %s (201); tenant B unchanged at %+v", row.ID, afterB)
	case rec.Code == http.StatusNotFound: // C3, C5, P1
		if afterA != beforeA {
			return statusFail, fmt.Sprintf("404, but tenant A's records and audit rows went from %+v to %+v: a refusal must write nothing", beforeA, afterA)
		}
		if err := r.sameAsUnknown(t, c, body); err != nil {
			return statusFail, err.Error()
		}
		return statusPass, fmt.Sprintf("refused 404, the unknown key's byte for byte; nothing written, tenant B unchanged at %+v", afterB)
	}
	// C3, P1: 5xx fails. So do 400, 409 and 429, narrower than C3's set:
	// they come from the envelope, the Idempotency-Key store or the limiter,
	// before the create reads the tenant, so they prove nothing. And so do
	// 403 and 422: tenant A has no such key, so any refusal but the unknown
	// key's 404 means the handler found the type somewhere, and tells the
	// caller so.
	return statusFail, fmt.Sprintf("%d, want the unknown key's 404 or 201 under tenant A: %s", rec.Code, body)
}

// sameAsUnknown sends c again to c.Unknown, with its own Idempotency-Key,
// and requires its 404 to match body byte for byte after withoutInstance.
func (r *apiRunner) sameAsUnknown(t *testing.T, c *HTTPCall, body string) error {
	t.Helper()
	if c.Unknown == "" {
		return errors.New("a StoredUnderA call needs Unknown for its 404 control")
	}
	u := *c
	u.Path = c.Unknown
	u.Header = c.Header.Clone()
	key := c.Header.Get("Idempotency-Key") + "-unknown"
	u.Header.Set("Idempotency-Key", key)
	defer func() {
		if _, err := suite.superConn(t).Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1`, key); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	rec := r.do(t, &u)
	if rec.Code != http.StatusNotFound {
		return fmt.Errorf("control: %s %s = %d %s, want 404", u.Method, u.Path, rec.Code, rec.Body)
	}
	got, err := withoutInstance(body)
	if err != nil {
		return err
	}
	want, err := withoutInstance(rec.Body.String())
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	if got != want {
		return fmt.Errorf("404 differs from the unknown key's, so it tells the caller the key exists:\n%s\n%s", got, want)
	}
	return nil
}
