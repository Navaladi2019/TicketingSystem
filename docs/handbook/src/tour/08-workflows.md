---
title: "8. Workflows"
section: tour
status: delivered
sources: [docs/acceptance/guide/s11b.html, docs/acceptance/s28-first-routed-record.md, docs/quality/x1a-first-routed-record.md, docs/acceptance/guide/bruno/S28/S28-13.bru, docs/acceptance/guide/bruno/S11b/S11b-18.bru, docs/acceptance/guide/bruno/S11b/S11b-24.bru, docs/acceptance/s11a-workflow-interpreter.md, docs/architecture/workflow-interpreter.md, cmd/nava-worker/interp_default.go, cmd/nava-worker/interp_ports.go, internal/workflow/def/schema.json, internal/configsurface/testdata/inventory.json, internal/configsurface/doc.go, internal/configsurface/cmd/compat/main.go, docs/acceptance/S32/README.md, internal/workflow/def/def.go, internal/workflowdefs/workflowdefs.go, internal/authz/policyref.go]
adrs: [ADR-001, ADR-003, ADR-008, ADR-010]
stories: [S11a, S11b, S27, S28, S32]
tables: [outbox]
packages: [internal/workflow/interp, cmd/nava-worker, internal/configsurface]
---

## In plain words

A workflow is a set of automatic steps attached to a record: wait two days, send it to a manager if the amount is high, wait for an approval, then mark it approved. An administrator publishes a workflow for a record type, and from then on every new record of that type (or every change, if the workflow says so) starts it. The steps run in the background and change the record the same way a person would, and the record's history shows that the workflow made the change. Workflows can wait, branch on a record's values and update the record. There is no screen to build them yet: an administrator sends the definition to the API.

## What you do

As the tenant administrator, publish a workflow definition with `publishWorkflowDefinitionVersion` (`POST /api/v1/workflow-definitions/{key}/versions`). Then create a record of the type it names (step 9) and watch the record change on its own.

## Try it

This is the demo the S11b acceptance kit publishes on the `incident` type, which every new tenant has. It waits one minute after an incident is created, then sets its state to `in_progress`:

```sh
curl -s -X POST "$CORE/api/v1/workflow-definitions/s11b_demo/versions" \
  -H "Authorization: Bearer $ADMIN" \
  -H "Idempotency-Key: tour-s11b-demo-v1" \
  -H "Content-Type: application/json" \
  -d '{
    "format": "nava.workflow/v1",
    "trigger": {"on": "record.created", "entity": "incident"},
    "start": "wait",
    "steps": [
      {"key": "wait",   "type": "timer",         "duration": "PT1M", "next": "triage"},
      {"key": "triage", "type": "update_record", "set": {"state": "in_progress"}, "next": "done"},
      {"key": "done",   "type": "end"}
    ]
  }'
```

`$ADMIN` is the administrator's token from [the setup](index.md#before-you-start); on a tenant you onboarded yourself, use its first admin's token ([step 2](02-sign-in.md#a-newly-onboarded-tenants-first-admin)). On such a tenant `incident` is still version 1 unless you published a new version in step 6; either works.

The `timer` step is optional. Without it (`"start": "triage"`), the record routes within a second or so of the create; with it, about a minute later.

Then create an incident (step 9) as the admin or as an agent. A newly onboarded tenant has only its first admin, so create it as the admin. `short_description` is the one field you must send (`state` starts at `new`). This body works on `incident` version 1 and version 2; add `"affected_service"` only after you published version 2 from the example in step 6, because version 1 refuses fields it does not list:

```sh
curl -i -X POST $CORE/api/v1/records/incident -H "Authorization: Bearer $ADMIN" \
  -H "Idempotency-Key: tour-inc-1" -H 'Content-Type: application/json' \
  -d '{"data":{"short_description":"Printer is offline","priority":3}}'
```

Read it again after a minute with `getRecord` (`GET /api/v1/records/incident/{id}`), and its history with `getRecordHistory` (`.../{id}/history`).

## What you see

These are the checks the S11b kit makes (Bruno folder `S11b`, requests `S11b-18` and `S11b-24`). The S28 kit (Bruno `S28.10` to `S28.13`) publishes this same definition on a newly onboarded tenant and sees the same answers; it runs in CI as the stack case `s28_first_routed_record`.

- The publish answers `201` with `current: true`, `entity: "incident"` and a server-assigned id for each step (`step_ids` has `wait`, `triage` and `done`).
- About a minute after the create (the kit allows 60 to 180 s), the incident has `state: "in_progress"` and `version: 2`.
- The history entry for version 2 has `source: workflow`, `action: update`, `changed: ["/data/state"]` and no `actor_id`.

The history says that *a* workflow made the change, not *which* one: there is no definition key or version on the entry. If the type has more than one published workflow, the history alone does not tell you which wrote the change; look the run up in the Temporal UI (namespace `nava`). The S28 timed run relies on a new tenant having one definition (finding F7).

## What happens behind it

Each record write also writes a row to a work queue (the "outbox") in the same database transaction. When a published definition's trigger matches the write, the write pipeline's event sink claims a cascade slot and queues a workflow start in that same transaction. The relay (`nava-relay`) hands the queued start to the workflow engine (Temporal), and the production worker (`nava-worker`) runs the interpreter. When the interpreter updates the record, it goes through the same write pipeline as a person, with the same checks, and the history records `source=workflow`.

- Definition versions never change once published. A run that is waiting when a new version is published finishes on the version it started with; new runs use the new one.
- A step that names a field the record type does not have is refused at publish time with `422`.
- One tenant's writes never start or signal another tenant's workflow.

Details: [workflow interpreter](../workflows/nava-interpreter.md), [outbox, relay and worker](../workflows/outbox-relay-worker.md), [S11b](../stories/s11b.md).

## The named-policy test

A `condition` step can test a field (`field`, `op` `eq`/`ne`/`in`, `value`) or, since [S32](../stories/s32.md), a named access policy (ADR-010 D9). The policy test asks whether the record's assignee or requester may read or change it under one policy of the entity type's access document:

```json
{ "key": "check", "type": "condition",
  "policy": { "entity_type": "incident", "name": "agents_write" },
  "who": "assignee", "op": "write",
  "then": "triaged", "else": "needs_owner" }
```

- `who` is `assignee` or `requester`; `op` is `read` or `write`. The step has either `policy`, `who` and `op`, or `field`, `op` and `value`, never both.
- `policy.entity_type` must be the trigger entity. Another type is refused at publish with `422 entity_type_mismatch` at `/steps/<key>/policy/entity_type`.
- The name is not looked up at publish. It is resolved each time the step runs, against the entity type's **current** access version, so a policy narrowed while a run waits applies from its next check. The definition version stays pinned; only the policy is live.
- Only that one policy is asked, not the OR of every policy. The answer is "no" when the person column is empty, the person is not a live principal of the tenant, the policy is voided, or the policy does not grant `op`.
- A name the entity type does not have (including a policy only another tenant has) is "no" as well, and is also logged, counted in `authz_policy_ref_missing_total` and audited as `policy_ref_refused`. See [S32](../stories/s32.md#finding-the-missing-policy-entry) for how an admin finds the entry.

## Adding or removing a step type

The step catalog today is `condition`, `branch`, `timer`, `wait_signal`, `update_record` and `end`, with condition operators `eq`, `ne`, `in`, the condition's named-policy test (`policy`, `who`, `op` `read`/`write`, S32) and triggers `record.created`, `record.updated`. A tenant's published definition keeps working after an upgrade within a major version: the catalog is additive-only (ADR-008 D12), and the upgrade-compatibility suite (M-X2, S27) checks that on every PR.

- **Adding** a step type, step field, operator or trigger: change `internal/workflow/def` (its `schema.json` and `Parse`), then list the new item in `internal/configsurface/testdata/inventory.json` and add a fixture `internal/configsurface/testdata/fixtures/workflow/<name>.json` that uses it. Without them `task test` fails with `unlisted: …` or `no fixture: …`.
- **Deprecating** one: keep it in the code and in the inventory, and add `{"item": "workflow.step_types/<name>", "since": "<version>"}` to the inventory's `deprecated` list. It must stay listed and its fixture must keep parsing until the next major.
- **Removing** one: only with a new major. Raise `"major"` in the inventory and write `docs/upgrade/v<major>.md` naming the item (for example `workflow.step_types/timer`) with the migration. Otherwise `task test` fails with `removed: …` and `task lint` with `compat: removed: …`. No `docs/upgrade/` note exists yet: nothing has been removed.

What each failure line means: [Troubleshooting](../operations/troubleshooting.md#a-lint-or-test-check-fails).

## Not built yet

- A workflow builder screen.
- Disabling or deleting a definition. The kit retires its demo by publishing a version that waits a day and writes nothing.
- An API to send a signal (such as `approve`) to a running workflow, so approvals are not a product feature yet.
- Conditions on state changes.
- The record history does not name the workflow or version that wrote a change.
