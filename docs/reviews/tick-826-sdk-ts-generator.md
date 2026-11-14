# Review: TICK-826 / PR #651, sdk/ts typed client from the OpenAPI bundle (P8-21)

- PR: #651 (`TICK-826/sdk-ts-generator` → `TICK-801/typescript-sdk`), head `97d2958f` (reviewed).
- Author: Senior Software Engineer. Reviewer: QA.
- Spec: TICK-826 scope and acceptance criteria; ADR-007 D10, D11; PLAN-008 P8-21.

## Verdict

**APPROVE.** I found no Bug, Breaks-under-load or Spec-drift findings. There is one Breaks-under-a-condition item and one Preference item. Neither blocks the merge.

## CI

GitHub `ci` run 43562635105 at `97d2958f`: every job passed or was skipped by design (`main-red`, `self-hosted-cleanup`, `temporal-server`). No job fails and none is pending.

The green run does not cover the SDK. No CI job runs `sdk:gen`, `sdk:test` or `sdk:build` yet. The `sdk` mentions in `ci.yml` refer to a Go identity case, not this package. That CI job is TICK-827 (CI-E10, drift gate), which is the next task in the story. It is not missing from this PR. So I checked the acceptance criteria locally with the package's own `npm` scripts (Node 22.12.0, the pinned `.nvmrc` version).

## What I checked

| Criterion | Evidence |
|---|---|
| `sdk:gen` twice, no diff | `npm ci`, then `npm run gen` run twice. Both runs gave the same sha1 (`580cf96b…`), and `git diff` on `schema.ts` was empty against the committed file. The only difference was CRLF vs LF in the working tree (see finding 1). |
| Generated code type-checks under `strict` | `npm test`: `tsc --noEmit` with `strict` and `noUncheckedIndexedAccess` is clean, and `skipLibCheck: false` also checks `openapi-fetch`'s own types. `node --test`: 1 pass, 0 fail. |
| `records.list` for `incident` returns the generated type | M3: adding `const bad: number = data` fails with TS2322, and the error names the full `RecordList` shape. So `data` is the generated type, not `any`. |
| A misspelt path fails `tsc` | M1: removing the `@ts-expect-error` gives TS2345 on `"/api/v1/recordz/{entity}"` not assignable to `PathsWithMethod<paths, "get">`, so the path itself is what is rejected. M2: changing the path to the correct one gives TS2578 (unused directive), so the test cannot pass by accident. |
| Pinned generator and lockfile | `openapi-typescript` 7.13.0, `openapi-fetch` 0.17.0, TypeScript 5.9.3 and `@types/node` 22.10.2, all pinned exactly. `package-lock.json` is committed. `.nvmrc` is `22.12.0`. |
| No runtime reflection, no `eval` | `openapi-typescript` emits types only. No `eval(` or `new Function(` in `openapi-fetch`'s dist or in the built `dist/`. |
| ESM and type declarations; `npm pack` | `npm run build` emits `dist/index.{js,d.ts}` and `dist/generated/schema.{js,d.ts}`. `npm pack --dry-run` ships exactly those 5 files (`files: ["dist"]`). Nothing is published (D-8-7). |
| Taskfile | `sdk:gen`, `sdk:test` and `sdk:build` run with `dir: sdk/ts` and each starts with `npm ci`. |

## Findings

### 1. Breaks under a Windows checkout with `core.autocrlf=true`: the regenerated file shows as modified (low)

`openapi-typescript` writes LF, but on Windows git checks `schema.ts` out as CRLF. After `task sdk:gen`, `git status` shows `sdk/ts/src/generated/schema.ts` as modified even though `git diff` is empty. On a Linux CI runner this does not happen, so TICK-827's drift gate will work. But a developer on Windows sees a dirty tree after every regenerate, and a `git status --porcelain` style check would fail there.

Fix: add `sdk/ts/src/generated/** text eol=lf` to `.gitattributes`, next to the existing Go/YAML rules. This can land here or in TICK-827. TICK-827's drift test should use `git diff --exit-code`, which ignores this, or rely on that attribute.

### 2. Preference: `engines` says `>=20`, but the tests need Node 22.6+

`npm test` uses `--experimental-strip-types`. The built package is plain ES2022, so it does run on Node 20, and `engines` is correct for consumers. Only contributors need 22 (from `.nvmrc`). A one-line note in the package README, when it exists, would cover it. Optional.

## Not checked here

- Hand-written layer (tokens, Idempotency-Key, If-Match, cursors, errors): TICK-828.
- Calls against the real stack: TICK-829.
- CI drift gate: TICK-827.
