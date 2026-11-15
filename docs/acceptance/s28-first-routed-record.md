# Timed kit: S28, from onboarding to the first record a workflow routed (M-X1a)

Story TICK-879, kit TICK-884 (PLAN-009 D-9-3). Story branch `TICK-879/first-routed-record`.
ADR: ADR-008 X1a, M-X1a, D9, R4 (`docs/adr/ADR-008-competitive-gate.md`); RFC-020 §8 AC-2.
Recorded runs: [`docs/quality/x1a-first-routed-record.md`](../quality/x1a-first-routed-record.md).

## What this measures

How long a **newly onboarded** tenant's first administrator takes to get from "onboarding finished" to "a record was routed by a workflow I published, on a type I published from an example". The target is 8 hours; the aim is 4.

- **Clock starts** when the onboarding call that creates the tenant and its first admin returns `201` (tour step 1). Not a seeded tenant.
- **Clock stops** at the first record whose history shows a change with `source: workflow` from that admin's workflow. The new tenant has no other workflow, so a `source: workflow` entry is the admin's.

## How to try

You are the first administrator of a customer that has just been onboarded, working from the documentation only. Follow these steps on the stack you were given, keep a stopwatch log, and stop at the first record a workflow you published has routed.

**Allowed help:** the handbook (`docs/handbook/`, open `docs/handbook/index.html` or read the pages under `docs/handbook/src/`; the product tour starts at `docs/handbook/src/tour/index.md`), this kit page, and the public API of the stack (nava-core, plus nava-onboard for the operator's onboarding call). The Bruno collection `docs/acceptance/guide/bruno/` is part of the kit and may be read.

**Not allowed:** `psql` or any other database access, operator or admin routes other than the onboarding call itself, platform-staff actions (the Keycloak master admin, `docker` commands, restarting services), and reading Go code or anything else outside the handbook and this kit. A step you cannot finish with the allowed help is a **finding**: write it down in the log ("needed help", and what you would have needed) instead of working around it.

**The one exception, sign-in on the local stack.** A real customer signs in through its own company login. The local stack has none, so the first admin cannot sign in with the allowed help alone. The kit's stand-in for the company login is Bruno `S28.3` to `S28.5` (the Keycloak master admin creates the user that the company login would create, links it to the tenant's login and adds it to the tenant's organization), after which the admin signs in with the password grant (`S28.6`). Using it is a platform-staff action: log that step as "needed help: platform staff (stand-in for the company login)", then use it, so the rest of the path can be timed. Do not use the master admin for anything else.

**Before the clock (the operator's part).** The stack is up and its operator tunnels are open (`sh docs/acceptance/guide/tools/tunnels.sh up`): nava-core on `http://127.0.0.1:18080`, Keycloak on `http://127.0.0.1:18081`, nava-onboard on `http://127.0.0.1:18082`. Get an operator token as the tour's "Before you start" describes. Pick a domain nobody has used, such as `x1a-<something random>.test`; the first admin's address is `admin@` that domain.

Do the steps by hand (curl, or single Bruno requests). Do not run the Bruno folder `S28` as a whole for a timed run: it is the kit's automated check, and it would not measure the handbook path.

1. **Onboard the tenant** (tour step 1). Your domain, your admin address, a new `Idempotency-Key`. The clock starts when it answers `201`; write that time down.
2. **Sign in as the first admin** (tour step 2). See the exception above. Then confirm with `GET /api/v1/me` that the platform knows you as the new tenant's admin (in `/admins`); the invite is placed by the next group-sync cycle (tour step 4), so this can take up to one cycle.
3. **Publish a type from an example** (tour step 6, and `docs/handbook/examples/`). Use one of the handbook's example types, through the public type-definition routes.
4. **Publish a workflow on that type** (tour step 8). Write the definition yourself from the handbook's example: on `record.created`, an `update_record` step that sets `state`. Use a key of your own.
5. **Create a record of that type** (tour step 9).
6. **Wait for it to route.** Read the record until the workflow's change shows, then read its history: the clock stops at the entry with `source: workflow`. Give up 5 minutes after the create (the handbook's example routes in about a minute) and log it as a blocker.

**The stopwatch log.** One row per step; times in UTC to the second; minutes to one decimal.

| Step | Start | End | Minutes | Help used | Blocker |
|---|---|---|---|---|---|
| 1 Onboard | | | | | |
| 2 Sign in | | | | | |
| 3 Type from an example | | | | | |
| 4 Workflow | | | | | |
| 5 Record | | | | | |
| 6 Routed | | | | | |
| **Total** (step 1 end to step 6 end) | | | | | |

"Help used" names the handbook page or kit section that got you through the step. "Blocker" is empty, or what stopped you and what you needed. Write down every wrong turn that cost time (a `4xx` you had to work out, a page that sent you the wrong way): they are the findings.

## The automated check

The Bruno folder `S28 first routed record (M-X1a)` runs the same path in about 80 seconds plus one group-sync cycle:

```sh
sh docs/acceptance/guide/tools/tunnels.sh up
sh docs/acceptance/guide/tools/api-test.sh S28
```

| Request | Step | Asserts |
|---|---|---|
| S28.1 | operator token | `200`, issuer |
| S28.2 | 1 onboard (clock start) | `201`, `state: active` |
| S28.3 to S28.5 | 2 stand-in for the company login (platform staff) | `201`, `204`, `201` |
| S28.6 | 2 the admin signs in | `200`; the token names only the new tenant's organization |
| S28.7 | 2 placed in `/admins` | `200` `/me` is the new tenant, `/admins`; polls up to `s28_place_limit_s` (360 s) |
| S28.8 | 3 starter types | `200`, `incident` at version 1 |
| S28.9 | 3 type from the example | `201`, `incident` version 2 from `docs/handbook/examples/incident.json` |
| S28.10 | 4 workflow | `201`, `s28_route` version 1 on `incident` |
| S28.11 | 5 record | `201`, `new`, schema version 2 |
| S28.12 | 6 routed | `200`, `in_progress`; polls once a second, fails 300 s after the create |
| S28.13 | 6 history (clock stop) | `200`, newest entry `source: workflow` changing `/data/state`; logs the total, below 8 h |

In CI the stack case `s28_first_routed_record` (`deploy/compose/test/cases/s28_first_routed_record.sh`) runs the folder on a fresh stack and uploads its report.

## Known limitations

- The sign-in step on the local stack is a platform-staff stand-in (finding F1 in the quality page). A real tenant's admin signs in through the company login in a browser; this kit cannot time that.
- The record history says `source: workflow` but not which workflow. The kit relies on the new tenant having one definition.
- There is no screen: every step is the REST API.
