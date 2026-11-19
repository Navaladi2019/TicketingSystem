package keycloak

import (
	"net/url"
	"strings"
)

// DevIdPPathPrefix is the only path a dev-allowed host may serve: the local
// stack's stand-in company realm. It is fixed, not configured, so setting
// NAVA_ONBOARD_DEV_IDP_HOSTS never opens another realm (TICK-986).
const DevIdPPathPrefix = "/realms/standin-company/"

// DevIdPHostAllowed reports whether raw is an http or https URL whose host
// (with its port, if any) is exactly one of hosts and whose path is under
// DevIdPPathPrefix. It is the one test behind the default-off dev allowance
// for the local stack's stand-in company IdP, shared by both gates: the
// allow-list is exact host[:port] strings, never a range or a wildcard, and an
// empty list allows nothing. A URL with userinfo, a percent-encoded path or a
// dot segment is never allowed.
func DevIdPHostAllowed(raw string, hosts []string) bool {
	if len(hosts) == 0 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Opaque != "" || u.User != nil || u.Host == "" {
		return false
	}
	if !devPathAllowed(u) {
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

// devPathAllowed accepts a path that starts with DevIdPPathPrefix, as written:
// no percent-encoding (so nothing is decoded into a different path), no `;`
// (a path parameter such as `..;/` is read as `..` by some servers), no empty
// or dot segment, and a query or fragment cannot move it.
func devPathAllowed(u *url.URL) bool {
	p := u.EscapedPath()
	if strings.ContainsAny(p, "%;") || !strings.HasPrefix(p, DevIdPPathPrefix) {
		return false
	}
	segs := strings.Split(strings.TrimPrefix(p, DevIdPPathPrefix), "/")
	for i, seg := range segs {
		if seg == "." || seg == ".." || (seg == "" && i < len(segs)-1) {
			return false
		}
	}
	return true
}
