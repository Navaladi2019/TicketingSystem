package admin

const groups = "/admin/realms/{realm}/groups"

func idp(c map[string]any) any { return c["idp"] }
