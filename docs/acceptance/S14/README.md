# S14 acceptance kit: one audit log (TICK-419)

Kit task TICK-437 (P5-20). Story branch `TICK-419/one-audit-log`, verified at `71ac70b` (code) on 2026-11-04. ADR-004, PLAN-005, grooming TICK-449.

**Result: PASS.** All four acceptance steps pass on a fresh stack, twice: by hand, then with `tools/upgrade-setup.sh` on a wiped stack. Each run sent 29 requests and passed 95 of 95 tests, with 0 failures.

## What you can now do

- As a tenant admin or a tenant auditor, see every entity-type publish and every incident change of your tenant on one timeline, newest first, with who did it, from where, and what changed.
- Entity-type versions that existed before this release are on that timeline too, marked as backfilled.
- As an agent, see an incident's own history, but not the tenant's timeline. Another tenant sees nothing of yours.
- A publish that would store a secret value in an entity-type schema is refused. The refusal names the path and never the value.

## Try it (about 10 minutes, most of it image builds)

Step 2 is about versions published *before* this release, so the stack has to start before S14 and be upgraded. A stack created at the story branch has no such versions.

1. From the repository root, on the story branch, with Docker running (Git Bash or any POSIX shell):

   ```sh
   sh docs/acceptance/S14/tools/upgrade-setup.sh
   ```

   This script:
   1. Starts a fresh stack at `origin/main` (no `audit_log`), with volumes deleted, in a sibling worktree `../s14-pre-release`.
   2. Runs that commit's `tools/seed.sh`: tenants A and B with `incident`, `request` and `change` at version 1.
   3. Has `admin@tenant-a.test` publish incident version 2 (`tools/prepublish.mjs`).
   4. Upgrades the same stack (same volumes) to this checkout. The `migrate` service applies the 13 S14 migrations, the backfill among them.
   5. Applies the S14 realm delta (`tools/realm-delta.mjs`): the `auditors` groups and `auditor@tenant-a.test`. Keycloak imports `realm-nava.json` only into a realm that does not exist yet, so an upgraded realm needs them added.
   6. Runs this checkout's `tools/seed.sh` and `tools/tunnels.sh up`, then checks `/readyz`.
2. Run the folder: `sh docs/acceptance/guide/tools/api-test.sh S14`. In Bruno desktop, open `docs/acceptance/guide/bruno/` with the `local` environment and run folder **S14 audit**.
3. Users (dev-only seed users, passwords in `bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | admin | `admin@tenant-a.test` |
   | auditor | `auditor@tenant-a.test` |
   | agent A | `alice@tenant-a.test` |
   | agent B | `bob@tenant-b.test` |

   These follow the TICK-449 grooming. The Contract line's `agent@tenant-*` names are not seeded.

You should see `api-test: PASS`, plus two log lines: `S14.20 timeline (newest first): record/update/record, record/create/record, config/publish/entity_definition` and `S14.22 backfilled: publish v2 user, create v1 onboarding, create v1 onboarding, create v1 onboarding`.

## Acceptance criteria

Evidence is in `evidence/`: one JSON file per request from S14.13 on, holding the request and the response, with no tokens. `run-summary.json` lists every test of the run, and `db-backfill-per-tenant.txt` holds the database check. S14.1–S14.10 are the reader's contract tests from TICK-435 and also pass.

| # | Criterion | Result | Bruno requests | Evidence |
|---|---|---|---|---|
| AC1 (CI-A11) | The admin publishes a new incident-type version and an agent creates and updates an incident. As the auditor, `GET /api/v1/audit/entries` shows the publish and both record changes on one timeline, with actor, source and diff | ✅ | S14.15–S14.21 | `S14-20.json`: newest first, update (v1→2, `replace /priority 1`), create (v1, `add ""`), publish (v2→3, `add /properties/impact_s14_<run>`). Each has `source: user`. The publish is by the admin and the record entries are by Alice. The record entries' `entity_definition_id` is the publish's `subject_id`. `S14-21.json`: the admin sees the same order. |
| AC2 | Entity-type versions published before this release appear as backfilled entries | ✅ | S14.22 | `S14-22.json`: three `create` v1 entries (`actor_system: onboarding`, `source: onboarding`, no diff, `content_sha256`). One `publish` v2 by the admin (`source: user`), published before the upgrade. All have `backfilled: true`; S14.16's live publish is `false`. `db-backfill-per-tenant.txt`: tenant A has 4 pre-upgrade versions and 4 backfilled entries, tenant B 3 and 3, each matching a version of the same tenant. |
| AC3 | `agent@tenant-a` gets 403 on `/api/v1/audit/entries` but sees the incident's history; `agent@tenant-b` sees nothing of tenant A | ✅ | S14.23–S14.27, S14.19–S14.20 | `S14-23.json`: Alice 403 `forbidden`. `S14-24.json`: Alice reads the history (update, create, by her). `S14-25.json`: Bob 403. `S14-27.json`: Bob reading the history of A's incident gets a 404 byte-identical to the 404 for an id that exists nowhere (`S14-26.json`). |
| AC4 | A publish whose schema puts a secret value in the content is refused with `config_secret_in_content`, naming the path only | ✅ | S14.28–S14.29 | `S14-28.json`: 422, `errors[0].pointer` is `/definition/schema/properties/api_token_s14_<run>/default`, and the value appears nowhere in the body. `S14-29.json`: the type is still at S14.16's version and no config entry was written. |

Not asserted, per the TICK-449 grooming: CI-A2 for configuration tables (D-5-8, whose commit check arrives in S15).

## Isolation check

- Bob created an incident in tenant B in the same window (S14.19). It is not on tenant A's timeline, and its id and text appear nowhere in the auditor's response (S14.20).
- Bob reading tenant A's incident history gets the same 404 bytes as for an id that exists nowhere (S14.26, S14.27).
- Bob has no `audit.read` in tenant B, so the timeline is a 403 for him (S14.25).
- The backfill wrote each tenant's versions into that tenant's own log only (`db-backfill-per-tenant.txt`).
- Not checked here: a tenant-B auditor. That is covered by P5-19 (TICK-449 grooming).

## Not included

- The commit check for configuration tables (CI-A2 on `entity_definitions` and `entity_definition_versions`): S15 (D-5-8).
- Migration B (P5-32): the second backfill run, the commit trigger on the entity tables and the chain head seeding. Until then, a version seeded by onboarding after the S14 migrations has no audit entry (see Known limits).
- An S14 page in the HTML guide (`docs/acceptance/guide/s*.html`). The Bruno folder and this kit cover the story.

## Known limits

- **AC2 needs the upgrade setup.** On a stack created at the story branch, `tools/seed.sh` writes the version-1 rows after the migrations, so nothing is backfilled and S14.22 fails by construction. That is why "Try it" upgrades a pre-S14 stack.
- **Tenants that existed before S14 have no auditors group after an upgrade.** Onboarding creates `/t/<tenant>/auditors` for new tenants only, and nothing creates it, or its `tenant_auditor_group` row, for existing ones. On the dev stack, `realm-delta.mjs` and `seed.sh` do it by hand. In a real upgrade, only tenant admins hold `audit.read` until someone creates the group. Routed to the Technical Lead as a question; not a criterion of this story.
- **Onboarding's version-1 seed writes no audit entry** (`internal/entitytypes` `Store.Seed`). A tenant onboarded after the S14 migrations and before migration B has its three templates off the timeline until B re-runs the backfill (ADR-004 R25).
- **`tools/seed.sh` on Windows with the mingw64 curl:** `curl -o /dev/null` fails with exit 23 because the script exports `MSYS_NO_PATHCONV=1`, and `set -e` stops the seed. The run used a host-local wrapper that maps `/dev/null` to `NUL`. The script itself is unchanged.
- The verified branch does not contain the latest `main` (`2fd9c74`, TICK-523). Merging `main` in is part of the story→main checklist (`docs/delivery/branching-strategy.md`).

## Runs

| Run | Setup | Requests | Tests | Result |
|---|---|---|---|---|
| 1, 17:41Z | Same steps as `upgrade-setup.sh`, by hand: `main` stack, seed, publish v2, upgrade, realm delta, seed, tunnels | 29 | 95/95 | PASS |
| 2, 17:42Z | Run 1's stack, folder again (re-runnable) | 29 | 95/95 | PASS |
| 3, 17:45Z | `upgrade-setup.sh` from a wiped stack (`pgdata` removed) | 29 | 95/95 | PASS; this run's evidence is in `evidence/` |

Bruno CLI 4.3.0 (`api-test.sh`), Node.js 22, Docker Desktop, Git Bash on Windows 11. The full reports (`docs/acceptance/guide/reports/`) are not published: they hold dev-only passwords and short-lived tokens.
