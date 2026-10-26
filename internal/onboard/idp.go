package onboard

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

// urlKeys are the IdP config keys whose value is a URL the Keycloak server
// will fetch or redirect to. Each must be https and must not resolve to a
// private, loopback or link-local host (SSRF through the IdP settings).
//
//nolint:gochecknoglobals // read-only allow-list
var urlKeys = map[string]bool{
	"authorizationUrl": true, "tokenUrl": true, "logoutUrl": true, "userInfoUrl": true,
	"jwksUrl": true, "metadataDescriptorUrl": true, "issuer": true,
	"singleSignOnServiceUrl": true, "singleLogoutServiceUrl": true,
}

// Resolver resolves a host to its addresses; net.DefaultResolver in
// production.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// IdPRequest is the tenant IdP the operator asks for. The alias, trustEmail,
// the first-login flow and mappers are never taken from the request.
type IdPRequest struct {
	ProviderID  string
	DisplayName string
	Config      map[string]string
}

const (
	maxConfigKeys  = 40
	maxConfigValue = 8192
	maxDisplayName = 128
)

// buildSpec validates the request and returns the spec to create. It pins
// validateSignature=true and syncMode=IMPORT whatever the request says. The
// returned string is the JSON pointer of the first failing field.
func buildSpec(ctx context.Context, res Resolver, in IdPRequest) (keycloak.IdPSpec, string) {
	if in.ProviderID != "oidc" && in.ProviderID != "saml" {
		return keycloak.IdPSpec{}, "/identity_provider/provider_id"
	}
	if len(in.DisplayName) > maxDisplayName {
		return keycloak.IdPSpec{}, "/identity_provider/display_name"
	}
	if len(in.Config) > maxConfigKeys {
		return keycloak.IdPSpec{}, "/identity_provider/config"
	}
	cfg := make(map[string]string, len(in.Config)+2)
	for k, v := range in.Config {
		// An allow-list, not a deny-list: a key that is not known for the
		// provider (alias, kc.org.*, artifactResolutionServiceUrl, ...) is
		// refused, so no unvetted URL-valued key reaches Keycloak.
		if !keycloak.IdPConfigKeyAllowed(in.ProviderID, k) {
			return keycloak.IdPSpec{}, "/identity_provider/config"
		}
		if len(v) > maxConfigValue {
			return keycloak.IdPSpec{}, "/identity_provider/config"
		}
		if urlKeys[k] {
			if err := checkPublicHTTPS(ctx, res, v); err != nil {
				return keycloak.IdPSpec{}, "/identity_provider/config"
			}
		}
		cfg[k] = v
	}
	cfg["validateSignature"] = "true"
	cfg["syncMode"] = "IMPORT"
	return keycloak.IdPSpec{ProviderID: in.ProviderID, DisplayName: in.DisplayName, Config: cfg}, ""
}

// checkPublicHTTPS accepts an https URL with a host name or address that
// resolves only to public addresses. The error never carries the URL.
func checkPublicHTTPS(ctx context.Context, res Resolver, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("not an https url")
	}
	host := u.Hostname()
	if host == "" || strings.EqualFold(host, "localhost") {
		return fmt.Errorf("host refused")
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return checkPublicAddr(ip)
	}
	addrs, err := res.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("host does not resolve")
	}
	for _, a := range addrs {
		if err := checkPublicAddr(a); err != nil {
			return err
		}
	}
	return nil
}

func checkPublicAddr(a netip.Addr) error {
	a = a.Unmap()
	if !a.IsValid() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() ||
		a.IsMulticast() || a.IsUnspecified() || a.IsInterfaceLocalMulticast() {
		return fmt.Errorf("address refused")
	}
	for _, p := range reservedPrefixes {
		if p.Contains(a) {
			return fmt.Errorf("address refused")
		}
	}
	// NAT64 embeds an IPv4 address in the low 32 bits: judge that address.
	if netip.MustParsePrefix("64:ff9b::/96").Contains(a) {
		b := a.As16()
		return checkPublicAddr(netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}))
	}
	return nil
}

// reservedPrefixes are special-purpose ranges that IsPrivate and friends do
// not cover.
//
//nolint:gochecknoglobals // read-only table
var reservedPrefixes = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24",
		"203.0.113.0/24", "240.0.0.0/4", "255.255.255.255/32", "64:ff9b:1::/48", "100::/64", "2001:db8::/32",
	} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

// netResolver adapts net.Resolver.
type netResolver struct{ r *net.Resolver }

// NewNetResolver returns the production Resolver.
func NewNetResolver() Resolver { return netResolver{net.DefaultResolver} }

func (n netResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return n.r.LookupNetIP(ctx, network, host)
}
