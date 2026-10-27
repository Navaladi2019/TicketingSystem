# Fixture pins

## Images

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `postgres` | 16.10 | fixture | fixture |
| `nava/keycloak`, `nava/nava-core` | dev | built locally | fixture |

## Go modules

| Component | Version | Why | Source |
| --- | --- | --- | --- |
| `github.com/jackc/pgx/v5` | v5.11.0 | fixture | fixture |
| `golang.org/x/crypto` | ≥ v0.56.0 | fixture | fixture |
| `google.golang.org/grpc` | v1.84.0 @., v1.79.3 @temporal-server | the platform must not take the server's vulnerable version | fixture |
| `github.com/coreos/go-oidc/v3` | v3.17.0 | a row does not make a forbidden module allowed | fixture |

## Forbidden

| Module | Replacement | Ruling |
| --- | --- | --- |
| `github.com/coreos/go-oidc` | `lestrrat-go/jwx/v3` | item 11 |
