# CI optimizations

Why local CI is slow, what we decided, and what is still open. Target: **a CI
run takes no more than 5 minutes** (human, 2026-10-24).

GitHub Actions is off (billing), so local Docker CI is the merge gate for every
PR. The host is shared by every Multica agent: 16 CPUs, about 20 GB for Docker.
For how the pipeline works, see [ci-pipeline.md](ci-pipeline.md).

## Baseline (2026-10-24)

A warm `make ci-isolated` run (TICK-147, same SHA, nothing changed) took
**1609 s, about 27 minutes**. A cold run took 2032 s.

| Stage | Warm time | Main cause |
| --- | --- | --- |
| Stack | 897 s | The full stack starts **3 times** (main tests, a restart for `dependencies`/`dbroles`, a fresh stack for `internal`). The 23 Temporal Go tests (`sdk`) run one at a time: 133 s |
| Integration | 339 s | `dbtest` 224 s: `freshCluster` drops every database and role and re-runs db-roles, usually all migrations too, for nearly every test, on a durable Postgres, with `-p 1` |
| Lint | 213 s | Includes two govulncheck runs (network-bound) |
| Unit | 152 s | Mostly cached; race detector |

The stages run one after another, so the total is the sum.

Under load the same stages took 2 to 5 times longer (lint 522 to 1066 s, unit
up to 1278 s), and one run hit a host out-of-memory error in `go test -race`.

A separate run of `scripts/ci/test-integration.sh` on `main` took 385 s:
dbtest 220 s, health 58 s, tenancy 19 s, pooltest 17 s.

### Caches

On a warm run, Docker layers, registry images, Go modules and the toolchain are
all reused. The one real miss is a **new checkout path**: without `-trimpath`,
Go's build cache key includes the package directory, so every new Multica
workspace recompiles the repo's own packages and reruns the unit tests.

## Decisions

| # | Decision | Status |
| --- | --- | --- |
| D1 | **Host-side stack tests run inside the compose network.** The Temporal (`internal/temporalstack`) and Keycloak (`internal/identitystack`) Go tests run in a container on the project network, so CI publishes no host ports and parallel stacks can't collide on ports. Fixed default ports stay for development. Exception: the `health` integration test keeps a fixed port, because it restarts its container and Docker Desktop on Windows can't reach container IPs. | Deferred by the Technical Lead (TICK-153 Phase 2), no owner now; see Open decisions. The Go suites already take their addresses from the environment (`NAVA_TEST_KEYCLOAK_URL`, `NAVA_TEST_TEMPORAL_ADDR`, `NAVA_TEST_CERT_DIR`), but the CI-6 tests start probe containers with `docker run -v <absolute certificate path>`, which the Docker daemon resolves on the host, so from inside a container they would mount nothing. Moving them means rewriting how the isolation tests reach the certificates, which changes a security test, not just CI. Port collisions, the reason for D1, are already prevented by `ci-isolated`'s per-run port slots (TICK-147), and D1 gives no speed |
| D2 | **govulncheck once per change.** Each module (root and `temporal-server/`, which have separate `go.mod` files) is scanned only when its `go.sum` changes, plus one daily run of both for new advisories (ADR-001-A1's R7 trigger). The two scans run in parallel. | Done (TICK-153 PR 3): `make vulncheck-ci`. A pass is stamped per module, `go.mod`/`go.sum` checksum, Go and govulncheck version and UTC day, so the first run each day is the daily scan. Both modules, first run of the day: 154 s (was two serial scans in lint's 260 s); stamped: 1 s |
| D3 | **Race-detector tests: one at a time on the host.** A host-wide lock around `go test -race` for every checkout and agent. Ship it with O5 (`-trimpath`) and O7 (`ci-changed`) so the queue stays short. | Done (TICK-153 PR 3): `tools/ci/racelock`, a listener on `127.0.0.1:47613` around both `go test -race` lines of `make test` and `task test`. Checked: a second holder waited 35 s and reported while waiting |
| D4 | **Stack slot limit, queued.** At most N compose stacks at once; extra runs wait. Measure N from one stack's real memory (`docker stats`). | Done (TICK-153 Phase 2): N = 7. One stack holds 1.34 GB while its tests run (two stacks up); (19.4 GiB - 4 GB for other containers) / (1.34 GB x 1.5 headroom) = 7.6. `tools/ci/racelock -slots 7` around `ci-stack`, one slot per run |
| D5 | **Verification cost rules for agents** (see `CLAUDE.md`, "Verification cost"): name the cheapest check for each acceptance criterion, with a full pipeline run once as the final gate; confirm `main` is green before starting; never re-run a full pipeline to re-prove something. | Written in `CLAUDE.md`, not yet merged. The Delivery squad pointer needs adding in Multica |

Isolation of networks, containers and volumes between compose projects needs no
proof; Docker guarantees it. What can collide across projects is host-level:
published ports, image tags, hard-coded project names, and host memory. Check
those statically (`docker compose config` diff and a grep), plus at most one
short parallel `STAGES=stack` smoke run.

## Optimizations

| # | Optimization | Stage | Expected gain | Status |
| --- | --- | --- | --- | --- |
| O1 | Test Postgres with `fsync=off`, `synchronous_commit=off`, `full_page_writes=off` in `scripts/ci/test-integration.sh` | Integration | Large cut to dbtest's 224 s | Done (TICK-153 PR 1). dbtest 117.9 s to 86.3 s on `main` + O1, host shared with another CI run |
| O2 | Faster `freshCluster`: a pre-migrated template database (`CREATE DATABASE … TEMPLATE`), keeping a full rebuild only for tests of role or migration behaviour from scratch. Needs care: roles are cluster-wide and some tests corrupt them on purpose | Integration | Most of the remaining dbtest time | Done (TICK-153 PR 2): `migratedCluster` copies the databases while the roles are unchanged, else rebuilds; `TestMigratedClusterMatchesFreshRun` proves a copy matches a real run. The role, D28 and from-scratch migrate tests keep `freshCluster` |
| O3 | Shard `dbtest` across 2 to 3 Postgres containers instead of `-p 1` | Integration | Parallel packages | Done (TICK-153 PR 2): 3 shard clusters; pooltest and tenancy on the shared cluster, health beside them. Integration 173 s to 86 s with O1 to O3 (dbtest shards 23 to 33 s), 49 of 49 dbtest tests run |
| O4 | Run the four stages in parallel inside `ci-isolated` | All | Total becomes the slowest stage, not the sum | Done (TICK-153 PR 4): all four stages at once, `PARALLEL=0` for the old order. Full run 1646 s to 1220 s; the stack is now the whole wall clock and runs slower beside the others (967 s alone, 1214 s in parallel) |
| O5 | `-trimpath` for the CI Go builds | Lint, unit | No cold compile in a new workspace | Done (TICK-153 PR 1), narrowed: `-trimpath` on the root module's `go test -race` only. New checkout: 94 s to 61 s. On every CI target it made `go vet` and navalint's analysistest runs slower (navalint 115 s to 241 s in a new checkout), so it is not in `GOFLAGS` |
| O6 | govulncheck per D2 | Lint | Most of the two scans | Done with D2 |
| O7 | `make ci-changed`: pick the stages from `git diff origin/main`, so most PRs skip integration and stack | All | Whole stages for most PRs | Done (TICK-153 PR 4): `tools/ci/cichanged` maps each changed file to stages through `go list -deps`; an unknown file runs all four. A code-only change in `internal/tenancy` runs lint, unit and integration: 186 s warm, 456 s in a new checkout |
| O8 | Stack stage: run `stack-internal` as its own project **at the same time** as the main stack | Stack | One full start-up off the wall clock; uses 2 slots | Measured both (TICK-153 Phase 2): N = 7 >= 4, so the rule says beside, but beside is slower on this host: two Keycloaks start at once and the main stack takes 233 s to healthy instead of about 120 s. Stack stage 698 s beside, 669 s after. Default `STACK_INTERNAL=after`, confirmed by the Technical Lead; `beside` stays a switch |
| O9 | `t.Parallel()` for the 23 `temporalstack` tests (confirm each uses its own workflow ids and tenants first) | Stack | 133 s → about 30 s | Partly done (TICK-153 Phase 2). Not `t.Parallel()` across the suite: every CI-6 test polls the one platform task queue (`nava-tq`) and leaves other workflows' tasks to time out, and some deny-log checks match a log window, so parallel tests would steal each other's tasks and could match another test's line. Instead the internode handshake test runs its 18 probe containers at once: 99 s to 4 s, the suite 133 s to 67-100 s (it now runs beside the other checks) |
| O10 | Run the read-only stack checks together (healthy, smoke, pgbouncer, temporal_*); `dependencies`, `dbroles` and `idempotent` stay last and sequential | Stack | 1 to 2 min | Done (TICK-153 Phase 2): `healthy`, then smoke, PgBouncer, every `temporal_*` case, hba and the two Go suites at once, then dependencies, trace, logs, redact, idempotent and dbroles in order. The same 65 checks and 24 Go stack tests pass |
| O11 | `fsync=off` for the stack Postgres, through a CI-only compose overlay | Stack | Faster db-roles and migrate | Done (TICK-153 Phase 2): `deploy/compose/compose.ci.yaml` sets `fsync`, `synchronous_commit` and `full_page_writes` off and `initdb --no-sync`; only `ci-stack` adds it, and that run ends in `down -v`. `compose.yaml`, the test overlay `make stack-up` uses on the dev project, and the internal overlay are unchanged |

O8 to O11 touch the same files as TICK-147 (compose, Makefile, stack-test
scripts), so they wait for it to merge. O1 to O3 do not.

### Measured result (Phase 1, 2026-10-24)

All on this host, shared with other agents' runs; seconds.

| Run | SHA | Lint | Unit | Integration | Stack | Total |
| --- | --- | --- | --- | --- | --- | --- |
| Baseline `make ci-isolated`, stages in order, new checkout | `3cae4c7` (`main`) | 260 | 239 | 173 | 967 | **1646** |
| `make ci-isolated`, stages at once | `dac8a32` | 154 (failed: lint findings in the new tools, fixed in `32a90e7`) | 761 | 256 | 1214 | **1220** |
| `STAGES="ci-lint ci-unit"`, the gate for the lint fix | `32a90e7` | 320 (one govulncheck rescan, new stamp key) | 290 | | | 325 |
| `make ci-changed`, code-only change in `internal/tenancy`, new checkout | `32a90e7` + 1 line | 274 | 446 | 222 | skipped | **456** |
| The same, warm (second run in that checkout) | `32a90e7` + 1 line | 78 | 177 | 131 | skipped | **186** |

Integration ran 70 top-level tests against the baseline's 69 (the new
`TestMigratedClusterMatchesFreshRun`), with the same one skip, and every
cross-tenant test passed. The stack ran the same 65 checks.

**Target.** A typical code-only PR (`make ci-changed`, warm) is at 3.1 minutes,
inside 5. A full run, or a PR that needs the stack, is not: the stack stage is
the whole wall clock (about 16 to 20 minutes), which is Phase 2 (D1, D4, O8 to
O11). The remaining floor after the stack:

- **Unit, 177 s warm.** Most of it is `tools/navalint`'s tests: Go re-hashes
  every file the analysistest runs opened before it can reuse a cached result
  (about 60 s per package here, cached or not). In a new checkout they rerun
  (about 115 s), and `-trimpath` makes them slower, so O5 leaves them alone.
- **Lint in a new checkout, 274 s,** with govulncheck skipped: golangci-lint
  and `go vet` rebuild their caches for the new path.

### Measured result (Phase 2, 2026-10-24)

The stack stage alone (`STAGES=ci-stack PARALLEL=0`), seconds:

| Step | Before (`72da3a7`) | After, internal beside | After, internal after (default) |
| --- | --- | --- | --- |
| Build, `compose up` | 47 | 39 | 35 |
| Start-up until `healthy` | 112 | 233 (two Keycloaks at once) | 140 |
| Read-only checks and Go suites | 218, one at a time (Temporal suite 133) | 192 at once (Temporal suite 67) | 119 at once (Temporal suite 100) |
| dependencies, trace, logs, redact | 66 | 66 | 56 |
| idempotent (down, up, healthy again) | 136 | 136 | 125 |
| dbroles, hba, down | 39 | 24 | 23 |
| Internal overlay | 157, after | beside | 173, after |
| **Stack stage** | **774** | **698** | **669** |

The same 65 stack checks and 24 Go stack tests pass in each.

Full `make ci-isolated` at the Phase 2 head (`b32827c`), all four stages at
once, from a new checkout: **PASS in 991 s** (lint 504, unit 983, integration
337, stack 976), against 1646 s at the start. Integration ran 70 top-level
tests (49 of 49 dbtest), the stack its 65 checks.

**Target, after Phase 2.** A code-only PR (`make ci-changed`, warm) stays at
about 3 minutes. A full run is not near 5 minutes, and the stack stage cannot
get there on this host while it starts Keycloak three times: the first
start-up, `idempotent`'s second `up` on the used volume, and the internal
overlay's fresh stack, each 100 to 140 s. The floor without dropping any of
them is about 8 to 9 minutes for the stack alone. On a new checkout, ci-unit
(navalint's analysistest runs, about 4 minutes cold) runs as long as the stack
when the four stages share the host. What would still cut it, none of them in
TICK-153's scope: a pre-augmented Keycloak image (`kc.sh build` at image build
time, so no start re-augments, about 40 s each start), and `ci-changed` as the
gate, which the Technical Lead ruled out for now.

## Open decisions

None left from TICK-153. Decided by the Technical Lead, 2026-10-24:

- **O8:** the internal overlay runs after the main stack by default. The
  "beside if N >= 4" rule assumed memory was the limit; measured, the limit is
  two Keycloaks starting at once (698 s beside, 669 s after).
  `STACK_INTERNAL=beside` stays as a switch.
- **D1:** deferred, no owner now: it saves no time, the per-run port slots
  (TICK-147) already prevent the collisions it targets, and moving the CI-6
  certificate mounts would put an isolation test at risk. Revisit when CI moves
  to a shared or remote runner where published host ports can collide, or a
  test must run inside a container; QA owns it then, with the Security &
  Multi-Tenancy Reviewer.
- **D5:** a full `ci-isolated` stays each PR's final gate; `ci-changed` is the
  quick check during development. The remaining full-run time is TICK-154.

## Lessons from TICK-147

TICK-147 (parallel compose stacks) took more than 4 hours, mostly verification:
8 to 10 full pipeline runs of about 30 minutes each, run one after another, on
a loaded host. Two failures already on `main` (TICK-148, TICK-149) were found
only partway through. Written well, with the cheapest check for each criterion
and `main` checked first, the same ticket takes about 2 to 2.5 hours, and
about 1 to 1.5 hours once O1 to O11 land. D5 records the rules that come from
this.
