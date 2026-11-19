---
title: "Troubleshooting"
section: operations
status: delivered
sources: [docs/acceptance/guide/troubleshooting.html, tools/adrgate/main.go, tools/navalint/uuidv7/uuidv7.go, docs/api/strict-json-bodies.md, internal/configsurface/doc.go, internal/configsurface/cmd/compat/main.go, docs/acceptance/s27-competitive-gate.md, deploy/compose/README.md, compose.yaml, docs/operator/onboarding.md, deploy/compose/prometheus/rules/audit.yml]
evidence: [docs/acceptance/guide/troubleshooting.html]
---

Where to look first, by symptom. The acceptance guide's [troubleshooting page](../../../acceptance/guide/troubleshooting.html) covers the Bruno walk-through in more detail.

## First three looks

1. `docker compose ps`: is every long-running service `healthy`? A one-shot job (`db-roles`, `migrate`, `temporal-schema`) that exited non-zero blocks everything after it.
2. `docker compose logs --since 5m <service>`: the reason is usually on one line. Every error response's `instance` is `urn:nava:request:<id>`; search the logs for that id.
3. `/healthz` of the service, through a [tunnel or `exec`](listeners.md): it names the failing dependency.

## The stack does not start

| Symptom | Cause | Fix |
|---|---|---|
| `ca.pem: no such file` | No dev certificates yet | `make dev-ca` (`task stack:up` runs it) |
| `all predefined address pools have been fully subnetted` | Old Docker networks | `docker network prune` (removes unused networks only) |
| A port is already in use | Another stack or program on 18080, 18081, 5080 or 9091 | Set `NAVA_CORE_PORT`, `KEYCLOAK_PORT` with `KEYCLOAK_PUBLIC_URL`, `OPENOBSERVE_PORT` or `PROMETHEUS_PORT` in `.env` |
| Startup takes minutes | Keycloak imports its realm on first boot; one-shot jobs run first | Normal on a busy machine. `docker compose up --wait` keeps waiting |

## Calls fail

| Symptom | Cause | Fix |
|---|---|---|
| 401 `invalid_token` on a fresh token | The token's issuer is not `KEYCLOAK_PUBLIC_URL/realms/nava`: Keycloak was reached through another host or port | Use the shipped URLs, or change `KEYCLOAK_PORT` and `KEYCLOAK_PUBLIC_URL` together and restart |
| 401 `invalid_token` after a while | Access tokens live 5 minutes | Sign in again |
| 401 `tenant_unresolved` | The tenants are not seeded, or no group sync has run since | `sh docs/acceptance/guide/tools/seed.sh` |
| 403 `forbidden` on `/api/v1/operator/tenants` | A tenant token was sent | Use the operator sign-in (client `nava-operator-cli`) |
| Connection refused on `18082`, `19090` to `19095` or `18233` | The tunnels are not running; they stop when the stack is recreated | `sh docs/acceptance/guide/tools/tunnels.sh up` |
| `18233` only | The Temporal UI is not started | Start with the `operator` profile (`task stack:up`), then the tunnels again |
| 502 from Keycloak | Keycloak is still importing its realm | Wait for `docker compose ps keycloak` to say healthy |
| Every token refused (`400 unauthorized_client`) after PostgreSQL restarted | Keycloak's database connections died with it | Keycloak validates its pool on every borrow, so this should recover on its own; nava-core's `/healthz` keycloak check shows `fail` until it does |
| 422 with `errors: [{"pointer": "/body/…", "rule": "duplicate"}]` or `"rule": "invalid_utf8"` | The body has the same member name twice in one object, or bytes that are not UTF-8 (a lone `\ud800` escape counts). Go 1.27 story branch | Send each name once and valid UTF-8; see [API errors and strict JSON bodies](../system/api-errors.md) |

## Something is slow or stuck

| Symptom | Where to look |
|---|---|
| One request is slow | Its trace in OpenObserve, by `request_id` ([Traces, logs and metrics](observability.md)) |
| A workflow did not start after a write | `nava-relay` logs and its admin `/metrics`; the workflow in the Temporal UI (namespace `nava`) |
| Audit entries are not sealed | Prometheus Alerts: `AuditSealLag`, `AuditDeadLineage`, `AuditDeadLineageCheckDown` |
| Users or groups changed in Keycloak do not show | Group sync runs every 5 minutes; run one cycle with `docker compose run --rm --no-deps nava-groupsync --once` |
| A tenant is stuck in provisioning | [docs/operator/onboarding.md](../../../operator/onboarding.md), "Steps and recovery" |

## A lint or test check fails

Two checks from S27 (ADR-008) fail a PR on documents and configuration, not on code style. Run one alone: `go run ./tools/adrgate`, `go run ./internal/configsurface/cmd/compat`, `go test ./internal/configsurface/...`.

**`adrgate` lines in `task lint`** (`path: CI-n: message`, exit 1):

| Line | Cause | Fix |
|---|---|---|
| `CI-1: no ## Competitive Gate section (ADR-008 D6)` | An ADR numbered above 008, or an amendment or erratum dated on or after 2026-11-15, has no gate | Copy the `## Competitive Gate (ADR-008)` section from `docs/adr/_template-adr.md` and fill it in |
| `CI-1: ## Competitive Gate has no **Result:** line before the next ## heading` | The section is there but has no result | Add `**Result:** PASS`, `PASS with accepted risk CP-…: <reason, owner, trigger>` or `N/A: <one line>` (N/A only for internal ADRs, D7) |
| `CI-1: **Result:** "…" does not start with PASS, PASS with accepted risk CP-, or N/A` or `… names no accepted risk CP- id` | The result value has another shape | Start it with one of the three forms; an accepted risk names its `CP-` id |
| `CI-1: amendment or erratum has no front-matter date: YYYY-MM-DD` | The gate cannot tell when it was written | Add `date:` to the front matter |
| `CI-2: CP-… is not a row in docs/adr/competitive/pain-points.md` | A typo, or a pain point not yet in the register | Fix the id, or add the row to the register (with evidence and a grade) in the same PR |
| `CI-3: CP-…: Evidence has no https:// link` or `Grade "…" is not A, B or C …` | A register row lacks a source or a grade | Add the link; set the grade to A, B or C (or `A/B`) |
| `CI-4: M-X… is not a method in ADR-008 "The axes"` | A method id that ADR-008 does not name | Use a method from the axes table (`M-X1a`, `M-X1b`, `M-X2`, `M-X3`, `M-X4`); a new method needs an ADR-008 amendment |

Exit 2 (no findings printed, an `adrgate:` line on stderr) means `docs/adr` or ADR-008 could not be read: run from the repository root.

**`uuid.New mints a random v4 id: persisted ids are UUIDv7, use uuid.NewV7 and handle the error (TICK-963)`** in `task lint` (navalint `uuidv7`, Go 1.27 story branch): a call to `uuid.New`, `uuid.NewString` or `uuid.NewRandom` in non-test code under `internal/` or `cmd/`. Use `uuid.NewV7()` and return its error; see [Id generation](../system/id-generation.md).

**Upgrade-compatibility lines** (M-X2, see [8. Workflows](../tour/08-workflows.md#adding-or-removing-a-step-type)):

| Line | Where | Cause | Fix |
|---|---|---|---|
| `removed: workflow.step_types/timer` (or another item) | `task test`, `internal/configsurface` | The code no longer has an item tenants can use | Put it back. Within a major version the configuration surface is additive-only (D12); deprecate instead of removing |
| `unlisted: …; add it to inventory.json` | `task test` | A new step type, field, keyword or extension is in the code but not in the inventory | Add it to `internal/configsurface/testdata/inventory.json`, with a fixture |
| `no fixture: …` / `unused: … is used by no accepted fixture` | `task test` | A listed item has no stored example | Add `internal/configsurface/testdata/fixtures/<workflow or typedef>/<name>.json` that uses it |
| `not accepted: <fixture>: …` | `task test` | A stored fixture no longer parses or compiles: tenant configuration like it would break | Restore what the fixture uses, or keep the old form accepted |
| `deprecated: … is no longer in the inventory` / `has no accepted fixture` | `task test` | A deprecated item was dropped before the next major | Keep it listed and accepted until the major is raised |
| `compat: removed: … (listed on the base, major 1); keep it, or raise major …` | `task lint` | The item was removed from both the code and the inventory | Put both back; a real removal needs a new major (`"major"` in the inventory) and `docs/upgrade/v<major>.md` naming the item |
| `compat: base origin/main not found; fetch it first (git fetch origin main)`, exit 2 | `task lint` | The base ref is missing, for example in a shallow clone | `git fetch origin main`, or set `BASE` to a ref you have |

`compat: no internal/configsurface/testdata/inventory.json on origin/main: nothing to compare` is not a failure: the base has no inventory yet.

## Start over

```sh
sh docs/acceptance/guide/tools/tunnels.sh down
task stack:down                 # deletes all data: tenants, users, workflows, telemetry
task stack:up                   # includes the operator profile
docker compose up --wait        # blocks until every service is healthy
sh docs/acceptance/guide/tools/seed.sh
sh docs/acceptance/guide/tools/tunnels.sh up
```
