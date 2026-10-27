package identity_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TestPrincipalFieldSet pins what an authorization decision may see (ADR-002
// AC7). Group paths and names are display data and stay out of Principal.
func TestPrincipalFieldSet(t *testing.T) {
	t.Parallel()
	want := []string{"TenantID", "PrincipalID", "Groups", "IdPAlias"}
	typ := reflect.TypeFor[identity.Principal]()
	got := make([]string, 0, typ.NumField())
	for f := range typ.Fields() {
		got = append(got, f.Name)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("identity.Principal fields = %v, want %v: changing the authorization context requires Security & Multi-Tenancy review (ADR-002 section 5)", got, want)
	}
}
