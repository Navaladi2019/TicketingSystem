# Decision log

**Entry rule:** a new decision is its own file, `docs/delivery/decisions/<YYYY-MM-DD>-<slug>.md`, starting with a `## <title>` heading; then run `go run ./tools/docindex` to refresh this index (`task lint` fails when it is stale). Entries below the index are the older ones, newest first.

Delivery decisions that change a task's contract, newest first.

<!-- docindex:start -->

- [2026-11-02: PLAN-005 groomed: 30 tasks, D-3-13 revised, `diff_truncated` inside the hash, v0 dropped in P5-17](decisions/2026-11-02-plan-005-grooming.md)
- [2026-11-02: PLAN-005 approved; S14 and S15 move S10 to S12 back about 4 to 5 weeks](decisions/2026-11-02-plan-005-approved.md)
- [2026-11-02: PLAN-005 drafted: ADR-004 lands as four stories; S14 goes before S10; the idempotency key leaves the audit log](decisions/2026-11-02-plan-005-adr-004.md)
- [2026-11-01: PLAN-004 approved: ADR-003 lands as two qa tasks inside S11a; the manifest-creating PR is human-merged](decisions/2026-11-01-plan-004-adr-003.md)
- [2026-11-01: D-3-13 amended: an interpreter start uses the existing outbox start key `nava.interpreter:<outbox_id>`](decisions/2026-11-01-interpreter-start-target-key.md)
- [2026-11-01: agents accept and merge stories and ADRs; `MergePr` retired](decisions/2026-11-01-agent-only-story-and-adr-acceptance.md)
- [2026-10-31: PLAN-003 drafted: records path in two lanes; four task-text defects fixed before grooming](decisions/2026-10-31-plan-003-records-path.md)
- [2026-10-31: PLAN-003 groomed twice (TICK-360, TICK-363); #293 kept, #294 closed; step ids are random v4; I9 ruling open with the CTO](decisions/2026-10-31-plan-003-grooming.md)
- [2026-10-31: PLAN-003 approved; golang-zod not adopted](decisions/2026-10-31-plan-003-approved-zod.md)
- [2026-10-28: shared tables are append-safe: story-prefixed ids, `merge=union`, one file per decision](decisions/2026-10-28-append-safe-shared-tables.md)

<!-- docindex:end -->

## 2026-10-28: migration order is checked against `main`, not the pull request's base
- **Decided by:** the Technical Lead on TICK-309, after PR #244 (TICK-281/TICK-284) went red only on `TestNewMigrationsLaterThanBase`.
- **Decision:** every pull request, including one into a story branch, checks its migrations against `origin/main`. A migration is new when `main` does not have it, and it must be later than `main`'s newest version. The ci `unit` job sets `BASE=origin/main` for every event, no longer `github.base_ref`. The invariant (TICK-299) protects databases that applied shipped migrations, and only `main` ships.
- **Allowed:** a story renumbers its own unshipped migrations (S5: `00008`–`00010` → `00010`–`00012`).
- **Accepted risk:** a long-lived database built from a story branch must be rebuilt after the story renumbers its migrations.
- **Unchanged:** `ci-stages`, `review-paths` and `check-openapi` still diff against the pull request's base.
- **Dissent:** none.

## 2026-10-28: Merge safety: `main` is protected and tested after every merge; shared files are split
- **Decided by:** the Technical Lead on TICK-288 (`docs/delivery/merge-safety.md`), at the request of vknavaladi after S5 (PR #221) and S7 (PR #223) broke on late merges of `main`.
- **Decision:**
  - D1: `main` gets branch protection. Every `ci.yml` job is required, the branch must be up to date, and direct pushes are blocked. The owner applies it.
  - D2: `ci.yml` runs on `push` to `main` and `TICK-*`, and a red `main` opens a P0 issue (TICK-289).
  - D3: new migrations use timestamp versions, and `task test` fails on a duplicate or malformed version (TICK-290).
  - D4–D7: `Makefile`, `api/openapi.yaml`, the shared doc tables and the stack tests are split so each story adds new files (TICK-293 to TICK-296).
  - D8: the Technical Lead merges `main` into every open story branch after each merge to `main`.
  - D9: a test checks that every token-validating service trusts the pinned issuer (TICK-291), and every landed story suite runs on `main` (TICK-292).
  - Review standard: a PR merges only if it is green against the current tip of its base branch.
- **Sequencing:** D1, TICK-289, TICK-290 and TICK-291 start now and do not touch `Makefile`. The splits wait for TICK-47 and TICK-49 to land, so that the split itself does not conflict with them.
- **Supersedes:** the 2026-10-25 "no run on push to main" decision (its accepted trade-off is what let S5 and S7 break unseen; vknavaladi approved the change on TICK-288), and the 2026-10-26 deferral of branch protection on `main` (TICK-231 entry). Pull requests keep the change-based job selection. Story branches stay unprotected.
- **Applied (2026-10-28):** protection is on `main` (the eight `ci.yml` jobs are required, the branch must be up to date, admins are included, no force pushes); the Technical Lead applied it with the owner's go-ahead on TICK-288. A new `ci.yml` job must be added to the required list in the same PR.
- **Deferred:**
  - Merge queue. Trigger: more than three PRs a day waiting on rebases.
  - One file per entry for `todo.md` and `accepted-risks.md`. Trigger: a union merge produces a malformed table.
- **Dissent:** none.

## 2026-10-28: PLAN-002-E1 implements ADR-002-E1 as one task, TICK-302, under S5
- **Decided by:** the Technical Lead on TICK-301 (`docs/delivery/plan-adr-002-e1.md`).
- **Decision:** ADR-002-E1 ships as TICK-302 (`qa`, reviewed by the Security & Multi-Tenancy Reviewer). It waits in `backlog` until nava signs TICK-272. It targets the S5 story branch, because the `api.go` exclusion exists only there, so `main` never carries R-b. TICK-302, TICK-298 and TICK-300 all edit `internal/server/api.go`; whichever merges second rebases. `docs/security/accepted-risks.md` stays with PR #225, which adds AR-039.
- **Fallback:** if TICK-272 is still unsigned when every other S5 task is `done`, TICK-302 leaves S5 and targets `main` after S5 merges, so S5 acceptance is not held up.
- **Deferred:** the `navalint` `grouppaths` analyzer (R-e). Trigger: a second `forbidigo` waiver is needed, or the directive line is moved once. Owner: the Solution Architect.
- **Dissent:** none.

## 2026-10-27: a principal who leaves the directory is tombstoned, not deleted
- **Decided by:** the Postgres & Data Modeling Specialist on TICK-220, on QA finding 2 of the T25 review (`docs/reviews/tick-78-groupsync-worker.md`, PR #121).
- **Decision:** option 1, soft-delete. `principals.deleted_at` (migrations `00010_principal_tombstone.sql` and `00011_principal_tombstone_validate.sql`) marks a principal removed from the Keycloak directory. `CHECK (deleted_at IS NULL OR NOT enabled)` makes every tombstone disabled, so the resolver's existing `enabled` test refuses it with no new read. Group sync tombstones the principal (`enabled = false, deleted_at = now()`) and removes its memberships. It no longer deletes the row. A user who comes back keeps the same row and id, and sync clears `deleted_at`.
- **Reason:**
  - The tables coming next point at people: ticket reporter and assignee, comment author, audit actor. With option 2 they would need `CASCADE`, which deletes a ticket when its reporter leaves, or `SET NULL`, which erases who did what. Both are wrong for a ticketing system. With a tombstone they keep the default `NO ACTION` and the history stays intact.
  - It is cheaper to reverse. Turning tombstones into hard deletes later is one `DELETE`. Rebuilding authorship after `SET NULL` cannot be done.
  - Cost (`docs/data/principal-tombstone.md`, 519,000 principals, PostgreSQL 16.10):
    - The migration holds ACCESS EXCLUSIVE for about 5 ms. The CHECK is validated in a separate migration that blocks nothing (86 to 172 ms).
    - Removing 1,000 users from a 50,000-user tenant takes 90.1 ms as a tombstone, against 95.8 ms for a hard delete.
    - A tombstone stays in the table, about 330 bytes with its index entries. No new index.
- **Rejected:** option 2. A lint cannot tell an explicit `NO ACTION` from the default (`pg_constraint` stores both as `a`), and every allowed action loses ticket data.
- **Contract (expand, then contract):**
  1. This change: the column, the CHECK (validated in its own migration), and `deleted_at` in `nava_groupsync`'s `UPDATE` grant. Sync is unchanged and still correct.
  2. A Senior Software Engineer task, filed from TICK-220: `applyPrincipals` tombstones instead of deleting. Its test is a table with a `NO ACTION` foreign key to `principals`, and sync still commits when that user leaves.
  3. In the same PR as step 2: revoke `DELETE` on `principals` from `nava_groupsync` and drop `principals_delete`. Sync can then no longer hard-delete, even by mistake. `TestT16DeletesCascadeWithinTenant` and the cross-tenant `DELETE` probe in `TestT16TenantRowsInvisible` move to the tombstone.
- **Rule for new tables:** a foreign key to `principals` keeps the default `NO ACTION`. Readers that list people filter `deleted_at IS NULL`. Readers that show history do not.
- **Open, same hazard:** `applyGroups` and `applyOrgUnits` also hard-delete. Today nothing outside the identity tables references `groups` or `org_units`. Trigger: the first such foreign key, for example a ticket queue that points at a group. Give that table the same tombstone first.
- **Deferred:** erasing a departed user's personal data (`email`) from the tombstone. Trigger: a right-to-erasure requirement, or a tenant asking for it.
- **Dissent:** none.

## 2026-10-27: PLAN-002 implements ADR-002 as one task, TICK-268
- **Decided by:** the Technical Lead on TICK-270 (`docs/delivery/plan-adr-002.md`).
- **Decision:** ADR-002 ships as TICK-268 (`backend-dev`, under S5). It restarts from the story branch, because commit `62d604e` changed `groups` itself and is not the design. The S5 explainer's section 10 moves from TICK-268 to the acceptance kit (PR #187), so no two open PRs share that file. AC5's new entry in `registry.go` needs CTO sign-off, which ADR §6 did not list; the implementer asks for it on TICK-268. TICK-268 and TICK-265 both touch `groupsync/apply.go` and run in parallel; whichever merges second rebases.
- **Deferred:** the ACL-compiler test (TODO-013). Trigger: the first I5 ACL compiler PR.
- **Dissent:** none.

## 2026-10-26: agents merge task PRs and close tasks; stories and ADRs stay with the human
- **Decided by:** vknavaladi (workspace merge and close policy), recorded by the Technical Lead on TICK-243.
- **Decision:**
  - **Tasks and subtasks** (anything that is not a story task or an ADR): once the reviewer's `VERDICT: APPROVE` is in, CI is green and the acceptance criteria are met, the reviewer or implementer merges the task PR into the story branch and sets the task `done`.
  - **Story tasks and story → `main` PRs** (superseded on 2026-11-01 by TICK-402: the Technical Lead accepts and merges; `MergePr` retired): agents never merge or close them. Add the `MergePr` label to the issue and the PR; the human merges after replying `ACCEPTED`.
  - **ADRs, amendments and errata** (superseded on 2026-11-01 by TICK-402: the Solution Architect accepts and merges; `MergePr` retired): agents never merge or close them. Add `MergePr` after the human's sign-off; the human merges. The ADR still needs its `.md` and self-contained `.html` under `docs/adr/`.
- **Supersedes:** any earlier "never merge the PR yourself" or "only the human sets a task `done`" instruction for task PRs.
- **Dissent:** none.

## 2026-10-26: Task PRs target story branches; branches are named `TICK-<no>/<description>`
- **Decided by:** nava (human), on TICK-231. Written up by the Technical Lead in `docs/delivery/branching-strategy.md`.
- **Decision:** the flow is task branch → story branch → `main`. Story branches start from `main` and the owner merges them after acceptance. Task PRs never target `main`.
- **Technical Lead calls:**
  - Open PRs #116, #126, #129 and #130 were retargeted to their story branches. Their `agent/...` head names stay, because renaming a PR's head branch closes the PR on GitHub.
  - Who merges is settled by the TICK-243 entry above: agents merge task PRs into story branches, and the owner merges story → `main` PRs and ADRs (superseded on 2026-11-01 by TICK-402: the Technical Lead merges story PRs, the Solution Architect merges ADRs).
  - TICK-231 itself has no story, so it targets `main`.
- **Addendum (owner, same day):** stories must not share files. A dependent story waits in `backlog` until the story it depends on has merged into `main`. A story → `main` PR needs: `main` merged into the story branch, a full CI re-run, a whole-diff review by the Principal Software Engineer (plus Security when tenant data, auth or ACLs are touched), QA's green CI and acceptance kit, and the list of files it changes.
- **Deferred:**
  - Branch protection on `main` and on story branches. Today nothing is protected. Owner: the CTO with the Quality, Testing & CI Advisor (TICK-232). The owner applies the settings.
  - Deleting about 60 legacy branches. Trigger: the owner approves the cleanup.
- **Dissent:** none.

## 2026-10-25: T27k classifies every id-shaped request field by a rule; the Authorizer gains D9h and closes D9g and D9c gaps
- **Decided by:** the Technical Lead on TICK-133 (PR #102), answering the Principal Software Engineer's three questions.
- **Decision:** T27k: no inert class; link and child-failure workflow ids get prefix rules; failure `workflow_type.name` classed not-id; D9g also applies to poll heartbeats; D9c pins Kind. Decided by the Technical Lead, 2026-10-25, per the ADR-001-A2 Annex A.2 statement.
  - **D9h-links:** every link on a relay start, signal-with-start or signal, or a core cancel (including `completion_callbacks[].links[]`), is a workflow event link whose workflow id has a `t:<tenant>:` prefix.
  - **D9h-failures:** every `ChildWorkflowExecutionFailureInfo` workflow id along a failure's whole `cause` chain has the prefix. This covers worker task failures, query failures, `FailWorkflowExecution`, `RecordMarker` and continue-as-new command failures, query results and the relay's `continued_failure`. It also covers update protocol `messages` on `RespondWorkflowTaskCompleted` and `RespondWorkflowTaskFailed` (`Rejection.failure`, `Response.outcome.failure`). A message body that is not an update `Acceptance`, `Rejection` or `Response`, or that does not decode, is denied, and the drift golden expands those three bodies of the `google.protobuf.Any` (Technical Lead review of PR #102). On worker task-token paths the child id must carry the task's own tenant, as E1's `checkTarget` requires, and another tenant's id is `command_cross_tenant`. The relay's start and `RespondQueryTaskCompleted` stay prefix-only. In the golden, `links[].batch_job.job_id` is classed `D9h-links: variant denied` (Security findings 2 and 3, ruled by the Technical Lead).
- **Supersedes:** TICK-133's contract wording "workflow type". A child failure's `workflow_type.name` is failure data, not an id or queue.
- **Revisit if:** a legitimate path carries an unprefixed id in a link or child failure. That is an ADR erratum for the Solution Architect, not a new class.
- **Deferred (Security re-review, Low):** relay and core `links`, and the relay's `continued_failure`, stay prefix-only. They are not bound to the tenant of the request's own workflow id. **Trigger:** the relay forwards links or failures that a client supplied. Then bind them to the request's tenant, which `checkStartRequest` and `requestIDDenial` already hold. Owner: Security & Multi-Tenancy Reviewer.
- **Dissent:** none.

## 2026-10-25: PLAN-001-E3 narrows nava_onboard's SELECT on tenants to a column list; E3's T21 condition joins T21
- **Decided by:** the Technical Lead on TICK-178 (`docs/delivery/plan-adr-001-e3.md`), after nava approved ADR-001-E3 on TICK-170.
- **Decision 1 (D-E3-2):** `db/grants.yaml` gives `nava_onboard` `SELECT [id, state, onboarding_request_hash, created_at]` on `tenants` instead of `SELECT: []` (T09d, TICK-180). Today nothing changes. A column added later isn't readable until it's granted on purpose. That closes RS8's "tenants gains a column" trigger by construction. Rejected: a tripwire test that fails on a new column. It fails open, because it depends on someone reading the failure.
- **Decision 2 (D-E3-3):** T21 (TICK-81) gains two acceptance tests from E3's condition: the onboarding tenant id is derived on the server, and a replayed Idempotency-Key from another operator sees 0 rows. Security reviews T21.
- **Decision 3 (D-E3-1):** PR #85 lands as T09c (TICK-179), owned by its author and reviewed by the Postgres & Data Modeling Specialist.
- **Precondition (D-E3-4):** PR #84 conflicts with `main`, where PR #87 took AR-026..028. RS8 is renumbered to the next free id before PR #84 merges. Owner: the Solution Architect.
- **Dissent:** none.

## 2026-10-25: T30a anchors the relay's max redelivery age on `created_at`
- **Decided by:** the Technical Lead on TICK-74 (PR #87), option (a) of the Principal Software Engineer's question.
- **Decision:** the 24 h max redelivery age (D-P8) is measured from `outbox.created_at`, not from the first attempt as D-P8 and ADR-001 A.2 say. `outbox` has no first-attempt column, and `nava_relay` cannot reconstruct one. `created_at` is never later than the first attempt, so redelivery stays inside the worker's 48 h signal window. No migration.
  - **Where it applies:** `Backoff` marks a failed row dead once it is 24 h old. `Claim` marks a row dead instead of claiming it when the row was attempted before (`attempts > 0`, its lease or backoff over) and is 24 h old. That covers a crash between dispatch and `MarkDispatched`, which never reaches `Backoff`. Without it the row would be redelivered past 48 h and could start a second workflow.
  - A row with no attempts is never marked dead.
- **Accepted cost:** a row that waited a long time before its first attempt (behind a backing-off head, or through a relay outage) gets fewer than 24 h of retries.
  - **Revisit if:** T33 or the operators see rows marked dead after fewer than 3 attempts. Then do option (b): a `first_attempt_at` column, owned by the Postgres & Data Modeling Specialist.
- **Accepted deferrals:**
  - A leased head is not backing off, so its available successors can fill its tenant's 200-row scan window until it settles. **Revisit if:** T33's flood test shows it.
  - A poll has no limit on tenants. **Revisit if:** T33 shows poll latency growing with the tenant count.
- **Accepted risks:** AR-026 (short retry window), AR-027 (successors behind a leased head) and AR-028 (no tenant limit per poll), all 🟡 Low, in `docs/security/accepted-risks.md`.
- **Dissent:** none.

## 2026-10-25: T29 carries the trace context through the outbox; signal name is `outbox.kind`
- **Decided by:** the Principal Software Engineer on TICK-70, the decision the TICK-93 audit asked for before coding. Resolves the TICK-89 deferral "trace continuity across the outbox" (trigger: T29 and T30a start).
- **Decision 1 (adopt, no migration):** `outbox.Enqueue` stores the W3C `traceparent` of the caller's span in the existing nullable `outbox.trace_context` column (T09, ADR-001 A.3). `nava_relay` already has `SELECT` on it in `db/grants.yaml`, so no migration and no grant change. NULL when the caller has no valid span.
  - **Only `traceparent` crosses.** `tracestate` is dropped (vendor-defined, free text), and baggage never crosses: the producer runs only `propagation.TraceContext{}` and keeps one key.
  - **Relay (T30a, T30b):** extracts it with `propagation.TraceContext{}` and starts the dispatch span as a **new root with a link** to it, never as its child, matching the edge rule of 2026-10-16.
- **Decision 2:** the pinned `Entry` has `Kind` and the target's kind or signal name, but `outbox` has no signal-name or start/signal column. So `outbox.kind` is the Temporal name the relay dispatches to: the workflow type for a start (`Entry.Kind` must equal `TargetStart.Kind`) and the signal name for a signal (`Entry.Kind` must equal `TargetSignal.SignalName`). The relay tells a start from a signal by `target_key`: only a start row's `target_key` ends in its own id. Kinds match `^[a-z][a-z0-9._-]{0,99}$`, so `target_key` (`<kind>:<uuid>`) carries no PII (TICK-94 item 3, for T30b).
- **Also decided:** the payload check is code (`outbox.ValidatePayload`): a JSON object whose values are UUID strings or arrays of them. T32b's CI-7 schema reuses or replaces it.
- **Dissent:** none. The Technical Lead confirms Decision 2 in the PR review.

## 2026-10-25: T31 worker guards land as a library; worker binary, alert rule and query guards deferred
- **Decided by:** the Technical Lead on TICK-65 (PR #73, merged as `85a9202`; reviews TICK-158 and TICK-162), with the Security & Multi-Tenancy Reviewer's approval of `e29682b`.
- **Decision:** `internal/workerguard` enforces every ADR-001 A.2 worker guard, writes the `nava-tenant` header on outbound commands, and dedupes signals for 48 h across continue-as-new. The shared Temporal telemetry constructor is `telemetry.NewTemporal`. T31 landed first, so T30b (TICK-85) reuses it and `workerguard.ConfigureClient`.
- **Also decided:** the Security Reviewer's two Low findings were fixed in this PR, and the worker now rejects both:
  - `NavaTenantId` set through a key type other than Keyword;
  - a continue-as-new header rewritten after the early check.
  - **Reason:** the contract says these commands are "rejected in the worker", and the Authorizer denying them only leaves the task stuck. PR #75's adversarial tests stay as regression guards.
- **Deferred:**
  - The `cmd/nava-worker` binary, its compose service and the `NAVA_ADMIN_ADDR` admin listener (TICK-94 item 1), as TICK-159 (T31b). Trigger: released now; needed before T32b (TICK-86) runs the worker in the stack.
  - The alert rule on `nava.worker.guard.decisions{outcome!="allow"}`. Owner: TODO-004, next to AR-011 and AR-012.
  - Query guards. Queries are read-only, and the Authorizer gates who can query. Trigger: a query handler returns tenant data that is not from its own workflow.
- **Accepted risk:** AR-025 (Low): history paging and task-token APIs need a compromised worker process.
- **Dissent:** none.

## 2026-10-25: T13 purge windows are floored by the RLS policies; follow-ups split out
- **Decided by:** the Technical Lead on TICK-68 (T13, PR #51). The Principal Software Engineer and the Security & Multi-Tenancy Reviewer approved.
- **Contract clarification:** the three retention windows are still config values, but they can only be *longer* than the policy floors. The floors are 7 days dispatched / 30 days dead for `outbox`, 30 days for `workflow_ref`, and `expires_at` for `idempotency_keys`, all hard-coded in the `*_purge` policies (T09). `LoadPurge` rejects a shorter window at startup, because otherwise it would silently delete nothing. *Trigger:* a D23 change to any window needs a policy migration under the 10-entry allow-list, not just a config change.
- **Deferred to follow-up tasks, released after PR #51 merged:**
  - TICK-143: Dockerfile and compose service for `nava-purge`.
  - TICK-145: startup assertion that the role is `nava_purge` and neither `BYPASSRLS` nor superuser. This was the Security Reviewer's Low finding.
  - TICK-146: indexes for the purge predicates, or evidence from a measured plan that none are needed.
- **Not taken:** an exec-mode integration test, quieter shutdown logging, and a timestamp check in `CannotWidenItsReach`. All three were optional review notes.
## 2026-10-25: GitHub CI runs only the jobs a pull request needs; no run on push to main
- **Decided by:** nava (human), in chat with the Quality, Testing & CI Advisor. This settles TICK-153 D5, which was left to the Technical Lead.
- **Decision:** a new `changes` job runs `make ci-stages` (`tools/ci/cichanged`, the rules of `make ci-changed`), and `unit`, `temporal-server`, `integration` and the three stack jobs run only when the change needs their stage. `lint` always runs, and a change under `.github/workflows` runs every job. The `push: main` trigger is gone. A full run on any branch is a manual `workflow_dispatch` from the Actions tab.
- **Supersedes:** "the full run stays the gate" (`docs/quality/ci-pipeline.md`, TICK-153 D5).
- **Trade-off:** `main` is no longer re-checked after a merge, and a docs-only PR skips `govulncheck` in temporal-server (the root module's still runs in `lint`). A new advisory or a merge-order conflict now shows up on the next PR that needs the stage, or on a manual run.
- **Dissent:** none.

## 2026-10-24: T15 operator token through IdentityResolver; JWKS client without a telemetry accessor
- **Decided by:** the Technical Lead on TICK-150 (review of PR #56, T15 / TICK-63). The Security & Multi-Tenancy Reviewer approved the client deviation on TICK-63.
- **Erratum to the T15 contract:** "an operator token through `IdentityResolver` returns `ErrTenantUnresolved`" now holds only when the token also carries `aud=nava-api`. With only `aud=nava-operator-api` it returns `ErrInvalidToken`, because TICK-94 item 11 gives each verifier one audience. Both map to 401, so T20 is unchanged.
- **Deviation accepted (T04 carry-over):** no telemetry accessor for the JWKS `*http.Client`. jwx and httprc take any `Do` client, so the adapter passes `*telemetry.OutboundClient` through `routedClient`. No `*http.Client` leaves telemetry, there is no forbidigo exception, and URL redaction stays on.
- **Deferred:**
  - `jwks_uri` canonicalisation and same-origin redirects: TICK-152 (T15a), released when T15 merges.
  - The operator role must not be reachable through IdP mappers or default/composite roles: an acceptance criterion on TICK-151.
  - navalint `idpclaims` flags any `"organization"`/`"idp"` constant outside the adapter. Trigger: the first false positive in a domain package; then add a scoped exemption.
- **Dissent:** none.

## 2026-10-23: PLAN-001-A2 splits ADR-001-A2 delivery into three tasks; R-CORE gates the first nava-core Temporal call
- **Decided by:** the Technical Lead on TICK-130 (`docs/delivery/plan-adr-001-a2.md`). Approved by nava on TICK-130 (2026-10-23). ADR-001-A2 was approved by nava on TICK-129 (2026-10-23).
- **Decision:** the ADR's single `core-dev` acceptance becomes three PRs under S7 (TICK-49), each waiting for PR #38 to merge:
  - T27i (TICK-131): D9b+, D9d (both clauses), D9e, D9f and D9g. Un-skips F2-1 to F2-6.
  - T27j (TICK-132), in parallel: the D6-K row `frontend.enableTokenNamespaceEnforcement` must be `true`. Un-skips F2-3b.
  - T27k (TICK-133), after T27i: the CI-C request-field drift golden.
- **R-CORE gate:** R-CORE-1 and R-CORE-2 aren't a task, because no planned task makes `nava-core` call Temporal yet. The first task that does must carry both as acceptance tests, plus the single workflow-id constructor, with Security reviewing (RS6). *Trigger:* that task is planned (likely Epic 09, TICK-12). It may use T23's (TICK-58) sealed cursor for the opaque-token clause. Owner: Technical Lead.
- **No grooming meeting:** same reason as D-A1-6.
- **Dissent:** none.

## 2026-10-20: PLAN-001-A1 splits ADR-001-A1 delivery into D9 first, then the 1.29.7 upgrade
- **Decided by:** the Technical Lead on TICK-119 (`docs/delivery/plan-adr-001-a1.md`). Approved by nava on TICK-119 (2026-10-20).
- **Decision:** RFC-002 §9's single T27d PR becomes two PRs, merged in order:
  - T27e (TICK-120): the D9 workload allow-list on 1.28.4. It closes S1, which is live on every version.
  - T27f (TICK-121): server 1.29.7 / api 1.53.0, schema 1.18, the D6-K guard and the vulncheck gate.
  - CI-A to CI-I must all be green on T27f's PR. T27d (TICK-111) closes when T27f merges.
- **E4:** no 1.28.4 data needs keeping (M1 is dev-only and single-node). T27f still checks the upgrade in place, schema-before-binary, with an in-flight execution. *Trigger:* the first non-dev deployment needs a runbook, together with F1.
- **RS5:** accepted. Each new workflow type or task queue is a policy PR plus an image rebuild. *Trigger:* more than about one such change a week, or PLAN-002 Epic 09 needing a type per definition.
- **Deferred:** F1 (mixed-version node smoke). *Trigger:* before any multi-node deployment. Owner: Technical Architect. F2 is T27g (TICK-122, `security`), released after T27f.
- **No grooming meeting.** The task text is lifted directly from the ADR, which was reviewed twice. The PR reviews carry that weight instead.
- **Dissent:** none recorded. RFC-002 §9 proposed one PR; the split keeps every test it named.

## 2026-10-20: CI wall clock under 5 minutes; stack tests split over three jobs
- **Decided by:** the Technical Lead on TICK-59, on the human's escalation (CI took about 22 minutes).
- **Decision:** `ci.yml` runs seven parallel jobs: `lint`, `unit`, `temporal-server`, `integration`, `stack`, `stack-db` and `stack-internal`. This replaces `lint-unit` and the single `stack` job. Every gate and every stack test still runs, and none is skipped or weakened. `make ci` and `make ci-stack` are unchanged locally.
- **Caching:** each job keeps a Go build cache keyed per job. CI tools are cached by Makefile hash, and image layers live in the Actions cache (`make image-ci`, `IMAGE_CACHE=gha`).
- **temporal-server image:** it no longer runs `go test -short` during the build. The `temporal-server` job already runs the full suite, which is a superset. Its Temporal dependencies now compile in a layer keyed on `go.mod`/`go.sum`.
- **Accepted cost:** a run that changes `go.sum` or a base image is cold and takes about 8 to 10 minutes. Three stacks per run use more Actions minutes.
- **Dissent:** none.

## 2026-10-19: T28 operator UI reached only through a tunnel; UI login deferred
- **Decided by:** the Technical Lead on TICK-59, on the Security Reviewer's Finding 1 (PR #21).
- **Decision:** `temporal-ui` publishes no port. Operators reach it with `make operator-tunnel`, which lives only while it runs. `stack-test.sh temporal_operator_reach` checks the default state, with no tunnel.
- **Accepted risk (Low):** while the tunnel is open, a compromised nava-core can reach the UI through `host.docker.internal` and act as `nava-operator`.
- **Deferred:** Temporal UI login through Keycloak OIDC (TODO-006). Trigger: the UI must be reachable without a tunnel, the stack runs in a shared or non-local environment, or Milestone 1 closes, whichever is first.
- **Also decided:** `temporal-server` pins `go.temporal.io/server/cmd/tools/sql` with a `tool` directive. Its indirect dependencies (`urfave/cli`, `go-md2man`, `blackfriday`) come only from the pinned server module and are accepted under the no-new-dependencies rule.
- **Dissent:** none.

## 2026-10-19: T05 keeps docker-CLI integration containers; CI gates run as parallel required checks
- **Decided by:** the Technical Lead on TICK-56 (testcontainers, 2026-10-18; gate order, 2026-10-19), after the PR #19 review.
- **Decision 1:** the integration tests keep starting containers through the docker CLI. `testcontainers-go` and `moby/go-archive` are approved, not adopted.
  - **Reason:** T03 and T04 already use this pattern, and converting would add a dependency that fixes nothing.
  - **Intent kept (TICK-94 item 16):** real containers, image tags from `docs/pins.md` (`check-pins` fails on an inline tag), and per-test cleanup. Go-started containers are removed in `t.Cleanup`. The shared container that `scripts/ci/test-integration.sh` starts is removed by its trap on exit, Ctrl-C or TERM.
  - **Trigger:** if testcontainers is adopted later, the `moby/go-archive` ≥ v0.3.0 pin comes with it (GO-2026-6253).
- **Decision 2:** "Gates, in order" means each gate is a separate required check that blocks merge. The four CI jobs (`lint-unit`, `temporal-server`, `integration`, `stack`) run in parallel, with no `needs: lint-unit`. That keeps the pipeline under the 10-minute target (6m15s). `make ci` still runs the stages in order locally.
- **Dissent:** none.

## 2026-10-17: TICK-94 rulings folded into PLAN-001; T01g owns span redaction
- **Decided by:** the Technical Architect on TICK-94 (verdicts for items 1–16, 2026-10-16). Folded in by the Technical Lead on TICK-104.
- **Decision:** each affected task (T03, T04, T05, T07, T15, T18, T20, T24, T30a, T30b, T31, T33) has an "Audit TICK-93 / TICK-94" section that quotes its conditions. T05 commits the pin rows and a forbidden-module list to `docs/pins.md`.
- **Rejected, now forbidden:** go-oidc → `jwx/v3`; gocloak → a hand-written client in `internal/identity/keycloak`; ogen → `oapi-codegen` + `kin-openapi`; golang-migrate and tern → goose; gofrs/uuid → google/uuid; golangci-lint module plugins → `go vet -vettool` (`tools/navalint`).
- **New task:** T01g (TICK-105, `core-dev`) turns the span export filter into an allow-list (item 3). T07 and T15 depend on it.
- **Technical Lead calls:** T04 renames T01d's `NAVA_METRICS_ADDR` to `NAVA_ADMIN_ADDR` and builds the shared outbound HTTP client. T15's token skew tightens from ±60 s to ±30 s (item 11). The CI-11 depguard exclusion is narrowed to `internal/identity/keycloak`, to match D-P6.
- **Dissent:** none.

## 2026-10-16: T01 telemetry on OpenTelemetry; the edge is a trace trust boundary
- **Decided by:** nava (human), on TICK-89. Shaped by the Technical Lead's review.
- **Decision:** nava-core uses the OTel API for traces and metrics. Prometheus stays only as the `/metrics` scrape format, through the OTel Prometheus exporter. The public listener uses `otelhttp.WithPublicEndpoint()`, so an inbound `traceparent` is linked and never continued. The edge propagator is TraceContext only, which drops inbound baggage. Sampling is `parentbased_traceidratio`. The request id stays server-generated.
- **Supersedes:** T01's "inbound `traceparent` is honoured" and the metric names `http_requests_total` and `http_request_duration_seconds`.
- **Dissent:** the Technical Lead preferred a separate T01c, merged after T01, to keep the PLAN-001 root unblocked. The human chose the same PR. T02 and T20 wait on it.
- **Deferred:**
  - OTLP log export (`otelslog`). Trigger: the T02 Collector exists.
  - Collector tail sampling. Trigger: T02.
  - Trace continuity across the outbox (the producer stores the trace context on the outbox row, and the relay starts the workflow as a linked span). Trigger: T29 and T30a start.
- **Erratum (2026-10-16):** otelhttp v0.72.0 removed `WithPublicEndpoint()`. The code uses `WithPublicEndpointFn(func(*http.Request) bool { return true })`, which does the same thing (otelhttp `handler.go`: new root span, and a remote inbound context is added as a link).
