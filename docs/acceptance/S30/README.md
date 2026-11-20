# S30 acceptance kit: admins compose access rules from named conditions with AND, OR and NOT

Story TICK-935 · kit task TICK-944 (PLAN-011 P11-12) · ADR-010 D6, R7 · verified 2026-11-19 on a fresh stack, story branch `TICK-935/composable-access-rules` at `34e5299b`.

## What you can now do

As a tenant admin, you can build the Incident type's access from named yes/no tests, such as "it is assigned to my group" or "I raised it", and combine them with "and", "or" and "not" into named policies. You can change one test and every policy that uses it changes with it, in one save.

A broken document is refused, with a pointer to the bad part, and nothing is saved. The "why can this user see this ticket" answer names the policy that decided and the tests it reached. The rules you had before keep giving the same answers.

## Try it (about 5 minutes)

From the repository root, on the story branch, with Docker, Node.js 20 or later and Go:

```sh
make dev-ca                       # once per checkout: Temporal's dev certificates
docker compose down -v            # no volume from an earlier run
docker compose up -d --wait       # every service healthy
sh docs/acceptance/S30/run.sh     # seed, the S30 and S10 Bruno folders, the metric, the D6 check
```

`run.sh` does the following, in order:

1. Seeds tenants A and B.
2. Shows how the incident ACL is stored.
3. Runs the Bruno folder **`S30 composable access`** (47 requests), then **`S10 access`** (66 requests).
4. Shows the stored versions again.
5. Reads `authz_policy_invalid_total`.
6. Ends with the D6 check. Its last line is `D6: ... threshold 16: PASS`.

The output of the verified run is `fresh-stack-run.txt`, with no tokens or passwords.

To click through by hand, open the collection `docs/acceptance/guide/bruno/` in Bruno, select the `local` environment, and run the folder `S30 composable access` top to bottom. Each request's Docs tab says what it shows. The story users are the seeded realm users; their dev-only passwords are in `local.bru`.

| Story user | Seeded user | Role |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test` | tenant A's admin |
| `agent@tenant-a` | `alice@tenant-a.test` | in `/service-desk` |
| `requester@tenant-a` | `auditor@tenant-a.test` | in neither `/service-desk` nor admins |
| tenant B | `bob@tenant-b.test` | the isolation rows |

The document the admin publishes in S30.28 has three conditions and five policies:

| Policy | Ops | When |
|---|---|---|
| `admins` | read, write | `member_of admins` |
| `agents_see_group_not_raised` | read | `member_of /service-desk AND in_assigned_group AND NOT is_requester` |
| `agents_edit_group_new` | write | `member_of /service-desk AND in_assigned_group AND is_new` |
| `requesters_see_own` | read | `NOT member_of /service-desk AND is_requester` |
| `requesters_edit_own_new` | write | `NOT member_of /service-desk AND is_requester AND is_new` |

The three conditions:

- `in_assigned_group`: `assignment_group_id in caller.my_groups`.
- `is_requester`: `requester_id eq caller.self`.
- `is_new`: `state eq "new"`.

## Acceptance criteria

| # | Criterion | Result | Evidence (`fresh-stack-run.txt`) |
|---|---|---|---|
| 1 | The admin publishes "agents see tickets in their group that they did not raise" (`in_assigned_group AND NOT is_requester`). The agent's list shows exactly those tickets, `total` matches, and opening one the agent raised returns the same 404 as a missing id | ✅ | S30.28: published, 3 conditions and 5 policies, lint `not_over_caller_leaf` at `/rules/policies/1/when/all/2/not`. S30.29: Alice's list has the `/service-desk` incident the requester raised, not the one she raised and not the admins one. Every item is in her groups and not raised by her, and `total` equals the items. S30.30: her own incident gives a 404 byte-identical to S30.13's missing id. S30.31: the requester still sees its own two |
| 2 | A document with a reference to an unknown condition, a reference cycle, or another tenant's group id returns 422 with a pointer to the bad node, and `GET` still shows the previous version | ✅ | S30.32: `unresolved_ref` at `/rules/policies/1/when/all/1/ref`. S30.33: `ref_cycle` inside the `loop_a`/`loop_b` conditions. S30.34: `foreign_id` at `/rules/policies/1/when/all/0/member_of/groups/0` (Bob's tenant-B group), and the response does not echo the id. S30.35: same version, same document. `run.sh` "after": the refused publishes wrote no version (v3 composed, v4 the AC3 edit) |
| 3 | Editing the condition `in_assigned_group` once changes what every policy using it grants, after one publish | ✅ | S30.36: Alice gets 404 on the admins incident. S30.37 changes only `in_assigned_group` to "my groups or admins": one new version, policies unchanged. S30.38: her list now has the admins incident (read policy). S30.39 and S30.40: she opens it and edits it, 200 (write policy). S30.41: her own incident is still 404 |
| 4 | The explainer for (agent, ticket) names the granting policy and lists each named condition reached with its value | ✅ | S30.42: allow, `policy` and `rule_id` `agents_see_group_not_raised`, trace `in_assigned_group` true and `is_requester` false, no voided policy, `acl_version` the S30.37 version. S30.43: for the incident Alice raised, deny, no policy, trace `is_requester` true |
| 5 | The Incident type's existing S10 rules, read back with `GET`, come out as format 2 and give the same list and single-read answers as before the upgrade | ✅ | S30.9: the seeded S10 default reads back as format 2: `default_admins`, `default_service_desk`, `default_requester`, `default_requester_new`. S30.14 to S30.19 record Alice's and the requester's lists and single reads. S30.21 republishes the `GET` document. S30.22 to S30.27 give the same ids, totals, statuses and 404 bytes. Then the **S10 folder passes unchanged in its answers on the same stack**: 66/66 requests, 220/220 tests. A row stored as format 1 before the upgrade is proven equal in CI by `TestI5Format1ConvertedMatchesS10` (TICK-942, D12); see Known limits |

Result: `S30 composable access` 47/47 requests, 145/145 tests. `S10 access` 66/66 requests, 220/220 tests.

S30 has no UI: every criterion is an API call, so the evidence is the recorded Bruno run, not screenshots.

### D6 re-check (outcome 3)

The last step of `run.sh`, on the fresh stack after both folders:

| Check | Result |
|---|---|
| PLAN-011 D6 SQL, `d6.sql` (format 1 only) | 0 tenants, no row: no format-1 document is stored on a fresh stack (see below) |
| `tools/acceptance/acld6` over every newest document (`d6-docs.sql`), with `authz.ReadBranchBound` (TICK-938 P11-06) | 2 documents, both format 2, in 2 tenants, **max 4 read branches**, threshold 16: **PASS** |

The issue expected format 1 on a fresh stack. Since TICK-939, onboarding's `PolicyStore.PublishIfNone` (`nava-onboard reconcile-default-acl`, which `tools/seed.sh` runs) stores the default incident ACL as format 2. Only the test seam `PolicyStore.Publish` still writes format 1. So the SQL now finds nothing to count. The counter decodes both formats: on an environment that still holds format-1 rows, it gives the same number as the SQL. Its test, `tools/acceptance/acld6/main_test.go`, checks 4 for the format-1 default. The 4 here equals the 2026-11-15 result in the issue.

### Benchmark (outcome 1)

The 800k benchmark on format 2, with one composed policy, is in `docs/quality/acl-perf-800k.md`, section "S30". The support-agent list p95 is 108.7 ms plain and 81.4 ms composed (page 1), against a 150 ms budget. Format 2 emits the same SQL and plan as format 1.

### R7 alert

`authz_policy_invalid_total > 0` is the alert **`AuthzPolicyInvalid`**:

- Rule: `deploy/compose/prometheus/rules/authz.yml`, added by this task.
- Test: `deploy/compose/prometheus/tests/authz_test.yml`, run by `lineage.TestAlertRules`.
- Fires per `entity_type` when `increase(authz_policy_invalid_total[10m]) > 0`.

Local Prometheus (`operator` profile) loads it. Local dev has no Alertmanager, so it shows on the Alerts page. On this run the counter stayed at zero: OpenTelemetry exports no series until the first increment, and `run.sh` prints that. Production alert routing is outside this story, the same as every other rule in that directory.

## Isolation check

| Tried | Result |
|---|---|
| Tenant A's admin publishes a policy naming tenant B's `/service-desk` group (S30.34) | 422 `foreign_id` at the id, the same as a group of no tenant; the response does not echo the id; nothing published |
| Bob (tenant B) lists incidents (S30.44) | none of the three tenant A incidents |
| Bob opens tenant A's incident (S30.45) | 404 `not_found` |
| Bob publishes incident access in tenant B (S30.46) | 403 `forbidden`, before any body check |
| Tenant B's incident ACL after the whole run (`run.sh` "after") | still version 1, the seeded default |

The adversarial isolation tests for format 2 are TICK-943's (`internal/server/access_s30_isolation_integration_test.go`).

## Not included

- `under` (org-unit subtree) decodes but is refused as `not_available` until S31.
- No UI for composing documents: the admin writes JSON through the access route.
- No production alert routing for `AuthzPolicyInvalid`, as for every local rule.

## Known limits

- **No stored format-1 row on a fresh stack.** AC5's "before the upgrade" can't be a literal format-1 row here, because no product path writes one since TICK-939. The kit shows the S10 rules reading back as format 2 with unchanged answers: the before/after republish and the S10 folder. For format-1 rows, the I5 suite in CI shows they are converted with identical grants: `TestI5Format1ConvertedMatchesS10`, `TestTICK940MigrationOverFormat1Rows`.
- **TICK-996 (fixed, PR #778, after this run):** a format-1 rule whose converted tree breaks the child cap (for example 16 `where` conditions) was answered 500; it is now 422 `child_count` (`TestPrepareFormat1OverTheChildCapIsChildCount`). No step of this kit hits it: the S10 default rules have at most one `where` condition, and every S30 document is format 2.
- The lint `not_over_caller_leaf` on `NOT is_requester` is expected. A caller with no id would pass the `not`, but every signed-in user has one (TICK-937 security review, finding 2).
- The folder adds four ACL versions per pass (S30.21, S30.28, S30.37, S30.47) and three incidents. S30.47 republishes the document S30.9 read, so S10 and S30 run again on the same stack.
- The benchmark host is shared with other work. Contended windows are identified in the bench section, by the round-trip floor and identical plans.
