# Review: TICK-1164, check-pins fails on a mapped Dockerfile whose Go image it cannot read (V-5 follow-up)

PR #917 at `f688ad89`. Reviewer: Quality, Testing & CI Advisor.
Read first: `docs/adr/ADR-001/E4/vulncheck-contract.md` rule 8, test V-5, and finding 1 of `docs/reviews/tick-1151-scanned-go-is-shipped-go.md`.

**VERDICT: APPROVED.** No Bug and no Spec drift. Two non-blocking findings are listed below.

## Evidence

- **CI on `f688ad89`.**
  - The first run had no label and ran lint only: a PR into `main` without `ci:run` runs lint only (TICK-563), so `unit` was skipped and the new tests had not run in CI.
  - I added `ci:run`. Run 45761809296 then passed `changes`, `lint` (includes `make check-pins` on the real tree), `unit`, `integration`, `coverage`, `commit-check-budget`, `runner` and `story-branches`.
  - The `unit` log shows `ok tools/ci/checkpins` under `-race`.
  - The other jobs (`stack`, `temporal-server`, `sdk`, `isolation`, `full-suite`, `main-red`, `self-hosted-cleanup`) skip by design on this diff.
- **Local.** `go test ./tools/ci/checkpins/` passes.
- **Mutation A: rule 3 removed** (`readableImage` returns nil instead of the "neither golang, an earlier stage nor an allowed runtime image" error). Two subtests fail:
  - `non-golang Go stage beside a decoy golang stage`;
  - `build stage from a non-golang Go image`.

  So the decoy test really depends on the allowlist.
- **Mutation B: `samego.go` from `main`.** Seven new subtests fail:
  - `image from a build ARG` (AC1);
  - `build stage from a non-golang Go image` (AC2);
  - `image from a bare $VAR`;
  - `runtime image from a variable`;
  - the decoy;
  - the line continuation;
  - `a missing tracked Dockerfile is named`.

  All the tests red before the fix are also green after it.

## Acceptance criteria

| # | Criterion | Test | Result |
|---|---|---|---|
| 1 | Red with `ARG GO_IMAGE=golang:1.27.3-alpine` + `FROM ${GO_IMAGE}` in `nava-relay` | `.../image from a build ARG` | pass (red on `main`'s code) |
| 2 | Red with `FROM cgr.dev/chainguard/go` as `nava-relay`'s build stage | `.../build stage from a non-golang Go image` | pass (red on `main`'s code) |
| 3 | Green on `main`; existing `checkpins` tests pass | `.../repository`, the whole package, `make check-pins` in `lint` | pass |

## `runtimeImages` against today's Dockerfiles

The eight mapped Dockerfiles are the seven `deploy/compose/nava-*/Dockerfile` files and `temporal-server/Dockerfile`. Each one uses only these bases:

- `golang:1.27.4-alpine` and `alpine:3.22.1` in each `nava-*` file;
- `golang:1.27.4-bookworm` and `debian:bookworm-20261005-slim@sha256:...` in `temporal-server`.

Both runtime bases are on the list, and `make check-pins` is green in CI. The other Dockerfiles are `Dockerfile.ci`, `keycloak`, `otel-collector` and the testdata file. None of them is mapped, so rule 3 does not apply to them.

## Findings (most severe first)

1. **Breaks under: a Dockerfile line that starts with `from` but is not a `FROM` instruction** (`samego.go`, `fromAnyRe`).
   - The regex is `(?i)^\s*FROM` with no word boundary, and the image after it is optional. Two kinds of line therefore match:
     - a line inside a `RUN <<EOF` heredoc or a `\` continuation, such as Python's `from os import path`, which is read as image `os` and rejected;
     - a word that only starts with `FROM`, such as `FROMX`, which reports "FROM without an image".
   - The check fails closed and names the line, so nothing unsafe ships, and none of today's Dockerfiles has a heredoc. The cost is a confusing false red on the first heredoc.
   - Fix: require whitespace or end of line after `FROM` (`^\s*FROM(?:\s|$)`), and skip heredoc bodies and continuation lines.
   - Missing test: a mapped Dockerfile with `RUN <<EOF` / `from x import y` / `EOF` stays green.
2. **Preference: package runtime.** Locally the `checkpins` package now takes about 6 minutes on Windows (95 s at TICK-1151), because 8 more subtests each `git init` a full copy. On CI Linux it takes 20 s, so this is a laptop cost only. Finding 3 of TICK-1151 is still open. If it grows, build the green-case copy once with `sync.Once` inside the test function, which needs no package globals.

## Accepted residual risk (recorded, not a finding)

- A runtime image on the list could `RUN apk add go && go build`. The check does not read `RUN`, so only diff review catches it. The Technical Lead accepted this risk on the issue, and it is indexed as `AR-TICK-1164-01` in `docs/security/accepted-risks.md`.
- `golang` under any registry path matches. Which registry an image comes from is the job of the pins check.

## Handbook

No handbook page mentions check-pins today. When T-E4-9 (TICK-1156) documents rule 8, it should describe the readable-image rule and `runtimeImages`, as the PR description says.
