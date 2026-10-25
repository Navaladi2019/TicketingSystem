package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const secretEmail = "carol.private@tenant-a.test"

//nolint:gochecknoglobals // read-only test fixture
var tenantA = uuid.MustParse("00000000-0000-4000-8000-0000000000a1")

type noRegistry struct{}

func (noRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", ErrRegistryNotFound
}

func (noRegistry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", ErrRegistryNotFound
}

func (noRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) { return nil, nil }

func directoryFor(t *testing.T, baseURL string) (*Directory, *tracetest.InMemoryExporter) {
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
	d, err := NewDirectory(DirectoryConfig{
		BaseURL: baseURL, Runtime: ClientCredentials{ClientID: "nava-platform", ClientSecret: "s"}, Registry: noRegistry{},
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	return d, spans
}

// fakeKeycloak answers exactly the calls InviteUser makes for a new email.
func fakeKeycloak(t *testing.T) *httptest.Server {
	t.Helper()
	root := rootPath(tenantA)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+tenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, tenantA.String(), root)
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
	d, spans := directoryFor(t, srv.URL)
	if err := d.InviteUser(t.Context(), tenantA, secretEmail); err != nil {
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

// InviteUser against a closed port returns an error with no email and no query
// string, and nothing it logs from it can carry them either.
func TestInviteUserClosedPortErrorHasNoEmailOrQuery(t *testing.T) {
	t.Parallel()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	d, _ := directoryFor(t, "http://"+addr)
	err = d.InviteUser(t.Context(), tenantA, secretEmail)
	if err == nil {
		t.Fatal("InviteUser to a closed port succeeded")
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
	err := d.InviteUser(t.Context(), tenantA, secretEmail)
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
		"invite bad email":  d.InviteUser(ctx, tenantA, "not an email"),
		"invite display":    d.InviteUser(ctx, tenantA, "Eve <eve@b.test>"),
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
	if err := d.DisableUser(ctx, tenantA, uuid.New()); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("disable unknown principal: err = %v, want ErrNotFound", err)
	}
	if err := d.SetGroupMembers(ctx, tenantA, uuid.New(), nil); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("set members unknown group: err = %v, want ErrNotFound", err)
	}
	// Onboarding is off on a runtime-only Directory.
	if _, err := d.CreateTenantOrg(ctx, tenantA, "tenant-a", nil); !errors.Is(err, ErrOnboardingDisabled) {
		t.Errorf("CreateTenantOrg without onboarding: err = %v", err)
	}
}

func TestSubtreeRules(t *testing.T) {
	t.Parallel()
	other := uuid.MustParse("00000000-0000-4000-8000-0000000000b1")
	for path, want := range map[string]bool{
		"/t/" + tenantA.String():             true,
		"/t/" + tenantA.String() + "/ou/eng": true,
		"/t/" + other.String():               false,
		"/t/" + other.String() + "/ou/eng":   false,
		"/t/" + tenantA.String() + "x":       false, // prefix of a longer id
		"/t":                                 false,
		"/other/t/" + tenantA.String():       false,
	} {
		if got := inTenant(tenantA, path); got != want {
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
