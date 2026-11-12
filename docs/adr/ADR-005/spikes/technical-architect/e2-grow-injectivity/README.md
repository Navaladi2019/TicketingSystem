# ADR-005-E2: is `grow` injective when a digit touches `{seq:N}`? (Technical Architect spike, TICK-667)

Source: TICK-667 comment 01a1d155-4d94 (verdict on PR #615, finding 4).

- **Hypothesis:** in S17, `{prefix}` is fixed per type and `{seq:N}` is the only variable-width token, so under `grow` the rendered number is injective over (scope, counter) whatever sits next to `{seq:N}`. The grow rule is then a readability rule, not a uniqueness rule.
- **Pass:** 0 clashes, i.e. no number rendered from two different (scope, counter) pairs. **Fail:** any clash.
- **Setup:** pure Go, no database. Dates 2000-01-01 to 2030, every 11th day. Counters 1..1199, every 4999th value up to 110000, plus 100000 and 100001, so every template overflows its width. Prefix `INC`.
- **Verdict:** PASS for all five templates, including the three the stricter rule refuses for a digit **before** `{seq:N}` and the two it refuses for a digit **after** it.
- **Rerun:** `go run grow-injectivity-main.go` (the file has `//go:build ignore`, so it is not part of the module build). Rerun by the Solution Architect on TICK-775 with go1.22.2, windows/amd64: identical output.

"Distinct numbers" is lower than "allocations" because many sampled days share one scope (for example the same `yy`), so the same (scope, counter) is rendered more than once; that is the same number, not a clash.

## Raw output

```
{prefix}{yy}{seq:3}          grow: 1259690 allocations, 37913 distinct numbers, 0 clashes
{prefix}2{seq:1}             grow: 1259690 allocations, 1223 distinct numbers, 0 clashes
{prefix}-{yyyy}{MM}{seq:4}   grow: 1259690 allocations, 454956 distinct numbers, 0 clashes
{prefix}{dd}{seq:1}{MM}      grow: 1259690 allocations, 447618 distinct numbers, 0 clashes
{prefix}-{seq:2}{yy}         grow: 1259690 allocations, 37913 distinct numbers, 0 clashes
```

## Limits

- The matcher covers S17's grammar only. Once `{field:…}` compiles (S19), a field token is a second variable-width token and the "after" half of the rule becomes load-bearing for uniqueness (`{field:br}{seq:3}`); this spike says nothing about that case.
- Overlap **between epochs** of one type is `Disjoint`'s job (ADR-005 §3.3), not the grow rule's; see `TestDisjointProperty` and `FuzzDisjoint` on PR #615.
