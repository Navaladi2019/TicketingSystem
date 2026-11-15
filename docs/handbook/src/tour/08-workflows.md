---
title: "8. Workflows"
section: tour
status: delivered
sources: [docs/acceptance/guide/s11b.html, docs/acceptance/guide/bruno/S11b/S11b-18.bru, docs/acceptance/guide/bruno/S11b/S11b-24.bru, docs/acceptance/s11a-workflow-interpreter.md, docs/architecture/workflow-interpreter.md, cmd/nava-worker/interp_default.go, cmd/nava-worker/interp_ports.go, internal/workflow/def/schema.json, internal/configsurface/testdata/inventory.json, internal/configsurface/doc.go, internal/configsurface/cmd/compat/main.go]
adrs: [ADR-001, ADR-003, ADR-008]
stories: [S11a, S11b, S27]
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
  -H "Authorization: Bearer $ADMIN_TOKEN" \
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

Then create an incident as an agent (step 9) and read it again after a minute with `getRecord`, and its history with `getRecordHistory`.

## What you see

These are the checks the S11b kit makes (Bruno folder `S11b`, requests `S11b-18` and `S11b-24`); this page was not re-run against a tour stack.

- The publish answers `201` with `current: true`, `entity: "incident"` and a server-assigned id for each step (`step_ids` has `wait`, `triage` and `done`).
- About a minute after the create (the kit allows 60 to 180 s), the incident has `state: "in_progress"` and `version: 2`.
- The history entry for version 2 has `source: workflow`.

## What happens behind it

Each record write also writes a row to a work queue (the "outbox") in the same database transaction. When a published definition's trigger matches the write, the write pipeline's event sink claims a cascade slot and queues a workflow start in that same transaction. The relay (`nava-relay`) hands the queued start to the workflow engine (Temporal), and the production worker (`nava-worker`) runs the interpreter. When the interpreter updates the record, it goes through the same write pipeline as a person, with the same checks, and the history records `source=workflow`.

- Definition versions never change once published. A run that is waiting when a new version is published finishes on the version it started with; new runs use the new one.
- A step that names a field the record type does not have is refused at publish time with `422`.
- One tenant's writes never start or signal another tenant's workflow.

Details: [workflow interpreter](../workflows/nava-interpreter.md), [outbox, relay and worker](../workflows/outbox-relay-worker.md), [S11b](../stories/s11b.md).

## Adding or removing a step type

The step catalog today is `condition`, `branch`, `timer`, `wait_signal`, `update_record` and `end`, with condition operators `eq`, `ne`, `in` and triggers `record.created`, `record.updated`. A tenant's published definition keeps working after an upgrade within a major version: the catalog is additive-only (ADR-008 D12), and the upgrade-compatibility suite (M-X2, S27) checks that on every PR.

- **Adding** a step type, step field, operator or trigger: change `internal/workflow/def` (its `schema.json` and `Parse`), then list the new item in `internal/configsurface/testdata/inventory.json` and add a fixture `internal/configsurface/testdata/fixtures/workflow/<name>.json` that uses it. Without them `task test` fails with `unlisted: …` or `no fixture: …`.
- **Deprecating** one: keep it in the code and in the inventory, and add `{"item": "workflow.step_types/<name>", "since": "<version>"}` to the inventory's `deprecated` list. It must stay listed and its fixture must keep parsing until the next major.
- **Removing** one: only with a new major. Raise `"major"` in the inventory and write `docs/upgrade/v<major>.md` naming the item (for example `workflow.step_types/timer`) with the migration. Otherwise `task test` fails with `removed: …` and `task lint` with `compat: removed: …`. No `docs/upgrade/` note exists yet: nothing has been removed.

What each failure line means: [Troubleshooting](../operations/troubleshooting.md#a-lint-or-test-check-fails).

## Not built yet

- A workflow builder screen.
- Disabling or deleting a definition. The kit retires its demo by publishing a version that waits a day and writes nothing.
- An API to send a signal (such as `approve`) to a running workflow, so approvals are not a product feature yet.
- Conditions on state changes.
