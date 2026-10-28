package cursor

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// ClaimRoute binds a cursor to the endpoint that issued it, so a cursor from
// one list endpoint does not decode on another.
const ClaimRoute = "route"

// Paging parameters, left out of the query hash so it is the same on every
// page of one listing.
const (
	paramCursor = "cursor"
	paramLimit  = "limit"
)

// ErrNoPrincipal is returned by ClaimsFor outside an authenticated request.
var ErrNoPrincipal = errors.New("cursor: no authenticated principal in context")

// NewKeyringFromConfig builds the keyring from the loaded configuration. It
// returns nil, nil when no keys are configured.
func NewKeyringFromConfig(c config.Cursor) (*Keyring, error) {
	if len(c.Keys) == 0 {
		return nil, nil
	}
	keys := make([]Key, len(c.Keys))
	for i, k := range c.Keys {
		keys[i] = Key{ID: k.ID, Secret: k.Secret}
	}
	return NewKeyring(c.ActiveKeyID, keys...)
}

// ClaimsFor builds the claims for a list endpoint, for both Encode and Decode.
// It is the only way endpoints build claims: the tenant and principal come
// from the authenticated request context (identity.FromContext), never from
// the query, a header or the body, and the signature has nowhere to pass
// them in. route names the endpoint, such as its OpenAPI operationId. q is the
// request's query; cursor and limit are dropped from a copy before hashing.
func ClaimsFor(ctx context.Context, route string, q url.Values) (Claims, error) {
	p, ok := identity.FromContext(ctx)
	if !ok || p.TenantID == uuid.Nil || p.PrincipalID == uuid.Nil {
		return nil, ErrNoPrincipal
	}
	if route == "" {
		return nil, errors.New("cursor: route must not be empty")
	}
	filter := make(url.Values, len(q))
	for k, v := range q {
		filter[k] = v
	}
	filter.Del(paramCursor)
	filter.Del(paramLimit)
	return Claims{
		ClaimTenant:    p.TenantID.String(),
		ClaimPrincipal: p.PrincipalID.String(),
		ClaimQuery:     QueryHash(filter),
		ClaimRoute:     route,
	}, nil
}

// WriteInvalid sends the 400 invalid_cursor problem for a Decode error. It is
// one code for every failure, so a client cannot tell a tampered cursor from
// another tenant's.
func WriteInvalid(reg *problem.Registry, w http.ResponseWriter, r *http.Request) {
	reg.Write(w, r, problem.InvalidCursor)
}
