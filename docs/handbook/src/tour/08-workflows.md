---
title: "8. Workflows"
section: tour
status: delivered
sources: [docs/acceptance/s11a-workflow-interpreter.md, docs/architecture/workflow-interpreter.md, cmd/nava-worker/interp_default.go, cmd/nava-worker/interp.go]
adrs: [ADR-001, ADR-003]
tables: [outbox]
packages: [internal/workflow/interp, cmd/nava-worker]
---

## In plain words

A workflow is a set of automatic steps attached to a record: wait two days, send it to a manager if the amount is high, wait for an approval, then mark it approved. **This is not available to customers yet.** The engine that runs such steps is built and tested, but it is switched off in the running product, and creating a record does not start one. Today a record changes state only when a person changes it (step 9).

## What you do

Nothing to do yet. There is no API to define a workflow or start one.

## Try it

There is no request. To see what a record write does today, check the work queue after step 9:

```sh
docker compose exec -T postgres sh -c 'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -qAt -c \
 "select kind, count(*) from outbox where tenant_id='"'"'00000000-0000-4000-8000-0000000000a1'"'"' group by 1"'
```

## What you see

```
audit.seal|5
```

The only background work on `main` is sealing the audit log (step 10). Creating and changing loan records added no workflow work.

## What happens behind it

Each record write also writes a row to a work queue (the "outbox") in the same database transaction, and a relay hands queued rows to the workflow engine (Temporal). Two things stop a record from starting a workflow today:

- The producer that turns a record write into a workflow start (story S11b) is not on `main`. The write pipeline's sink is a no-op.
- The worker (`nava-worker`) registers the interpreter only in a test build (`interptest`). The production build registers the audit sealer only, because the stores the interpreter needs (definitions, record reader and writer) do not exist yet.

The interpreter itself runs a definition with timers, branches on a record's values, waits for a signal such as `approve`, and updates the record. In the test build it was shown to keep in-flight runs on the version they started with and to apply an update once after a worker crash (acceptance kit S11a). Details: [workflow interpreter](../workflows/nava-interpreter.md), [outbox, relay and worker](../workflows/outbox-relay-worker.md).

## Not built yet

- Starting a workflow from a record write (S11b).
- Storing and publishing workflow definitions through the API, and the interpreter in the production worker.
- A workflow builder screen.
- Approvals as a product feature, and conditions on state changes.
