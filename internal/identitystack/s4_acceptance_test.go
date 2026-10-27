//go:build identitystack && acceptance

package identitystack

// Story S4 acceptance (TICK-46): an operator onboards two tenants, each admin
// signs in through their tenant's IdP and GET /api/v1/me is scoped to that
// tenant, and a user already federated to another tenant's IdP cannot be
// invited. Also the cross-tenant check of the acceptance kit. Not a CI test:
// it runs once per acceptance, on a fresh stack, and writes the recorded
// requests and responses (no token, no secret) to NAVA_S4_EVIDENCE.
//
//	docker compose up --wait
//	NAVA_S4_EVIDENCE=docs/acceptance/evidence/operator-onboards-tenant.md \
//	  go test -tags 'identitystack acceptance' -count=1 -v -run '^TestS4Acceptance$' ./internal/identitystack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// evidence is the markdown transcript of the run.
type evidence struct{ b strings.Builder }

func (e *evidence) h(format string, a ...any) { fmt.Fprintf(&e.b, "\n## "+format+"\n\n", a...) }
func (e *evidence) p(format string, a ...any) { fmt.Fprintf(&e.b, format+"\n\n", a...) }

func (e *evidence) check(t *testing.T, ok bool, format string, a ...any) {
	t.Helper()
	mark := "PASS"
	if !ok {
		mark = "FAIL"
		t.Errorf(format, a...)
	}
	fmt.Fprintf(&e.b, "- **%s** %s\n", mark, fmt.Sprintf(format, a...))
}

func (e *evidence) exchange(title, req string, code int, resp []byte) {
	if len(resp) == 0 {
		resp = []byte("(no body: busybox wget keeps none for an error status)")
	}
	fmt.Fprintf(&e.b, "%s\n\n```http\n%s\n```\n\n```http\nHTTP/1.1 %d\n\n%s\n```\n\n", title, req, code, bytes.TrimSpace(resp))
}

// redactBody hides the IdP client secret, a dev literal but still a secret.
func redactBody(body []byte) string {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return string(body)
	}
	if idp, ok := m["identity_provider"].(map[string]any); ok {
		if cfg, ok := idp["config"].(map[string]any); ok {
			cfg["clientSecret"] = "<redacted>"
		}
	}
	out, _ := json.MarshalIndent(m, "", "  ")
	return string(out)
}

// onboardPostRaw is onboardPost through busybox nc, which, unlike wget,
// returns the body of an error status. stdin stays open while the server
// answers (a half-close would cancel the request); the token is passed in the
// environment, never argv.
func onboardPostRaw(t *testing.T, token, key string, body []byte) (int, []byte) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "docker", "compose", "exec", "-T", "-e", "NAVA_OPERATOR_TOKEN", "nava-onboard", "sh", "-c",
		`{ printf 'POST /api/v1/operator/tenants HTTP/1.1\r\nHost: 127.0.0.1\r\nAuthorization: Bearer %s\r\nIdempotency-Key: %s\r\n`+
			`Content-Type: application/json\r\nContent-Length: %s\r\nConnection: close\r\n\r\n%s' "$NAVA_OPERATOR_TOKEN" "$1" "${#2}" "$2"; sleep 15; } | nc -w 20 127.0.0.1 8081`,
		"sh", key, string(body))
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "NAVA_OPERATOR_TOKEN="+token)
	out, _ := cmd.Output()
	head, rest, _ := bytes.Cut(out, []byte("\r\n\r\n"))
	m := statusLine.FindSubmatch(head)
	if m == nil {
		t.Fatalf("no HTTP response from nava-onboard: %.400s", out)
	}
	code, _ := strconv.Atoi(string(m[1]))
	return code, rest
}

// feedback is the sentence around Keycloak's error on a login page's text.
func feedback(page string) string {
	i := strings.Index(strings.ToLower(page), "exist")
	if i < 0 {
		return "no error text on the page"
	}
	return strings.TrimSpace(page[max(0, i-90):min(len(page), i+60)])
}

func onboardReq(key string, body []byte) string {
	return "POST /api/v1/operator/tenants (nava-onboard :8081)\nAuthorization: Bearer <operator@nava.test token>\n" +
		"Idempotency-Key: " + key + "\nContent-Type: application/json\n\n" + redactBody(body)
}

func meReq(query string, header http.Header, who string) string {
	s := "GET /api/v1/me" + query + " (nava-core)\nAuthorization: Bearer <" + who + " token>"
	for k, v := range header {
		s += "\n" + k + ": " + strings.Join(v, ",")
	}
	return s
}

// onboardCounts is what one onboarding can create, for the "creates nothing
// new" half of criterion 1.
func onboardCounts(t *testing.T, alias string) string {
	t.Helper()
	rows := psqlSuper(t, "SELECT (SELECT count(*) FROM tenants) || ' tenants, ' || "+
		"(SELECT count(*) FROM tenant_identity_binding) || ' bindings, ' || "+
		"(SELECT count(*) FROM tenant_email_domain) || ' email domains, ' || "+
		"(SELECT count(*) FROM pending_invite) || ' invites'")
	return fmt.Sprintf("%s, %d Keycloak organizations %s", rows, len(orgsByAlias(t, alias)), alias)
}

type s4Tenant struct {
	tenantFx
	key  string
	body []byte
}

func s4NewTenant(t *testing.T) s4Tenant {
	t.Helper()
	key, tenant, domain, email := newOnboarding(t)
	f := s4Tenant{tenantFx: tenantFx{
		tenant: tenant, alias: onboard.Alias(tenant), idpAlias: keycloak.TenantIdPAlias(tenant),
		realm: "onb-" + tenant.String()[:8], domain: domain, admin: email, pass: "dev-only-" + uuid.NewString()[:8],
	}, key: key, body: onboardBody(email, domain)}
	createIdPRealm(t, f.realm, email, f.pass)
	return f
}

func TestS4Acceptance(t *testing.T) {
	out := os.Getenv("NAVA_S4_EVIDENCE")
	if out == "" {
		t.Skip("NAVA_S4_EVIDENCE is not set")
	}
	var e evidence
	e.p("# S4 acceptance evidence (TICK-46)\n\nRecorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on %s. "+
		"Tokens are never recorded; the IdP client secret is redacted.", time.Now().UTC().Format(time.RFC3339))
	t.Cleanup(func() {
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err == nil {
			_ = os.WriteFile(out, []byte(e.b.String()), 0o644)
		}
	})

	stopGroupsync(t)
	op := operatorToken(t)
	a, b := s4NewTenant(t), s4NewTenant(t)
	e.p("Tenant A: `%s`, domain `%s`, admin `%s`.\nTenant B: `%s`, domain `%s`, admin `%s`.", a.tenant, a.domain, a.admin, b.tenant, b.domain, b.admin)

	// Criterion 1: 201 with the tenant id; the same key and body again gives
	// the same body and creates nothing.
	e.h("Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing")
	for _, f := range []s4Tenant{a, b} {
		code, first := onboardPost(t, op, f.key, f.body)
		e.exchange("First call, tenant "+f.domain+":", onboardReq(f.key, f.body), code, first)
		var got onboarded
		_ = json.Unmarshal(first, &got)
		e.check(t, code == http.StatusCreated && got.TenantID == f.tenant && got.State == "active",
			"first call: %d, tenant_id `%s`, state `%s` (want 201, `%s`, `active`)", code, got.TenantID, got.State, f.tenant)

		before := onboardCounts(t, f.alias)
		code, again := onboardPost(t, op, f.key, f.body)
		e.exchange("Replay, same key and body:", onboardReq(f.key, f.body), code, again)
		after := onboardCounts(t, f.alias)
		e.check(t, code == http.StatusOK, "replay status %d (documented: 200 for a replay of a completed request)", code)
		e.check(t, bytes.Equal(bytes.TrimSpace(first), bytes.TrimSpace(again)), "replay body is byte-identical to the first body")
		e.check(t, before == after, "nothing new: before replay `%s`, after `%s`", before, after)

		changed := onboardBody(f.admin, f.domain)
		changed = bytes.Replace(changed, []byte(`"Onboarded `), []byte(`"Changed `), 1)
		code, resp := onboardPost(t, op, f.key, changed)
		e.exchange("Same key, different body:", onboardReq(f.key, changed), code, resp)
		e.check(t, code == http.StatusConflict, "same key with a different body: %d (want 409 `idempotency_key_reuse`)", code)
		e.check(t, onboardCounts(t, f.alias) == after, "nothing new after the conflicting call")
		e.check(t, kcUserByEmail(t, f.admin) == "" && inviteState(t, f.tenant, f.admin) == "pending",
			"the admin `%s` has a pending invite and no Keycloak user yet", f.admin)
		pointIdPAt(t, f.idpAlias, f.realm)
		ids := orgsByAlias(t, f.alias)
		if len(ids) != 1 {
			t.Fatalf("organizations %s: %v", f.alias, ids)
		}
	}
	a.orgID, b.orgID = orgsByAlias(t, a.alias)[0], orgsByAlias(t, b.alias)[0]

	// Criterion 2: each admin signs in through their tenant's IdP and is
	// scoped to their tenant.
	e.h("Criterion 2: the new admin signs in and is scoped to the new tenant")
	toks := map[uuid.UUID]string{}
	for _, f := range []s4Tenant{a, b} {
		tok := brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
		c := claims(t, tok)
		org, _ := c["organization"].(map[string]any)
		_, inOrg := org[f.alias]
		e.check(t, len(org) == 1 && inOrg, "`%s` signed in through IdP `%s`; the token's `organization` claim is exactly `%s`", f.admin, f.idpAlias, f.alias)
		toks[f.tenant] = tok
	}
	groupsyncOnce(t)
	e.p("One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).")
	for _, f := range []s4Tenant{a, b} {
		code, m, raw := getMe(t, toks[f.tenant], "", nil)
		e.exchange("Admin of "+f.domain+":", meReq("", nil, f.admin), code, raw)
		e.check(t, code == http.StatusOK && m.TenantID == f.tenant && m.IdPAlias == f.idpAlias && m.UserID != uuid.Nil,
			"`/me` is tenant `%s` via `%s` (got %d, tenant `%s`, idp `%s`)", f.tenant, f.idpAlias, code, m.TenantID, m.IdPAlias)
		e.check(t, f.placed(t, f.admin) && inviteState(t, f.tenant, f.admin) == "consumed",
			"`%s` is in `/t/%s/` and the invite is consumed", f.admin, f.tenant)
	}

	// Isolation: neither admin reaches the other tenant.
	e.h("Isolation check: tenant B cannot see or change what onboarding created for tenant A")
	for _, pair := range [][2]s4Tenant{{b, a}, {a, b}} {
		self, other := pair[0], pair[1]
		for _, try := range []struct {
			query  string
			header http.Header
		}{
			{header: http.Header{"X-Tenant-Id": {other.tenant.String()}}},
			{query: "?tenant_id=" + other.tenant.String()},
		} {
			code, m, raw := getMe(t, toks[self.tenant], try.query, try.header)
			e.exchange("Admin of "+self.domain+" names tenant "+other.domain+":", meReq(try.query, try.header, self.admin), code, raw)
			e.check(t, code == http.StatusOK && m.TenantID == self.tenant, "still tenant `%s`, never `%s`", self.tenant, other.tenant)
		}
		id := kcUserByEmail(t, self.admin)
		e.check(t, !userInGroup(t, id, kcGroupID(t, "/t/"+other.tenant.String())) && !slices.Contains(orgMembers(t, other.orgID), id),
			"`%s` is not in `/t/%s/` and not a member of organization `%s`", self.admin, other.tenant, other.alias)
		e.check(t, !slices.Contains(fedLinks(t, self.admin), other.idpAlias), "`%s` holds no link to `%s`", self.admin, other.idpAlias)
	}
	// Claiming A's domain for another tenant is 409 domain_claimed (TICK-271):
	// the body names neither A nor its organization, and the refused tenant
	// leaves no tenants row and no Keycloak organization.
	{
		key := uuid.NewString()
		refused := onboard.TenantID(operatorSub, key)
		stolen := onboardBody("admin@"+a.domain, a.domain)
		cleanupTenantOrg(t, refused, onboard.Alias(refused))
		code, resp := onboardPostRaw(t, op, key, stolen)
		e.exchange("A new tenant claims tenant A's domain:", onboardReq(key, stolen), code, resp)
		var problem struct{ Code string }
		_ = json.Unmarshal(resp, &problem)
		e.check(t, code == http.StatusConflict && problem.Code == "domain_claimed", "%d `%s` (want 409 `domain_claimed`)", code, problem.Code)
		leaks := []string{a.tenant.String(), a.alias, a.orgID, a.idpAlias}
		e.check(t, !slices.ContainsFunc(leaks, func(s string) bool { return bytes.Contains(resp, []byte(s)) }),
			"the body names neither tenant A (`%s`) nor its organization (`%s`, id `%s`) nor its IdP", a.tenant, a.alias, a.orgID)
		owner := psqlSuper(t, fmt.Sprintf("SELECT tenant_id FROM tenant_email_domain WHERE domain = '%s'", a.domain))
		e.check(t, owner == a.tenant.String(), "`%s` is still owned by tenant A (got `%s`)", a.domain, owner)
		rows := psqlSuper(t, fmt.Sprintf("SELECT count(*) FROM tenants WHERE id = '%s'", refused))
		e.check(t, rows == "0", "no `tenants` row (provisioning or otherwise) for the refused tenant `%s` (got %s)", refused, rows)
		orgs := orgsByAlias(t, onboard.Alias(refused))
		e.check(t, len(orgs) == 0, "no Keycloak organization `%s` (got %v)", onboard.Alias(refused), orgs)
		e.check(t, inviteState(t, a.tenant, a.admin) == "consumed" && psqlSuper(t,
			fmt.Sprintf("SELECT count(*) FROM pending_invite WHERE email = 'admin@%s'", a.domain)) == "1",
			"no second invite for `admin@%s`", a.domain)
	}
	// The operator route is not on the tenant ingress.
	{
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, coreURL()+"/api/v1/operator/tenants", bytes.NewReader(a.body))
		req.Header.Set("Authorization", "Bearer "+toks[b.tenant])
		req.Header.Set("Idempotency-Key", a.key)
		req.Header.Set("Content-Type", "application/json")
		code, raw := do(t, &http.Client{Timeout: 30 * time.Second}, req)
		e.exchange("Tenant B's admin replays A's onboarding on nava-core:",
			"POST /api/v1/operator/tenants (nava-core)\nAuthorization: Bearer <"+b.admin+" token>\nIdempotency-Key: "+a.key, code, raw)
		e.check(t, code == http.StatusNotImplemented || code == http.StatusForbidden || code == http.StatusNotFound,
			"refused with %d; nothing changes for tenant A (`%s`)", code, onboardCounts(t, a.alias))
	}

	// Criterion 3: a user already federated to another tenant's IdP. B's IdP
	// asserts an address in tenant C's domain; the first broker login creates
	// that user, linked to B's IdP. Onboarding C with them as admin must not
	// invite them.
	e.h("Criterion 3: inviting a user already federated to another tenant's IdP is rejected")
	cT := s4NewTenant(t)
	x := "lead-" + uuid.NewString()[:8] + "@" + cT.domain
	seedIdPUser(t, b.realm, x, x, b.pass)
	deleteUserAfter(t, x)
	xTok, page := brokeredAttempt(t, b.idpAlias, x, b.pass)
	if xTok == "" {
		e.p("Tenant B's IdP refused the first login of `%s` (page: %.200s). Linking the user to B's IdP through the admin API instead.", x, page)
		code, raw := adminRealm(t, "nava", http.MethodPost, "/users", map[string]any{"username": x, "email": x, "enabled": true})
		if code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", x, code, raw)
		}
		code, raw = adminRealm(t, "nava", http.MethodPost, "/users/"+kcUserByEmail(t, x)+"/federated-identity/"+b.idpAlias,
			map[string]any{"identityProvider": b.idpAlias, "userId": uuid.NewString(), "userName": x})
		if code != http.StatusNoContent {
			t.Fatalf("link %s to %s: %d %s", x, b.idpAlias, code, raw)
		}
	}
	e.check(t, slices.Equal(fedLinks(t, x), []string{b.idpAlias}), "precondition: `%s` exists and is federated to tenant B's IdP only (`%v`)", x, fedLinks(t, x))

	cT.admin = x
	cT.body = onboardBody(x, cT.domain)
	seedIdPUser(t, cT.realm, x, x, cT.pass)
	code, resp := onboardPost(t, op, cT.key, cT.body)
	e.exchange("Onboard tenant C naming that user as admin:", onboardReq(cT.key, cT.body), code, resp)
	e.p("Status %d. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.", code)
	pointIdPAt(t, cT.idpAlias, cT.realm)
	cT.orgID = orgsByAlias(t, cT.alias)[0]
	e.check(t, inviteState(t, cT.tenant, x) == "missing", "no invite for `%s` in tenant C (got `%s`)", x, inviteState(t, cT.tenant, x))
	cTok, cPage := brokeredAttempt(t, cT.idpAlias, x, cT.pass)
	reason := feedback(cPage)
	e.check(t, cTok == "" && strings.Contains(strings.ToLower(reason), "exists"),
		"signing in through tenant C's IdP as `%s` is refused because the account already exists (Keycloak says: %q)", x, reason)
	// Control: C's IdP itself works; its other user signs in.
	ctlTok, ctlPage := brokeredAttempt(t, cT.idpAlias, "admin@"+cT.domain, cT.pass)
	e.check(t, ctlTok != "", "control: `admin@%s` signs in through the same IdP of tenant C (%s)", cT.domain, feedback(ctlPage))
	deleteUserAfter(t, "admin@"+cT.domain)
	groupsyncOnce(t)
	id := kcUserByEmail(t, x)
	e.check(t, !userInGroup(t, id, kcGroupID(t, "/t/"+cT.tenant.String())) && !slices.Contains(orgMembers(t, cT.orgID), id),
		"after a group sync cycle `%s` is not in `/t/%s/` and not a member of `%s`", x, cT.tenant, cT.alias)
	e.check(t, !slices.Contains(fedLinks(t, x), cT.idpAlias), "`%s` holds no link to tenant C's IdP", x)
	if xTok != "" {
		code, m, raw := getMe(t, xTok, "", http.Header{"X-Tenant-Id": {cT.tenant.String()}})
		e.exchange("`"+x+"` (signed in through B's IdP) names tenant C:", meReq("", http.Header{"X-Tenant-Id": {cT.tenant.String()}}, x), code, raw)
		e.check(t, m.TenantID != cT.tenant, "`/me` never resolves tenant C (got %d, tenant `%s`)", code, m.TenantID)
	}
}
