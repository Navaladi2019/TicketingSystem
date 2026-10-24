package identitystack

func helper(c map[string]any) any { return c["organization"] } // want `Keycloak claim "organization" read outside`
