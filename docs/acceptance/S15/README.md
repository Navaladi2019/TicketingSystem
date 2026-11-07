# S15 acceptance kit: each tenant's audit entries are sealed into a hash chain (TICK-420)

Kit task TICK-443 (P5-36), timings updated for ADR-004-E7 by TICK-603 (P5-36b). Story branch `TICK-420/audit-sealer` at `7fb7727` (this kit, PR #496; the AC4 fix TICK-598, PR #497; the E7 sealer change TICK-602, PR #501). Verified on 2026-11-07 at 16:48Z on a fresh stack. ADR-004 (R13 to R17, R-A2), ADR-004-E7, ADR-001-A6, PLAN-005, PLAN-005-E7, grooming TICK-449.

**Result: PASS.** All four acceptance criteria pass on a fresh stack. The final run had 21 requests and 71 tests, all passed (`run-summary.json`). It is after TICK-598 (AC4) and TICK-602 (E7): the second write after idle (S15.9) was sealed **9.3 s** after it was written, where it waited 80 s before E7.

## What you can now do

- Every audit entry of a tenant gets a place in that tenant's chain (`seq`) and a hash chained to the entry before it, without anyone doing anything.
- While a tenant is busy, its entries are sealed about 10 seconds after they are written. A tenant that has been quiet is still sealed within 5 minutes of its next write, and from the write after that on it is back to about 10 seconds.
- If the worker that seals dies, nothing is lost or numbered twice. The next worker carries on where the chain stopped.
- A newly onboarded tenant is sealed from its first entry on.

## Try it (about 30 minutes: 10 to build and set up, 20 to run)

Tenants A and B in the seed have to get their chain head the way an existing tenant does in production, from migration B (P5-32). So the stack starts just before migration B and is then upgraded (see Known limits).

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell). Optional: put other ports in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT`) when another stack holds the defaults:

   ```sh
   sh docs/acceptance/S15/tools/upgrade-setup.sh
   ```

   It starts a fresh stack (volumes deleted) at `080dd74` in the sibling worktree `../s15-pre-b` and seeds it. Tenants A and B then have **no** chain head. It then upgrades the same stack to this checkout: the `migrate` service applies migration B, which gives both a head and a first seal start row. Last, it runs `tools/seed.sh` and `tools/tunnels.sh up`, and checks `/readyz` on nava-core and nava-worker.
2. Run the kit:

   ```sh
   sh docs/acceptance/S15/tools/run-kit.sh
   ```

   It runs the Bruno folders **S15 seal** (`bruno/S15/`), **S15 seal after a worker kill** (`bruno/S15K/`) and **S15 seal after idle** (`bruno/S15I/`), in that order. In between it runs `docker compose kill nava-worker` / `docker compose up -d nava-worker` and the SQL check, and writes everything to `evidence/`. In Bruno desktop you can run folder S15 by hand with the `local` environment. S15K and S15I only make sense after the kill and the idle wait that the script does.
3. Users (dev-only seed users, passwords in `bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | operator (onboards the new tenant) | `operator@nava.test` |
   | auditor | `auditor@tenant-a.test` |
   | agent, tenant A | `alice@tenant-a.test` |
   | agent, tenant B | `bob@tenant-b.test` |

You should see `api-test: PASS` for each folder and `s15: PASS` at the end.

## Acceptance criteria

The evidence is in `evidence/`. There is one JSON file per API request, with the request and the response and no credentials (sign-ins are left out). `run-summary.json` lists every test of the run. The `*.txt` files hold the database checks.

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| AC1 | A new tenant is onboarded; an agent creates an incident; within about 10 s its audit entry is sealed (`seq`, `entry_hash`) | ✅ | S15.5–S15.11 | **New tenant:** `S15-5.json` (201, tenant `4da7f908-…`). `ac1-seal-latency.txt`: its three onboarding entries sealed **1.1 s** after they were written. **Agent's incidents:** the first write after idle (`S15-7.json`) waited for the run already booked at the backed-off interval: **48.9 s**, under AC3's 300 s bound. That run's A1 saw the entry pending and booked the next run 10 s ahead (ADR-004-E7), so the second write was sealed **9.3 s** after it was written (`S15-9.json`, `seq` 5; the check fails above 30 s) and the third **9.9 s** after (`S15-11.json`, `seq` 6). `S15-13.json`: tenant A's chain from `seq` 4 on has no gap and no duplicate. `entry_hash` is checked in the database (`ac2-seq-check.txt`, `head_hash_ok`, `broken_links`), not in the API (see Spec notes). |
| AC2 | Kill the worker mid-run, start it again: the backlog is sealed with no gap and no duplicate `seq` | ✅ | run-kit.sh, then S15K.2 | `ac2-kill-restart.txt`: while Alice and Bob wrote about 13 incidents/s each (`load.json`: 790 in A, 788 in B, all 201), `nava-worker` was SIGKILLed at 16:50:47Z. That was two seconds before tenant A's next run was due, with 106 entries unsealed in A and 204 in B. It stayed down 28 s. Writes went on, the unsealed backlog grew to 475 and 573, and `head_seq` did not move. After `up -d` the backlog was sealed. `ac2-seq-check.txt` (`tools/seq-check.sql`), for every tenant: 0 unsealed, 0 duplicate `seq`, 0 missing `seq` (1..796 in A, 1..792 in B, 1..3 in the new tenant), 0 broken `prev_hash` links, head on the last entry, `ok = t`. 0 entries without a head. `S15K-2.json`: the 593 tenant-A entries written since the kill, read through the API, have consecutive `seq` 204..796. |
| AC3 | A tenant left idle is still sealed within 300 s of its next write | ✅ | run-kit.sh, then S15I.3–S15I.4 | `ac3-idle.txt`: tenant A at `seal_interval_s = 300`. Alice's write, the *first* after idle, was sealed by the run already booked, **139.6 s** later (`S15I-4.json`, `seq` 797); that run then booked the next one 10 s ahead (`seal_interval_s = 10` after the write, ADR-004-E7). Tenant B's backlog after the kill waited for B's back-off run, max **123.1 s** (`ac2-seal-latency.txt`). |
| AC4 | `audit_unsealed_oldest_seconds` and `audit_seal_failures_total` are on `/metrics` | ✅ | S15.15 | `S15-15.json`: nava-worker's `/metrics` has `audit_unsealed_oldest_seconds … 0` and `audit_seal_failures_total` at 0 for each `reason` (`error`, `reschedule`, `unsealable`), both with a `# HELP` line and no tenant label. Before TICK-598 (PR #497) the counter was missing until the first failure (first verification, 13:23Z). |

Not asserted, per TICK-449: the exact A1/A2 kill window (P5-34b's fault-injection test covers it), poison batches and reseed (S16).

## Isolation check

- Bob wrote an incident in tenant B in the same window (S15.12). It is not in tenant A's chain, and its id appears nowhere in the auditor's response (S15.13).
- Bob asking for tenant A's incident history gets `404 not_found` (S15.14).
- Each tenant has its own chain: `seq` starts at 1 per tenant and the heads are per tenant (`ac2-seq-check.txt`). The kill and restart sealed A's and B's backlogs into their own chains, with no entry in the other's.
- Not repeated here: a tenant-B auditor, and Bob's 403 on the timeline (S14 kit, S14.27).

## Spec notes (for the Technical Lead)

- **ADR-004-E7 (TICK-600, implemented by TICK-602).** A1 now books the next run 10 s ahead when the tenant has unsealed entries, so an entry written after idle is sealed within the delay already booked when it was written (≤ 300 s) plus one run. The first verifications measured the R16 drift this fixes: S15.9 waited 80 s and 160 s. In this run it was 9.3 s. The 300 s bound (AC3) still applies to the *first* write after idle.
- **The story text says `sealed: true` and `entry_hash`.** The API has no `sealed` field and no `entry_hash`: S15 has no `api/` change (D-5-9). A sealed entry is shown as `seq` plus `sealed_at`, and the hash chain is checked from the database. The export (S16) carries the hashes.

## Not included

- `chain:reseed` and its kit (S16, D-5-9).
- Recomputing `entry_hash` from the envelope: that is the offline verifier on an export (`cmd/nava verify`, S16). The kit checks the chain links (`prev_hash` = previous `entry_hash`) and the head.
- An S15 page in the HTML guide. The Bruno folders (generated by `docs/acceptance/guide/build/collection.py`) and this kit cover the story.
- A CI stack case for this kit. The kill step and the 5-minute idle wait are too slow for the stack job. CI-A16's kill and stack cases run in CI in P5-34b's tests (`internal/audit/seal/workflow_integration_test.go`, `internal/workerstack/stack_test.go`).

## Known limits

- **No reseed in S15** (TICK-449). A lineage that dies or is poisoned stays stopped until S16's `chain:reseed`. `AuditDeadLineage` and `AuditSealLag` alert on it. The reseed row of CI-A16 is tested only in P5-35 and P5-43.
- **Migration B needs a maintenance window** (TICK-442, R-A6). It holds the six R27 ACCESS EXCLUSIVE locks for about **4.3 s at 10,000 tenants** (4.29 s, 300,972 versions) and **0.48 s at 1,000 tenants** (30,093 versions), on postgres 16.10 with 16 CPUs. For that time, tenant resolution, S8 publishes, audited writes and the relay wait. Evidence: the TICK-442 RESULT comment, `docs/data/audit-log.md`.
- **A stack created at the branch never seals the seeded tenants.** `tools/seed.sh` writes tenants A and B after the migrations ran. Migration B has already run, and onboarding gives a head only to the tenants it creates, so A and B have no `audit_chain_head` (checked: 2 tenants, 0 heads). Their entries stay unsealed. Hence `upgrade-setup.sh`. Production tenants are not affected: they exist before migration B, or they are onboarded.
- **The access token lives a few minutes,** and the S15 polls wait up to 330 s, so every S15 request signs in again when its token is about to expire (`bruno/lib/s15.js`).
- **`tools/seed.sh` on Windows with the mingw64 curl:** `curl -o /dev/null` exits 23 under `MSYS_NO_PATHCONV=1`, as in the S14 kit. The runs used a host-local wrapper that maps `/dev/null` to `NUL`. The script is unchanged.
- **The verified branch does not contain the latest `main`.** Merging `main` in is part of the story→main checklist (`docs/delivery/branching-strategy.md`).

## Runs (2026-11-07, each from a fresh stack)

| Run | Setup | Result | Notes |
|---|---|---|---|
| 1, 12:48Z | `upgrade-setup.sh` | stopped after S15 | The kit's own bugs: the operator signed in with the wrong client (401), and the second incident's 60 s bound did not allow for the R16 drift fixed by E7. Showed AC4 failing. |
| 2, 13:05Z | `upgrade-setup.sh` | stopped after S15 | The kit's own bugs: access tokens expired during the polls (401), and the Bruno report path was not converted on Windows. |
| 3, 13:23Z | `upgrade-setup.sh` at `d3fb7c3` | AC1–AC3 pass, AC4 fails | 21 requests, 69/71 tests. AC4: `audit_seal_failures_total` missing. Fix task TICK-598. |
| 4, 15:02Z | `upgrade-setup.sh` at `9e95869` (after TICK-598) | PASS | 21 requests, 71/71 tests. Before E7: S15.9 sealed after 80 s. |
| 5, 16:48Z | `upgrade-setup.sh` at `7fb7727` (after TICK-602, E7) | **PASS** | 21 requests, 71/71 tests. S15.9 sealed after 9.3 s, bound 30 s (TICK-603). This run's evidence is in `evidence/`. |

Bruno CLI 4.3.0 (`api-test.sh`), Node.js 22, Docker Desktop, Git Bash on Windows 11. The stack ran as compose project `s15acc` on ports 28080/28081/28082/29090–29095, beside another local stack. The full reports (`docs/acceptance/guide/reports/`) are not published: they hold dev-only passwords and short-lived tokens.
