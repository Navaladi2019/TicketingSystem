// Package v4probe is a throwaway for ADR-001-E4 test V-4 (b), TICK-1155.
// It calls a function with a known advisory (GO-2020-0017) so govulncheck
// reports it as reachable. Never merged.
package v4probe

import jwt "github.com/dgrijalva/jwt-go"

// Audience calls the vulnerable MapClaims.VerifyAudience.
func Audience(aud string) bool {
	c := jwt.MapClaims{"aud": aud}
	return c.VerifyAudience("x", true)
}
