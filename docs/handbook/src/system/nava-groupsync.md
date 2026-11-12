---
title: "nava-groupsync"
section: system
status: delivered
sources: [compose.yaml, cmd/nava-groupsync/main.go, db/grants.yaml]
adrs: [ADR-001, ADR-002]
packages: [cmd/nava-groupsync, internal/groupsync]
invariants: [I3]
tests: [TestE5GroupsyncCannotRekeyGroups, TestTwoSyncersNeverSyncOneTenantConcurrently, TestGroupsyncAccountFloor]
---

Mirrors each tenant's users, groups, memberships and org units under `/t/<tenant_id>/` from Keycloak into the identity tables, every `NAVA_GROUPSYNC_INTERVAL` (default 5m, at most 5m). A principal resolves (`GET /api/v1/me`) only after a cycle has synced it. `--once` runs one cycle and exits.

| | |
|---|---|
| Network | `internal` |
| Listeners | `:9090` admin only |
| Host port | none (tunnel 19092) |
| DB role | `nava_groupsync` through PgBouncer |
| Talks to | keycloak (Admin API read, client `nava-groupsync`); pgbouncer; otel-collector |
| Introduced by | PLAN-001 T21b (TICK-234) |

## Boundary it enforces

**I3.** `nava_groupsync` may write only the identity tables, and cannot re-key a group: `TestE5GroupsyncCannotRekeyGroups`. Its Keycloak account holds only the roles the sync needs: `TestGroupsyncAccountFloor`.

[Back to the system map](index.md)
