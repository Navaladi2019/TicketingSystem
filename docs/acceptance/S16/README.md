# S16 acceptance kit: an auditor exports a filtered audit log and proves it complete offline (TICK-421)

Kit task TICK-448 (P5-43). Verified on 2026-11-08 at 16:28Z on a fresh stack built from `19899341`: this kit on top of the story branch `TICK-421/audit-export` at `91b84143`. That head holds export JSONL #510, the CSV writer #512, reseed #513, the seed fix #507, the reader grants #508, CSV routing #516 (TICK-624) and erasure plus export isolation #515 (TICK-447). The Go code on the stack is `91b84143`'s. References: ADR-004 (R17, R19 to R24), ADR-001-A6, PLAN-005, grooming TICK-449.

**Result: PASS.** All five acceptance criteria and the isolation check pass. The Bruno folder ran 21 requests and 65 tests, all passed (`run-summary.json`). `kit.mjs` ran 33 checks, all passed (`kit-summary.json`). Per TICK-449, AC5's evidence is P5-42's CI-A7 job plus the S13 samples. There is no on-stack erasure until Epic 30.

## What you can now do

- Download your tenant's audit log as one file, as JSONL or CSV, with all of it or only what matches a time window, a subject or a domain.
- Check that file on your own laptop with `nava audit verify`, with no access to the platform. It says `ok` when nothing in the range was removed or changed. It says `tampered`, and names the line, when even one byte was.
- Open the CSV in a spreadsheet without any cell running as a formula.
- Only one export per tenant runs at a time, and every export is recorded in the audit log itself.

## Try it (about 25 minutes: 15 to build and set up, 10 to run)

Tenants A and B in the seed need a chain head, as in S15, so the stack starts just before migration B and is then upgraded (S15 kit, Known limits).

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell). If another stack holds the default ports, put other ones in `.env`: `COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT` and `ACCEPT_UI_PORT`. On Windows, see Known limits first.

   ```sh
   sh docs/acceptance/S15/tools/upgrade-setup.sh
   ```

2. Run the kit:

   ```sh
   sh docs/acceptance/S16/tools/run-kit.sh
   ```

   The script does three things, and writes everything to `evidence/`:
   - It builds `bin/nava`.
   - It runs the Bruno folder **S16 export** (`docs/acceptance/guide/bruno/S16/`).
   - It runs `tools/kit.mjs` for the steps that need a saved file, a second concurrent request or the verifier.

   In Bruno desktop, run folder S16 by hand with the `local` environment. Save the body of S16.6 (JSONL) or S16.20 (CSV), then run `bin/nava audit verify <file>`.
3. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user |
   |---|---|
   | auditor (holds `audit.export`) | `auditor@tenant-a.test` |
   | agent, tenant A (writes the incidents) | `alice@tenant-a.test` |
   | agent, tenant B (isolation) | `bob@tenant-b.test` |

You should see `api-test: PASS`, `s16 kit: PASS (33 checks)` and `s16: PASS`.

## Acceptance criteria

Evidence is in `evidence/`:
- `S16-*.json`: one Bruno request each, request and response, with no credentials (sign-ins are left out).
- `run-summary.json`: every Bruno test.
- `kit-summary.json`: every kit check.
- `*.txt`: the verifier's output, written as in the S13 kit (the exit code, then stdout).

| # | Criterion | Result | Steps | Evidence |
|---|---|---|---|---|
| AC1 | An export filtered to one incident verifies `ok`; deleting a line or editing a byte makes it fail | ✅ | S16.4–S16.6, kit AC1 | `S16-6.json`: the export filtered to Alice's incident is 200 `application/x-ndjson`, sent as an attachment. Its header records the filter. The incident is in full at its `seq`, every other seq is a hash-only stub, and the trailer says `verifiable: true`. `ac1-export-filtered.jsonl` is the kit's copy (1 entry, 15 stubs). **`ac1-verify-ok.txt`: `exit 0`, `ok: seq 1..16 from genesis to head_hash 40f8bb95…; 1 entries, 15 stubs, 0 unsealed`.** The incident's line deleted (`ac1-export-line-deleted.jsonl`): **`exit 1`, `tampered: line 12 (seq 11): seq_missing: seq 10..10 missing`**. One byte edited, the first hex digit of its `request_id` (`ac1-export-byte-edited.jsonl`): **`exit 1`, `tampered: line 12 (seq 10): envelope_sha256_mismatch`**. |
| AC2 | A second export started while one runs gets 409 `export_in_progress`; the export appears in the log as an `export` entry | ✅ | kit AC2, S16.10 | Alice writes 600 incidents with a 30,000-character description, which makes the whole-log export 18.9 MB. The kit opens it and pauses reading, so the server blocks on a full socket and keeps the lease. A second export then gets **409 `export_in_progress`** (`ac2-second-export-409.json`). The first export is drained to its trailer (`ac2-first-export-summary.json`: 18,850,460 bytes, 621 lines) and verifies `ok` (`ac2-verify-first-export.txt`). `ac2-export-entries.json`: the audit/export entries go up by exactly one. The streamed export wrote it, with its request in the diff (filter, `stubs`, `format`); the refused export wrote none. `S16-10.json`: S16.6's export is an `audit/export` entry that names its filter. |
| AC3 | The CSV opens in a spreadsheet with no formula executed (cells starting `=`, `+`, `-`, `@` are prefixed `'`) | ✅ | S16.20–S16.21, kit AC3 | `format=csv` returns 200 `text/csv; charset=utf-8` as a `.csv` attachment (S16.20, `ac3-export-filtered.csv`). It verifies **`ok`, exit 0** (`ac3-verify-filtered-csv.txt`), with one row per line the trailer counts. **The six leading characters:** Alice wrote one incident per character, each with a `short_description` that starts with `=`, `+`, `-`, `@`, tab or CR (`=HYPERLINK(…)`, …). They were exported as CSV over their time window (`ac3-export-six-leads.csv`, `ac3-six-leads.json`). Each is a full row, and each text comes back verbatim inside its `diff` cell, which starts with `[`. **Of 44,104 cells, 0 start with a formula character**, so a spreadsheet executes nothing. That file also verifies `ok`, exit 0 (`ac3-verify-six-leads-csv.txt`). `format=xml` gets 400 `validation_failed` (S16.21, `ac3-format-xml-400.json`). The `'` prefix itself is applied to every cell that starts with one of the six characters, or with `'`. That is CI-A15's unit test `TestCSVFormulaInjectionPrefixing`, run at this commit in `ac3-ci-a15-unit.txt` (PASS). See Spec notes for why no on-stack cell needs it. |
| AC4 | `POST /api/v1/audit/chain:reseed` returns 409 `chain_alive` while sealing runs | ✅ | S16.11, kit AC4 | `S16-11.json` and `ac4-reseed-409.json`: **409 `chain_alive`**, problem+json. The seal lineage is alive: Alice's incident was sealed within 8 s of being written. A dead lineage is reseeded in P5-35's integration tests. |
| AC5 | An entry erased by the test erasure helper still verifies, with the erase entry included in the export | ✅ | CI-A7, kit AC5 | Per TICK-449, the evidence is **P5-42's CI-A7 job**: GitHub Actions `ci` run 42495607369, `integration` job, on #515's merged head `0811c48`, conclusion success (`ac5-ci-a7-run.txt`). `TestCIA7EraseVerifiesPendingThenPass` erases a sealed entry through the helper. The export verifies `pending` while the audit/erase entry is unsealed. Once it is sealed, the export verifies `ok`, both unfiltered and under a filter that leaves out the erase entry itself, which the export still includes (R24). In the same job: `TestCIA7SecondEraseIsRejected`, `…ErasureCannotReadDiff`, `…AppAndWorkerCannotErase` and `…NulledDiffWithoutEraseEntryIsTampered` pass. **S13 samples**, run on this stack's `bin/nava`: `ac5-export-erased-pending.txt` gives **`exit 3`, `pending`** (`seq 6: pending: erased; its audit/erase entry is not sealed yet`). `ac5-export-erased-without-erase-entry.txt` gives **`exit 1`, `tampered: line 8 (seq 6): erasure_unlisted`**. Both match `docs/acceptance/S13/expected/` byte for byte. |

## Isolation check

- Bob (tenant B, no `audit.export`) gets **403** on the export (S16.12, `iso-bob-export-403.json`) and on the reseed (S16.13, `iso-bob-reseed-403.json`). The tenant is always the token's, and nothing in the request can name tenant A.
- Tenant A's whole-log exports (S16.7, and the 18.9 MB kit export) hold only envelopes with `tenant_id` `…0000a1`. The id of Bob's incident, written in the same run, appears in neither the whole-log export nor the filtered one.
- Alice (tenant A, an agent) gets 403 on both (S16.15, S16.17).
- In CI on #515 (`ac5-ci-a7-run.txt`): `TestCIA4EraseNeverCrossesTenants`, `TestCIA4ExportAndStubsNeverCrossTenants`, `TestCIA4ExportLeaseIsPerTenant`, `TestCIA4ReseedReadsOnlyItsOwnHead`, `TestCIA4NoTenantContextReachesNothing` and `TestCIA5ExportBytesCarryNoCanary` pass. The database isolation review belongs to the Security & Multi-Tenancy Reviewer.

## Spec notes (for the Technical Lead)

- **No on-stack cell needs the `'` prefix.** User text reaches the CSV only inside the `diff` cell, which always starts with `[`. The other cells are ids, enums, timestamps, hashes and server-generated request ids, and none of them can start with `=`, `+`, `-`, `@`, tab or CR. TICK-624 (PR #516) states the same, and this run shows it with all six characters. So "no formula executed" is proven on the stack, and "prefixed `'`" is proven by CI-A15's unit test, which forces each leading character into a cell. If a future envelope column carries user text (for example `source_ref`), the kit's AC3 check fails on any unprefixed cell.
- **History of this kit:** the first run (12:56Z) failed AC3, because `format=csv` was not routed. That became TICK-624, merged in #516.

## Not included

- **Erasure on the stack.** There is no erasure API until Epic 30. The test erasure helper runs only in P5-42's CI-A7 job.
- **A dead-lineage reseed on the stack.** It needs the lineage more than 10 minutes overdue. P5-35's integration tests cover it.
- **The export's 429 and 501 answers, and the 15-minute cap.** They are in the coverage page and P5-40's tests.
- **An S16 page in the HTML guide.** The Bruno folder (generated by `docs/acceptance/guide/build/collection.py`) and this kit cover the story.

## Known limits

- **No on-stack erasure until Epic 30** (TICK-449). AC5's evidence is CI-A7 plus the S13 samples.
- **A stack created at the branch cannot export for tenants A and B.** They have no chain head (S15 kit, Known limits), so the export would answer 500. Hence `upgrade-setup.sh`.
- **`upgrade-setup.sh` on Windows needs one patch.** The pre-migration-B commit `080dd74` still has the `curl -o /dev/null` seed bug that TICK-609 fixed. The runs applied TICK-609's `seed.sh` change to the throwaway `../s15-pre-b` worktree: `git show 8de9798f -- docs/acceptance/guide/tools/seed.sh > seed.patch && git -C ../s15-pre-b apply ../TicketingSystem/seed.patch`. Linux and macOS do not need it.
- **The concurrent-export step needs the 600 large incidents** (`S16_LOAD`). Without them the first export finishes before the second request arrives.
- **The AC2 whole-log export has 21 unsealed lines.** They are the newest load entries, not yet sealed when the export started. They are reported as unsealed, outside the verified range, and the file still verifies `ok`.
- **The verified branch does not contain the latest `main`.** Merging `main` in is part of the story→main checklist (`docs/delivery/branching-strategy.md`).

## Runs (2026-11-08, compose project `s16acc` on ports 28080/28081/28082/29090–29095)

| Run | Setup | Result | Notes |
|---|---|---|---|
| 1, 12:45Z | fresh, `25cd3c1` | stopped in AC1 | Bug in the kit: the byte edit looked for `content_sha256`, which is null on a record entry. |
| 2, 12:56Z | same stack | AC3 fails | `format=csv` returned JSONL. Fix task TICK-624. AC1, AC2, AC4 and isolation passed. |
| 3, ended 16:23Z | fresh, `19899341` (story head `91b84143`) | 2 failures, both kit bugs | The Bruno CSV check misread an empty last cell before CRLF as a CR-led cell. The kit compared the CSV row count with an export taken before the load moved the head. Both checks are fixed. The product passed everything. |
| 4, 16:28Z–16:31Z | fresh, `19899341` (story head `91b84143`) | **PASS** | Bruno: 21 requests, 65 tests. Kit: 33 checks. This run's evidence is in `evidence/`. |

Bruno CLI 4.3.0 (`api-test.sh`), Node.js 22, Docker Desktop, Git Bash on Windows 11. The full Bruno reports (`docs/acceptance/guide/reports/`) are not published, because they hold dev-only passwords and short-lived tokens.
