//go:build lintfixture

package lintfixture

import (
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// ParseOutsideAdapter imports the JWT library outside
// internal/identity/keycloak. depguard's identity-port rule must reject the
// import (CI-11, D-P6).
func ParseOutsideAdapter(raw string) (jwt.Token, error) {
	return jwt.ParseInsecure([]byte(raw))
}

// ReadOrganization reads a Keycloak-specific claim outside the adapter, and
// AdminUsers spells the admin API. navalint idpclaims must report both (CI-11).
func ReadOrganization(claims map[string]any) any {
	return claims["organization"]
}

// AdminUsers builds a Keycloak admin API URL outside the adapter.
func AdminUsers(base string) string {
	return base + "/admin/realms/nava/users"
}
