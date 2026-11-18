package keycloak

import (
	"net/url"
	"strings"
)

// DevIdPHostAllowed reports whether raw is an http or https URL whose host
// (with its port, if any) is exactly one of hosts. It is the one test behind
// the default-off dev allowance for the local stack's stand-in company IdP:
// the allow-list is exact host[:port] strings, never a range or a wildcard,
// and an empty list allows nothing. A URL with userinfo is never allowed.
func DevIdPHostAllowed(raw string, hosts []string) bool {
	if len(hosts) == 0 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Opaque != "" || u.User != nil || u.Host == "" {
		return false
	}
	h := strings.ToLower(u.Host)
	for _, a := range hosts {
		if h == a {
			return true
		}
	}
	return false
}
