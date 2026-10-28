# Regression: milestone-1 on main after S5 and S7 (TICK-286)

**REGRESSION: FAIL (4 findings)**

- **main SHA:** `4392c26ceba541adbc172e725669f80d083cea34` (S5 squash, #221; S7 is `c3671e5`, #223, an ancestor)
- **Date:** 2026-10-28
- **Stack:** fresh clone, `docker compose down -v`, `make dev-ca`, `docker compose build`, `docker compose up --wait`, project `nava-regr`, ports 28080/28081, `KEYCLOAK_PUBLIC_URL=http://localhost:28081`
- **Evidence:** `docs/acceptance/regression-milestone-1-evidence/` (one file per story plus `cross-story.md`; no tokens: every JWT is cut to 10 characters, the IdP client secret is redacted)

## Findings (most severe first)

| # | Severity | Finding | Route to |
|---|---|---|---|
| F1 | **Bug** | **Story S6 (TICK-48) never reached `main`.** The story is `done`, but `TICK-48/api-ground-rules` was never merged and no story PR exists. Seven commits (4 non-merge) are only on that branch: T24 per-tenant rate limits (TICK-77, #136, review #154), T22a idempotency lease (TICK-212, #141) and T22b request-hash hardening (TICK-213, #175). On the live stack, `GET /api/v1/me` sends no `RateLimit*`/`Retry-After` headers, and 60 rapid calls all return 200. Reproduce: `git merge-base --is-ancestor origin/TICK-48/api-ground-rules origin/main` exits 1, and `git log --oneline origin/main..origin/TICK-48/api-ground-rules` lists 7 commits. Expected: S6's tasks are on `main`, as S1–S5 and S7 are. | Technical Lead (story → `main`: merge `main` into TICK-48, full CI, story PR) |
| F2 | **Breaks under merge** | **Three open PRs target the S5 story branch, which is already squash-merged:** #251 (TICK-297), #252 (TICK-282) and #255 (TICK-283), all with base `TICK-47/group-changes-within-10-minutes`. If merged as they are, their changes land on a dead branch and never reach `main`. | Technical Lead (retarget to `main` and re-stamp any migrations, or close) |
| F3 | **Missing test** (kit tooling drift) | **The S3 acceptance script fails when run verbatim (10 of 16).** `docs/acceptance/s3-tenant-signin/accept.py` hard-codes `ISSUER = "http://keycloak:8080/realms/nava"` and the `Host: keycloak:8080` trick. Since TICK-160, Keycloak pins `iss` to `KEYCLOAK_PUBLIC_URL`, so no binding matches and Alice gets `401 tenant_unresolved`. With the bindings re-seeded to the pinned issuer, 15 of 16 pass. The 16th, "untrusted issuer via missing Host header", is now obsolete: the same realm is no longer a different issuer. A master-realm token is still refused (`401 invalid_token`, logged `issuer`). The product holds, but the kit can no longer prove it. "Try it" step 3 is stale for the same reason. | S3 kit owner (QA): bind `${KEYCLOAK_PUBLIC_URL}/realms/nava`, drop the Host trick, use a different-realm token for the issuer case |
| F4 | **Bug** (test code) | **The S4 acceptance harness does not compile on `main`.** `go vet -tags 'identitystack acceptance' ./internal/identitystack` fails with `s4_acceptance_test.go:78:6: onboardPostRaw redeclared`, which clashes with `onboard_e2e_test.go:226`. The 3-result `onboardPostRaw` came from `1f5a376` (TICK-253). CI never builds the `acceptance` tag, so nothing caught it. For this run I used a `go test -overlay` copy with the S4 helper renamed: all 43 checks pass. The product holds, but the S4 kit cannot be run as written. | S4 owner: rename the helper. QA (me): a `make` target plus a lint-job step that compiles every build-tag combination (`go vet -tags 'identitystack acceptance'`) |

Not counted in the verdict:

| # | Severity | Note |
|---|---|---|
| N1 | **Spec drift** (in this task, not the code) | The task asks for migrations "`down` to zero, `up` again". Every migration on `main` is forward-only by repo rule (`db/README.md`: "Forward only"; no `-- +goose Down` in any of the 12 files), so down/up cannot run and should not be added. I verified up-from-empty instead (see Migrations). Flagged to the Technical Lead so the regression template can drop the step. |
| N2 | Flaky (infrastructure) | Run 1's `temporal-server` job failed before any test ran: `proxy.golang.org ... stream error: INTERNAL_ERROR` while downloading `google.golang.org/genproto`. Re-running the failed job went green (attempt 2). The failure made `main-red` open P0 issue #257 ("main is red: 4392c26"), which is still open but stale: main is green on this SHA. |
| N3 | Preference (kit wording) | S5 kit says the stack has no `nava-groupsync` service or onboarding API; main has both. S7 kit says `nava_app` gets `permission denied` reading `outbox`; `grants.yaml` now grants `SELECT (tenant_id, id, dedupe_key)` behind the tenant policy. Isolation still holds in both cases. |
| N4 | Observation, routed to Security & Multi-Tenancy Reviewer | Relay-started workflows carry no `NavaTenantId` search attribute (`NavaTenantId='<tenant>'` returns 0 for every tenant). The worker guard only refuses a wrong one. The tenant is still bound by the workflow id prefix `t:<tenant>:`. Not a kit criterion. |

## 1. Merge-integrity audit

Method: for every child task of S1–S7, take the PRs from its `PR #N` labels. Where a task has no label, search PR titles and branches for the task id, then commit messages. A PR counts as **on main** when its merge commit is an ancestor of `4392c26` (`git merge-base --is-ancestor`). For a task PR merged into a story branch that was squash-merged (S5, S7), it counts when the merge commit is an ancestor of the story PR's head and that story PR is on `main`. Squash integrity: the trees of the S5 PR #221 head and `4392c26` are identical (`git diff --stat` empty), and likewise for the S7 PR #223 head and `c3671e5`. Both branch tips equal their PR heads, so nothing was pushed after the merge.

Story PRs: S1 #100, S2 #135/#167, S3 #207, S4 #215, S5 #221, S7 #223 are all merged and on `main`. **S6: none (F1).** Cancelled tasks (TICK-92, 138, 238, 239, 246, 247) are excluded.

Story branches with commits not on `main`:

| Branch | Non-merge commits not on `main` | Why |
|---|---|---|
| `TICK-45/tenant-user-signin`, `TICK-46/operator-onboards-tenant` | 0 | merge-merged |
| `TICK-47/group-changes-within-10-minutes`, `TICK-47/acceptance-kit` | 34, 14 | squash-merged as `4392c26`; tree identical |
| `TICK-49/outbox-row-runs-one-workflow`, `TICK-49/ci6-report-artifact` | 31, 28 | squash-merged as `c3671e5`; tree identical |
| **`TICK-48/api-ground-rules`** | **4 (7 with merges)** | **never merged (F1)** |

Done tasks: 135 task-PR rows. 130 are on main or have a legitimate reason they don't need to be (closed throwaway or superseded PR). The 5 exceptions are all S6 (F1).

<details><summary>Full table: task, PR, merged?, on main?</summary>

`*` = PR found by searching for the task id, not by label.

| Story | Task | PR | Merged? | On main? |
|---|---|---|---|---|
| S1 | TICK-50 | #2* | yes | yes |
| S1 | TICK-51 | #9* | yes | yes |
| S1 | TICK-54 | none | n/a | yes, direct commits (T03, `7a4df46`) |
| S1 | TICK-55 | none | n/a | yes, direct commits (T04, `b8aba0b`) |
| S1 | TICK-56 | #19 | yes | yes |
| S1 | TICK-60 | #25 | yes | yes |
| S1 | TICK-90 | none | n/a | yes, `b4fe705` (delivered by TICK-117) |
| S1 | TICK-98 | #6* | yes | yes |
| S1 | TICK-99 | #7* | yes | yes |
| S1 | TICK-100 | none | n/a | yes, direct commits (T01d, `b3a6610`) |
| S1 | TICK-101 | #47 | yes | yes |
| S1 | TICK-103 | #10* | yes | yes |
| S1 | TICK-107 | #50 | yes | yes |
| S1 | TICK-142 | #49 | yes | yes |
| S1 | TICK-144 | #55 | yes | yes |
| S1 | TICK-156 | #72 | yes | yes |
| S1 | TICK-164 | #95 | yes | yes |
| S1 | TICK-166 | #90 | yes | yes |
| S1 | TICK-225 | #184 | yes | yes |
| S2 | TICK-61 | #46 | yes | yes |
| S2 | TICK-62 | #23 | yes | yes |
| S2 | TICK-64 | #44 | yes | yes |
| S2 | TICK-66 | #48 | yes | yes |
| S2 | TICK-67 | #76 | yes | yes |
| S2 | TICK-68 | #51 | yes | yes |
| S2 | TICK-68 | #69 | yes | yes |
| S2 | TICK-71 | #80 | yes | yes |
| S2 | TICK-105 | #14* | yes | yes |
| S2 | TICK-105 | #13* | yes | yes |
| S2 | TICK-113 | #24 | yes | yes |
| S2 | TICK-114 | #104 | yes | yes |
| S2 | TICK-143 | #127 | yes | yes |
| S2 | TICK-145 | #120 | yes | yes |
| S2 | TICK-146 | #128 | yes | yes |
| S2 | TICK-163 | #79 | yes | yes |
| S2 | TICK-179 | #85 | yes | yes |
| S2 | TICK-179 | #8 | yes | yes |
| S2 | TICK-180 | #99 | yes | yes |
| S2 | TICK-217 | #123 | yes | yes |
| S2 | TICK-233 | #139 | yes | yes |
| S2 | TICK-233 | #143 | closed | n/a: throwaway probe PR, not meant to merge; #139 carries the fix |
| S2 | TICK-235 | #145 | yes | yes |
| S3 | TICK-57 | #43 | yes | yes |
| S3 | TICK-57 | #45 | yes | yes |
| S3 | TICK-63 | #56 | yes | yes |
| S3 | TICK-69 | #88 | yes | yes |
| S3 | TICK-72 | #97 | yes | yes |
| S3 | TICK-151 | #89 | yes | yes |
| S3 | TICK-151 | #92 | yes | yes |
| S3 | TICK-152 | #103 | yes | yes |
| S3 | TICK-184 | #94 | yes | yes |
| S3 | TICK-189 | #151 | yes | yes |
| S3 | TICK-190 | #131 | yes | yes |
| S4 | TICK-73 | #101 | yes | yes |
| S4 | TICK-81 | #130 | yes | yes |
| S4 | TICK-194 | #115 | yes | yes |
| S4 | TICK-195 | #117 | yes | yes |
| S4 | TICK-195 | #119 | yes | yes |
| S4 | TICK-204 | #182 | yes | yes |
| S4 | TICK-219 | #124 | yes | yes |
| S4 | TICK-234 | #162 | yes | yes |
| S4 | TICK-242 | #157 | yes | yes |
| S4 | TICK-249 | #161 | yes | yes |
| S4 | TICK-252 | #163 | yes | yes (via #162 → S4 #215) |
| S4 | TICK-254 | #164 | yes | yes |
| S4 | TICK-257 | #165 | yes | yes |
| S4 | TICK-271 | #209 | yes | yes |
| S4 | TICK-271 | #213 | yes | yes |
| S4 | TICK-276 | #219 | yes | yes |
| S4 | TICK-278 | #217 | yes | yes |
| S4 | TICK-279 | #218 | yes | yes |
| S4 | TICK-279 | #217 | yes | yes |
| S5 | TICK-75 | #159 | yes | yes (via story PR #221) |
| S5 | TICK-78 | #113 | yes | yes |
| S5 | TICK-78 | #121 | yes | yes |
| S5 | TICK-78 | #125 | yes | yes |
| S5 | TICK-82 | #153 | yes | yes (via story PR #221) |
| S5 | TICK-82 | #155 | yes | yes (via story PR #221) |
| S5 | TICK-220 | #186 | yes | yes (via story PR #221) |
| S5 | TICK-221 | #152 | yes | yes (via story PR #221) |
| S5 | TICK-222 | #129 | yes | yes (via story PR #221) |
| S5 | TICK-223 | #147 | yes | yes |
| S5 | TICK-263 | #192 | yes | yes (via story PR #221) |
| S5 | TICK-264 | #191 | yes | yes (via story PR #221) |
| S5 | TICK-265 | #198 | yes | yes (via story PR #221) |
| S5 | TICK-266 | #197 | yes | yes |
| S5 | TICK-267 | #201 | yes | yes (via story PR #221) |
| S5 | TICK-268 | #204 | yes | yes (via story PR #221) |
| S5 | TICK-274 | #211 | yes | yes (via story PR #221) |
| S5 | TICK-281 | #224 | yes | yes (via story PR #221) |
| S5 | TICK-281 | #238 | yes | yes (via story PR #221) |
| S5 | TICK-281 | #242 | closed | n/a: superseded; the other TICK-281 PRs merged |
| S5 | TICK-281 | #244 | yes | yes (via story PR #221) |
| S5 | TICK-281 | #250 | yes | yes (via story PR #221) |
| S5 | TICK-284 | #226 | closed | n/a: superseded; renumbering is on main (00010–00012) |
| S5 | TICK-298 | #233 | yes | yes (via story PR #221) |
| S5 | TICK-300 | #240 | yes | yes (via story PR #221) |
| S5 | TICK-302 | #239 | yes | yes (via story PR #221) |
| S5 | TICK-308 | #244 | yes | yes (via story PR #221) |
| S6 | TICK-52 | none | n/a | yes, direct commits (T20, `a5a6db1`) |
| S6 | TICK-58 | #66 | yes | yes |
| S6 | TICK-58 | #68 | yes | yes |
| S6 | TICK-76 | #111 | yes | yes |
| S6 | TICK-76 | #114 | yes | yes |
| S6 | TICK-77 | #136 | yes | **NO** (F1) |
| S6 | TICK-77 | #154 | yes | **NO** (F1) |
| S6 | TICK-102 | #67 | yes | yes |
| S6 | TICK-157 | #74 | yes | yes |
| S6 | TICK-161 | #174 | yes | yes |
| S6 | TICK-212 | #141 | yes | **NO** (F1) |
| S6 | TICK-213 | #172 | closed | n/a: superseded by #175 (which is **not** on main, see F1) |
| S6 | TICK-213 | #175 | yes | **NO** (F1) |
| S7 | TICK-53 | none | n/a | yes, direct commits (T27a, `9411535`) |
| S7 | TICK-59 | #21 | yes | yes |
| S7 | TICK-65 | #73 | yes | yes |
| S7 | TICK-65 | #75 | yes | yes |
| S7 | TICK-65 | #77 | yes | yes |
| S7 | TICK-70 | #82 | yes | yes |
| S7 | TICK-74 | #87 | yes | yes |
| S7 | TICK-79 | #39 | yes | yes |
| S7 | TICK-80 | #156 | closed | n/a: superseded; `internal/relay/flood_integration_test.go` is on main via S7 #223 |
| S7 | TICK-84 | #20 | yes | yes |
| S7 | TICK-85 | #96 | yes | yes |
| S7 | TICK-85 | #98 | yes | yes |
| S7 | TICK-86 | #116 | yes | yes (via story PR #223) |
| S7 | TICK-110 | #22 | yes | yes |
| S7 | TICK-111 | #26 | yes | yes |
| S7 | TICK-118 | #30 | yes | yes |
| S7 | TICK-120 | #31 | yes | yes |
| S7 | TICK-121 | #34 | yes | yes |
| S7 | TICK-122 | #36 | yes | yes |
| S7 | TICK-125 | #33 | yes | yes |
| S7 | TICK-127 | #35 | yes | yes |
| S7 | TICK-128 | #37 | yes | yes |
| S7 | TICK-131 | #41 | yes | yes |
| S7 | TICK-132 | #42 | yes | yes |
| S7 | TICK-133 | #102 | yes | yes |
| S7 | TICK-133 | #109 | yes | yes |
| S7 | TICK-159 | #83 | yes | yes |
| S7 | TICK-171 | #118 | yes | yes |
| S7 | TICK-173 | #110 | yes | yes |
| S7 | TICK-174 | #126 | yes | yes (via story PR #223) |
| S7 | TICK-191 | #144 | yes | yes (via story PR #223) |
| S7 | TICK-211 | #122 | yes | yes |
| S7 | TICK-214 | #148 | yes | yes (via story PR #223) |
| S7 | TICK-215 | #146 | yes | yes (via story PR #223) |
| S7 | TICK-216 | #149 | yes | yes (via story PR #223) |
| S7 | TICK-224 | #179 | yes | yes |
| S7 | TICK-227 | #173 | yes | yes |
| S7 | TICK-228 | #177 | yes | yes (via story PR #223) |
| S7 | TICK-240 | #158 | yes | yes (via story PR #223) |
| S7 | TICK-241 | #169 | yes | yes (via story PR #223) |
| S7 | TICK-245 | #189 | yes | yes (via story PR #223) |
| S7 | TICK-245 | #200 | yes | yes (via story PR #223) |
| S7 | TICK-248 | #160 | yes | yes (via story PR #223) |
| S7 | TICK-250 | #170 | yes | yes (via story PR #223) |
| S7 | TICK-275 | #212 | yes | yes (via story PR #223) |
| S7 | TICK-277 | #216 | yes | yes (via story PR #223) |
| S7 | TICK-285 | #229 | yes | yes (via story PR #223) |
| S7 | TICK-305 | #243 | closed | n/a: superseded; S7 merged as #223 |

</details>

## 2. Migrations

| Check | Result | Evidence |
|---|---|---|
| Versions unique and contiguous | PASS | `00001`–`00012`, no gaps or duplicates; `BASE=origin/main go test -count=1 ./db/`: `TestMigrationVersions`, `TestMigrationsParseWithGoose`, `TestNewMigrationsLaterThanBase` PASS |
| `up` from an empty DB | PASS | fresh stack `migrate` job: `"migrate done","migrations_applied":12,"grants_granted":134,"grants_revoked":0` (s2 evidence); CI `stack-db` idempotent re-run green ×3 |
| `down` to zero, `up` again | N/A (N1) | forward-only by design |
| RLS on every tenant table | PASS | superuser query on the fresh stack: tables without RLS+FORCE = `goose_db_version` only; 12 tenant tables all FORCE. CI-2 isolation report: 200 PASS, 4 ALLOWLISTED, 0 real FAIL (the 12 FAIL rows are the negative fixtures, which are expected to fail) |
| Outbox CHECKs (T32c/T32f) | PASS | `00008_outbox_payload_references.sql`, `00009_outbox_payload_keys.sql` applied; CI-7 rows of the CI-6/7 report PASS; integration job green ×3 |

## 3. CI on main in GitHub Actions (SHA `4392c26`)

| Run | Trigger | lint (incl. `vulncheck`) | unit | temporal-server (incl. `vulncheck-temporal-server`) | integration (isolation report) | stack (CI-5, CI-6/7 reports) | stack-db | stack-internal |
|---|---|---|---|---|---|---|---|---|
| [40729414061](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40729414061) | push to `main` | ✅ | ✅ | ❌ attempt 1 (N2), ✅ attempt 2 | ✅ | ✅ | ✅ | ✅ |
| [40729696092](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40729696092) | push of `TICK-286/regression-milestone-1` at the same SHA | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| [40729696277](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40729696277) | `workflow_dispatch` on `main` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

Reports (artifacts of run 40729696092):
- `ci5-report`: CI-5 adversarial identity suite, 14/14 PASS
- `ci6-report`: CI-6 certificate/network/Authorizer/relay/worker plus CI-7 workerguard, 46/46 PASS
- `isolation-report`: CI-2/5/8/9/10/T26, result PASS

## 4. Story acceptance kits, one fresh stack, S1 → S7

"Covered by CI" means the kit step is a CI script that agents must not run locally. Its evidence is the green jobs in §3.

| Story | Criterion | Result | Evidence |
|---|---|---|---|
| S1 | 1 healthy < 3 min | ✅ 159 s | `docker compose up --wait` exit 0, 12 services Healthy (s1) |
| S1 | 2 healthz lists deps; Keycloak down → degraded, readyz 200 | ✅ | `{"status":"degraded","checks":{"keycloak":"fail",...}}`, readyz 200 ×3 (s1) |
| S1 | 3 Postgres down → readyz non-200 < 5 s | ✅ ~0.55 s | `HTTP/1.1 503` (s1) |
| S1 | 4 pins | ✅ | `make check-pins` → `ok` |
| S1 | 5 failing test turns CI red | covered by CI | kit PRs #71/#78; `main-red` fired on this very SHA (N2) |
| S2 | 1 no/empty/leftover tenant → 0 rows | ✅ | direct and through PgBouncer, 4 role paths: 0/0/0/0, own 3; cross-tenant INSERT refused by RLS (s2); report half: CI isolation report |
| S2 | 2 no superuser/BYPASSRLS/ownership/membership | ✅ | all runtime roles `f`; `pg_auth_members` 0; owner `nava_migrator` (41 relations) |
| S2 | 3 grants == `grants.yaml`; hand grant fails CI | ✅ / covered by CI | `grants_revoked:0`; probe PR #143 |
| S2 | 4 relay cannot read tenant data; backup no SQL | ✅ | `permission denied` ×4; `pg_hba.conf rejects` ×3 |
| S3 | kit script verbatim | ❌ **F3** (10/16) | `s3-accept-run1-verbatim.txt` |
| S3 | 1 Alice /me → tenant A + groups | ✅ (re-seeded issuer) | `200 ... "path":"/service-desk" ... tenant ...a1` |
| S3 | 2 two orgs / no org → 401 | ✅ | 3/3 `401 tenant_unresolved` |
| S3 | 3 ID token / untrusted issuer rejected | ✅ | ID token and master-realm token `401 invalid_token` |
| S3 | forged tenant B in header/query/body/baggage → A | ✅ | 5/5 tenant A |
| S3 | isolation: Bob → B only | ✅ | disjoint groups; forging A → B |
| S4 | `TestS4Acceptance` verbatim | ❌ **F4** (does not build) | `onboardPostRaw redeclared` |
| S4 | 1 onboard 201, replay identical, nothing new; reuse → 409 | ✅ (overlay copy) | `TestS4Acceptance`, two tenants (s4) |
| S4 | 2 new admin signs in, scoped to the new tenant | ✅ | `/me` tenant = onboarded id, invite consumed |
| S4 | 3 invite of a user federated elsewhere rejected | ✅ | no invite; sign-in refused "already exists" |
| S4 | isolation | ✅ | header/query forging ignored; claimed domain 409 `domain_claimed` without leaking A |
| S5 | 1 group removal < 10 min | ✅ 75 s | same token `/me` `group_paths:[]` (s5) |
| S5 | 2 disabled user's token stops < 10 min | ✅ 285 s | `401 tenant_unresolved`, token unexpired |
| S5 | 3 synced tables never show another tenant | ✅ | every role × tenant: other-tenant rows 0 |
| S5 | 4 kc.org tamper, ID token, foreign-IdP user rejected | ✅ by hand + CI-5 | `401` each |
| S5 | 5 B's admin gets no token accepted for A | ✅ by hand + CI-5.13 | |
| S5 | 6 tenant admin 403 operator; operator 401 tenant | ✅ | 403 nava-onboard; 401 nava-core |
| S6 | (no kit; story not on main) | ❌ **F1** | ground rules on main checked in §5c |
| S7 | 1 one workflow per start row, delivered twice | ✅ | `make demo-outbox` ×2 → 3 rows, 1 execution (s7) |
| S7 | 2 signals in commit order, each once | ✅ + CI-6 21–23 | history events 3/4 in order |
| S7 | 3 worker signalling tenant-B workflow denied + alert | covered by CI | CI-6 case 8 PASS |
| S7 | 4 relay killed between claim and start: no loss, one workflow | ✅ | `relaytest` drill: attempts 1→2, 1 execution |
| S7 | 5 CI-6 Authorizer cases | covered by CI | `ci6-report` 46/46 |
| S7 | isolation | ✅ | B sees 0 of A's outbox; forged signal row marked dead, A's history unchanged |

## 5. Cross-story checks (`cross-story.md`)

| Check | Result | Evidence |
|---|---|---|
| a. S4-onboarded tenant signs in (S3), gets groups synced (S5), and its outbox rows run exactly one tenant-scoped workflow (S7), all on one issuer config | ✅ | tenant `e948c248…`: 201; brokered sign-in; `/me` gains `/regr-team` after one sync; demo-outbox ×2 → 3 rows, 1 execution `t:e948c248…:nava.smoke.relay:…` |
| b. Group sync and relay running together: no cross-tenant rows | ✅ | 21 role×tenant lines, 0 other-tenant rows (only the allow-listed `binding_lookup`); superuser sweep 0 except the deliberately forged S7 probe row (marked dead) |
| c. S6 ground rules on `/api/v1/me` (S5) | ✅ for what is on main | 401/404/405 all `application/problem+json` RFC 9457 with `X-Request-Id` and `instance`; rate limit and idempotency absent (F1) |
| d. Tenant admin 403 on `/api/v1/operator/*`; operator token 401 on tenant endpoints | ✅ | 403 `urn:nava:problem:forbidden` (also for the onboarded admin); operator → `/me` 401 `invalid_token` |

Security & Multi-Tenancy Reviewer: review of §5 requested on TICK-286.

## 6. Flakiness

`integration` and `stack` (plus every other job) ran 3 times on `4392c26`, all green. **Flaky tests: none.** One infrastructure flake: the Go module proxy download in `temporal-server`, before any test ran (N2).

## Reproduce

- Audit: `git fetch origin '+refs/pull/*/head:refs/remotes/pr/*'`, then `git merge-base --is-ancestor <merge-oid> 4392c26` per PR (`gh pr list --state all --json number,state,baseRefName,headRefOid,mergeCommit`)
- Migrations: `BASE=origin/main go test -count=1 ./db/`
- CI: `gh workflow run ci --ref main`, then read the run
- Kits: each kit's "Try it" from a fresh `docker compose down -v`, deviations as listed in the evidence files
