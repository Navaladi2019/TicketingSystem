package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const secretEmail = "carol.private@tenant-a.test"

//nolint:gochecknoglobals // read-only test fixture
var dirTenantA = uuid.MustParse("00000000-0000-4000-8000-0000000000a1")

type noRegistry struct{}

func (noRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", ErrRegistryNotFound
}

func (noRegistry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", ErrRegistryNotFound
}

func (noRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) {
	return []string{"idp-a"}, nil
}

func (noRegistry) TenantOwnsEmailDomain(_ context.Context, _ uuid.UUID, domain string) (bool, error) {
	return domain == "tenant-a.test", nil
}

type userRegistry struct{ noRegistry }

func (userRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "u-1", nil
}

func withUser(c *DirectoryConfig) { c.Registry = userRegistry{} }

func withOnboarding(c *DirectoryConfig) {
	c.Onboarding = &ClientCredentials{ClientID: "nava-onboarding", ClientSecret: "s"}
	c.IdPs = func(context.Context, uuid.UUID) (IdPSpec, error) {
		return IdPSpec{ProviderID: "oidc", Config: map[string]string{"clientId": "x"}}, nil
	}
}

// recordedInvites is an InviteWriter that keeps what it was asked to write.
type recordedInvites struct {
	mu   sync.Mutex
	rows []recordedInvite
}

type recordedInvite struct {
	tenant  uuid.UUID
	email   string
	expires time.Time
}

func (r *recordedInvites) RecordInvite(_ context.Context, tenant uuid.UUID, email string, exp time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, recordedInvite{tenant, email, exp})
	return nil
}

func withInvites(r *recordedInvites) func(*DirectoryConfig) {
	return func(c *DirectoryConfig) { c.Invites = r }
}

func directoryFor(t *testing.T, baseURL string, opts ...func(*DirectoryConfig)) (*Directory, *tracetest.InMemoryExporter) {
	t.Helper()
	spans := tracetest.NewInMemoryExporter()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry, telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		t.Fatal(err)
	}
	dc := DirectoryConfig{
		BaseURL: baseURL, Runtime: ClientCredentials{ClientID: "nava-platform", ClientSecret: "s"}, Registry: noRegistry{},
	}
	for _, o := range opts {
		o(&dc)
	}
	d, err := NewDirectory(dc, out)
	if err != nil {
		t.Fatal(err)
	}
	return d, spans
}

// fakeKeycloak answers exactly the calls InviteUser makes for a new email.
func fakeKeycloak(t *testing.T) *httptest.Server {
	t.Helper()
	root := rootPath(dirTenantA)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, dirTenantA.String(), root)
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `[]`) })
	mux.HandleFunc("POST /admin/realms/nava/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://x/admin/realms/nava/users/u-new")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("PUT /admin/realms/nava/users/u-new/groups/g-root", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// After InviteUser no exported span carries the email or a Keycloak user id in
// its URL attributes (TICK-94 item 2).
func TestInviteUserSpansCarryNoEmailOrUserID(t *testing.T) {
	t.Parallel()
	srv := fakeKeycloak(t)
	d, spans := directoryFor(t, srv.URL, withInvites(&recordedInvites{}))
	if err := d.InviteUser(t.Context(), dirTenantA, secretEmail); err != nil {
		t.Fatal(err)
	}
	got := spans.GetSpans()
	if len(got) == 0 {
		t.Fatal("no spans exported")
	}
	for _, s := range got {
		for _, a := range s.Attributes {
			v := a.Value.String()
			if strings.Contains(v, secretEmail) || strings.Contains(v, "u-new") || strings.Contains(v, "%40") {
				t.Errorf("span %q attribute %s = %q leaks an email or user id", s.Name, a.Key, v)
			}
			if (a.Key == "url.query" || a.Key == "http.url") && v != "" {
				t.Errorf("span %q carries %s = %q", s.Name, a.Key, v)
			}
		}
		if strings.Contains(s.Name, secretEmail) {
			t.Errorf("span name %q leaks the email", s.Name)
		}
	}
}

// InviteUser against a connection that drops returns an error with no email
// and no query string. The fake answers the token and group calls, then closes
// the connection on GET /users?email=..., so the request that carries the
// email is the one that fails (and the test asserts it was received).
func TestInviteUserDroppedConnectionErrorHasNoEmailOrQuery(t *testing.T) {
	t.Parallel()
	var gotLookup atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, dirTenantA.String(), rootPath(dirTenantA))
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("email") == secretEmail {
			gotLookup.Store(true)
		}
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL)
	err := d.InviteUser(t.Context(), dirTenantA, secretEmail)
	if err == nil {
		t.Fatal("InviteUser over a dropped connection succeeded")
	}
	if !gotLookup.Load() {
		t.Fatal("the server never received GET /users?email=...: the test does not reach the request that carries the email")
	}
	for _, bad := range []string{secretEmail, "carol", "%40", "?", "email=", "exact="} {
		if strings.Contains(err.Error(), bad) {
			t.Errorf("error %q contains %q", err, bad)
		}
	}
}

// A Keycloak error body that quotes the email never reaches the error.
func TestStatusErrorDropsResponseBody(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"errorMessage":"User exists with same email %s"}`, secretEmail)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL)
	err := d.InviteUser(t.Context(), dirTenantA, secretEmail)
	if !errors.Is(err, identity.ErrAlreadyExists) {
		t.Fatalf("err = %v, want ErrAlreadyExists", err)
	}
	if strings.Contains(err.Error(), "carol") {
		t.Errorf("error %q quotes the response body", err)
	}
}

func TestDirectoryRefusesBadInput(t *testing.T) {
	t.Parallel()
	d, _ := directoryFor(t, "http://127.0.0.1:1")
	ctx := t.Context()
	for name, err := range map[string]error{
		"invite nil tenant": d.InviteUser(ctx, uuid.Nil, "a@b.test"),
		"invite bad email":  d.InviteUser(ctx, dirTenantA, "not an email"),
		"invite display":    d.InviteUser(ctx, dirTenantA, "Eve <eve@b.test>"),
		"disable nil":       d.DisableUser(ctx, uuid.Nil, uuid.New()),
		"set members nil":   d.SetGroupMembers(ctx, uuid.Nil, uuid.New(), nil),
	} {
		if !errors.Is(err, identity.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, _, err := d.ChangesSince(ctx, uuid.Nil, ""); !errors.Is(err, identity.ErrInvalidInput) {
		t.Errorf("changes nil tenant: err = %v", err)
	}
	// Unknown ids from the registry are not found, with no HTTP call (port 1
	// is closed, so a call would be a different error).
	if err := d.DisableUser(ctx, dirTenantA, uuid.New()); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("disable unknown principal: err = %v, want ErrNotFound", err)
	}
	if err := d.SetGroupMembers(ctx, dirTenantA, uuid.New(), nil); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("set members unknown group: err = %v, want ErrNotFound", err)
	}
	// Onboarding is off on a runtime-only Directory.
	if _, err := d.CreateTenantOrg(ctx, dirTenantA, "tenant-a", nil); !errors.Is(err, ErrOnboardingDisabled) {
		t.Errorf("CreateTenantOrg without onboarding: err = %v", err)
	}
}

func TestSubtreeRules(t *testing.T) {
	t.Parallel()
	other := uuid.MustParse("00000000-0000-4000-8000-0000000000b1")
	for path, want := range map[string]bool{
		"/t/" + dirTenantA.String():             true,
		"/t/" + dirTenantA.String() + "/ou/eng": true,
		"/t/" + other.String():                  false,
		"/t/" + other.String() + "/ou/eng":      false,
		"/t/" + dirTenantA.String() + "x":       false, // prefix of a longer id
		"/t":                                    false,
		"/other/t/" + dirTenantA.String():       false,
	} {
		if got := inTenant(dirTenantA, path); got != want {
			t.Errorf("inTenant(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestDigestIsStableAndSensitive(t *testing.T) {
	t.Parallel()
	a := identity.DirectoryChanges{Users: []identity.DirectoryUser{{KeycloakUserID: "u1", Email: "a@b.test", Enabled: true}}}
	b := a
	b.Users = []identity.DirectoryUser{{KeycloakUserID: "u1", Email: "a@b.test", Enabled: false}}
	da1, _ := digestOf(a)
	da2, _ := digestOf(a)
	db, _ := digestOf(b)
	if da1 != da2 || da1 == db {
		t.Errorf("digests: a=%s a=%s b=%s", da1, da2, db)
	}
}

func TestNewDirectoryValidates(t *testing.T) {
	t.Parallel()
	cfg, _ := config.Load(func(string) (string, bool) { return "", false })
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, _ := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	run := ClientCredentials{ClientID: "r", ClientSecret: "s"}
	for name, c := range map[string]DirectoryConfig{
		"no base":          {Runtime: run, Registry: noRegistry{}},
		"query in base":    {BaseURL: "http://h?x=1", Runtime: run, Registry: noRegistry{}},
		"no runtime":       {BaseURL: "http://h", Registry: noRegistry{}},
		"no registry":      {BaseURL: "http://h", Runtime: run},
		"same account":     {BaseURL: "http://h", Runtime: run, Registry: noRegistry{}, Onboarding: &run},
		"empty onboarding": {BaseURL: "http://h", Runtime: run, Registry: noRegistry{}, Onboarding: &ClientCredentials{}},
	} {
		if _, err := NewDirectory(c, out); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewDirectory(DirectoryConfig{BaseURL: "http://h", Runtime: run, Registry: noRegistry{}}, nil); err == nil {
		t.Error("nil outbound client accepted")
	}
}

// existingUserServer serves one existing user with the given groups and
// federated links, and records whether anything was written.
func existingUserServer(t *testing.T, groups, links string) (*httptest.Server, *atomic.Bool) {
	t.Helper()
	var wrote atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, dirTenantA.String(), rootPath(dirTenantA))
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `[{"id":"u-1","email":%q}]`, secretEmail)
	})
	mux.HandleFunc("GET /admin/realms/nava/users/u-1/groups", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, groups) })
	mux.HandleFunc("GET /admin/realms/nava/users/u-1/federated-identity", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, links)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			wrote.Store(true)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &wrote
}

func TestInviteExistingUserRules(t *testing.T) {
	t.Parallel()
	inA := fmt.Sprintf(`[{"id":"g","path":"/t/%s/ou/eng"}]`, dirTenantA)
	inB := `[{"id":"g","path":"/t/0b000000-0000-4000-8000-00000000000b"}]`
	for _, c := range []struct {
		name, groups, links string
		want                error
	}{
		{"already in the tenant", inA, `[]`, nil},
		{"local user in no tenant (operator)", `[]`, `[]`, identity.ErrInviteRefused},
		{"local user of another tenant", inB, `[]`, identity.ErrInviteRefused},
		{"federated to the tenant, in no group", `[]`, `[{"identityProvider":"idp-a"}]`, nil},
		{"federated to another tenant's IdP", `[]`, `[{"identityProvider":"idp-b"}]`, identity.ErrInviteRefused},
		{"federated to both", `[]`, `[{"identityProvider":"idp-a"},{"identityProvider":"idp-b"}]`, identity.ErrInviteRefused},
		{"federated to the tenant but in another tenant's group", inB, `[{"identityProvider":"idp-a"}]`, identity.ErrInviteRefused},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, wrote := existingUserServer(t, c.groups, c.links)
			d, _ := directoryFor(t, srv.URL)
			err := d.InviteUser(t.Context(), dirTenantA, secretEmail)
			if c.want == nil {
				if err != nil || !wrote.Load() {
					t.Fatalf("err = %v, wrote = %v, want accepted", err, wrote.Load())
				}
				return
			}
			if !errors.Is(err, c.want) || wrote.Load() {
				t.Fatalf("err = %v, wrote = %v, want %v and no write", err, wrote.Load(), c.want)
			}
		})
	}
}

// An address in a domain the tenant does not own creates no user.
func TestInviteUnownedDomainIsRefusedWithoutWrite(t *testing.T) {
	t.Parallel()
	var wrote atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","path":%q}`, rootPath(dirTenantA))
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `[]`) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { wrote.Store(true) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL)
	if err := d.InviteUser(t.Context(), dirTenantA, "squatter@tenant-b.test"); !errors.Is(err, identity.ErrInviteRefused) {
		t.Fatalf("err = %v, want ErrInviteRefused", err)
	}
	if wrote.Load() {
		t.Error("a write was made")
	}
}

func TestDisableUserRefusesAnyGroupOutsideTheTenant(t *testing.T) {
	t.Parallel()
	for name, path := range map[string]string{
		"other tenant":        "/t/0b000000-0000-4000-8000-00000000000b/ou/x",
		"top-level t":         "/t",
		"other tenant's root": "/t/0b000000-0000-4000-8000-00000000000b",
	} {
		t.Run(name, func(t *testing.T) {
			groups := fmt.Sprintf(`[{"id":"a","path":"/t/%s"},{"id":"b","path":%q}]`, dirTenantA, path)
			srv, wrote := existingUserServer(t, groups, `[]`)
			d, _ := directoryFor(t, srv.URL, withUser)
			if err := d.DisableUser(t.Context(), dirTenantA, uuid.New()); !errors.Is(err, identity.ErrInviteRefused) {
				t.Fatalf("err = %v, want ErrInviteRefused", err)
			}
			if wrote.Load() {
				t.Error("a write was made")
			}
		})
	}
}

func TestIdPConfigAllowList(t *testing.T) {
	t.Parallel()
	ok := IdPSpec{ProviderID: "oidc", Config: map[string]string{"clientId": "x", "tokenUrl": "https://i/t"}}
	if err := checkIdPConfig(ok); err != nil {
		t.Errorf("allowed config refused: %v", err)
	}
	for _, key := range []string{"kc.org.domain", "kc.org.broker.redirect.mode.email-matches", "hideOnLoginPage", "trustEmail", "postBrokerLoginFlowAlias", "authenticateByDefault"} {
		bad := IdPSpec{ProviderID: "oidc", Config: map[string]string{"clientId": "x", key: "v"}}
		if err := checkIdPConfig(bad); !errors.Is(err, identity.ErrInvalidInput) {
			t.Errorf("key %q: err = %v, want ErrInvalidInput", key, err)
		}
	}
	if err := checkIdPConfig(IdPSpec{ProviderID: "saml", Config: map[string]string{"clientId": "x"}}); err == nil {
		t.Error("an OIDC key is allowed on a SAML IdP")
	}
}

// If the IdP create fails other than with a 409 the server-derived alias is
// deleted (the create may have landed though the reply was lost); on a 409 it
// is not, because the IdP is not ours. The organization is removed either way.
func TestCreateTenantOrgRollsBackIdP(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name        string
		idpExists   bool // the derived alias exists before the call
		status      int  // answer to the IdP create
		wantErr     error
		wantIdPDel  bool
		wantOrgMade bool
		wantOrgGone bool
	}{
		{name: "server error on a fresh alias", status: http.StatusInternalServerError, wantIdPDel: true, wantOrgMade: true, wantOrgGone: true},
		{name: "conflict on a fresh alias", status: http.StatusConflict, wantOrgMade: true, wantOrgGone: true},
		// Keycloak answers 400, not 409, for a bad config on an existing
		// alias: the live IdP must survive, and nothing is written.
		{name: "existing idp, create returns 400", idpExists: true, status: http.StatusBadRequest, wantErr: identity.ErrAlreadyExists},
	} {
		t.Run(c.name, func(t *testing.T) {
			var delIdP, madeOrg, delOrg atomic.Bool
			var idpMade atomic.Bool
			mux := http.NewServeMux()
			mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
			})
			idpPath := "/admin/realms/nava/identity-provider/instances/" + TenantIdPAlias(dirTenantA)
			mux.HandleFunc("GET "+idpPath, func(w http.ResponseWriter, _ *http.Request) {
				if c.idpExists || idpMade.Load() {
					fmt.Fprintf(w, `{"alias":%q,"organizationLinks":[]}`, TenantIdPAlias(dirTenantA))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			})
			mux.HandleFunc("POST /admin/realms/nava/organizations", func(w http.ResponseWriter, _ *http.Request) {
				madeOrg.Store(true)
				w.Header().Set("Location", "http://x/admin/realms/nava/organizations/org-1")
				w.WriteHeader(http.StatusCreated)
			})
			mux.HandleFunc("POST /admin/realms/nava/identity-provider/instances", func(w http.ResponseWriter, _ *http.Request) {
				idpMade.Store(c.status == http.StatusInternalServerError) // the create landed, the reply failed
				w.WriteHeader(c.status)
			})
			mux.HandleFunc("DELETE "+idpPath, func(w http.ResponseWriter, _ *http.Request) {
				delIdP.Store(true)
				w.WriteHeader(http.StatusNoContent)
			})
			mux.HandleFunc("DELETE /admin/realms/nava/organizations/org-1", func(w http.ResponseWriter, _ *http.Request) {
				delOrg.Store(true)
				w.WriteHeader(http.StatusNoContent)
			})
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			d, _ := directoryFor(t, srv.URL, withOnboarding)
			_, err := d.CreateTenantOrg(t.Context(), dirTenantA, "tenant-a", nil)
			if err == nil {
				t.Fatal("CreateTenantOrg succeeded")
			}
			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}
			if delIdP.Load() != c.wantIdPDel || madeOrg.Load() != c.wantOrgMade || delOrg.Load() != c.wantOrgGone {
				t.Errorf("idp deleted %v (want %v), org created %v (want %v), org deleted %v (want %v)",
					delIdP.Load(), c.wantIdPDel, madeOrg.Load(), c.wantOrgMade, delOrg.Load(), c.wantOrgGone)
			}
		})
	}
}

// The IdP that the rollback finds linked to another organization is never
// deleted, whatever the create said.
func TestCreateTenantOrgRollbackSparesAnIdPLinkedElsewhere(t *testing.T) {
	t.Parallel()
	var delIdP atomic.Bool
	var reads atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	idpPath := "/admin/realms/nava/identity-provider/instances/" + TenantIdPAlias(dirTenantA)
	mux.HandleFunc("GET "+idpPath, func(w http.ResponseWriter, _ *http.Request) {
		// Absent at the pre-check; by rollback time it is linked elsewhere.
		if reads.Add(1) == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, `{"organizationLinks":[{"organizationId":"someone-else"}]}`)
	})
	mux.HandleFunc("POST /admin/realms/nava/organizations", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://x/admin/realms/nava/organizations/org-1")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("POST /admin/realms/nava/identity-provider/instances", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("DELETE "+idpPath, func(w http.ResponseWriter, _ *http.Request) { delIdP.Store(true) })
	mux.HandleFunc("DELETE /admin/realms/nava/organizations/org-1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL, withOnboarding)
	if _, err := d.CreateTenantOrg(t.Context(), dirTenantA, "tenant-a", nil); err == nil {
		t.Fatal("CreateTenantOrg succeeded")
	}
	if delIdP.Load() {
		t.Error("an IdP linked to another organization was deleted")
	}
}
