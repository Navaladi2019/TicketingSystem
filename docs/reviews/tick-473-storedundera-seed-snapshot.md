# Review: TICK-473, StoredUnderA seed-id snapshot check (PR #382)

Reviewer: Quality, Testing & CI Advisor. Second reviewer under D-P10: Postgres & Data Modeling Specialist.
Scope: ADR-001 I2, CI-2. Follow-ups from TICK-366 / PR #378.

**Verdict: APPROVE** (re-review at `61ffe54`). The first pass at `555db90` was REQUEST CHANGES; finding 1 is fixed.

## Findings

### 1. Breaks under a non-canonical UUID in the 201 body: the seed row can still be deleted

`internal/tenancy/isolation/http_isolation_test.go`, `runStoredUnderA`: the lines with `uuid.Validate(row.ID)` and `_, existed = seedA[row.ID]`.

- `recordIDs` keys the snapshot by Postgres's canonical text, which is lowercase and hyphenated (`id::text`).
- The 201 id is checked with `uuid.Validate`. That function also accepts uppercase ids, `{...}` braces, the `urn:uuid:` prefix and 32-hex ids without hyphens.
- The lookup (`WHERE id = $1`) and the cleanup (`DELETE ... AND id = $2`) both bind the raw string to a `uuid` parameter. pgx's UUID codec decodes uppercase and 32-hex forms to the same bytes, so those match the seed row. The braced and `urn:` forms fail to encode and end as a failed case, which is safe.

Failing scenario: take the PR's mutant and change one thing. It returns `0A000000-0000-4000-8000-0000000000A2` (uppercase) instead of the lowercase id. Then:

1. `seedA[row.ID]` misses, so `existed` is false.
2. The lookup still finds exactly one row, in tenant A, and tenant A has +1 record.
3. The cleanup is registered and **deletes seed row `…a2`**. The case reports "stored under A".

This is the exact hole the task closes, reached through a different spelling of the id. The PR's claim that "a 201 naming a seed row can never delete it" does not hold yet.

**Fix (test code):** canonicalize once, right after the unmarshal, and use the canonical value everywhere after that:

```go
id, err := uuid.Parse(row.ID)
if err != nil { /* existing "no record id the runner can resolve" failure */ }
row.ID = id.String()
```

The alternative is to fail any id where `row.ID != id.String()`. A real handler emits the canonical form, so either option is fine. My preference is to fail it, so the runner never silently accepts a body the API contract would not.

**Red then green:** rerun the PR's mutant with an uppercase seed id. Before the fix it passes and the seed count drops to 1. After the fix it fails with "already held before the call", and the seed counts stay at A 2, B 3.

## Checked and fine

- **Ordering:** the snapshot is taken before `r.do`, and `existed` is evaluated before the cleanup `defer` is registered. A 201 that names a pre-existing id never registers the cleanup, even when it also inserted a row.
- **Switch precedence:** the tenant B delta, the tenant B id in the body, a replay and a 401 all still fail first. `existed` fails before the stored and +1 checks, so its message names the real cause.
- **Concurrency:** the runner already assumes no concurrent writers to tenant A between the before and after counts (`afterA == beforeA+1`). The snapshot adds no new race; it shares the window the count check already has.
- **`registry.go`:** the diff is the single comment line "201 or 200" → "201 ... new record". It is comment-only, so the CTO's D-P10 sign-off on #378 still covers it.
- **Evidence:** the red mutant run (three forged cases fail on the new check, seed counts unchanged) and the green full `isolation.sh` run are both in the PR description. Not committing the mutant is right, and matches #378.

## Re-review at `61ffe54`: finding 1 fixed

- `canonicalID` runs `uuid.Parse(raw).String()`. `row.ID` is replaced with the canonical form right after the unmarshal, so the `seedA` lookup, the `stored` query, the cleanup `DELETE` and every message all use the same form as `id::text`.
- Non-uuids still fail with "201 with no record id the runner can resolve". The braced and `urn:uuid:` forms are now canonicalized as well, which is stricter than the previous behaviour.
- `TestCanonicalID` pins the five spellings of `…a2` to the seed key and rejects three non-uuids. This is the regression test for the finding.
- Red: the mutant returned the uppercase id and then the unhyphenated id, and both runs failed with "already held before the call". The seed counts stayed at A 2, B 3. Green: the full `isolation.sh` exits 0. Both runs are in the PR description.
- `registry.go` has not changed since the comment-only commit, so D-P10 coverage holds.
