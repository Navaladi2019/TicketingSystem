package keycloak

import (
	"errors"
	"net/netip"
	"net/url"
	"regexp"
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
	netip.MustParsePrefix("::/96"), // IPv4-compatible
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), // NAT64
}

var numericLabelRE = regexp.MustCompile(`^(0[xX][0-9a-fA-F]*|[0-9]+)$`)

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
	if numericLastLabel(host) {
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

// numericLastLabel catches forms such as 0x7f.0xa, 10.0xa or 2130706433 that
// glibc's getaddrinfo reads as IPv4 (inet_aton) although netip does not parse
// them: a last label of decimal digits or a 0x-prefixed hex number.
func numericLastLabel(h string) bool {
	return numericLabelRE.MatchString(h[strings.LastIndex(h, ".")+1:])
}
