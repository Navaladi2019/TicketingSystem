package keycloak

type orgClaims struct {
	Org map[string]any `json:"organization"`
}

func read(c map[string]any) any { return c["realm_access"] }

const usersPath = "/admin/realms/{realm}/users"
