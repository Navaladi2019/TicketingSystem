---
title: "Workflows"
section: workflows
status: delivered
sources: [internal/outbox/outbox.go:244, cmd/nava-worker/main.go:200-203]
---

Durable work runs on Temporal, and it is entered only through the transactional outbox (ADR-001 I6). There are exactly two workflow kinds on `main`:

| Kind | Build | Runs in the stack | Page |
|---|---|---|---|
| `audit.seal` | production | yes | [Audit sealer](audit-seal.md) |
| `nava.interpreter` | `interptest` only | no | [Workflow interpreter](nava-interpreter.md) |

Start with [the durable path](outbox-relay-worker.md): how a row becomes a workflow start or signal, the relay's outcomes, the idempotency keys and the `workerguard` interceptor every workflow runs behind.

Onboarding, group sync and purge are batch jobs, not workflows: they import no Temporal package. Onboarding appears on the durable path page only as the producer of a tenant's first `audit.seal` row.

## How the names are checked

The build parses the names from the Go source (`GoRegistry` in `docs/handbook/build/build.py`):

- `workflows:` must be a kind in the closed list at `internal/outbox/outbox.go:244`, and every kind must be registered by a `RegisterWorkflow` function.
- `activities:` must be a name registered by `(*seal.Activities).Register`, `(*interp.Activities).Register` or `(*workerguard.Activities).Register`.
- Every `.Register(w)` and `RegisterWorkflow(w)` call in `cmd/nava-worker` must go through one of those five functions, so a registration added in the worker cannot bypass the parsed names.
- A page in this section that names workflows also says which build registers them (`build: production` or `build: interptest`). The build cannot see that, because `package main` decides it; a reviewer checks it against `cmd/nava-worker`.

The build fails if the parser finds no kind or no registration, so it cannot pass silently. Timeouts and retry policies are quoted by hand, with the `file:line` of each in the page's `sources:`; the build checks that each file has those lines, and the reviewer checks what they say against the footer sha.
