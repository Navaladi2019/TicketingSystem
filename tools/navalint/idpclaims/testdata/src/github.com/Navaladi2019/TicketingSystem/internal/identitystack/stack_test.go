package identitystack

func claimsOf(c map[string]any) (any, any) { return c["organization"], c["realm_access"] }

const adminUsers = "/admin/realms/nava/users"
