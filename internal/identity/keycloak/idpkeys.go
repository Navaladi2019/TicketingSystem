package keycloak

// IdPConfigKeyAllowed reports whether key may appear in the config of an IdP
// of the given provider ("oidc" or "saml"). It is the same allow-list
// CreateTenantOrg enforces, exported so the operator handler can refuse an
// unknown key before it creates anything.
func IdPConfigKeyAllowed(providerID, key string) bool {
	if providerID == "saml" {
		return samlConfigKeys[key]
	}
	return oidcConfigKeys[key]
}
