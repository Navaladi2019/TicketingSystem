package keycloaktest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Vanish is a fake Keycloak admin API for group sync tests (TICK-1113): per
// tenant one group and two users, numbered 1 and 2. A test can make a listed
// user answer 404 on its federated-identity read and a status of its choice on
// its own record, as a user deleted during a snapshot does. It lives here
// because only this tree may know the admin API.
type Vanish struct {
	mu       sync.Mutex
	tenants  []uuid.UUID
	fedGone  map[string]bool
	userCode map[string]int
	listed   map[string]bool
	userReqs []string
	faults   map[uuid.UUID]Fault
	dir      *keycloak.Directory
}

// Fault is something other than Keycloak answering for a tenant (TICK-1141):
// a proxy's 404 page, or a 200 that is not the group asked for.
type Fault int

const (
	// NoFault is: Keycloak answers as usual.
	NoFault Fault = iota
	// ProxyUsers404 is: both user reads (federated identities and the user)
	// answer a 404 with an HTML body, as a proxy does.
	ProxyUsers404
	// ProxyRoot404 is: the root group read answers the same HTML 404.
	ProxyRoot404
	// ProxyRootEmpty is: the root group read answers 200 with {}.
	ProxyRootEmpty
	// KeycloakRootGone is: the root group read answers Keycloak's own 404,
	// {"error":"Group path does not exist"}: the group was deleted.
	KeycloakRootGone
)

const proxyHTML404 = `<html>not found</html>`

// SetFault switches the fault for one tenant; NoFault switches it off.
func (v *Vanish) SetFault(tenant uuid.UUID, f Fault) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.faults[tenant] = f
}

func (v *Vanish) faultOfUser(id string) Fault {
	for t, f := range v.faults {
		if strings.HasPrefix(id, "kc-"+t.String()+"-") {
			return f
		}
	}
	return NoFault
}

// NewVanish starts the server, closed with the test, and builds a real
// keycloak.Directory on it. Every user starts listed and present.
func NewVanish(t *testing.T, tenants ...uuid.UUID) *Vanish {
	t.Helper()
	v := &Vanish{tenants: tenants, fedGone: map[string]bool{}, userCode: map[string]int{}, listed: map[string]bool{}, faults: map[uuid.UUID]Fault{}}
	for _, tn := range tenants {
		v.listed[v.User(tn, 1)], v.listed[v.User(tn, 2)] = true, true
	}
	srv := httptest.NewServer(http.HandlerFunc(v.handle))
	t.Cleanup(srv.Close)
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		t.Fatal(err)
	}
	v.dir, err = keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL: srv.URL, Runtime: keycloak.ClientCredentials{ClientID: "nava-groupsync", ClientSecret: "s"},
		Registry: noRegistry{}, Logger: slog.New(slog.DiscardHandler),
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// Directory is the real adapter reading this fake.
func (v *Vanish) Directory() identity.DirectoryAdmin { return v.dir }

// User is the Keycloak id of user n (1 or 2) of the tenant.
func (v *Vanish) User(tenant uuid.UUID, n int) string { return fmt.Sprintf("kc-%s-%d", tenant, n) }

// Delete makes the user's federated-identity read answer 404 and its own
// record answer userStatus, while the user stays in the group's member list:
// the deletion happened after the list was read.
func (v *Vanish) Delete(id string, userStatus int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.fedGone[id], v.userCode[id] = true, userStatus
}

// Unlist removes the user from the group's member list.
func (v *Vanish) Unlist(id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.listed[id] = false
}

// CheckUserReads fails the test for a /users/ request about a user that no
// member list of the tenants named.
func (v *Vanish) CheckUserReads(t *testing.T) {
	t.Helper()
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, r := range v.userReqs {
		id := strings.TrimSuffix(r, "/federated-identity")
		if _, known := v.listed[id]; !known {
			t.Errorf("a /users/ request for %q, which no member list named", r)
		}
	}
}

func (v *Vanish) members(tenant uuid.UUID) string {
	var us []string
	for n := 1; n <= 2; n++ {
		if id := v.User(tenant, n); v.listed[id] {
			us = append(us, fmt.Sprintf(`{"id":%q,"email":"u%d@%s.test","enabled":true}`, id, n, tenant))
		}
	}
	return "[" + strings.Join(us, ",") + "]"
}

func (v *Vanish) handle(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock()
	defer v.mu.Unlock()
	p := r.URL.Path
	if strings.HasSuffix(p, "/protocol/openid-connect/token") {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
		return
	}
	p = strings.TrimPrefix(p, "/admin/realms/nava")
	for _, t := range v.tenants {
		root, grp := "root-"+t.String(), "grp-"+t.String()
		switch p {
		case "/group-by-path/t/" + t.String():
			switch v.faults[t] {
			case ProxyRoot404:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, proxyHTML404)
				return
			case ProxyRootEmpty:
				fmt.Fprint(w, `{}`)
				return
			case KeycloakRootGone:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"error":"Group path does not exist"}`)
				return
			}
			fmt.Fprintf(w, `{"id":%q,"name":"r","path":"/t/%s","subGroupCount":1}`, root, t)
			return
		case "/groups/" + root + "/children":
			fmt.Fprintf(w, `[{"id":%q,"name":"agents","path":"/t/%s/agents","subGroupCount":0}]`, grp, t)
			return
		case "/groups/" + root + "/members":
			fmt.Fprint(w, `[]`)
			return
		case "/groups/" + grp + "/members":
			fmt.Fprint(w, v.members(t))
			return
		}
	}
	if rest, ok := strings.CutPrefix(p, "/users/"); ok {
		v.userReqs = append(v.userReqs, rest)
		id, fed := strings.CutSuffix(rest, "/federated-identity")
		if v.faultOfUser(id) == ProxyUsers404 {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, proxyHTML404)
			return
		}
		if fed {
			if v.fedGone[id] {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"error":"User not found"}`)
				return
			}
			fmt.Fprint(w, `[]`)
			return
		}
		code := v.userCode[rest]
		if code == 0 {
			code = http.StatusOK
		}
		w.WriteHeader(code)
		if code == http.StatusNotFound {
			fmt.Fprint(w, `{"error":"User not found"}`)
			return
		}
		fmt.Fprint(w, `{}`)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// noRegistry is the Registry of a process that only reads the directory.
type noRegistry struct{}

func (noRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", nil
}

func (noRegistry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", nil
}
func (noRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) { return nil, nil }
func (noRegistry) TenantOwnsEmailDomain(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}
