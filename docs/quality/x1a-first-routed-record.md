# M-X1a: onboarding to the first workflow-routed record (S28, TICK-884)

ADR-008 X1a and its measurement M-X1a (D9, R4); RFC-020 §8 AC-2; PLAN-009 D-9-3.
Kit and "How to try": [`docs/acceptance/s28-first-routed-record.md`](../acceptance/s28-first-routed-record.md).
Bruno folder: `docs/acceptance/guide/bruno/S28/` (generated from `build/collection.py`, `s28()`).

**Target:** 8 hours from the onboarding `201` to the first record a workflow the new admin published has routed. **Aim:** 4 hours.

## Scope

- **Clock starts** when the onboarding call (`POST /api/v1/operator/tenants` on nava-onboard, tour step 1) returns `201` for a new tenant.
- **Clock stops** at the first record whose history has an entry with `source: workflow`. The new tenant has no other workflow definition, so that entry is the admin's workflow.
- **Path:** sign in as the first admin, publish a type from a handbook example, publish a workflow on it, create a record, wait for the route.
- **Allowed help:** the handbook, the kit, the public API. **Not allowed:** `psql`, operator routes beyond the onboarding call, platform-staff actions, reading Go code. A step that needs one of those is a finding.

## Risks, ranked, and what covers each

| # | Risk | Covered by | Layer |
|---|---|---|---|
| 1 | A newly onboarded tenant cannot reach a routed record at all: onboarding, the invite, group sync, the type publish, the workflow publish and the route do not compose for a tenant that is not seeded. | Bruno `S28.1`–`S28.13`, CI stack case `s28_first_routed_record` | end to end (stack) |
| 2 | The handbook does not get a new admin through the path without help, so M-X1a is far over 8 h for a real customer. | The timed runs below; each finding is a follow-up | agent proxy, then human |
| 3 | The route never happens and the kit hangs. | `S28.12` polls at most 300 s after the create, then fails; with no record it reads once and fails; every call inside the `S28.7` and `S28.12` loops is bounded at 15 s, and a timed-out call counts as "not yet" | end to end (stack) |
| 4 | The first admin's token expires while group sync places them (token 5 min, default cycle 5 min). | `S28.7` signs in again each round, polls up to `s28_place_limit_s` (360 s) | end to end (stack) |

## Fixtures

- Per pass: a new run id, so a new domain `s28-<8 hex>.test`, a new tenant and the admin `admin@` that domain, with a dev-only password made up for the run and kept in a Bruno runtime variable.
- The type body is `docs/handbook/examples/incident.json`'s `definition`, read by the generator at build time. The workflow is the handbook's example (tour step 8) under the key `s28_route`.
- The stand-in for the company login (`S28.3`–`S28.5`) uses the dev-only Keycloak master admin of `environments/local.bru`.

## Deliberately not covered, and why

- **The real brokered sign-in.** The local stack has no customer IdP, and onboarding refuses private IdP addresses. `TestS4Acceptance` and `TestOnboardedAdminSignsInAndMeShowsTenant` (`internal/identitystack`) cover the browser login through a test realm. The kit replays its result through the Keycloak admin API instead, and counts the step as `needed help` (F1).
- **Database checks.** The kit stays on the public API, as the timed admin must. The S11b stack case already checks the start rows and `workflow_ref` behind a route.
- **Human reading time.** The agent proxy reads and types far faster than a person; the human run measures that.

## How to run

- Locally, on a fresh stack with the tunnels up (story acceptance or a timed run, not a CI substitute): `sh docs/acceptance/guide/tools/tunnels.sh up`, then `sh docs/acceptance/guide/tools/api-test.sh S28`.
- In CI: the `stack` job runs the case `deploy/compose/test/cases/s28_first_routed_record.sh` (order 61, serial) and uploads `s28-first-routed-record-report`.
- A timed run: give the runner only the kit's "How to try", the handbook and the stack URL, and record it below.

## Runs

### Run 1: agent proxy

- **Runner:** `agent proxy`, a fresh subagent that got only the kit's "How to try" text, the handbook and the stack URL (not the task body or this page).
- **Date:** 2026-11-15.
- **Commit:** story branch `TICK-879/first-routed-record` at `c461d547` plus this kit (branch `TICK-884/x1a-timed-run`).
- **Stack:** fresh local compose stack (`docker compose build`, `docker compose up --wait` from a clean state), `NAVA_GROUPSYNC_INTERVAL=15s`, tunnels up. No `seed.sh`. Before the run the Bruno folder `S28` passed on the same stack: 13 of 13 requests, 30 of 30 tests, routed 60 s after the create.
- **What it published:** tenant `709f9f61-baf5-51a7-a2aa-01902f715c64`; `incident` version 2 from `docs/handbook/examples/incident.json`; workflow `x1a_triage_ea468c9b` version 1 (`record.created` on `incident`, `update_record` sets `state` to `in_progress`, no timer); record `INC0000001`. The `source: workflow` history entry is at `2026-11-15T07:21:18.231Z`.

| Step | Start | End | Minutes | Help used | Blocker |
|---|---|---|---|---|---|
| 1 Onboard | 07:19:46 | 07:20:01 | 0.3 | Tour index "Before you start", tour step 1 | none |
| 2 Sign in | 07:20:05 | 07:20:36 | 0.5 | **Needed help: platform staff** (stand-in for the company login, Bruno S28.3–S28.5, with `lib/nava.js` and `environments/local.bru`), S28.6, tour step 2 | F1; one `401` while waiting for group sync (F2) |
| 3 Type from an example | 07:20:40 | 07:20:55 | 0.3 | Tour step 6, `docs/handbook/examples/` | one `409` first (F3) |
| 4 Workflow | 07:20:59 | 07:21:07 | 0.1 | Tour step 8 | none (F4, F5) |
| 5 Record | 07:21:13 | 07:21:18 | 0.1 | Tour step 9 | none |
| 6 Routed | 07:21:18 | 07:21:24 | 0.1 | Tour steps 8 and 9 | none |
| **Total** (onboarding `201` to the routed record) | 07:20:01 | 07:21:24 | **1.4** | | |

**Reading the total.** 1.4 minutes is far below the 8 h target, but it is the time of an agent that sends a request right after reading a page. It shows the handbook path works end to end, with one platform-staff step. It does not show how long a person takes. The runner's workflow had no timer, so the route took 0.16 s; the handbook example's one-minute timer would add a minute.

### Run 2: acceptance run

| Step | Start | End | Minutes | Help used | Blocker |
|---|---|---|---|---|---|
| 1 Onboard | | | | | |
| 2 Sign in | | | | | |
| 3 Type from an example | | | | | |
| 4 Workflow | | | | | |
| 5 Record | | | | | |
| 6 Routed | | | | | |
| **Total** | | | | | |

### Run 3: human (if the human does one)

| Step | Start | End | Minutes | Help used | Blocker |
|---|---|---|---|---|---|
| 1 Onboard | | | | | |
| 2 Sign in | | | | | |
| 3 Type from an example | | | | | |
| 4 Workflow | | | | | |
| 5 Record | | | | | |
| 6 Routed | | | | | |
| **Total** | | | | | |

## Findings, follow-ups for the Technical Lead

Ordered by impact on M-X1a. F1 is the only step that needed help.

| # | Severity | Finding | Where | Follow-up |
|---|---|---|---|---|
| F1 | **Needed help** (blocks a staff-free run on the local stack) | A newly onboarded tenant's first admin cannot sign in with the allowed help. The admin has no local account; sign-in goes through the tenant's company login, and the local stack has none (onboarding refuses private IdP addresses). The kit's stand-in uses the Keycloak master admin, and running it by hand also needs `lib/nava.js` (`kcAdmin()`), the master-admin credentials in `environments/local.bru`, and the organization lookup hidden in S28.5's pre-request script. The handbook does not describe first sign-in on the local stack at all. | Tour steps 1 and 2 | Decide the local path: a stand-in IdP realm the operator creates at onboarding on a dev stack (a `make` target), or a handbook section "first sign-in on the local stack" that names the stand-in. Until then every local M-X1a run carries this step as needed help. |
| F2 | Spec drift (handbook) | Before group sync places a new admin, `GET /api/v1/me` answers `401 tenant_unresolved`. Tour step 2 does not say so, and the troubleshooting row for `tenant_unresolved` says "not seeded … run `seed.sh`", which is wrong for an onboarded tenant (the answer is: wait one group-sync cycle, up to 5 min by default). | `docs/handbook/src/tour/02-sign-in.md`, `docs/handbook/src/operations/troubleshooting.md:32` | Add the "wait one cycle" case to tour step 2 and the troubleshooting row. |
| F3 | Spec drift (handbook) | Every new tenant already has `incident` (tour step 1), so `POST /api/v1/entity-types` with the handbook's `incident.json` answers `409 entity_type_exists`. No page says how to publish an example over a seeded type; the runner found it from one sentence in tour step 6 (`POST /entity-types/{key}/versions` with `{"definition": …}`). The examples README describes it only through `seed-examples.sh`, written for tenant A. The other examples (`loan_application`) have no default access rule and need group ids for `__ADMINS__`. | `docs/handbook/src/tour/06-create-an-entity-type.md`, `docs/handbook/examples/README.md` | Show publishing `incident.json`'s definition as the next version of `incident` in tour step 6, for a new tenant. |
| F4 | Bug (handbook) | The tour step 8 curl sends `$ADMIN_TOKEN`; the tour's setup defines `$ADMIN`. Pasted as written, it sends an empty bearer token (`401`). | `docs/handbook/src/tour/08-workflows.md:26` | Use `$ADMIN`. |
| F5 | Spec drift (handbook) | Tour step 8 says "create an incident as an agent (step 9)"; a newly onboarded tenant has no agent user. The admin can create it (the runner did). | `docs/handbook/src/tour/08-workflows.md` | Say "as the admin, or an agent" and note that a new tenant has only its admin. |
| F6 | Preference | Tour step 8 says it "was not re-run against a tour stack". This run confirms the publish response and the `source: workflow` history entry on a new tenant. | `docs/handbook/src/tour/08-workflows.md` | Drop the caveat when the page is next re-run. |
| F7 | Missing test (observability of the stop condition) | The history entry says `source: workflow` but not which workflow or version wrote it. The clock-stop rule "from that admin's workflow" holds here only because the new tenant has one definition. | `RecordChange` in `api/openapi.bundled.yaml` | Not a blocker for M-X1a. If tenants with several workflows are measured later, the history would need the definition key. |
