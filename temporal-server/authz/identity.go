// Package authz holds the frontend Authorizer and ClaimMapper of the custom
// Temporal server (ADR-001 D22, Annex A.2). Access is deny by default: a call is
// allowed only when the caller's certificate identity is listed for the API in
// policy.go and the call targets the `nava` namespace (or no namespace, for APIs
// whose request carries none). A nava-worker RespondWorkflowTaskCompleted is
// also denied if any command reaches another tenant (commands.go, erratum E1).
package authz

import (
	"go.temporal.io/server/common/authorization"
)

// Identity is a caller identity, derived only from the client certificate's CN.
type Identity string

const (
	Relay    Identity = "nava-relay"
	Core     Identity = "nava-core"
	Worker   Identity = "nava-worker"
	Operator Identity = "nava-operator"
	// System is the server's own system worker, on the `systemWorker` certificate.
	System Identity = "temporal-system"
)

// Namespace is the single Temporal namespace of an installation (D22).
const Namespace = "nava"

// Identities lists every identity the ClaimMapper can produce.
var Identities = []Identity{Relay, Core, Worker, Operator, System}

func identityFromCN(cn string) (Identity, bool) {
	for _, id := range Identities {
		if cn == string(id) {
			return id, true
		}
	}
	return "", false
}

// ClaimMapper maps the verified client certificate's CN to an Identity. Any
// other CN, no certificate, or a bearer token alone yields empty claims, which
// the Authorizer denies. The installation CA is enforced by the TLS handshake
// (global.tls.frontend.server.requireClientAuth), not here.
type ClaimMapper struct{}

var _ authorization.ClaimMapper = ClaimMapper{}

// GetClaims never returns an error: an unknown caller is "no identity", which is
// denied (and logged) by the Authorizer with the API and namespace it tried.
func (ClaimMapper) GetClaims(ai *authorization.AuthInfo) (*authorization.Claims, error) {
	if ai == nil || ai.TLSSubject == nil {
		return &authorization.Claims{}, nil
	}
	id, ok := identityFromCN(ai.TLSSubject.CommonName)
	if !ok {
		return &authorization.Claims{}, nil
	}
	return &authorization.Claims{Subject: string(id)}, nil
}
