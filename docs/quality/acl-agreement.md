# I5 agreement suite: SQL list predicate vs Go check (P3-32)

PLAN-003 P3-32 (TICK-617), story S10 (TICK-336). This proves ADR-001 I5 for the record ACL: for every policy, principal and row, `authz.Allows(pol, p, "read", r)` is true exactly when `SELECT … WHERE <authz.CompileRead(pol, p)>` returns `r`.

## Scope

- **Generator** (`internal/tenancy/isolation/i5_gen_test.go`, no build tag). It builds one tenant with 6 users, 3 outsider principals, 5 groups and 2 outsider groups. It also adds a decoy principal that has a group's id and a decoy group that has a user's id. There are 160 records. Requester, assignee and assignment group can each be NULL. Each record has 0 to 3 shares, which can name anyone, outsiders and decoys included.
- **Policies.** Each run has 500 policies. Each policy has 1 to 4 rules over every `Subject` kind and every `Cond` form: state and uuid literals, `self`, `my_groups`, write-only and empty op sets. Malformed conditions are included too: unknown column, `ne`, a bad uuid, eq with 2 values, and a caller reference with values. Nil and empty policies also appear.
- **Principals.** Each policy is checked against 50 principals. Some have no id. Some are in no group. Some are the decoy principal or members of the decoy group.
- **`Row.SharedWith`** follows A1's pair-matched rule: a user share for the caller's id, or a group share for one of the caller's groups. A caller with no id gets no shares. The database holds every share, so a compiler that ignores the pair or the caller admits rows the Go check does not.
- **Database half** (`i5_agreement_isolation_test.go`, `TestI5Agreement`). It seeds the world as the superuser in one transaction. It then runs `SET LOCAL ROLE nava_app` with the tenant set, and runs one query per policy, with one `UNION ALL` arm per principal. Each arm has its own `CompileRead` predicate, numbered after the arms before it, so `argBase` is exercised too. The transaction is rolled back, so tenants A and B keep their seed and the deferred commit checks never run.

## Report rows (`isolation-report.md`, case I5)

| Row | What fails it |
|---|---|
| I5 agreement | Any case where the two sides disagree: the report row shows the seed, policy, principal, row and predicate. Also fails when fewer than 20% of cases admit a proper subset of the records, or when the run takes over 60 s. |
| I5 loader | `authz.SharesFor` disagrees with the generator's pair-matched `SharedWith` for 20 callers × every record. Also fails when no caller has a share. |
| I5 mutant `drop_my_groups` / `share_id_only` / `ignore_ops` | The broken compiler agrees on every case: the suite cannot see it. A caught mutant is a PASS row with its first counterexample. |
| I5 data filters | PENDING on P3-34 (TICK-619). P3-38 does not start until this row is a PASS. |
| I5 list route | PENDING on P3-34 (TICK-619), PR #523 security finding 2. |

## Risks, ranked, and the layer that covers each

1. **SQL admits a row Go refuses, or the reverse** (wrong list or wrong 404). Covered by the agreement row (testcontainers-style real PostgreSQL, under RLS as `nava_app`).
2. **Shares matched by id alone** (AR-S10-01). Covered by the decoy shares in the agreement row, the `share_id_only` mutant and the loader row.
3. **A NULL fixed column matched by a condition.** Covered by the NULL columns in the agreement row.
4. **Malformed rules handled differently on the two sides.** Covered by the malformed conditions in the agreement row.
5. **A suite too weak to fail.** Covered by the mutant rows, the proper-subset guard, and `TestI5GeneratorCoversTheRiskyShapes` / `TestI5MutantsChangeTheSQL` (unit tests).

## Not tested here, and why

- **Write agreement.** `CompileRead` is read only. Write is the Go check alone (`CanWrite`), and `internal/records/pipeline/acl_integration_test.go` covers it.
- **Cross-tenant ids in a policy.** The publish route rejects them (P3-35), and the Security & Multi-Tenancy Reviewer owns that review.
- **Data-field filters and the list route.** They do not exist until P3-34 ships (see the PENDING rows).

## Commands

- Unit (generator, determinism, mutants are not no-ops): `task test`.
- Database half: the `integration` job of the GitHub `ci` workflow (`make isolation`). Read the `isolation-report` artifact. Agents never run it locally.
- Replay a seed: `NAVA_I5_SEED=<n> make isolation`. The seed is logged on every run and printed in a failing row.
- Mutation run: `NAVA_I5_MUTANT=drop_my_groups make isolation` turns the agreement row red.
