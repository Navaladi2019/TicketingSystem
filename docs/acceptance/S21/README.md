# S21 acceptance kit: filter a large ticket type on a flagged field in milliseconds (TICK-739)

Kit task TICK-749 (PLAN-007 P7-09). Story branch `TICK-739/filterable-fields` at `7284e2f2` (every task merged: TICK-741 to TICK-748, the P7-07 fixes TICK-989 to TICK-992, and the ADR-006-A1 budget row, TICK-1001). Design: ADR-006 D1 to D10, D12, D13 as amended by ADR-006-A1, D14. Budget row: `docs/budgets.md`, "Filtering on a flagged data field (S21)".

__RESULT__

## What you can now do

- Mark a top-level field of a ticket type as filterable (`"x-nava-filterable": true` in the schema you publish). The platform copies its existing values into an index in the background.
- While it does, a list that filters on the field answers 422 `field_not_ready` with `index_progress`, the whole percentage done. When it is done, the list filters on the field: equality, a list of values, and number and date ranges.
- Filtering a 400,000-incident type takes tens to a few hundred milliseconds, not a second.
- A field you did not flag is refused (400 `field_not_filterable`), and so is a field your access policy hides from you (400 `masked_field`). Neither is ever a scan of every record.

## Try it (about 35 minutes, most of it the two backfills)

1. From the repository root, on this branch, with Docker running (Git Bash or any POSIX shell) and Node.js 22. If another stack holds the default ports, put other ones in `.env` (`COMPOSE_PROJECT_NAME`, `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `ACCEPT_ONBOARD_PORT`, `ACCEPT_ADMIN_PORT`). The verified run used project `nava-s21` and the default ports.

   ```sh
   sh docs/acceptance/S21/tools/run-kit.sh
   ```

   It deletes the stack's volumes (`docker compose down -v`), then:
   1. starts a fresh stack from this checkout and runs `docs/acceptance/guide/tools/seed.sh` (tenants A and B and their users);
   2. `tools/kit.mjs setup`: as `admin@tenant-a`, adds an optional `due_at` (date-time) to the incident type, not flagged, and masks `description` to tenant A's admins group;
   3. `tools/seed.sql`: writes **400,000 incidents in tenant A** and 50 in tenant B, as the database superuser, in one statement per tenant (see "The seed" below; __SEED_TIME__). `tools/expect.sql` then computes what a plain JSON scan of the records gives for each of the kit's filters, and the script passes those numbers to Bruno;
   4. runs the Bruno folder **S21 filterable fields** (`docs/acceptance/guide/bruno/S21/`). It flags `priority` and `due_at`, watches the backfill, then filters;
   5. `tools/verify.sql`: compares both fields' index tables with a JSON scan of every tenant A incident (none missing, none extra, none wrong) and checks that tenant B has no index rows.

   `S21_KEEP_STACK=1` skips step 1 on a stack that is up and has had nothing but `seed.sh`. `S21_P95_SECONDS` (default 15) sets the length of each p95 cell.

2. Users (dev-only seed users, passwords in `docs/acceptance/guide/bruno/environments/local.bru`):

   | Story user | Seeded user | What they do here |
   |---|---|---|
   | `admin@tenant-a` | `admin@tenant-a.test` | publishes the flags; reads every incident |
   | `agent@tenant-a` | `alice@tenant-a.test` (service-desk group) | lists; under the default incident ACL she reads about 5% of the incidents |
   | `agent@tenant-b` | `bob@tenant-b.test` | the other tenant |

You should see `api-test: PASS` and `s21: PASS`. The folder runs once per prepared stack: S21.8 publishes the flags, so a second pass fails at S21.4.

## The seed

`tools/seed.sql` writes records directly, as the P7-07 measurement does (`perfSeedSQLWith`): through the API, 400k creates take about 35 minutes. Every value is a hash of the record's ordinal, so reruns give the same rows:

| Field | Values in tenant A |
|---|---|
| `priority` | 1: 1% · 2: 1% · 3: 48% · 4: 50% |
| `due_at` | a whole minute in 2026, spread evenly |
| assignment group | service-desk for 5%, none for the rest (only the admins read those) |
| marked records | one with `due_at` 2026-12-31T23:59:59Z and no group (Alice cannot read it); one with 23:57:59Z in service-desk (she can). 23:58:59Z exists nowhere. |

Tenant B gets 50 incidents, all priority 1, on its unchanged incident type.

## Acceptance criteria

__AC_TABLE__

### The p95 cells (criterion 2)

Each cell: 8 clients list with the filter for 15 s after 2 s of warm-up. They are paced to 40 requests a second in total, because the per-tenant rate limit is 50 a second and a 429 would fail the cell. Latency is measured at the client, through nava-core and PgBouncer, with the default page of 50. Limits are ADR-006 D13 as amended by ADR-006-A1: 350 ms for the 0.5–2% band, with or without a narrowing ACL, and for the ≥ 40% band with one (Alice); 150 ms for every other filter. **No 2–40% filter is asserted.** Those bands are unmeasured until TICK-1009 (S21b). The 0.25% `in` cells are not rerun here; TICK-1002 does that.

__P95_TABLE__

## Isolation check

- Tenant B's 50 priority-1 incidents never appear in tenant A's `filter[priority]=1`. Tenant A's count is the tenant A scan, not the all-tenant one (S21.18). Tenant B has no index rows (`verify.txt`).
- A value that only a record Alice cannot read has gives her the same status and body as a value that exists nowhere: 200, no items, `total` 0 (S21.17).
- The SQL-level attacks on the side tables and the registry are the isolation suite's (TICK-748, `docs/reviews/tick-748-s21-isolation-attacks.md`), in CI. This kit does not repeat them.

## Not included

- Removing a flag (S22; the publish answers 422 `filterable_field_removal_unavailable`).
- The 2–40% bands (TICK-1009, S21b) and the rerun of the provisional D13 cells (TICK-1002).
- Insert throughput with flagged fields (D10, ≥ 40% of the unflagged tps): `docs/budgets.md`, measured by P7-07, not by this kit.

## Known limits

- **Runs once per prepared stack** (S21.8 publishes the flags; `seed.sql` refuses a tenant that already has incidents).
- **The seed bypasses the API.** It writes records as the superuser with `session_replication_role = replica`, so the seeded records have no audit entries and no create history. Every record the kit itself creates or edits goes through the API.
- **p95 at the client, paced.** The D13 numbers in `docs/budgets.md` come from 8 unpaced clients on the handler inside the test process (P7-07). The kit measures through the whole stack instead, and has to stay under the rate limit, so its numbers are close to D13's but not the same measurement.
- **Docker on Windows:** run the script from Git Bash. `run-kit.sh` handles `MSYS_NO_PATHCONV` (TICK-623).
