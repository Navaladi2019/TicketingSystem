# Identity port and access-token validation (T15)

The core reaches the identity provider only through `internal/identity`
(ADR-001 A.4, D-P6). `internal/identity/keycloak` is the one adapter in M1 and
the only package that imports the JWT/JWKS library, calls the Keycloak admin
API or reads Keycloak-specific claims.

| Piece | Where | Who fills it |
| --- | --- | --- |
| `IdentityResolver`, `OperatorResolver`, `DirectoryAdmin`, `ErrInvalidToken`, `ErrTenantUnresolved` | `internal/identity` | T15 |
| Access-token validation (A.1 step 1), `ResolveOperator` | `internal/identity/keycloak` | T15 |
| Tenant resolution (A.1 steps 2–4) | `keycloak.Adapter.Resolve`, reading `identity.Store` (`internal/identity/pgstore`) | T17, see [tenant-resolution.md](tenant-resolution.md) |
| `DirectoryAdmin` adapter | `internal/identity/keycloak` | T18 |
| Contract suite every adapter must pass | `internal/identity/contracttest` | T15 harness; T17, T18, T19 add cases |

## Validation order

```mermaid
flowchart TD
  T[raw access token] --> S{compact JWS,<br/>≤ 16 KiB, alg RS256, kid set?}
  S -- no --> X[ErrInvalidToken]
  S -- yes --> I{iss on the<br/>allow-list?}
  I -- no, no HTTP request --> X
  I -- yes --> K{kid in iss's JWKS?}
  K -- no --> G{iss fetched in<br/>the last 30 s?}
  G -- yes --> X
  G -- no --> R[one forced JWKS refresh,<br/>shared by concurrent callers] --> K2{kid now?}
  K2 -- no --> X
  K2 -- yes --> V
  K -- yes --> V{signature; iss exact; aud;<br/>exp required, exp/nbf ±30 s;<br/>typ = Bearer; azp; sub}
  V -- no --> X
  V -- yes, tenant resolver --> O{exactly one organization?}
  O -- no --> U[ErrTenantUnresolved]
  O -- yes --> T17[T17: binding, idp claim,<br/>federated links]
  V -- yes, operator resolver --> Role{realm role<br/>nava-operator?}
  Role -- no --> X
  Role -- yes --> AL{iss, sub on<br/>Config.Operators?}
  AL -- no --> X
  AL -- yes --> Op[Operator]
```

- **Keys come only from the issuer's own discovery document.** On first use of
  an issuer the adapter reads `<iss>/.well-known/openid-configuration`; its
  `issuer` must equal `iss` exactly and its `jwks_uri` must lie under `iss`:
  as text (no `..`, `@`, `\`, query or fragment) and parsed with `url.Parse`
  (same scheme and host as `iss`, empty `RawPath`, so no `%2e%2e` or `%2f`, and
  a `path.Clean`ed path still under the issuer's; T15a, TICK-152). The path
  after `iss` may hold no `%` and no `//` at all, so a proxy that decodes twice
  (`%252e`) or merges slashes sees the path Go sent. The JWKS URL
  then goes on the cache's URL allow-list. So a token cannot point key fetches at another host, and a key
  of realm X never verifies a token with `iss=Y`.
- **Fetch budget** (TICK-94 item 11): background refresh no more often than
  every 5 minutes (`WithMinInterval(5m)`); an unknown `kid` forces at most one
  refresh per issuer per 30 s, and discovery and the first fetch count toward
  that window. A rotated key is accepted at the first unknown-`kid` token once
  the window allows, without a restart. Spike, in real time: 1,000 random-`kid`
  tokens in 60 s caused 2 JWKS fetches; a key rotated at +40 s was accepted
  after about 19 s (`go test -tags spike -run TestSpike ./internal/identity/keycloak/`).
- **One audience per resolver:** `nava-api` for `Resolve`, `nava-operator-api`
  for `ResolveOperator`. The operator resolver also requires the realm role
  `nava-operator`: audience alone is never enough (TICK-57).
- **Operator allow-list** (AR-030, TICK-195): `ResolveOperator` also requires
  the token's `(iss, sub)` to be on `Config.Operators`, which comes from
  platform config, never from a request. An empty list refuses every operator
  token. So a role mapping alone, such as one made with a stolen runtime-account
  secret, never makes anyone an operator. On the stack the list is the pinned
  id of `operator@nava.test`, and `TestOperatorRoleMembersAllowListed` fails if
  `nava-operator` has any other member, directly or through a group.
- **Errors** wrap one of the two sentinels with a fixed reason (`expired`,
  `issuer`, `kid`, …). Library error text, which can quote token values, never
  leaves the adapter.

## Outbound calls

Discovery and JWKS fetches go through `telemetry.OutboundClient` (TICK-94
item 2), whose timeouts also bound the background refresh. jwx's cache takes
any `Do(*http.Request)` client, so the adapter passes the outbound client
itself, wrapped to set the route template and `telemetry.WithoutRedirects`;
no `*http.Client` leaves
`internal/platform/telemetry` and no lint exception is needed. Spans are named
`GET /realms/{realm}/.well-known/openid-configuration` and
`GET /realms/{realm}/protocol/openid-connect/certs`. The span filter exports an
outbound client span only under a method and a route template, else `_OTHER`.

**No redirects** (T15a, TICK-152). Both fetches refuse every 3xx, same-origin
included, and fail closed with `telemetry.ErrRedirectRefused`: the checked URL
is the only one fetched. A same-host redirect from realm X's certs to realm
Y's would otherwise undo the `jwks_uri` path check. Keycloak serves both
documents directly, so nothing legitimate is lost. Other outbound callers keep
the default policy (up to 10 redirects, any host).

Metric: `nava.identity.jwks.fetch{trigger=initial|unknown_kid, result=ok|error|throttled}`.

## CI-11

| Check | How | Fixture (`task lint:fixture`) |
| --- | --- | --- |
| Only the adapter imports a Keycloak client or jwx/httprc | depguard rule `identity-port` in `.golangci.yml`, `files: ["$all", "!**/internal/identity/keycloak/**"]` | `internal/platform/lintfixture/identityport.go` imports jwx |
| No Keycloak claim read (`organization`, `idp`, `realm_access`, `resource_access`) or admin API path (`/admin/realms`) outside the adapter | navalint `idpclaims` (`tools/navalint/idpclaims`); the `_test.go` files of `internal/identitystack`, the realm's own acceptance tests, are exempt | the same fixture reads `organization` and builds an admin URL |
| Every adapter meets the port contract | `contracttest.Suite.Run`, run by `internal/identity/keycloak/contract_test.go` | — |

depguard sees direct imports only; a re-exporting wrapper would get around it,
which is why the contract suite stays as the backstop.

## IdP spec URLs and SSRF through Keycloak

`checkIdPConfig` validates every URL-valued key of an `IdPSpec` (`authorizationUrl`,
`tokenUrl`, `logoutUrl`, `userInfoUrl`, `issuer`, `jwksUrl`, `metadataDescriptorUrl`,
`singleSignOnServiceUrl`, `singleLogoutServiceUrl`) before any admin call: `https` only, no
userinfo, no IP literal in a loopback, private, link-local, CGNAT, multicast or reserved range,
and no `localhost`, `.local`, `.internal` or single-label host.

This is a syntax check only. The adapter cannot check what a public-looking name resolves to,
and Keycloak resolves it later. The deployment must therefore apply a network egress policy on
the Keycloak container that denies Postgres, Temporal, the management port and the cloud
metadata address. The local compose stack has no such policy; that is a known limit of local
development, so do not feed tenant-supplied IdP specs to a local stack that holds real data.
