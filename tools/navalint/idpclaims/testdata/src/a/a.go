package a

type token interface {
	Get(name string, dst any) error
}

func claims(c map[string]any, tok token) {
	_ = c["organization"]       // want `Keycloak claim "organization" read outside internal/identity/keycloak`
	_ = c["realm_access"]       // want `Keycloak claim "realm_access" read outside`
	_ = c["resource_"+"access"] // want `Keycloak claim "resource_access" read outside`
	var v any
	_ = tok.Get("idp", &v) // want `Keycloak claim "idp" read outside`
	_ = c["sub"]
	_ = c["organizations"]
	_ = "the organization claim" // prose is fine
}

const claimName = "organization" // want `Keycloak claim "organization" read outside`

type orgClaims struct {
	Org  map[string]any `json:"organization"`  // want `Keycloak claim "organization" read outside`
	IdP  string         `json:"idp,omitempty"` // want `Keycloak claim "idp" read outside`
	Name string         `json:"name"`
}

func admin(base string) string {
	return base + "/admin/realms/nava/users" // want `Keycloak admin API outside internal/identity/keycloak`
}
