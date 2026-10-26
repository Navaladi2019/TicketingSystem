# TicketingSystem

Multi-tenant ticketing platform (`nava-core`). The platform goal and the
architecture decisions are in
[`docs/adr/ADR-001-platform-blueprint.md`](docs/adr/ADR-001-platform-blueprint.md);
contributor and agent instructions are in [`CLAUDE.md`](CLAUDE.md).

## Getting started

Commands run through [go-task](https://taskfile.dev) (`Taskfile.yml`):

```bash
task build
```

```bash
task test
```

```bash
task lint
```

Run `task` to list every available task, and `make ci` to run the whole CI
pipeline locally when debugging (agents use the GitHub `ci` workflow instead; see [`docs/quality/ci-pipeline.md`](docs/quality/ci-pipeline.md)).

## Design principles

All code, by people or agents, must follow sound object-oriented and
architectural design. Use the patterns the problem calls for; do not add a
pattern where a plain function is enough.

### SOLID

- **Single responsibility:** a package, type or function has one reason to
  change. Split HTTP handling, business rules and persistence.
- **Open/closed:** add behaviour by adding a new implementation of an
  interface, not by editing a `switch` on provider or type names.
- **Liskov substitution:** every implementation of an interface honours its
  full contract, including its documented errors. Contract test suites
  (for example `internal/identity/contracttest`) prove it.
- **Interface segregation:** keep interfaces small and shaped by the caller
  (`IdentityResolver`, `OperatorResolver` and `DirectoryAdmin` are separate,
  not one large `Keycloak` interface).
- **Dependency inversion:** core code depends on interfaces it owns; concrete
  implementations are built and injected at the composition root in
  `cmd/nava-core`.

### Third-party services: always use the Adapter pattern

Every external system (identity provider, workflow engine, database, message
broker, email, storage, payment, telemetry backends and so on) is reached only
through a **port** (an interface owned by the core) and an **adapter** (the one
package that talks to the vendor). The Keycloak integration is the reference
example:

| Piece          | Location                                   | Role                                                                                  |
| -------------- | ------------------------------------------ | ------------------------------------------------------------------------------------- |
| Port           | `internal/identity`                        | Vendor-neutral interfaces, types (`Principal`) and sentinel errors (`ErrInvalidToken`) |
| Adapter        | `internal/identity/keycloak`               | The only code that imports a Keycloak/JWT client or reads Keycloak-specific claims     |
| Contract tests | `internal/identity/contracttest`           | Suite every adapter of the port must pass                                             |
| Test double    | `internal/identity/keycloak/keycloaktest`  | Fake used by tests instead of a live Keycloak                                         |
| Wiring         | `cmd/nava-core/identity.go`                | Composition root: builds the adapter and injects it as the port                       |

Rules for a new integration:

1. Define the port in the core package, in domain terms. No vendor types,
   claim names or SDK errors in its signatures.
2. Put the vendor SDK in a single adapter sub-package. Nothing else imports
   it; add a `depguard` rule in `.golangci.yml` to enforce that.
3. Translate vendor errors into the port's sentinel errors so callers match
   with `errors.Is`, never on vendor error text.
4. Write a contract test suite for the port and run every adapter, including
   the fake, against it.
5. Wire the adapter only in `cmd/nava-core`. Swapping vendors must not change
   any code outside the adapter and the wiring.

### Other patterns to apply where they fit

- **Repository:** persistence behind an interface (see `internal/identity/pgstore`).
- **Factory / constructor injection:** `New(ctx, cfg, deps...)` returns a
  ready, validated value; no package-level globals.
- **Strategy:** interchangeable algorithms or policies behind one interface.
- **Decorator / middleware:** cross-cutting concerns (auth, tracing, retries,
  metrics) wrap a component instead of being written into it.
- **Null object:** a safe default implementation instead of `nil` checks
  (for example `refuseAll` in `cmd/nava-core/identity.go`).
- **Outbox:** reliable publication of events with the database write
  (see `internal/outbox`).

Prefer composition over inheritance-like embedding, keep interfaces at the
consumer side, and keep each design choice that changes the architecture in an
ADR under `docs/adr/`.
