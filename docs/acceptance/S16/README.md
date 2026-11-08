# S16 acceptance kit: an auditor exports a filtered audit log and proves it complete offline (TICK-421)

Kit task TICK-448 (P5-43). Story branch `TICK-421/audit-export` at `25cd3c1` (export JSONL #510, CSV writer #512, reseed #513, seed fix #507, reader grants #508). Verified on 2026-11-08 at 12:45Z–12:58Z on a fresh stack. ADR-004 (R17, R19 to R24), ADR-001-A6, PLAN-005, grooming TICK-449.

**Result: NOT READY.** AC1, AC2 and AC4 pass, and so do the isolation checks. **AC3 fails:** the CSV export cannot be downloaded, because `format=csv` is not routed and the API returns JSONL. Fix task: TICK-624. **AC5 waits for TICK-447** (P5-42's CI-A7 job). The S13 sample files give their documented verdicts today. The final fresh-stack run happens after TICK-624 and TICK-447 merge.

## What you can now do

- Download your tenant's audit log as one file, either all of it or filtered by time, subject or domain.
- Check that file on your own laptop with `nava audit verify`, with no access to the platform. It says `ok` when nothing in the range was removed or changed. It says `tampered`, and names the line, when even one byte was.
- Only one export per tenant runs at a time. Every export is recorded in the audit log itself.
- *Not yet:* the same file as CSV for a spreadsheet (TICK-624).

## Try it (about 25 minutes: 15 to build and set up, 10 to run)

Tenants A and B in the seed need a chain head, as in S15, so the stack starts just before migration B and is then upgraded. See the S15 kit, Known limits.

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell). Optional: put other ports in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT`, `ACCEPT_UI_PORT`) when another stack holds the defaults:

   ```sh
   sh docs/acceptance/S15/tools/upgrade-setup.sh
   ```

2. Run the kit:

   ```sh
   sh docs/acceptance/S16/tools/run-kit.sh
   ```

   It builds `bin/nava` and runs the Bruno folder **S16 export** (`docs/acceptance/guide/bruno/S16/`). Then it runs `tools/kit.mjs` for the steps that need a saved file, a second concurrent request or the verifier. Everything goes to `evidence/`. In Bruno desktop you can run folder S16 by hand with the `local` environment. Save the body of S16.6 and run `bin/nava audit verify <file>`.
3. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | auditor (holds `audit.export`) | `auditor@tenant-a.test` |
   | agent, tenant A (writes the incident) | `alice@tenant-a.test` |
   | agent, tenant B (isolation) | `bob@tenant-b.test` |

You should see `api-test: PASS` and `s16 kit: PASS`. Today it ends with `FAILED 1 of 21`: the AC3 check.

## Acceptance criteria

Evidence is in `evidence/`. `S16-*.json` hold one Bruno request each, request and response, with no credentials (sign-ins are left out). `run-summary.json` lists every Bruno test, and `kit-summary.json` lists every kit check. The `*.txt` files are the verifier's output, written as in the S13 kit: the exit code, then stdout.

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| AC1 | An export filtered to one incident verifies `ok`; deleting a line or editing a byte makes it fail | ✅ | S16.4–S16.6, kit AC1 | `S16-6.json`: the export filtered to Alice's incident is 200 `application/x-ndjson`, sent as an attachment. The header records the filter, the incident is in full at its `seq`, and every other seq is a hash-only stub. The trailer says `verifiable: true`. `ac1-export-filtered.jsonl` is the same export from the kit (1 entry, 15 stubs), and `ac1-verify-ok.txt` gives **`exit 0`, `ok: seq 1..16 from genesis to head_hash 2022feaf…; 1 entries, 15 stubs, 0 unsealed`**. With the incident's line deleted (`ac1-export-line-deleted.jsonl`), `ac1-verify-line-deleted.txt` gives **`exit 1`, `tampered: line 18: count_mismatch`**. With one byte edited, the first hex digit of its `request_id` (`ac1-export-byte-edited.jsonl`), `ac1-verify-byte-edited.txt` gives **`exit 1`, `tampered: line 18 (seq 16): envelope_sha256_mismatch`**. |
| AC2 | A second export started while one runs gets 409 `export_in_progress`; the export appears in the log as an `export` entry | ✅ | kit AC2, S16.10 | Alice writes 600 incidents with a 30,000-character description, so the whole-log export is 18.8 MB. The kit opens it and pauses reading, so the server blocks on a full socket and keeps the lease. A second export then gets **409 `export_in_progress`** (`ac2-second-export-409.json`). The first one is drained to its trailer (`ac2-first-export-summary.json`: 18,827,619 bytes, 621 lines) and verifies `ok` (`ac2-verify-first-export.txt`). `ac2-export-entries.json`: the audit/export entries went from 10 to 11. The streamed export wrote one, with the request (filter, `stubs`) in its diff, and the refused one wrote none. `S16-10.json`: S16.6's export is an `audit/export` entry that names its filter. |
| AC3 | The CSV opens in a spreadsheet with no formula executed (cells starting `=`, `+`, `-`, `@` prefixed `'`) | ❌ | kit AC3 | **Fails.** `GET …/entries:export?subject_id=…&format=csv` returns 200 **`application/x-ndjson`**, the JSONL file (`ac3-format-csv-response.txt`, `ac3-export-filtered.csv`). `Exporter.WriteCSV` (TICK-445) exists and is unit tested, but neither `api/paths/audit.yaml` nor `internal/server/audit_export.go` reaches it. The unknown parameter is silently ignored. Fix task **TICK-624**. The kit already holds the check for after the fix: `text/csv`, no cell starting `=` `+` `-` `@` tab or CR without the `'`, and `nava audit verify` exits 0 on the CSV. Alice's incident has `=HYPERLINK(…)` and `@SUM(…)` text for it. |
| AC4 | `POST /api/v1/audit/chain:reseed` returns 409 `chain_alive` while sealing runs | ✅ | S16.11, kit AC4 | `S16-11.json` and `ac4-reseed-409.json`: **409 `chain_alive`**, problem+json. The seal lineage is alive: Alice's incident was sealed 4 to 14 s after it was written. A dead lineage is reseeded in P5-35's integration tests. |
| AC5 | An entry erased by the test erasure helper still verifies, with the erase entry included in the export | ⏳ | kit AC5 | Per TICK-449, the evidence is P5-42's CI-A7 job plus S13's erased and pending samples, with **no on-stack erasure until Epic 30**. S13 samples, today: `ac5-export-erased-pending.txt` gives **`exit 3`, `pending: … 1 unsealed`** with `seq 6: pending: erased; its audit/erase entry is not sealed yet`. `ac5-export-erased-without-erase-entry.txt` gives **`exit 1`, `tampered: line 8 (seq 6): erasure_unlisted`**. Both are byte for byte the outputs in `docs/acceptance/S13/expected/`. **Waiting for TICK-447** (P5-42, CI-A7: the erased entry with its sealed erase entry verifies `ok`). Its CI run gets linked here when it merges. |

Bruno run: 19 requests, all tests passed (`run-summary.json`). Kit: 21 checks, 20 passed, the AC3 check failed (`kit-summary.json`).

## Isolation check

- Bob (tenant B, no `audit.export`) gets **403** on the export (S16.12, `iso-bob-export-403.json`) and on the reseed (S16.13, `iso-bob-reseed-403.json`). The tenant is always the token's. Nothing in the request can name tenant A.
- Tenant A's whole-log export (S16.7, and the 18.8 MB kit export) holds only envelopes with `tenant_id` `…0000a1`. The id of Bob's incident, written in the same run, is in neither tenant A export.
- Alice (tenant A, an agent) gets 403 on both (S16.15, S16.17).
- Not repeated here: a tenant-B auditor exporting its own chain, and the export's database isolation (P5-42, CI-A4 export, stubs and reseed rows), which the Security & Multi-Tenancy Reviewer owns.

## Spec notes (for the Technical Lead)

- **CSV route missing (Bug, TICK-624).** TICK-445's outcome is "`format=csv` returns the same lines as JSONL as CSV". The writer landed without a route, and the OpenAPI has no `format` parameter. TICK-445's own acceptance tests are unit tests of the writer, so CI could not catch it.
- **The prefix rarely shows on a real export.** User text reaches the CSV only inside the `diff` cell, which always starts with `[`. On today's envelope no column can start with `=`, `+`, `-` or `@`. AC3's "no formula executed" therefore holds trivially for the cells that exist today. The `'` prefix is proven by CI-A15's unit test. TICK-624 asks the engineer to say whether any cell can carry a leading formula character through the API.
- **The deleted line is reported as `count_mismatch`.** The incident's line was the last in the range, so the remaining links still hold and the trailer count is what catches it. That is the documented behaviour (`audit-export-v1.md` §4.1).

## Not included

- Erasure on the stack. There is no erasure API until Epic 30. The test erasure helper (`internal/audit/erasetest/`) runs only in P5-42's CI-A7 job.
- A dead-lineage reseed on the stack. The lineage would have to be more than 10 minutes overdue. P5-35's integration tests cover it.
- 429 and 501 answers of the export, and the 15-minute cap. They are in the coverage page and P5-40's tests.
- An S16 page in the HTML guide. The Bruno folder (generated by `docs/acceptance/guide/build/collection.py`) and this kit cover the story.

## Known limits

- **No on-stack erasure until Epic 30** (TICK-449). AC5's evidence is CI-A7 plus the S13 samples.
- **A stack created at the branch cannot export for tenants A and B.** They have no chain head (S15 kit, Known limits), so the export would answer 500. Hence `upgrade-setup.sh`.
- **`upgrade-setup.sh` on Windows:** the pre-migration-B commit `080dd74` still has the `curl -o /dev/null` seed bug that TICK-609 fixed. The run applied TICK-609's `seed.sh` change to the throwaway `../s15-pre-b` worktree (`git show 8de9798f -- docs/acceptance/guide/tools/seed.sh | git -C ../s15-pre-b apply`). Linux and macOS do not need it.
- **The concurrent-export step takes most of the run** (about 8 minutes on Docker Desktop): the 600 large incidents. Without them the first export ends before the second request arrives. `S16_LOAD` sets the count.
- **The whole-log export in AC2 has 176 unsealed lines.** They are the newest load entries, not yet sealed when it started. They are reported as unsealed, outside the verified range, and the file still verifies `ok`.
- **The verified branch does not contain the latest `main`.** Merging `main` in is part of the story→main checklist (`docs/delivery/branching-strategy.md`).

## Runs (2026-11-08, fresh stack, compose project `s16acc` on ports 28080/28081/28082/29090–29095)

| Run | Setup | Result | Notes |
|---|---|---|---|
| 1, 12:45Z | `upgrade-setup.sh` at `25cd3c1` | stopped in AC1 | The kit's own bug: the byte edit looked for `content_sha256`, which is null on a record entry. Bruno S16 passed. |
| 2, 12:56Z | same stack | **AC3 fails**, all else passes | Bruno 19/19 requests pass; kit 20/21 checks pass. This run's evidence is in `evidence/`. Fix task TICK-624. |

Bruno CLI 4.3.0 (`api-test.sh`), Node.js 22, Docker Desktop, Git Bash on Windows 11. The full Bruno reports (`docs/acceptance/guide/reports/`) are not published: they hold dev-only passwords and short-lived tokens.
