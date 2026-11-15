# S25 acceptance kit: a developer installs the TypeScript SDK and lists, reads and creates records

Story TICK-801 · kit task TICK-830 (PLAN-008 P8-25) · ADR-007 D10, D11 · verified 2026-11-14 on a fresh stack.

## What you can now do

Install `@nava/sdk` (a tarball built by CI) in a Node 20 project, hand it a function that returns a token, and list, read and create incidents with typed results and typed errors, without writing any HTTP code. A repeated create with the same key makes one record; a stale update, an invalid body and rate limiting each arrive as their own error type.

## Try it (about 10 minutes)

From the repository root, on the story branch `TICK-801/typescript-sdk`, with Docker and Node.js 20 or later:

```sh
make dev-ca                                  # once per checkout: Temporal's dev certificates (deploy/compose/certs/, git-ignored)
docker compose down -v                       # no volume from an earlier run
docker compose up -d --wait                  # every service healthy
sh docs/acceptance/S25/run.sh                # seed, pack, install, run the example and every check
```

`run.sh` seeds tenants A and B, runs `task sdk:build`'s steps (build, `npm pack`), installs the tarball into `examples/sdk-node/`, runs the example and the criteria walk, and counts the replayed create's audit rows in Postgres. It ends with `ALL PASS` and `PASS  C2 one audit_log row`.

To try it by hand after `run.sh` (or after `task sdk:build` and `npm install` in `examples/sdk-node/`):

```sh
cd examples/sdk-node
node list-read-create.mjs              # lists every incident of tenant A page by page, creates one, reads it back
node list-read-create.mjs my-key-1     # run this twice: the second run prints the same INC number, no new record
node criteria.mjs                      # criteria 1 to 4 and the isolation checks, one PASS/FAIL line each
```

You are `alice@tenant-a.test` (tenant A's agent). The token comes from the kit's dev helper `examples/sdk-node/dev-token.mjs`, which signs in to the local Keycloak with the realm's dev-only seed password; nothing to type. The SDK's whole setup is one call:

```js
const client = createClient({ baseUrl: "http://127.0.0.1:18080", getToken: () => devToken("alice@tenant-a.test") });
```

If another compose project on the host already holds ports 5080 or 9091, set `OPENOBSERVE_PORT` and `PROMETHEUS_PORT` to free ports before `docker compose up` (this run used 15080 and 19091).

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | `client.records.list("incident")` iterates every page through cursors; `client.records.get("incident", id)` returns the record and its version | ✅ | `fresh-stack-run.txt`: 5 records at `pageSize: 2` arrive once each in 3 list requests; `get` returns the record and `version 1`. `examples/sdk-node/list-read-create.mjs` is the ten-line script (9 lines) |
| 2 | `create` twice with the same `idempotencyKey` creates one record | ✅ | `fresh-stack-run.txt`: the example run twice with one key prints `INC0000002` both times; the walk gets the same id back, finds one record by body, and Postgres holds **one** `audit_log` row for it |
| 3 | stale version → `VersionConflict`; invalid body → `NavaProblem` `record_invalid` with field errors; 429 → `retryAfter` | ✅ | `fresh-stack-run.txt`: `VersionConflict 412 version_conflict`, the stale write changed nothing; `NavaProblem 422 record_invalid` with `/data/priority enum, /data/short_description required`, nothing written; a burst of 200 as tenant B gives 47 `RateLimited` (`429 rate_limited`, `retryAfter 1`) and no other error |
| 4 | every request carries `Nava-SDK: ts/<version>`; package version is `1.x` | ✅ | `fresh-stack-run.txt`: create, get, list and update all sent `Nava-SDK: ts/1.0.0-alpha.1`; installed `@nava/sdk@1.0.0-alpha.1`, equal to `SDK_VERSION` (see Known limits on the pre-release tag) |
| 5 | changing `api/openapi.bundled.yaml` without regenerating fails CI (`sdk` job) | ✅ | Probe PR #662 (TICK-827, closed, not merged). Commit `ec3074e8` edits the bundle only: run [43577782547](https://github.com/Navaladi2019/TicketingSystem/actions/runs/43577782547), job `sdk` **failure**, `check-sdk-generated: sdk/ts/src/generated is stale against api/openapi.bundled.yaml; run task sdk:gen and commit the result (CI-E10)`. Commit `6f762e64` adds `task sdk:gen`: run [43577934429](https://github.com/Navaladi2019/TicketingSystem/actions/runs/43577934429), job `sdk` **success** |

Also on Node 20 (the story's runtime): `node20-run.txt`, the example and the whole walk under `node v20.20.2`, `ALL PASS`. The SDK's own integration tests (TICK-829, stack case `sdk_ts`) cover the same paths in CI, plus the 401 retry.

## Isolation check

Run as `bob@tenant-b.test` against what `alice@tenant-a.test` created (`fresh-stack-run.txt`, lines `ISO`):

| Tried | Result |
|---|---|
| tenant B `get` of tenant A's record id | `404 not_found` (not 403: the record's existence is not revealed) |
| tenant B `update` of tenant A's record | `404 not_found`; tenant A reads it back unchanged |
| tenant B `list` filtered to tenant A's marker | 0 records |
| tenant B `create` with the same `idempotencyKey` tenant A used | a new tenant B record (`INC0000001` in B), not tenant A's: keys are per tenant |

## Setup actually run

Date 2026-11-14, a developer laptop (Windows 11, Docker Desktop, Git Bash, Node 22.12.0 and Node 20.20.2 via `npx -p node@20`). Story branch at `1da27457` (PR #670 merged).

1. `make dev-ca`, `docker compose build` (every image from this branch), `docker compose down -v`, `docker compose up -d --wait --no-build` with `OPENOBSERVE_PORT=15080 PROMETHEUS_PORT=19091` (another project's stack held 5080 and 9091). All 12 services healthy; nava-core `/readyz` on the admin listener answered `ok`.
2. `sh docs/acceptance/S25/run.sh > docs/acceptance/S25/fresh-stack-run.txt`: exit 0.
3. On the same stack: the example and walk under Node 20 → `node20-run.txt`.

`main` is not merged into the story branch at `1da27457`. The commits it lacks (`945b6763` and earlier) change docs, `scratch/` spikes and CI files only, no service code, so the stack under test is the one `main` would run. The story → `main` checklist still merges `main` and runs the full CI.

The transcripts hold no token or password; one host path from `npm ls` was replaced with `examples/sdk-node`.

## Not included

- Publishing to an npm registry: the tarball is a CI artifact only (PLAN-008 D-8-7).
- History, attachments, entity-type and admin calls: the v1 SDK covers records list, get, create and update (the integration test reads history with a raw request).
- The embeddable widget built on the SDK: later ADR-007 stories.
- A browser example: the kit is Node only.

## Known limits

- **Pre-release version.** The package is `1.0.0-alpha.1`: major 1, so criterion 4 holds, but it is a pre-release tag of 1.0.0. Whether the first CI-built tarball should drop `-alpha.1` is D11's call (Solution Architect / Technical Lead), not a failing criterion.
- **ESM only.** `exports` has only an `import` condition, so `require("@nava/sdk")` throws `ERR_PACKAGE_PATH_NOT_EXPORTED`; CommonJS projects must use `await import("@nava/sdk")`. The kit's own script first hit this when resolving the package's `package.json` and now uses `import.meta.resolve`.
- **User name in the story.** The story says `agent@tenant-a`; the seed's tenant A agent is `alice@tenant-a.test`. Same role, different name.
- **Windows host, 429 burst.** With 300 requests at once (the CI test's number), Docker Desktop's published-port proxy refused about 70 connections (`ECONNREFUSED`, never reaching nava-core, which logged no error). The kit sends 200, which still empties the bucket (burst 100, 50/s). Linux CI is not affected.
- **Fresh checkout needs `make dev-ca`.** Without it, `docker compose up` creates empty directories in place of the certificates and `temporal-server` stays unhealthy (`read /etc/temporal/certs/ca.pem: is a directory`); delete those directories (`find deploy/compose/certs -mindepth 1 -type d -empty -delete`) and run `make dev-ca`.
- **No lock file in the example.** Every `npm pack` makes a new tarball hash, so `examples/sdk-node/.npmrc` turns `package-lock` off; the tarball path in `package.json` names the version and needs updating with it.
