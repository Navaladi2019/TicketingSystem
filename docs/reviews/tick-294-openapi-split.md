# Review: TICK-294 (MS-6) split api/openapi.yaml, bundle, regenerate-and-diff gate

PR #261, head `ccc9bc1`. Reviewer: Quality, Testing & CI Advisor.

**Verdict: changes requested (one finding).** The bundler, the gate and the
consumer switch to `api.SpecFile` are correct. The two new migration tests
pin today's spec exactly, so the next PR that adds an endpoint has to edit them.
That brings back the merge conflict MS-6 exists to remove.

CI on `ccc9bc1` (run 40737802019): `changes`, `lint` (includes
`make lint` with the new gate and `make check-openapi` against main's pre-split
file), `unit`, `integration`, `temporal-server`, `stack`, `stack-db` and
`stack-internal` all pass.

## Findings (most severe first)

### 1. Breaks under: the next story that adds an endpoint (must fix)

`api/api_test.go` `TestBundleOperationIDsMatchPreSplit` asserts
`want := []string{"getMe", "onboardTenant"}` with `slices.Equal`.
`TestBundleIsSemanticallyThePreSplitSpec` asserts `reflect.DeepEqual` between the
whole bundle and `api/testdata/openapi.presplit.yaml`.

Scenario: stories A and B each add an operation in their own
`api/paths/<group>.yaml`. Each PR fails both tests until it appends its id to
`want` and adds its path to `openapi.presplit.yaml`. Both PRs then edit the
same line and the same file, so they conflict, and a "pre-split" snapshot ends
up holding post-split endpoints.

The issue says "operation ids equal the pre-split set". That fits the PR that
does the split, but nothing after it. The additive-only rule is already
enforced by `make check-openapi` against main's bundle.

Fix (owner: backend-dev): keep the tests and make them prove that nothing
was lost.
- Operation ids: assert that every pre-split id is still in the bundle
  (subset), not that the two sets are equal.
- Semantic diff: for each path and component in `openapi.presplit.yaml`, assert
  the bundle's entry is `DeepEqual`, and ignore entries the bundle adds. Top-level
  `openapi`, `servers` and `security` stay equal.

Test that catches it: add a throwaway `paths/me.yaml` operation locally. Both
tests must still pass, and `check-openapi` must still pass because the change is
additive.

### 2. Breaks under: running `task lint` before staging regenerated files (Preference-level friction, optional)

`scripts/ci/openapi-generated.sh` uses `git diff --exit-code -- api/`, which
compares the working tree with the index. A developer who edits
`paths/me.yaml`, runs `task generate` and then runs `task lint` before
`git add` gets a failure even though the tree is consistent. The same happens
for any uncommitted, consistent edit under `api/`. CI is not affected because
its checkout is clean. Option: compare against a fresh generate in a temp
copy, or document "stage first" in `api/README.md`.

### 3. Preference: cross-group component refs are not rejected

A group file can use `#/components/schemas/X` where `X` lives in another
group's file. It bundles because all components are merged into one map. The
README says "group-only schemas live in the group file", but nothing enforces
it, and moving `X` later would break the other group. A check is optional:
after the merge, a `#/components/...` ref in a group file must resolve to the
root or to the same file.

### 4. Preference: fixed temp file name in `scripts/ci/check-openapi.sh`

`$tmp/openapi.base.yaml` is shared by concurrent runs on one host. `mktemp`
would avoid the collision. This is low risk.

## Checked, no finding

- Bundler: inlines path items. Rewrites `../openapi.yaml#` refs. Rejects a
  non-`$ref` root path, a wrong group, a duplicate component, an unreferenced
  group file and a dangling ref. Output is deterministic (sorted file
  merge) and validated by kin-openapi. Each failure has a table test.
- No new dependencies (kin-openapi and yaml.v3 already pinned). `api.gen.go`
  is unchanged.
- `.gitattributes`: `*.yaml`/`*.go` are already `eol=lf`, so the gate does not
  flap on Windows checkouts. Generated files are marked `linguist-generated`.
- `check-openapi.sh` falls back to `api/openapi.yaml` when the base has no
  bundle. That makes the first run on this PR compare against main's
  pre-split contract (passed in CI).
- All embed consumers (`internal/server`, `apidiff`, isolation
  `boundary_paths.go`) read `api.SpecFile`. The served spec changed only by the
  leading `# Code generated` comment.
