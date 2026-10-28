package contracttest

// Entry is one case of an ADR-001 Annex B catalogue.
type Entry struct {
	ID    string
	Title string
}

// CI-5 catalogue ids (ADR-001 Annex B CI-5, PLAN-001 T19). Tag a case with
// one of these to count it towards Require: "CI-5".
const (
	CI5ZeroOrganizations     = "CI-5.01"
	CI5TwoOrganizations      = "CI-5.02"
	CI5UnknownIssuer         = "CI-5.03"
	CI5ForeignRealmKey       = "CI-5.04"
	CI5IDTokenAsAccessToken  = "CI-5.05"
	CI5OrgPickerTamper       = "CI-5.06"
	CI5BrokerExistingEmail   = "CI-5.07"
	CI5BrokerOutOfDomain     = "CI-5.08"
	CI5IdPNotBound           = "CI-5.09"
	CI5ForeignIdPAtLogin     = "CI-5.10"
	CI5ForeignIdPAtInvite    = "CI-5.11"
	CI5DuplicateBinding      = "CI-5.12"
	CI5OtherTenantAdmin      = "CI-5.13"
	CI5InDomainBrokeredAllow = "CI-5.14"
)

// CI5Cases is the CI-5 catalogue, in the order of ADR-001 Annex B. Every case
// but the last must be refused; the last is the positive control.
func CI5Cases() []Entry {
	return []Entry{
		{CI5ZeroOrganizations, "a token naming zero organizations is refused"},
		{CI5TwoOrganizations, "a token naming two organizations is refused"},
		{CI5UnknownIssuer, "a token from an unknown issuer is refused"},
		{CI5ForeignRealmKey, "a token signed with realm X's key and iss=Y is refused"},
		{CI5IDTokenAsAccessToken, "an ID token used as an access token is refused"},
		{CI5OrgPickerTamper, "the kc.org organization-picker tamper to a non-member organization is refused"},
		{CI5BrokerExistingEmail, "a brokered login asserting an existing user's email is refused"},
		{CI5BrokerOutOfDomain, "an out-of-domain brokered user gets no tenant"},
		{CI5IdPNotBound, "an idp not bound to the tenant is refused"},
		{CI5ForeignIdPAtLogin, "a user federated to a foreign IdP is refused at login"},
		{CI5ForeignIdPAtInvite, "a user federated to a foreign IdP is refused at invitation"},
		{CI5DuplicateBinding, "a duplicate (issuer, organization) binding is refused"},
		{CI5OtherTenantAdmin, "tenant B's admin, with full rights in B, gets no token accepted for tenant A"},
		{CI5InDomainBrokeredAllow, "allowed: an in-domain brokered user gets the organization and idp"},
	}
}

// Catalogues are the catalogues Suite.Require may name.
func Catalogues() map[string][]Entry {
	return map[string][]Entry{"CI-5": CI5Cases()}
}
