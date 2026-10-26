# CLAUDE.md

## Platform goal

For the full platform goal and architecture decisions, read
[`docs/adr/ADR-001-platform-blueprint.md`](docs/adr/ADR-001-platform-blueprint.md)
when the task needs that context, for example a new component or a design
choice.

## Design principles

Follow the design rules in [`README.md`](README.md#design-principles) for any
code change: apply SOLID, and reach every third-party service (identity
provider, workflow engine, broker, storage, email and so on) only through a
core-owned port and a single adapter package, as `internal/identity` and
`internal/identity/keycloak` do. Use other patterns only where the problem
needs them.

## Known gaps

Deferred work is tracked as a table in
[`docs/delivery/todo.md`](docs/delivery/todo.md). Each row has a matching
`TODO(TODO-NNN)` comment in the code. When you add a TODO, add both.

Do not read `docs/delivery/todo.md` up front; it is not needed for most tasks.
As the **final step** of any task that changed code, check lazily:

1. Run `grep -rn "TODO(TODO-" --include=*.go .` and see whether any listed
   site is in code you touched.
2. Only if your change resolved one, grep the tracker for that ID's row
   (`grep -n "TODO-NNN" docs/delivery/todo.md`), set its status to `Fixed`
   with the date and PR or commit, and remove the `TODO(TODO-NNN)` comment
   from the code.
3. Otherwise leave the tracker alone.

## Accepted risks

Every knowingly accepted risk is indexed, with a severity flag and a link to
its source, in
[`docs/security/accepted-risks.md`](docs/security/accepted-risks.md). Read it
when a task touches tenancy, Temporal authorization, telemetry or the
database roles, or when asked what to fix next.

- When a decision, ADR, review or doc accepts a new risk, add an `AR-NNN` row
  in the same change.
- When a change fixes one, delete its row and note the fix at the source.
- If the risk becomes deferred code work, move it to `docs/delivery/todo.md`
  instead.

## Commands

Use the reusable task commands from `Taskfile.yml` (go-task v3) instead of
typing raw `go` / `golangci-lint` commands.

| Command           | What it does                                                  |
| ----------------- | ------------------------------------------------------------- |
| `task`            | List the available tasks                                      |
| `task build`      | Build the nava-core binary into `bin/`                        |
| `task run`        | Run nava-core locally, loading `.env` (copy `.env.example`)   |
| `task test`       | Run all tests with the race detector (needs cgo)              |
| `task test:short` | Run all tests without the race detector, for hosts with no cgo |
| `task test:integration` | Run the database integration tests against a throwaway Postgres container (Docker) |
| `task lint`       | Run golangci-lint (config in `.golangci.yml`)                 |

Before calling a change done, run `task test` and `task lint`. `make ci` runs
the whole CI pipeline locally (see `docs/quality/ci-pipeline.md`).