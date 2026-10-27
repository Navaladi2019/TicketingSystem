package keycloak

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// idpURLKeys are the IdPSpec config keys whose values Keycloak fetches or
// redirects to from its own network position.
//
//nolint:gochecknoglobals // read-only allow-list
var idpURLKeys = toSet("authorizationUrl", "tokenUrl", "logoutUrl", "userInfoUrl", "issuer", "jwksUrl",
	"metadataDescriptorUrl", "singleSignOnServiceUrl", "singleLogoutServiceUrl")

// blockedPrefixes are IP ranges Keycloak must never be pointed at, beyond what
// netip classifies as loopback, private, link-local, multicast or unspecified.
//
//nolint:gochecknoglobals // read-only list
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), // NAT64
}

var errIdPURL = errors.New("url refused")

// checkIdPURL checks the syntax of a tenant-supplied IdP URL: https only, no
// userinfo, no IP literal in a blocked range, no localhost or single-label
// host. It cannot see what a name resolves to; that needs an egress policy on
// the Keycloak container (see docs/architecture/identity-port.md).
func checkIdPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" {
		return errors.Join(errIdPURL, errors.New("must be an https URL"))
	}
	if u.User != nil {
		return errors.Join(errIdPURL, errors.New("userinfo is not allowed"))
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return errors.Join(errIdPURL, errors.New("host is required"))
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if blockedAddr(addr.Unmap()) {
			return errors.Join(errIdPURL, errors.New("host address is in a blocked range"))
		}
		return nil
	}
	if !strings.Contains(host, ".") || host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".local") {
		return errors.Join(errIdPURL, errors.New("host must be a public multi-label name"))
	}
	if allDigitsAndDots(host) {
		return errors.Join(errIdPURL, errors.New("host is a non-canonical IP literal"))
	}
	return nil
}

func blockedAddr(a netip.Addr) bool {
	if a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() || a.IsLinkLocalMulticast() ||
		a.IsMulticast() || a.IsUnspecified() || a.IsInterfaceLocalMulticast() {
		return true
	}
	for _, p := range blockedPrefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// allDigitsAndDots catches forms such as 0x7f.1 or 1.2.3 that some resolvers
// read as IPv4 although netip does not parse them.
func allDigitsAndDots(h string) bool {
	last := h[strings.LastIndex(h, ".")+1:]
	if last == "" {
		return true
	}
	for _, r := range last {
		if (r < '0' || r > '9') && r != 'x' && r != 'X' {
			return false
		}
	}
	return true
}
