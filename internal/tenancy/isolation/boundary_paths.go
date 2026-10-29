package isolation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// boundaryPaths are the eight cases of T26 (TICK-82, PLAN-001 D-P9, Epic 03
// boundary), one path each. Cases that need the real Keycloak are Stack
// paths: their tests live in internal/identitystack.
func boundaryPaths() []Path {
	return []Path{
		{Name: "T26-1 deprovisioning: a group removal reaches an unexpired token within one sync cycle", Case: "T26", Role: "nava_groupsync", API: groupRemovalWithinBound},
		{Name: "T26-2 deprovisioning: a disabled user's unexpired token is refused within one sync cycle", Case: "T26", Role: "nava_groupsync", API: disabledUserWithinBound},
		{Name: "T26-3 self-service cannot change any attribute feeding tenant, role or group", Case: "T26", Stack: "TestSelfServiceCannotChangeTenantRoleOrGroupAttributes"},
		{Name: "T26-4 a service-account token cannot act in any tenant", Case: "T26", Stack: "TestServiceAccountTokenCannotActInAnyTenant"},
		{Name: "T26-5 token exchange stays disabled", Case: "T26", Stack: "TestTokenExchangeStaysDisabled"},
		{Name: "T26-6 operator boundary: a tenant admin token is refused on /api/v1/operator/*", Case: "T26", API: tenantAdminRefusedOnOperatorRoutes},
		{Name: "T26-7 operator boundary: a user cannot self-assign nava-operator", Case: "T26", Stack: "TestUserCannotSelfAssignOperatorRole"},
		{Name: "T26-8 operator boundary: an operator token on a tenant endpoint gets 401", Case: "T26", API: operatorTokenRefusedOnTenantRoutes},
	}
}

// D-P9: a change in Keycloak reaches every API call within the access token
// lifespan (5 min) plus one sync interval (5 min). The API reads groups and
// the enabled flag from the synced rows on every request, never from the
// token, so after one sync cycle an unexpired token already sees the change;
// the token lifespan bounds only what the IdP itself still accepts.
const (
	deprovisionBound = 10 * time.Minute
	maxTokenLifespan = 5 * time.Minute
)

// Tenant A's principals the two deprovisioning paths add and remove, and the
// seeded tenant A group they join.
const (
	deprovGroupUser    = "kc-deprov-group"
	deprovDisabledUser = "kc-deprov-disabled"
	groupA1            = "0a000000-0000-4000-8000-0000000004a1"
)

// groupRemovalWithinBound (T26 case 1): a tenant A user in group A1 holds an
// unexpired token, and GET /api/v1/me lists A1. Keycloak removes the user
// from the group; one nava-groupsync cycle (the real Syncer, as
// nava_groupsync through PgBouncer) applies it, and the same token's next
// GET /api/v1/me no longer lists A1. The sync interval and the realm's token
// lifespan add up to at most D-P9's 10 minutes, and tenant B's identity rows
// are untouched.
func groupRemovalWithinBound(ctx context.Context, c Cluster, s Server) error {
	return deprovision(ctx, c, s, deprovGroupUser, func(u *identity.DirectoryUser, groups []identity.DirectoryGroup) {
		for i := range groups {
			groups[i].MemberKeycloakUserIDs = slices.DeleteFunc(groups[i].MemberKeycloakUserIDs,
				func(id string) bool { return id == u.KeycloakUserID })
		}
	}, func(status int, body string) error {
		if status != http.StatusOK {
			return fmt.Errorf("after the sync: %d %s, want 200 without group A1", status, body)
		}
		if me, err := readMe(body); err != nil || me.TenantID != TenantA || slices.Contains(me.Groups, groupA1) {
			return fmt.Errorf("after the sync: %s, want tenant A without group A1 (%w)", body, err)
		}
		return nil
	})
}

// disabledUserWithinBound (T26 case 2): as case 1, but Keycloak disables the
// user. After one sync cycle the same unexpired token gets 401
// tenant_unresolved: the resolver refuses a disabled principal.
func disabledUserWithinBound(ctx context.Context, c Cluster, s Server) error {
	return deprovision(ctx, c, s, deprovDisabledUser, func(u *identity.DirectoryUser, _ []identity.DirectoryGroup) {
		u.Enabled = false
	}, func(status int, body string) error {
		if status != http.StatusUnauthorized || !strings.Contains(body, "tenant_unresolved") {
			return fmt.Errorf("after the sync: %d %s, want 401 tenant_unresolved", status, body)
		}
		if strings.Contains(body, TenantA) || strings.Contains(body, TenantB) {
			return fmt.Errorf("the refusal names a tenant: %s", body)
		}
		return nil
	})
}

// deprovision seeds subject as an enabled tenant A principal in group A1,
// proves its token reads tenant A with A1, applies change to the directory
// snapshot of tenant A, runs one sync cycle, and hands the same token's next
// GET /api/v1/me to check. It removes what it seeded, and fails when tenant
// A's or B's identity rows are not as they were.
func deprovision(ctx context.Context, c Cluster, s Server, subject string,
	change func(*identity.DirectoryUser, []identity.DirectoryGroup), check func(int, string) error,
) error {
	if err := withinBound(); err != nil {
		return err
	}
	super, err := c.Super()
	if err != nil {
		return err
	}
	before, err := identityRows(ctx, super)
	if err != nil {
		return err
	}
	pid := uuid.NewString()
	if _, err := super.Exec(ctx, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at)
		VALUES ($1, $2, $3, true, '[]', now())`, TenantA, pid, subject); err != nil {
		return fmt.Errorf("seed %s: %w", subject, err)
	}
	defer func() {
		_, _ = super.Exec(context.Background(), "DELETE FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2", TenantA, subject)
	}()
	if _, err := super.Exec(ctx, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)",
		TenantA, groupA1, pid); err != nil {
		return fmt.Errorf("seed %s in A1: %w", subject, err)
	}

	// Positive control: before the change the token reads tenant A with A1.
	tok := s.Mint(Token{Subject: subject})
	status, body := s.Do(ctx, http.MethodGet, "/api/v1/me", tok, "", nil)
	if me, err := readMe(body); status != http.StatusOK || err != nil || me.TenantID != TenantA || !slices.Contains(me.Groups, groupA1) {
		return fmt.Errorf("before the change: %d %s, want 200 for tenant A in group A1", status, body)
	}

	// Keycloak's side: tenant A's directory as the tables hold it, with the
	// change applied. Every other tenant reports no change.
	snap, err := directoryOf(ctx, super, TenantA)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(snap.Users, func(u identity.DirectoryUser) bool { return u.KeycloakUserID == subject })
	if i < 0 {
		return fmt.Errorf("%s is not in tenant A's snapshot", subject)
	}
	change(&snap.Users[i], snap.Groups)

	pool, err := c.Pool("nava_groupsync")
	if err != nil {
		return err
	}
	syncer, err := groupsync.New(pool, oneTenantDirectory{tenant: uuid.MustParse(TenantA), snap: snap},
		metricnoop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{})
	if err != nil {
		return err
	}
	results, err := syncer.Once(ctx)
	if err != nil {
		return fmt.Errorf("sync cycle: %w", err)
	}
	synced := false
	for _, r := range results {
		if r.Err != nil || r.Skipped {
			return fmt.Errorf("sync of tenant %s: skipped=%v err=%w", r.TenantID, r.Skipped, r.Err)
		}
		synced = synced || r.TenantID.String() == TenantA && !r.Unchanged
	}
	if !synced {
		return fmt.Errorf("the cycle did not apply tenant A's snapshot: %+v", results)
	}

	status, body = s.Do(ctx, http.MethodGet, "/api/v1/me", tok, "", nil)
	if err := check(status, body); err != nil {
		return err
	}

	// Only the seeded principal changed: removing it must leave both
	// tenants' identity rows exactly as they were.
	if _, err := super.Exec(ctx, "DELETE FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2", TenantA, subject); err != nil {
		return err
	}
	after, err := identityRows(ctx, super)
	if err != nil {
		return err
	}
	if after != before {
		return fmt.Errorf("the sync changed rows it should not have:\nbefore %s\nafter  %s", before, after)
	}
	return nil
}

// withinBound checks the two halves of D-P9 against the code and the realm:
// the longest sync interval nava-groupsync accepts (config.MaxGroupsyncInterval,
// the cap on NAVA_GROUPSYNC_INTERVAL) and its default, and the realm's access
// token lifespan.
func withinBound() error {
	for name, d := range map[string]time.Duration{
		"config.MaxGroupsyncInterval": config.MaxGroupsyncInterval,
		"groupsync.DefaultInterval":   groupsync.DefaultInterval,
	} {
		if d > deprovisionBound-maxTokenLifespan {
			return fmt.Errorf("%s = %v: with a %v token the bound exceeds %v", name, d, maxTokenLifespan, deprovisionBound)
		}
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("cannot locate the realm export")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "deploy", "compose", "keycloak", "import", "realm-nava.json"))
	if err != nil {
		return err
	}
	var realm struct {
		AccessTokenLifespan int `json:"accessTokenLifespan"`
	}
	if err := json.Unmarshal(raw, &realm); err != nil {
		return err
	}
	if l := time.Duration(realm.AccessTokenLifespan) * time.Second; l <= 0 || l > maxTokenLifespan {
		return fmt.Errorf("realm nava accessTokenLifespan = %v, want 0 < lifespan <= %v (D-P9)", l, maxTokenLifespan)
	}
	return nil
}

type me struct {
	TenantID string   `json:"tenant_id"`
	Groups   []string `json:"groups"`
}

func readMe(body string) (me, error) {
	var m me
	err := json.Unmarshal([]byte(body), &m)
	return m, err
}

// identityRowsSQL is every identity row of tenants A and B, in a fixed order.
const identityRowsSQL = `SELECT concat_ws(' | ',
  (SELECT string_agg(concat_ws(',', tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at), ';' ORDER BY tenant_id, id) FROM principals),
  (SELECT string_agg(concat_ws(',', tenant_id, id, keycloak_group_id, path), ';' ORDER BY tenant_id, id) FROM groups),
  (SELECT string_agg(concat_ws(',', tenant_id, group_id, principal_id), ';' ORDER BY tenant_id, group_id, principal_id) FROM group_members),
  (SELECT string_agg(concat_ws(',', tenant_id, id, parent_id, keycloak_group_id, name), ';' ORDER BY tenant_id, id) FROM org_units))`

func identityRows(ctx context.Context, super *pgx.Conn) (string, error) {
	var s string
	err := super.QueryRow(ctx, identityRowsSQL).Scan(&s)
	return s, err
}

// directoryOf is tenant's directory snapshot as the identity tables hold it:
// a cycle that applies it unchanged writes nothing.
func directoryOf(ctx context.Context, super *pgx.Conn, tenant string) (identity.DirectoryChanges, error) {
	var out identity.DirectoryChanges
	rows, err := super.Query(ctx, `SELECT keycloak_user_id, coalesce(email, ''), enabled, coalesce(federated_links, '[]')
		FROM principals WHERE tenant_id = $1 ORDER BY keycloak_user_id`, tenant)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var u identity.DirectoryUser
		var links []byte
		if err := rows.Scan(&u.KeycloakUserID, &u.Email, &u.Enabled, &links); err != nil {
			rows.Close()
			return out, err
		}
		var ls []struct {
			IdentityProvider string `json:"identityProvider"`
			UserID           string `json:"userId"`
		}
		if err := json.Unmarshal(links, &ls); err != nil {
			rows.Close()
			return out, err
		}
		for _, l := range ls {
			u.FederatedLinks = append(u.FederatedLinks, identity.FederatedLink{IdPAlias: l.IdentityProvider, ExternalUserID: l.UserID})
		}
		out.Users = append(out.Users, u)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	rows, err = super.Query(ctx, `SELECT g.keycloak_group_id, g.path,
		coalesce(array_agg(p.keycloak_user_id ORDER BY p.keycloak_user_id) FILTER (WHERE p.id IS NOT NULL), '{}')
		FROM groups g
		LEFT JOIN group_members m ON m.tenant_id = g.tenant_id AND m.group_id = g.id
		LEFT JOIN principals p ON p.tenant_id = m.tenant_id AND p.id = m.principal_id
		WHERE g.tenant_id = $1 GROUP BY g.keycloak_group_id, g.path ORDER BY g.keycloak_group_id`, tenant)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var g identity.DirectoryGroup
		if err := rows.Scan(&g.KeycloakGroupID, &g.Path, &g.MemberKeycloakUserIDs); err != nil {
			rows.Close()
			return out, err
		}
		g.Name = g.Path[strings.LastIndex(g.Path, "/")+1:]
		out.Groups = append(out.Groups, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	// org_units keeps no path; any path under /t/<tenant>/ou/ is accepted.
	rows, err = super.Query(ctx, `SELECT u.keycloak_group_id, coalesce(p.keycloak_group_id, ''), u.name
		FROM org_units u LEFT JOIN org_units p ON p.tenant_id = u.tenant_id AND p.id = u.parent_id
		WHERE u.tenant_id = $1 ORDER BY u.keycloak_group_id`, tenant)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var g identity.DirectoryGroup
		if err := rows.Scan(&g.KeycloakGroupID, &g.ParentKeycloakGroupID, &g.Name); err != nil {
			rows.Close()
			return out, err
		}
		g.Path = "/t/" + tenant + "/ou/" + g.KeycloakGroupID
		out.OrgUnits = append(out.OrgUnits, g)
	}
	rows.Close()
	return out, rows.Err()
}

// oneTenantDirectory is Keycloak as group sync sees it: snap for tenant, no
// change for every other tenant, so the cycle writes nothing outside tenant.
type oneTenantDirectory struct {
	identity.DirectoryAdmin
	tenant uuid.UUID
	snap   identity.DirectoryChanges
}

func (d oneTenantDirectory) ChangesSince(_ context.Context, tenantID uuid.UUID, _ string) (identity.DirectoryChanges, string, error) {
	if tenantID != d.tenant {
		return identity.DirectoryChanges{Unchanged: true}, "unchanged", nil
	}
	return d.snap, "snapshot", nil
}

// tenantAdminRefusedOnOperatorRoutes (T26 case 6): admin@tenant-a holds a
// valid token with tenant-admin, which GET /api/v1/me honours for tenant A.
// The operator resolver, which authenticates every /api/v1/operator/* route,
// refuses it, while it accepts the allow-listed operator (the control). On
// every operator route of the contract it gets 403 from the suite's handler,
// built with server.WithOperator, with or without the route's required
// header and body, and reaches no operator handler.
func tenantAdminRefusedOnOperatorRoutes(ctx context.Context, _ Cluster, s Server) error {
	admin := s.Mint(Token{Subject: isoTenantSubject, RealmRoles: []string{"tenant-admin"}})
	if status, body := s.Do(ctx, http.MethodGet, "/api/v1/me", admin, "", nil); status != http.StatusOK || !strings.Contains(body, TenantA) {
		return fmt.Errorf("the tenant admin's token on GET /api/v1/me: %d %s, want 200 for tenant A", status, body)
	}
	if err := s.ResolveOperator(ctx, operatorToken(s, false)); err != nil {
		return fmt.Errorf("control: the allow-listed operator's token was refused: %w", err)
	}
	if err := s.ResolveOperator(ctx, admin); !errors.Is(err, identity.ErrInvalidToken) {
		return fmt.Errorf("ResolveOperator(tenant admin) = %w, want ErrInvalidToken", err)
	}
	routes, err := operatorRoutes()
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		return errors.New("the contract has no /api/v1/operator/* route: T26 case 6 proves nothing")
	}
	// A contract-valid request, so the 403 comes from authentication and not
	// from the validator rejecting the input; and requests missing the
	// required header or body, so authentication answers before the contract
	// does (TICK-298).
	key := http.Header{"Idempotency-Key": {"t26-6"}}
	var errs []error
	for _, rt := range routes {
		for _, c := range []struct {
			name   string
			header http.Header
			body   string
		}{
			{"a contract-valid request", key, onboardRequestBody},
			{"no Idempotency-Key", nil, onboardRequestBody},
			{"no body", key, ""},
		} {
			status, body := s.Do(ctx, rt.method, rt.path, admin, c.body, c.header)
			if status != http.StatusForbidden || !strings.Contains(body, `"forbidden"`) {
				errs = append(errs, fmt.Errorf("%s %s with a tenant admin token (%s): %d %s, want 403 forbidden", rt.method, rt.path, c.name, status, body))
			}
		}
	}
	return errors.Join(errs...)
}

type operatorRoute struct{ method, path string }

// operatorRoutes lists every operation the contract has under
// /api/v1/operator/, path parameters filled with tenant B's id.
func operatorRoutes() ([]operatorRoute, error) {
	data, err := api.Files.ReadFile(api.SpecFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", api.SpecFile, err)
	}
	spec, err := apihttp.LoadSpec(data)
	if err != nil {
		return nil, err
	}
	var out []operatorRoute
	for p, item := range spec.Paths.Map() {
		if !strings.HasPrefix(p, "/api/v1/operator/") {
			continue
		}
		path := pathParam.ReplaceAllString(p, TenantB)
		for m := range item.Operations() {
			out = append(out, operatorRoute{m, path})
		}
	}
	slices.SortFunc(out, func(a, b operatorRoute) int { return strings.Compare(a.path+" "+a.method, b.path+" "+b.method) })
	return out, nil
}

// pathParam is a {param} segment of a contract path.
var pathParam = regexp.MustCompile(`\{[^}]+\}`)

// onboardRequestBody is a well-formed POST /api/v1/operator/tenants body.
const onboardRequestBody = `{"admin_email":"admin@acme.test","email_domains":["acme.test"],"identity_provider":{"provider_id":"oidc","config":{"clientId":"c"}}}`

// operatorTokenRefusedOnTenantRoutes (T26 case 8): the allow-listed
// operator's token (the control: the operator resolver accepts it) on GET
// /api/v1/me gets 401 invalid_token, since its audience is
// nava-operator-api. A token that also names nava-api, as a mapper mistake
// could make it, passes the audience check and gets 401 tenant_unresolved,
// since an operator has no organization. Neither response names a tenant.
func operatorTokenRefusedOnTenantRoutes(ctx context.Context, _ Cluster, s Server) error {
	for _, c := range []struct {
		both bool
		code string
	}{{false, "invalid_token"}, {true, "tenant_unresolved"}} {
		tok := operatorToken(s, c.both)
		if err := s.ResolveOperator(ctx, tok); err != nil {
			return fmt.Errorf("control: the operator token (both audiences %v) was refused: %w", c.both, err)
		}
		status, body := s.Do(ctx, http.MethodGet, "/api/v1/me", tok, "", nil)
		if status != http.StatusUnauthorized || !strings.Contains(body, `"`+c.code+`"`) {
			return fmt.Errorf("operator token (both audiences %v) on GET /api/v1/me: %d %s, want 401 %s", c.both, status, body, c.code)
		}
		if strings.Contains(body, TenantA) || strings.Contains(body, TenantB) {
			return fmt.Errorf("the refusal names a tenant: %s", body)
		}
	}
	return nil
}

// isoTenantSubject is tenant A's seeded principal kc-user-shared.
const isoTenantSubject = "kc-user-shared"

// operatorToken is the allow-listed operator's token: realm role
// nava-operator, no organization, audience nava-operator-api (and nava-api
// too when both).
func operatorToken(s Server, both bool) string {
	aud := []string{"nava-operator-api"}
	if both {
		aud = append(aud, "nava-api")
	}
	return s.Mint(Token{Subject: OperatorSubject, NoOrg: true, Audiences: aud, RealmRoles: []string{"nava-operator"}})
}
