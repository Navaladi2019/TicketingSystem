# Branching strategy

Owner: the Technical Lead. Decided by the owner (nava) on TICK-231 and in effect from 2026-10-26. Merge authority updated on 2026-11-01 (TICK-402): no human approval gate remains for stories or ADRs.

Nothing at task level merges straight into `main`. Work flows from a task branch to its story branch, and from the story branch to `main`.

## Branch model

| Branch | Name | Created from | PR targets | Merged by | Merge method |
|---|---|---|---|---|---|
| Story | `TICK-<story-no>/<story-slug>`, e.g. `TICK-46/operator-onboards-tenant` | `main`, by the Technical Lead when the story's first task starts | `main` | the Technical Lead, after `VERDICT: ACCEPTED` on the story | squash or merge commit, per repo setting |
| Task | `TICK-<task-no>/<short-kebab-description>`, e.g. `TICK-81/operator-onboarding-api` | the parent story branch | the parent story branch, never `main` | see [Who merges](#who-merges) | squash, then delete the branch |

## Naming

- The name is the ticket identifier, a `/`, then a lowercase kebab description. Keep it to about 50 characters.
- No other prefixes: no `feat/`, `fix/`, `agent/`, `docs/` or user names.
- Because the name contains `TICK-<no>`, Multica links the PR to the issue automatically.

## Story branches

- The Technical Lead creates the story branch from `main` and names it on the story and on every task under it.
- Keep it fresh by merging `main` into it. Story branches are shared, so never force-push them and never rebase them.
- Create a story branch only when its story is free to start (see [Story independence](#story-independence)).

## Closing a story without a story PR

CI cannot read Multica. The `story-branches` job (`make check-story-branches`, TICK-379) treats a story branch with merged task PRs and no story PR as in flight, and never reports it, until the repo says the story is closed. When a story is `cancelled`, or `done` without its work reaching `main`:

1. Prefer deleting the story branch. A deleted branch needs no entry.
2. If the branch must stay, the Technical Lead adds an entry to `tools/ci/storybranches/closed.yaml` in the same change that closes the story:

   ```yaml
   closed:
     - branch: TICK-<story-no>/<story-slug>
       status: cancelled   # or done, as on the story issue
       issue: TICK-<story-no>
   ```

3. The entry goes in a standalone PR into `main` (see [Exceptions](#exceptions)). From then on the job reports the branch as orphaned until it is deleted.

A story that merged through its story PR needs no entry. A bad entry (unknown status, or a branch that is not `TICK-<n>/...`) makes the tool exit 2 and turns the job red.

## Story independence

- **Stories must be independent: no two stories may touch the same file.** When a plan or grooming splits work into stories, list the files and packages each story is expected to change. Check the lists for overlap before any story starts.
- **A dependent story waits.** If story 2 depends on story 1, because they share files or story 2 needs story 1's code:
  - keep story 2 in `backlog` with `Depends on: <story 1>`;
  - do not create its story branch until story 1 has merged into `main`;
  - then create it from the updated `main`.
- **An overlap found mid-flight stops the later story.** Tell the CTO and re-sequence. Never resolve an overlap by merging stories into each other.

## Story branch → `main`

Take the utmost care here. Before the story PR opens:

1. Merge the current `main` into the story branch and resolve any conflict there, never on `main`. Re-run the full GitHub Actions suite on the result, including the tenancy-isolation tests (`gh workflow run ci --ref <story-branch>`).
2. The Principal Software Engineer reviews the whole story diff against `main`, not task by task. The Security & Multi-Tenancy Reviewer joins when the diff touches tenant data, auth or ACLs.
3. The Quality, Testing & CI Advisor confirms that every job is green and that the story's acceptance kit passed on a fresh stack (`story-acceptance`).
4. The story PR lists the files it changes and confirms that no other open story touches them.

Every task PR must already have merged into the story branch. Then the Technical Lead verifies the acceptance kit end-to-end on a fresh stack, posts `VERDICT: ACCEPTED` on the story, approves and merges the story PR (CI green, whole-story diff reviewed) and sets the story `done`. No `MergePr` label and no owner merge. On failure the Technical Lead posts `VERDICT: REJECTED` and creates a fix task per failure. Only the Technical Lead merges a story PR into `main`.

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

## Who merges

The owner's merge and close policy (decision log: TICK-243, updated by TICK-402) settles this. No PR waits for a human:

| PR | Merged by |
|---|---|
| A task PR into its story branch | the reviewer or the implementer, once every reviewer has approved and CI is green. Then the task is set to `done`. |
| A standalone task PR into `main` (no story, the Technical Lead's call) | the same: the reviewer, implementer or Technical Lead, once it is approved and CI is green |
| A story → `main` PR | the Technical Lead, after posting `VERDICT: ACCEPTED` (fresh-stack acceptance kit passed, CI green, whole-story diff reviewed). Then the story is set to `done`. No `MergePr`. |
| An ADR, amendment or erratum | the Solution Architect, after giving the `ACCEPTED` verdict (Technical Architect critique and any CTO tie-break done) and setting the status in the `.md` and `.html`. The CTO then creates `Plan ADR-NNN`. No `MergePr`. |

Only two things still go to the human: a tenant-isolation gap, and a business tradeoff (cost, new vendor, timeline vs correctness). Repository settings changes (see [Branch protection](#branch-protection-state-on-2026-10-26)) remain the owner's call.

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

**Overlap found at cutover (2026-10-26):** `cmd/nava-groupsync/main.go` is changed by #129 (TICK-222, story TICK-47) and by #126 (TICK-174, story TICK-49). That breaks the story-independence rule. The Technical Lead asked the owner on TICK-231 how to re-sequence; from 2026-11-01 (TICK-402) re-sequencing questions go to the CTO. No other pair of open task PRs shares a file.
