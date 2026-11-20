---
title: "Id generation"
section: system
status: in-progress
sources: [tools/navalint/uuidv7/uuidv7.go, tools/navalint/main.go, internal/entitytypes/entitytypes.go, internal/groupsync/apply.go, internal/workflowdefs/workflowdefs.go, internal/workflow/def/ids.go, internal/workerguard/child.go, docs/adr/ADR-001-platform-blueprint.md, docs/delivery/decisions/2026-11-15-tick-961-go-1-27-upgrade.md]
adrs: [ADR-001, ADR-004]
stories: [GO127]
packages: [tools/navalint/uuidv7]
tests: [TestReportsV4, TestReportsCmd, TestExemptPaths, TestIsolationSiblingIsReported, TestSyncMintsV7Ids, TestEntityTypeCreateAndPublishMintV7Ids, TestWorkflowDefinitionPublishMintsV7Ids]
---

Every id the platform stores or uses as a durable identifier is a **UUIDv7**, minted in Go with `uuid.NewV7()` from `github.com/google/uuid` (ADR-001: `id uuid` (UUIDv7); ADR-004 R1). Status: on the story branch `TICK-961/go-1-27-upgrade` until the Go 1.27 story merges to `main`.

## Why v7, and why in Go

- **Time-ordered.** The first 48 bits are a millisecond timestamp, so new rows land at the right-hand edge of a B-tree index instead of at random pages. Ids sort roughly by creation time.
- **Minted in Go, not in PostgreSQL.** PostgreSQL 16 has no `uuidv7()`, and no migration uses `DEFAULT gen_random_uuid()`. The service creates the id before the insert, so it can be put in the outbox row, the audit entry and the response in the same transaction. No database upgrade is needed.
- **`NewV7` can fail** (the random source). Every caller handles the error; none ignores it.

## Which ids

| Id | Where it is minted |
|---|---|
| Records, audit entries, outbox rows, tenants and the rest of the write pipeline | already v7 before this story |
| Entity type and entity type version | `internal/entitytypes/entitytypes.go` (was v4) |
| Synced users, groups and memberships | `internal/groupsync/apply.go` (was v4) |
| Workflow definition and version | `internal/workflowdefs/workflowdefs.go` (was v4) |
| Workflow step ids stored in the immutable version row | `internal/workflow/def/ids.go` (was v4) |
| Child workflow id | `internal/workerguard/child.go`, inside `workflow.SideEffect`, so replay reads the recorded value (was v4) |

Deterministic **v5** ids (`uuid.NewSHA1`) are allowed: the same input must give the same id. Ids already stored as v4 stay as they are; nothing is rewritten.

## The guard

`task lint` runs the `uuidv7` analyzer (`tools/navalint/uuidv7`). It reports a call to `uuid.New`, `uuid.NewString`, `uuid.NewRandom` or `uuid.NewRandomFromReader` in non-test code under `internal/` or `cmd/`:

```
internal/x/x.go:12:9: uuid.New mints a random v4 id: persisted ids are UUIDv7, use uuid.NewV7 and handle the error (TICK-963)
```

Fix: call `uuid.NewV7()` and return its error. Exempt: `_test.go` files, any `contracttest` package, and `internal/tenancy/isolation` (test helpers in non-test files). A sibling such as `internal/tenancy/isolationx` is not exempt (`TestIsolationSiblingIsReported`). Integration tests check that real writes produce version-7 ids (`TestSyncMintsV7Ids`, `TestEntityTypeCreateAndPublishMintV7Ids`, `TestWorkflowDefinitionPublishMintsV7Ids`).

## Not yet: the stdlib `uuid` package

Go 1.27 adds a standard-library `uuid` package. The move from `github.com/google/uuid` to it (340 files) is its own deferred story, TICK-966, because it would touch files every open story edits and adds no invariant. It restarts when no open story branch changes Go files under `internal/` or `cmd/`, or when `google/uuid` is deprecated. When it lands, the analyzer's package path and the depguard rule change in the same PR.
