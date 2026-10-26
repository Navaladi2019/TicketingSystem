package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TestInviteUserNewEmailWritesInviteAndNoUser: the invite of an unknown email
// in an owned domain records a pending invite and calls no write endpoint of
// Keycloak (TICK-252 condition 2a).
func TestInviteUserNewEmailWritesInviteAndNoUser(t *testing.T) {
	t.Parallel()
	var writes atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, dirTenantA.String(), rootPath(dirTenantA))
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `[]`) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writes.Add(1)
		w.WriteHeader(http.StatusTeapot)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	rec := &recordedInvites{}
	d, _ := directoryFor(t, srv.URL, withInvites(rec), func(c *DirectoryConfig) { c.InviteTTL = time.Hour })
	before := time.Now()
	if err := d.InviteUser(t.Context(), dirTenantA, "  Admin@Tenant-A.test "); err != nil {
		t.Fatal(err)
	}
	if writes.Load() != 0 {
		t.Errorf("%d unexpected Keycloak calls", writes.Load())
	}
	if len(rec.rows) != 1 || rec.rows[0].tenant != dirTenantA || rec.rows[0].email != "admin@tenant-a.test" {
		t.Fatalf("recorded = %+v", rec.rows)
	}
	if exp := rec.rows[0].expires; exp.Before(before.Add(time.Hour)) || exp.After(time.Now().Add(time.Hour)) {
		t.Errorf("expires_at = %v, want about now+1h", exp)
	}
}

func TestInviteUserForeignDomainWritesNothing(t *testing.T) {
	t.Parallel()
	srv := fakeKeycloak(t)
	rec := &recordedInvites{}
	d, _ := directoryFor(t, srv.URL, withInvites(rec))
	if err := d.InviteUser(t.Context(), dirTenantA, "squatter@tenant-b.test"); !errors.Is(err, identity.ErrInviteRefused) {
		t.Fatalf("err = %v, want ErrInviteRefused", err)
	}
	if len(rec.rows) != 0 {
		t.Errorf("recorded = %+v", rec.rows)
	}
}

func TestInviteUserWithoutWriterFailsClosed(t *testing.T) {
	t.Parallel()
	srv := fakeKeycloak(t)
	d, _ := directoryFor(t, srv.URL)
	if err := d.InviteUser(t.Context(), dirTenantA, "admin@tenant-a.test"); err == nil {
		t.Fatal("invite succeeded with no invite writer")
	}
}

func TestNewDirectoryRefusesNegativeInviteTTL(t *testing.T) {
	t.Parallel()
	cfg := DirectoryConfig{BaseURL: "http://h", Runtime: ClientCredentials{ClientID: "a", ClientSecret: "b"}, Registry: noRegistry{}, InviteTTL: -1}
	if _, err := NewDirectory(cfg, nil); err == nil {
		t.Fatal("accepted")
	}
}

// invitee describes the Keycloak state PlaceInvitee sees.
type invitee struct {
	email   string
	enabled bool
	links   string // JSON array
	groups  string // JSON array
	orgs    string // JSON array
}

// orgA is the Keycloak organization id the tenant is bound to.
var orgA = []string{"org-a"} //nolint:gochecknoglobals // read-only test fixture

func goodInvitee() invitee {
	return invitee{
		email: "admin@tenant-a.test", enabled: true,
		links:  `[{"identityProvider":"idp-a","userId":"x"}]`,
		groups: `[]`,
		orgs:   `[{"id":"org-a"}]`,
	}
}

func inviteeServer(t *testing.T, u invitee, placed *atomic.Bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/group-by-path/t/"+dirTenantA.String(), func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"id":"g-root","name":%q,"path":%q}`, dirTenantA.String(), rootPath(dirTenantA))
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `[{"id":"u1","email":%q,"enabled":%v}]`, u.email, u.enabled)
	})
	mux.HandleFunc("GET /admin/realms/nava/users/u1/federated-identity", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, u.links) })
	mux.HandleFunc("GET /admin/realms/nava/users/u1/groups", func(w http.ResponseWriter, r *http.Request) {
		if f := r.URL.Query().Get("first"); f != "" && f != "0" {
			fmt.Fprint(w, `[]`)
			return
		}
		fmt.Fprint(w, u.groups)
	})
	mux.HandleFunc("GET /admin/realms/nava/organizations/members/u1/organizations", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, u.orgs) })
	mux.HandleFunc("PUT /admin/realms/nava/users/u1/groups/g-root", func(w http.ResponseWriter, _ *http.Request) {
		placed.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPlaceInvitee(t *testing.T) {
	t.Parallel()
	other := uuid.MustParse("00000000-0000-4000-8000-0000000000b2")
	bad := func(f func(*invitee)) invitee { u := goodInvitee(); f(&u); return u }
	idpA := []string{"idp-a"}
	tests := []struct {
		name  string
		u     invitee
		bound []string
		want  bool
	}{
		{"good", goodInvitee(), idpA, true},
		{"no bound idps", goodInvitee(), nil, false},
		{"link to an idp of another tenant", bad(func(u *invitee) { u.links = `[{"identityProvider":"idp-b","userId":"x"}]` }), idpA, false},
		{"one good and one foreign link", bad(func(u *invitee) {
			u.links = `[{"identityProvider":"idp-a","userId":"x"},{"identityProvider":"idp-b","userId":"y"}]`
		}), idpA, false},
		{"local user without a link", bad(func(u *invitee) { u.links = `[]` }), idpA, false},
		{"disabled", bad(func(u *invitee) { u.enabled = false }), idpA, false},
		{"different email on the user", bad(func(u *invitee) { u.email = "other@tenant-a.test" }), idpA, false},
		{"not an org member", bad(func(u *invitee) { u.orgs = `[]` }), idpA, false},
		{"member of another org", bad(func(u *invitee) { u.orgs = `[{"id":"org-b"}]` }), idpA, false},
		{"member of two orgs", bad(func(u *invitee) {
			u.orgs = `[{"id":"org-a"},{"id":"org-b"}]`
		}), idpA, false},
		{"in another tenant's group", bad(func(u *invitee) { u.groups = fmt.Sprintf(`[{"id":"gb","path":%q}]`, rootPath(other)) }), idpA, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var placed, consumed atomic.Bool
			srv := inviteeServer(t, tc.u, &placed)
			d, _ := directoryFor(t, srv.URL)
			got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", tc.bound, orgA, func(context.Context) (bool, error) {
				consumed.Store(true)
				return true, nil
			})
			if err != nil || got != tc.want {
				t.Fatalf("placed = %v, %v; want %v", got, err, tc.want)
			}
			if placed.Load() != tc.want || consumed.Load() != tc.want {
				t.Errorf("group written = %v, invite consumed = %v; want both %v", placed.Load(), consumed.Load(), tc.want)
			}
		})
	}
}

func TestPlaceInviteeNoUserYet(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/users", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `[]`) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL)
	got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", []string{"idp-a"}, orgA, func(context.Context) (bool, error) {
		t.Error("consumed an invite with no user")
		return true, nil
	})
	if got || err != nil {
		t.Fatalf("= %v, %v", got, err)
	}
}

// An invite that is expired or already consumed (consume reports false) places
// nobody and writes no group.
func TestPlaceInviteeSpentInvitePlacesNobody(t *testing.T) {
	t.Parallel()
	var placed atomic.Bool
	srv := inviteeServer(t, goodInvitee(), &placed)
	d, _ := directoryFor(t, srv.URL)
	got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", []string{"idp-a"}, orgA, func(context.Context) (bool, error) { return false, nil })
	if got || err != nil || placed.Load() {
		t.Fatalf("placed = %v, %v, group written %v", got, err, placed.Load())
	}
}

func TestPlaceInviteeRefusesBadInput(t *testing.T) {
	t.Parallel()
	d, _ := directoryFor(t, "http://127.0.0.1:1")
	ok := func(context.Context) (bool, error) { return true, nil }
	if _, err := d.PlaceInvitee(t.Context(), uuid.Nil, "a@tenant-a.test", []string{"idp-a"}, orgA, ok); err == nil {
		t.Error("nil tenant accepted")
	}
	if _, err := d.PlaceInvitee(t.Context(), dirTenantA, "not an email", []string{"idp-a"}, orgA, ok); !errors.Is(err, identity.ErrInvalidInput) {
		t.Errorf("bad email err = %v", err)
	}
}

type consumedInvites struct{}

func (consumedInvites) RecordInvite(context.Context, uuid.UUID, string, time.Time) error {
	return ErrInviteConsumed
}

// A writer that reports a consumed invite makes InviteUser fail, never succeed.
func TestInviteUserConsumedInviteIsAnError(t *testing.T) {
	t.Parallel()
	srv := fakeKeycloak(t)
	d, _ := directoryFor(t, srv.URL, func(c *DirectoryConfig) { c.Invites = consumedInvites{} })
	if err := d.InviteUser(t.Context(), dirTenantA, "admin@tenant-a.test"); !errors.Is(err, ErrInviteConsumed) {
		t.Fatalf("err = %v, want ErrInviteConsumed", err)
	}
}

func TestPlaceInviteeNeedsABoundOrgID(t *testing.T) {
	t.Parallel()
	var placed atomic.Bool
	srv := inviteeServer(t, goodInvitee(), &placed)
	d, _ := directoryFor(t, srv.URL)
	got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", []string{"idp-a"}, nil, func(context.Context) (bool, error) { return true, nil })
	if got || err != nil || placed.Load() {
		t.Fatalf("placed = %v, %v, group written %v", got, err, placed.Load())
	}
}

// A user already in this tenant's group (a cycle placed them but did not
// commit) has the invite consumed and no second group write.
func TestPlaceInviteeAlreadyInTenantGroupConsumesWithoutWrite(t *testing.T) {
	t.Parallel()
	u := goodInvitee()
	u.groups = fmt.Sprintf(`[{"id":"g-root","path":%q}]`, rootPath(dirTenantA))
	var placed, consumed atomic.Bool
	srv := inviteeServer(t, u, &placed)
	d, _ := directoryFor(t, srv.URL)
	got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", []string{"idp-a"}, orgA, func(context.Context) (bool, error) {
		consumed.Store(true)
		return true, nil
	})
	if err != nil || !got || !consumed.Load() || placed.Load() {
		t.Fatalf("placed=%v err=%v consumed=%v group written=%v; want true, nil, true, false", got, err, consumed.Load(), placed.Load())
	}
}

// ... but if another condition fails, the invite is not consumed.
func TestPlaceInviteeAlreadyInTenantGroupStillNeedsTheOtherConditions(t *testing.T) {
	t.Parallel()
	u := goodInvitee()
	u.groups = fmt.Sprintf(`[{"id":"g-root","path":%q}]`, rootPath(dirTenantA))
	u.links = `[{"identityProvider":"idp-b","userId":"x"}]`
	var placed atomic.Bool
	srv := inviteeServer(t, u, &placed)
	d, _ := directoryFor(t, srv.URL)
	got, err := d.PlaceInvitee(t.Context(), dirTenantA, "admin@tenant-a.test", []string{"idp-a"}, orgA, func(context.Context) (bool, error) {
		t.Error("consumed")
		return true, nil
	})
	if got || err != nil {
		t.Fatalf("= %v, %v", got, err)
	}
}
