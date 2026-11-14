---
title: "Using the TypeScript SDK"
section: operations
status: in-progress
sources: [sdk/ts/package.json, sdk/ts/.nvmrc, sdk/ts/src/index.ts, sdk/ts/src/client.ts, sdk/ts/src/errors.ts, sdk/ts/src/version.ts, make/sdk.mk, scripts/ci/sdk-changed.sh, Taskfile.yml, .github/workflows/ci.yml, docs/quality/ci-pipeline.md, examples/sdk-node/README.md]
adrs: [ADR-007]
stories: [S25]
operations: [listRecords, getRecord, createRecord, patchRecord]
packages: [sdk/ts]
---

`@nava/sdk` is the TypeScript client for the core API (ADR-007 D10). It runs in Node 20 or later and in browsers. Most of it is generated from `api/openapi.bundled.yaml` into `sdk/ts/src/generated/` by a pinned `openapi-typescript`; a hand-written layer (`sdk/ts/src/client.ts`, `errors.ts`) adds tokens, idempotency keys, versions, cursors and typed errors. It is on the story branch `TICK-801/typescript-sdk` until S25 merges.

## Install

There is no registry package (PLAN-008 D-8-7). Build the tarball, then install it by path:

```sh
task sdk:build                    # npm ci, build into sdk/ts/dist, npm pack -> sdk/ts/nava-sdk-<version>.tgz
cd my-app && npm install ../sdk/ts/nava-sdk-<version>.tgz
```

The CI `sdk` job uploads the same tarball as the `sdk` artifact. The package is ESM only: `import { createClient } from "@nava/sdk"`. In CommonJS, use `await import("@nava/sdk")`; `require` throws `ERR_PACKAGE_PATH_NOT_EXPORTED`.

## Token provider

```js
import { createClient } from "@nava/sdk";

const client = createClient({ baseUrl: "http://127.0.0.1:18080", getToken: async () => myToken() });
```

`getToken` is called before every request and once more after a `401`, and the request is retried once with the new token. The SDK never stores the token. The tenant comes from the token, as for every caller (I2); the SDK cannot choose one. The token comes from your own Keycloak sign-in; `examples/sdk-node/dev-token.mjs` is a local-stack helper only.

## Records

| Call | Request | Notes |
|---|---|---|
| `records.list(type, { filter, pageSize })` | `GET /api/v1/records/{entity}` | async iterator; follows `next_cursor` until the last page. `filter: { status: "new" }` becomes `filter[status]=new`, an array becomes `filter[status][in]=a,b` |
| `records.get(type, id)` | `GET /api/v1/records/{entity}/{id}` | returns `{ record, version }` |
| `records.create(type, data, { idempotencyKey })` | `POST /api/v1/records/{entity}` | sends `Idempotency-Key`; a random UUID when you give none. The same key in the same tenant answers the same record |
| `records.update(type, id, patch, { version })` | `PATCH /api/v1/records/{entity}/{id}` | merge patch with `If-Match: W/"<version>"` and a fresh `Idempotency-Key` |

<!-- gen:operations ids=listRecords,getRecord,createRecord,patchRecord -->

History, attachments, entity types and admin calls are not in the v1 layer; the generated types for them are exported (`paths`, `components`, `operations`) and `createRawClient(baseUrl)` gives the raw generated client.

## Errors

Every non-2xx answer throws. The classes are exported from the package.

| Class | When | Fields |
|---|---|---|
| `NavaProblem` | any RFC 9457 problem | `status`, `code` (registered in `api/errors.yaml`), `title`, `detail`, `fields` (`[{ pointer, rule }]`, e.g. `/data/priority enum`) |
| `VersionConflict` | `412`: the `version` you sent is not the newest | as `NavaProblem` |
| `RateLimited` | `429` | adds `retryAfter`, the advised wait in seconds from `Retry-After` |

Branch on the class or on `code`, never on `title` or `detail`. A body that is not JSON still gives a `NavaProblem` with `code: "unknown"`. The SDK does not retry a 412 or a 429 for you.

## Versioning (D11)

SDK major = API major (`/api/v1` gives `1.x`); a minor is additive; a patch is an SDK fix. The version is in `sdk/ts/package.json` and `SDK_VERSION` (`sdk/ts/src/version.ts`) and is currently `1.0.0-alpha.1`. Every request sends `Nava-SDK: ts/<version>`. The header is diagnostic only: the server never trusts it or decides anything by it.

## Regeneration rule (D-8-5)

Whoever changes the contract regenerates the SDK. Any PR that changes `api/openapi.bundled.yaml` also runs `task sdk:gen` and commits `sdk/ts/src/generated/`. The output is a pure function of the bundle, so two stories that both change the bundle do not count as a new file overlap; a merge conflict in `generated/` is resolved by regenerating, never by hand. Never edit `generated/` by hand.

## The `sdk` CI job

The `sdk` job in `.github/workflows/ci.yml` (TICK-827, RFC-017 CI-E10) runs on the Node pinned in `sdk/ts/.nvmrc`:

1. `make check-sdk-generated`: reruns the generator over the bundle and fails when `sdk/ts/src/generated` differs or gains an untracked file, with `run task sdk:gen and commit the result (CI-E10)`.
2. `make sdk-lint` (strict type-check) and `make sdk-test` (unit and type tests).
3. `make sdk-pack`: the tarball becomes the `sdk` artifact. Nothing is published.

It runs only when `api/`, `sdk/`, `make/sdk.mk`, `scripts/ci/sdk-changed.sh` or `ci.yml` changed (`make ci-sdk-changed`), with the same full-run, lint-only, push and story-branch rules as `temporal-server`; `full-suite` accepts a skipped `sdk` only when that step said `false`. Budget: about 1 minute, timeout 10 (`docs/quality/ci-pipeline.md`).

The SDK's integration tests against the running stack (`sdk/ts/test/integration/`, TICK-829) run in the `stack` job as the stack case `sdk_ts`. On a laptop, with the stack up and seeded: `make sdk-integration`.

## Local commands

| Command | What it does |
|---|---|
| `task sdk:gen` | regenerate `sdk/ts/src/generated/` from the bundle |
| `task sdk:test` | strict type-check and the SDK tests |
| `task sdk:build` | build into `sdk/ts/dist` and `npm pack` the tarball |

A worked example is `examples/sdk-node/` (S25). See [S25](../stories/s25.md) and [ADR-007](../adr/adr-007.md).
