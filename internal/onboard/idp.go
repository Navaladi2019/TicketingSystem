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
		return keycloak.IdPSpec{}, "/idp/provider_id"
	}
	if len(in.DisplayName) > maxDisplayName {
		return keycloak.IdPSpec{}, "/idp/display_name"
	}
	if len(in.Config) > maxConfigKeys {
		return keycloak.IdPSpec{}, "/idp/config"
	}
	cfg := make(map[string]string, len(in.Config)+2)
	for k, v := range in.Config {
		if len(v) > maxConfigValue {
			return keycloak.IdPSpec{}, "/idp/config"
		}
		if urlKeys[k] {
			if err := checkPublicHTTPS(ctx, res, v); err != nil {
				return keycloak.IdPSpec{}, "/idp/config"
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
	// Carrier-grade NAT 100.64.0.0/10 is not covered by IsPrivate.
	if cgnat := netip.MustParsePrefix("100.64.0.0/10"); cgnat.Contains(a) {
		return fmt.Errorf("address refused")
	}
	return nil
}

// netResolver adapts net.Resolver.
type netResolver struct{ r *net.Resolver }

// NewNetResolver returns the production Resolver.
func NewNetResolver() Resolver { return netResolver{net.DefaultResolver} }

func (n netResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return n.r.LookupNetIP(ctx, network, host)
}
