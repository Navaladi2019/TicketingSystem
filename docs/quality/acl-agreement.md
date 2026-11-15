# I5 agreement suite: SQL list predicate vs Go check (P3-32)

PLAN-003 P3-32 (TICK-617), story S10 (TICK-336). This proves ADR-001 I5 for the record ACL: for every policy, principal and row, `authz.Allows(pol, p, "read", r)` is true exactly when `SELECT … WHERE <authz.CompileRead(pol, p)>` returns `r`.

## Scope

- **Generator** (`internal/tenancy/isolation/i5_gen_test.go`, no build tag). It builds one tenant with 6 users, 3 outsider principals, 5 groups and 2 outsider groups. It also adds a decoy principal that has a group's id and a decoy group that has a user's id. There are 160 records. Requester, assignee and assignment group can each be NULL. Each record has 0 to 3 shares, which can name anyone, outsiders and decoys included.
- **Policies.** Each run has 500 policies. Each policy has 1 to 4 rules over every `Subject` kind and every `Cond` form: state and uuid literals, `self`, `my_groups`, write-only and empty op sets. Malformed conditions are included too: unknown column, `ne`, a bad uuid, eq with 2 values, and a caller reference with values. Nil and empty policies also appear.
- **Principals.** Each policy is checked against 50 principals. Some have no id. Some are in no group. Some are the decoy principal or members of the decoy group.
- **`Row.SharedWith`** follows A1's pair-matched rule: a user share for the caller's id, or a group share for one of the caller's groups. A caller with no id gets no shares. The database holds every share, so a compiler that ignores the pair or the caller admits rows the Go check does not.
- **Route generator** (`internal/tenancy/isolation/i5_route_gen_test.go`, no build tag; TICK-900). One entity type, `iso_et_i5r`, in tenant A with two schema versions: v1 has `prio`, `tag` (strings), `kind` (integer) and `cost` (number, masked per case), and v2 adds `"default": "low"` to `prio` (`NeedsBackfill`; nothing backfills old rows, so they read the default). 48 records plus two twins are written past the API under `schema_version` 1 or 2, so each data field takes every shape an older row can have: present, JSON `null`, absent and defaulted on read (`prio`), absent with no default, and a different JSON type (`kind` stored as a string in a v1 row; the publish route refuses a type change as Breaking, so such rows come only from a write that bypassed validation). 24 policies (the agreement generator's rules, minus those `acl_versions`' shape CHECK refuses) × 6 callers, each with 5 random `eq`/`in` filter requests on data and fixed fields and one paged list with a random sort and page size.
- **Database half** (`i5_agreement_isolation_test.go`, `TestI5Agreement`). It seeds the world as the superuser in one transaction. It then runs `SET LOCAL ROLE nava_app` with the tenant set, and runs one query per policy, with one `UNION ALL` arm per principal. Each arm has its own `CompileRead` predicate, numbered after the arms before it, so `argBase` is exercised too. The transaction is rolled back, so tenants A and B keep their seed and the deferred commit checks never run.

## Report rows (`isolation-report.md`, case I5)

| Row | What fails it |
|---|---|
| I5 agreement | Any case where the two sides disagree: the report row shows the seed, policy, principal, row and predicate. Also fails when fewer than 20% of cases admit a proper subset of the records, or when the run takes over 60 s. |
| I5 loader | `authz.SharesFor` disagrees with the generator's pair-matched `SharedWith` for 20 callers × every record. Also fails when no caller has a share. |
| I5 mutant `drop_my_groups` / `share_id_only` / `ignore_ops` | The broken compiler agrees on every case: the suite cannot see it. A caught mutant is a PASS row with its first counterexample. |
| I5 data filters | For some caller and filter request, `GET /api/v1/records/iso_et_i5r?filter[...]` (the real handler, `authz.CompileRead` plus the route's filter compiler) returns a record the Go side refuses, or leaves out one it admits. The Go side is `authz.Allows` on the record's row and the typed Go filter (`i5rQuery.matches`) on the record as GET serves it, defaults applied; the run first checks that GET serves the generator's defaulted record. A filter on `cost` by a caller the mask hides it from must be `400 masked_field`, and no item may carry `cost`. The row reports the count of disagreements by the filtered field's shape and the first one with seed, policy, principal, request and row. It also fails when under a quarter of requests match a record, no request is masked or none hits a defaulted `prio`. P3-38 does not start until this row is a PASS. |
| I5 list route | A paged unfiltered list (random sort, limit 4–15, following `next_cursor`) puts a record `Allows` refuses on a page, puts a record on two pages, misses a readable record, or reports a `total` other than the readable count on any page (PR #523 security finding 2). Also fails when under 20% of callers read a proper subset. |
| I5 masked-field oracle | Two twins differ only in `cost`. A caller who reads them with `cost` masked gets one `400 masked_field` body for `filter[cost]` eq and in on either twin's value or neither's, alone or with another filter, and the body never carries the value or a total. Without `cost` in the request the twins are alike: both listed, total 2, items equal but for `id` and `number`, one page each at `limit=1`. Control: a caller the mask admits gets only twin A for `filter[cost]=1`. A fixed case (every user reads, only one user may read `cost`) guarantees both kinds of caller. Fails when either kind is missing. |
| I5 mutant `type_blind` / `null_matches` | The broken data-filter compiler (`server.WithListDataTerm`, integration builds only) disagrees nowhere the real list route agrees. `type_blind` compares values as text, so the v1 string `"1"` matches the integer filter `1`. `null_matches` lets a JSON `null` match any value. Each runs on the first 12 cases. `NAVA_I5_MUTANT=<name>` runs the data filters row against it. |

## Risks, ranked, and the layer that covers each

1. **SQL admits a row Go refuses, or the reverse** (wrong list or wrong 404). Covered by the agreement row (testcontainers-style real PostgreSQL, under RLS as `nava_app`).
2. **Shares matched by id alone** (AR-S10-01). Covered by the decoy shares in the agreement row, the `share_id_only` mutant and the loader row.
3. **A NULL fixed column matched by a condition.** Covered by the NULL columns in the agreement row.
4. **Malformed rules handled differently on the two sides.** Covered by the malformed conditions in the agreement row.
5. **A suite too weak to fail.** Covered by the mutant rows, the proper-subset guard, and `TestI5GeneratorCoversTheRiskyShapes` / `TestI5MutantsChangeTheSQL` (unit tests).
6. **A data filter that matches stored data while the read serves defaults** (a row written before a default existed is shown with the default and missed by a filter on it). Covered by the data filters row. **Open: the row is red today** (TICK-900): the list's data filter is `r.data @> {prop: value}` on the stored document, and GET and the list fill in the current version's defaults. Every disagreement on seeds 20261109, 7, 4242 and 31337 is of this shape.
7. **Type confusion and nulls in data filters** (a v1 string matching an integer filter, a `null` matching). Covered by the data filters row and the `type_blind` / `null_matches` mutants.
8. **A list that leaks an unreadable record or counts one** (a page, a cursor or `total`). Covered by the list route row.
9. **Masked fields used as an oracle** (filter, total, paging). Covered by the masked-field row and S10 attack row 3. `TestI5RouteGeneratorCoversTheRiskyShapes` (unit) fails when the default seed's route run lacks any of the five field shapes, a v1 row that reads v2's default, a filter on that default, eq or in, or callers on either side of the mask.

## Not tested here, and why

- **Write agreement.** `CompileRead` is read only. Write is the Go check alone (`CanWrite`), and `internal/records/pipeline/acl_integration_test.go` covers it.
- **Cross-tenant ids in a policy.** The publish route rejects them (P3-35), and the Security & Multi-Tenancy Reviewer owns that review.
- **Nested data properties, arrays, unions and objects.** The route has no filter on them (`filterValue` refuses them), so there is nothing to agree on.
- **Shares that cross kinds in the route world.** `record_shares` has foreign keys, so a decoy share cannot be stored; the SQL agreement row covers the pair-matched rule with decoys.
- **Malformed rules on the route.** `acl_versions`' shape CHECK refuses them, so the route can never read one; the SQL agreement row covers them.
- **Composable policies and performance.** Out of scope (sibling design issue under TICK-9; TICK-646, S21, S22).

## Commands

- Unit (generator, determinism, mutants are not no-ops): `task test`.
- Database half: the `integration` job of the GitHub `ci` workflow (`make isolation`). Read the `isolation-report` artifact. Agents never run it locally.
- Replay a seed: `NAVA_I5_SEED=<n> make isolation`. The seed is logged on every run and printed in a failing row.
- Mutation run: `NAVA_I5_MUTANT=drop_my_groups make isolation` turns the agreement row red. `NAVA_I5_MUTANT=type_blind make isolation` (or `null_matches`) runs the data filters row against that broken list route and turns it red.
