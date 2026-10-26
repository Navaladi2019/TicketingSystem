# Branching strategy

Owner: the Technical Lead. Decided by the owner (nava) on TICK-231 and in effect from 2026-10-26.

Nothing at task level merges straight into `main`. Work flows from a task branch to its story branch, and from the story branch to `main`.

## Branch model

| Branch | Name | Created from | PR targets | Merged by | Merge method |
|---|---|---|---|---|---|
| Story | `TICK-<story-no>/<story-slug>`, e.g. `TICK-46/operator-onboards-tenant` | `main`, by the Technical Lead when the story's first task starts | `main` | the owner, after story acceptance (`MergePr`) | squash or merge commit, per repo setting |
| Task | `TICK-<task-no>/<short-kebab-description>`, e.g. `TICK-81/operator-onboarding-api` | the parent story branch | the parent story branch, never `main` | see [Who merges task PRs](#who-merges-task-prs) | squash, then delete the branch |

## Naming

- The name is the ticket identifier, a `/`, then a lowercase kebab description. Keep it to about 50 characters.
- No other prefixes: no `feat/`, `fix/`, `agent/`, `docs/` or user names.
- Because the name contains `TICK-<no>`, Multica links the PR to the issue automatically.

## Story branches

- The Technical Lead creates the story branch from `main` and names it on the story and on every task under it.
- Keep it fresh by merging `main` into it. Story branches are shared, so never force-push them and never rebase them.
- The story → `main` PR opens only when every task PR has merged into the story branch and the acceptance kit has passed on the story branch (`story-acceptance`). The owner merges it. Agents never merge into `main`.

## Task branches

`multica repo checkout` creates an `agent/...` branch by default. Rename it before the first push:

```sh
multica repo checkout https://github.com/Navaladi2019/TicketingSystem.git --ref TICK-<story-no>/<story-slug>
git branch -m TICK-<task-no>/<short-kebab-description>
git push -u origin TICK-<task-no>/<short-kebab-description>
gh pr create --base TICK-<story-no>/<story-slug> --title "TICK-<task-no>: <title>"
```

- One task, one branch, one PR into its story branch.
- Before review, rebase on the story branch: `git fetch origin && git rebase origin/TICK-<story-no>/<story-slug>`, then `git push --force-with-lease`. Force-push only your own task branch.
- Delete the branch after merge.

## Who merges task PRs

The owner allowed the Technical Lead to merge task PRs into story branches once the reviewers approve and CI is green. The Technical Lead's standing agent instructions still say "never merge the PR yourself", and those instructions take precedence. **So until the owner updates the Technical Lead's agent instructions, task PRs follow the existing `MergePr` flow and the owner merges them.** To change this, update the "Pull request labeling" section of the Technical Lead's instructions to allow merges into `TICK-*` story branches only.

## Exceptions

| Case | Branch | PR target |
|---|---|---|
| A task with no parent story (process, docs or tooling work, e.g. TICK-231) | `TICK-<no>/<description>` | `main`, only when the Technical Lead says so on the issue |
| A bug with no story | `TICK-<no>/<description>` | `main`, only when the Technical Lead says so on the issue |
| Design artefacts: RFC, ADR, plan (Architecture squad, TICK-232) | `TICK-<no>/<description>` | the story branch if one exists, otherwise `main` |
| Throwaway CI probes (draft, "do not merge") | any | whatever base the probe tests. Never merged; closed after the result is recorded |

## CI

- `.github/workflows/ci.yml` runs on every `pull_request`, with no base-branch filter. So CI already runs on PRs into story branches as well as `main`.
- The `changes` job diffs against `origin/${{ github.base_ref }}`. On a task PR that is the story branch, so path filtering works without changes.
- The CI gate in the Delivery squad instructions (every GitHub Actions job green, confirmed by the Quality, Testing & CI Advisor) applies to both levels.

## Branch protection (state on 2026-10-26)

| Item | Today | Needed |
|---|---|---|
| `main` | Not protected. The one ruleset, "Mina branch" (blocks deletion and force-push), has an empty ref include list, so it applies to nothing. | Require a PR, require the CI checks, and block force-push and deletion. |
| Story branches | Not protected | Same as `main`. |
| Delete branch on merge | Off | On |

The owner's rule says "protect `TICK-*` story branches". Story and task branches share one name shape (`TICK-<no>/<slug>`), so a pattern like `TICK-*/*` also protects task branches, and that blocks the force-push a task rebase needs. Proposed fix: one ruleset for `main`, and a second ruleset for story branches whose include list holds exact names. The Technical Lead adds each story branch to it when creating the branch. The CTO confirms the required check names with the Quality, Testing & CI Advisor (TICK-232) and applies them. Changing repository settings is the owner's call.

## Open PRs at cutover

On 2026-10-26 these PRs were retargeted from `main` to their story branches. The story branches were created from `main` at `1e52944`, so each PR's diff is unchanged.

| PR | Task | Story branch |
|---|---|---|
| #130 | TICK-81 | `TICK-46/operator-onboards-tenant` |
| #129 | TICK-222 | `TICK-47/group-changes-within-10-minutes` |
| #126 | TICK-174 | `TICK-49/outbox-row-runs-one-workflow` |
| #116 | TICK-86 | `TICK-49/outbox-row-runs-one-workflow` |

`TICK-48/api-ground-rules` was also created, for in-flight TICK-212 and TICK-77.

The head branches of these four PRs keep their `agent/...` names. On GitHub, renaming a PR's head branch closes the PR ([GitHub docs](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-branches-in-your-repository/renaming-a-branch)), which would lose review threads and CI history. Every new task branch, and every PR still to be opened for an in-flight task, uses the new names. About 60 legacy branches (`agent/*`, `docs/*`, `sec/*` and others) remain. Deleting the merged ones is a cleanup for the owner to approve.
