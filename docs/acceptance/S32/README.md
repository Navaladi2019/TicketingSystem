# S32 acceptance kit: a workflow step can check a named access policy

Story TICK-955 · kit task TICK-958 (PLAN-011 P11-24) · ADR-010 D9 · verified 2026-11-21 on a fresh stack, story branch `TICK-955/workflow-policy-refs` at `f249b7b5`.

## What you can now do

An automated workflow can ask "may this incident's assignee (or its requester) read it (or change it) under policy X?" and branch on the answer. It names the policy instead of copying the rule. The check uses the policy as it is at the moment of the check, so a policy you narrow also changes the outcome for workflows that are already waiting. A name that does not exist makes the check say no, and you can find that in the audit log.

## Try it (about 3 minutes)

From the repository root, on the story branch, with Docker, Node.js 20 or later and Go:

```sh
make dev-ca                                  # once per checkout: Temporal's dev certificates
docker compose down -v                       # no volume from an earlier run
docker compose up -d --build --wait          # every service healthy, images from this branch
sh docs/acceptance/S32/run.sh                # seed, admin@tenant-b, the S32 Bruno folder, the evidence
```

If another stack holds the default ports, put other ones in `.env`. The verified run used `COMPOSE_PROJECT_NAME=nava-s32`, `NAVA_CORE_PORT=38080`, `KEYCLOAK_PORT=38081`, `KEYCLOAK_PUBLIC_URL=http://localhost:38081`, `OPENOBSERVE_PORT=35080`, `PROMETHEUS_PORT=39091`.

`run.sh` does the following, in order:

1. Seeds tenants A and B.
2. Makes `admin@tenant-b.test` tenant B's admin (`docs/acceptance/S18/tools/kit.mjs admin-b`), then runs one group sync.
3. Checks that tenant A has no other incident workflow. Every current workflow on incident fires on a create, so another one would also write the state.
4. Runs the Bruno folder **`S32 workflow policy`** (39 requests).
5. Checks from the database that the AC2 run had started before the policy was narrowed and finished after it. It stops with exit 1 unless there is exactly one such run (`AC2: PASS`). Bruno alone cannot tell a waiting run from one that started after the narrowing.
6. Lists the refused-reference audit entries per tenant, and reads `authz_policy_ref_missing_total` from the worker.

The output of the verified run is `fresh-stack-run.txt`. It holds no tokens or passwords.

To click through by hand, open the collection `docs/acceptance/guide/bruno/` in Bruno, select the `local` environment, and run the folder `S32 workflow policy` top to bottom. Each request's Docs tab says what it shows.

| Story user | Seeded user | Role |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test` | tenant A's admin, creates the incidents |
| `agent@tenant-a` | `alice@tenant-a.test` | in `/service-desk`, the assignee |
| `requester@tenant-a` | `auditor@tenant-a.test` | the requester; not in `/service-desk` |
| tenant B's admin | `admin@tenant-b.test` | seeded by `run.sh` step 2 |

### What the folder sets up

- **States.** The seeded incident type has no `triaged` or `needs_owner`. S32.10 publishes a compatible version that adds both, with `new -> triaged` and `new -> needs_owner`. On a second pass the type already has them, so the request is skipped.
- **Tenant A's incident access** (S32.12): `admins` (read, write), `agents_read` (read for `/service-desk`), **`agents_write` (write for `/service-desk`)** and `requesters_see_own` (read).
- **One workflow key, `s32_route`.** Each criterion publishes its next version, so exactly one version is current:

```json
{ "key": "check", "type": "condition",
  "policy": { "entity_type": "incident", "name": "agents_write" },
  "who": "assignee", "op": "write",
  "then": "triaged", "else": "needs_owner" }
```

`triaged` and `needs_owner` are `update_record` steps that set `state`. AC2's version puts a timer step `PT20S` before `check`.

## Acceptance criteria

| # | Criterion | Result | Evidence (`fresh-stack-run.txt`) |
|---|---|---|---|
| 1 | A workflow "if the assignee may write under `agents_write`, set state `triaged`, else `needs_owner`" routes an incident assigned to `agent@tenant-a` to `triaged` | ✅ | S32.13 publishes it. S32.16: Alice's incident is `triaged` 0.7 s after the create. S32.18: its history's newest entry has `source: workflow`. Control S32.17: the same workflow sends an incident assigned to the requester, who is not in `/service-desk`, to `needs_owner` |
| 2 | After the admin removes the agent's group from `agents_write`, a workflow that was already waiting on a timer before its check routes its incident to `needs_owner` once the timer fires | ✅ | S32.19: version with `PT20S` before the check. S32.20 creates an incident assigned to Alice. S32.21: 3 s later it is still `new`. S32.22: `agents_write` now names the admins group only, published 3.1 s after the run started. S32.23: `needs_owner` 20.9 s after the create, after the narrowing. `run.sh` step 5: `workflow_ref` started 21:20:23.02, policy narrowed 21:20:26.14, run closed 21:20:43.82, both checks `t`: `AC2: PASS` |
| 3 | A workflow naming policy `does_not_exist` routes to `needs_owner`, and the audit log shows a missing-policy entry | ✅ | S32.24 first gives `agents_write` back to `/service-desk`, so Alice may write again and only the name decides. S32.25: publish accepts the unknown name (resolved at run time). S32.27: `needs_owner`. S32.28: the admin's audit read has one `policy_ref_refused` entry with `content_sha256` = SHA-256 of `incident/does_not_exist`. See "How the admin finds the entry" below |
| 4 | A workflow in tenant A cannot test a policy of tenant B's entity type: same result as a missing name | ✅ | S32.30: tenant B's admin adds the policy `tenant_b_only` to tenant B's incident type. S32.31: tenant A's `s32_route` names it. S32.33: `needs_owner`. S32.34: tenant A's audit has a `policy_ref_refused` entry for `incident/tenant_b_only`, with the same action, domain, subject, actor and source as the AC3 entry. S32.35: tenant B's audit has no refused entry. S32.36: naming another entity type's policy is refused when the workflow is published: 422 `workflow_definition_invalid`, `entity_type_mismatch` at `/steps/check/policy/entity_type` |

Result: `S32 workflow policy` 39/39 requests, 102/102 tests. `authz_policy_ref_missing_total{entity_type="incident"}` went from no series to 2, one for AC3 and one for AC4.

S32 has no UI. Every criterion is an API call, so the evidence is the recorded Bruno run, not screenshots.

### How the admin finds the missing-policy entry (AC3)

1. Read the audit log as a tenant admin. The admin has `audit.read`:

   ```
   GET /api/v1/audit/entries?domain=config&subject_type=acl_policy&from=<time the workflow ran, minus a minute>
   ```

2. Find the entries with `"action": "policy_ref_refused"`. They have `actor_system: "workflow"` and `source: "workflow"`, and no `actor_principal_id`.
3. `subject_id` (and `entity_definition_id`) is the incident type's access policy. It is the same subject as the admin's own `PUT /entity-types/incident/access` entries in the same list, so the admin can tell which entity type refused the name.
4. **The entry does not hold the policy name.** `content_sha256` is the SHA-256 of `<entity_type>/<name>`. To check that an entry is for the name your workflow uses, hash it: `printf 'incident/does_not_exist' | sha256sum`. The worker log has the name in plain text: the warning `authz: workflow policy reference names no policy, it denies`, with `entity_type`, `ref_entity_type` and `policy`.

S32.28 does exactly this: it filters by `action` and the hash, and compares `subject_id` with the admin's access publishes.

## Isolation check

| Tried | Result |
|---|---|
| Tenant A's workflow names `tenant_b_only`, a policy only tenant B's incident type has (S32.31 to S32.33) | `needs_owner`, the same as `does_not_exist` |
| Tenant A's audit for that run (S32.34) | the same `policy_ref_refused` entry as AC3, for tenant A's own incident access policy |
| Tenant B's audit since the start of the folder (S32.35) | no `policy_ref_refused`; no entry with tenant A's subject id |
| Tenant A publishes a workflow whose policy test names another entity type (S32.36) | 422 `entity_type_mismatch` at `/steps/check/policy/entity_type`, nothing published. A test can name only the trigger entity's own type, whichever tenant has that key |
| `run.sh` step 6 | both refused entries are tenant A's; tenant B has none |

The adversarial tests under RLS (same entity type key and policy name in two tenants with different rules, another tenant's record, a run with no `workflow_ref` of the tenant) are TICK-957's: `internal/records/pipeline/policyref_integration_test.go`.

## Runs again on the same stack

The last three requests restore what the folder changed. S32.37 publishes a version of `s32_route` that ends at once: a definition cannot be deleted, and any current version fires on every incident create. S32.38 and S32.39 republish both tenants' incident access as the folder found it. The incident type keeps the two extra states.

Verified on the same stack after the fresh run:

- A second pass: 39 requests (38 passed, S32.10 skipped), 100/100 tests. S32.38 and S32.39 check that both tenants' `rules` and `masks` are back as they were.
- The S30 folder (first fix round): 47/47 requests, 144/144 tests.

The restore steps run only at the end of a pass. **After a failed pass, use a fresh stack.** A failure part-way can leave the S32 access document in place, and the next pass would then take it as the original.

## Not included

- No UI or canvas for the policy test. The admin writes the workflow JSON through `POST /api/v1/workflow-definitions/{key}/versions`.
- `who: requester` and `op: read` are not walked here. They take the same code path as `assignee` and `write`. TICK-957's integration tests cover both `who` values.
- A voided policy (`authz_policy_invalid_total`) is not walked. It is covered by TICK-957's unit tests.

## Known limits

- **The audit entry holds a hash of the policy name, not the name** (see AC3 above). An admin has to hash the name to match an entry, or read the worker log. This is how TICK-957 built it, not a kit choice. A follow-up could add the name to the entry; that is a product decision for the story owner.
- **A Temporal retry can audit and count one refusal twice.** The test commits its own transaction (TICK-957, `policyRefMissing`). Both refused references in the verified run wrote exactly one entry, and S32.28 and S32.34 accept one or more.
- **AC2 depends on the clock.** The narrowing has to be published within the 20-second timer. In the verified run it came 3.1 s after the run started. S32.22 fails if the narrowing is late, and `run.sh` step 5 fails if the run started late (after the narrowing).
- **Needs a tenant A with no other incident workflow.** On a stack where the S11b folder ran, its workflows also fire on S32's creates. `run.sh` step 3 stops on that, so use a fresh stack.
