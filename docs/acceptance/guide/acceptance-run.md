# Acceptance run: Bruno collection over the whole stack (TICK-331)

- Collection: `docs/acceptance/guide/bruno/`, environment `local`, run headless with `sh docs/acceptance/guide/tools/api-test.sh` (Bruno CLI `@usebruno/cli@4.3.0`, `--sandbox developer`, Node.js 22.12.0, Git Bash on Windows 11). Bruno desktop 4.3.0 installed on the same host.
- Commit: `5846dbd` (branch `agent/quality-testing-ci-advisor/432165182ab2`, on `main` at `2ce5bbc`); only this file and the zip changed after it.
- Stack: fresh. `task stack:down` (volumes deleted), `task stack:up` (images built from that commit), `docker compose up --wait` (healthy in 350 s), `tools/seed.sh`, `tools/tunnels.sh up` (8 forwards, including the 6 admin listeners), `make demo-outbox DEDUPE_KEY=accept` twice. `.env`: `NAVA_GROUPSYNC_INTERVAL=15s`, `COMPOSE_PROFILES=operator` (default rate limit).
- Started 2026-10-29T06:14:48Z; duration 342 s.

| | executed | failed |
|---|---|---|
| requests (poll repeats included) | 145 | 0 |
| tests | 466 | 0 |

**Result: PASS** (0 failures, `api-test.sh` exit 0)

| Folder | Requests sent | Tests passed | Failed |
|---|---|---|---|
| S1 — Local stack starts healthy | 6 | 16 | 0 |
| S2 — A query without the right tenant returns nothing | 7 | 28 | 0 |
| S3 — Tenant user signs in, scoped to tenant | 21 | 62 | 0 |
| S4 — Operator onboards a tenant | 14 | 44 | 0 |
| S5 — Group changes and deprovisioning (polls repeat) | 23 | 51 | 0 |
| S6 — API ground rules | 13 | 49 | 0 |
| S7 — A committed outbox row runs exactly one workflow | 2 | 8 | 0 |
| X1 — API suite: `getMe` | 17 | 63 | 0 |
| X2 — API suite: `onboardTenant` | 22 | 82 | 0 |
| X3 — API suite: every admin listener | 18 | 55 | 0 |
| Z — Expired tokens (S3.22, X2.23) | 2 | 8 | 0 |

- S5 polls: S5.6 sent 3 times, S5.10 twice, S5.13 3 times, S5.17 twice (15 s group sync).
- S6.12 burst: tenant A `{200: 106, 429: 194}`, tenant B `{200: 20}`; S6.13 200 for tenant A after `Retry-After`.
- S7.1 selected the `accept` demo workflow (latest-started tenant A `nava.smoke.relay`, the only one listed): one execution; S7.2 found two `demo.note` signals.
- The script's failure path: an earlier run on the same day, before two collection fixes, had 9 failed tests; `api-test.sh` printed `FAILED (bru exit 1)` and exited 1.

## Coverage against the previous collection

Every assertion of the previous collection (77 requests, 262 assertions, the TICK-329 run) has a Bruno counterpart with the same name in the same request (S1.1–S7.2, S3.22): 262 of 262. One request changed its token: S4.14 now uses tenant A's admin token from S4.2 (still a valid tenant token on nava-onboard's `/me`), so S4 runs on its own. The API suite adds 58 requests and 204 assertions (X1, X2, X3, X2.23).

## Findings from the suite

- **Spec drift** (`api/errors.yaml` vs `internal/server/api.go:159`): `invalid_idp_config` says "See errors for the failing fields", but the 422 has no `errors` array (only `validation_failed` writes the fields). X2.17–X2.19 assert the code and status only. The OpenAPI `Problem.errors` text ("Present on validation_failed") matches the server; the registry text does not.
- **Spec drift** (`api/openapi.bundled.yaml`): `onboardTenant` documents 429, but operator routes are not rate limited (`internal/server/api.go:81-84`), so nava-onboard never answers 429. Recorded as an exemption in `build/coverage-exempt.json`.

Earlier runs (TICK-287 to TICK-329, previous collection format) are in [`../acceptance-runs-tick287-329.md`](../acceptance-runs-tick287-329.md).

Tokens and response bodies are left out; the run's reports (`reports/all.*`) hold short-lived bearer tokens and are not published.
