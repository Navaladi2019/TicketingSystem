## CI-10

Coverage: partial → P3-42 (S11b): (g) signal ping-pong needs the interpreter; (f) the start key is per tenant, here; (a)–(e) are TestCI10_* in internal/records/pipeline and the outbox depth limit TestCI10_* in internal/outbox, both against real PostgreSQL in make test-integration.

### Registered paths

| Path | Role | Status | Detail |
|---|---|---|---|
| cascade_start: look a shared origin up in every tenant | nava_app | PASS | 1 rows, all tenant A's |
| cascade_start: read tenant B's claim by its record and definition | nava_worker | PASS | 0 rows, all tenant A's |
| cascade_start: claim a key for tenant B | nava_worker | PASS | denied: SQLSTATE 42501 |
| cascade_start: claim a key on tenant B's record from tenant A | nava_app | PASS | denied: SQLSTATE 23503 |
| cascade_start: rewrite tenant B's claims | nava_worker | PASS | denied: SQLSTATE 42501 |
| cascade_start: delete tenant B's claims | nava_app | PASS | denied: SQLSTATE 42501 |
| cascade_start: the same origin, record and definition in two tenants never collide | nava_worker | PASS |  |

