# CI pipeline (T05)

`make ci` runs every gate locally, for a human debugging a failure. **Agents
verify in GitHub, not locally:** push the branch or open a PR and read the `ci`
workflow, or run `gh workflow run ci --ref <branch>` for a full run. The GitHub workflow `.github/workflows/ci.yml`
calls only `make` targets. Install the pinned tools once with `make tools`.

## Gates, in order

| Stage | Make target | What fails it | Containers |
| --- | --- | --- | --- |
| Lint | `ci-lint` | tool versions differ from `docs/pins.md`, `gofmt`, `go vet`, `vet-tags` (`go vet` once per build-tag set the `//go:build` lines use), golangci-lint, `lint-fixture` (exactly 5 forbidigo findings), `check-taskfile`, `check-pins`, `check-generate` (`api/api.gen.go` differs from what `go generate ./api/...` writes), `check-guide` (`docs/acceptance/guide/`, among it `bruno/lib/nava.js` `REGISTERED`, differs from what `build/build.py` writes; TICK-391), `check-ci-steps` (`ci.yml` lost a step listed in `tools/ci/cisteps/required.yaml`), `check-openapi`, `govulncheck ./...` in both modules at once (`vulncheck-ci`, below) | none |
| Unit | `ci-unit` | `go test -race ./...` under the host-wide race lock (below), temporal-server unit tests | none |
| Integration | `ci-integration` | any `//go:build integration` test (`internal/db/dbtest`, `internal/db/pooltest`, `internal/tenancy`, `internal/platform/health`) against throwaway PostgreSQL clusters and two PgBouncers (tags from `docs/pins.md`), or the dbtest shards running fewer tests than the package lists | Docker |
| Isolation (in `ci-integration`, after `test-integration`) | `isolation` | the tenant isolation suite (`internal/tenancy/isolation`, T11, ADR-001 CI-2): any role × tenant table, registered path or fixture failing; writes the report `isolation-report.md` at the repository root (uploaded as the `isolation-report` artifact, even when the job fails) | Docker |
| ci-isolated `--down` (in `ci-integration`, after `isolation`; its own step in the `integration` job) | `test-ci-isolated-down` | `make ci-isolated-down ID=<ID>` leaves a `nava/*:<ID>` tag or removes one of another ID (TICK-211, throwaway tags on an empty imported image). Not part of `make test-integration`, so the local integration loop does not pay for it (TICK-224) | Docker |
| Stack | `ci-stack` | image build, every container healthy, `GET /healthz` smoke from inside the compose network, then the stack tests (`pgbouncer`, `temporal`, `dependencies`, `trace`, `redact`, `idempotent`, `dbroles`, `hba`, `internal`) | compose |

In GitHub, the `temporal-server` job runs the module's tests and then
`make vulncheck-temporal-server`, which must report zero findings: there is no
exception and no allowlist (ADR-001 A1 Decision 3, TICK-121).
`lint`, `unit`, `temporal-server`, `integration`, `stack`,
`stack-db` and `stack-internal` run as seven parallel jobs (the ones a pull
request needs, see "In GitHub" below), each a separate
check (decision log, 2026-10-20). Inside each job, steps follow the order above
and the first failure stops the job. The stack tests are split over three jobs,
each with its own compose stack: `stack` (healthy, smoke, PgBouncer, every
Temporal case, SDK, dependencies, trace, redact), `stack-db` (idempotent,
dbroles, hba) and `stack-internal` (the internal overlay).

`make ci-stack` runs all of them locally (TICK-153): the main stack, then the
internal overlay as a project of its own (`<project>-int`, host ports 0 so
Docker picks free ones). `STACK_INTERNAL=beside` runs the two at once; on this
host that is slower (698 s against 669 s), since two Keycloaks then start
together. On the main stack, `healthy` gates
the rest; the read-only checks and the two Go suites (`mode=parallel` in the case header:
smoke, PgBouncer, every `temporal_*` case, hba, SDK, identity) then run at
once, and `worker`, `dependencies`, `trace`, `logs`, `redact`, `idempotent` and `dbroles`
run one at a time after them (`mode=serial`): `worker` polls `nava-tq`
like the SDK suite's workers, `dependencies` stops
Keycloak, `trace` fires its own requests and waits on the Collector's sampler, `idempotent` restarts
the stack and `dbroles` changes roles. `ci-stack` alone adds the CI overlay
(`deploy/compose/compose.ci.yaml`), which runs the stack's PostgreSQL with
`fsync`, `synchronous_commit` and `full_page_writes` off; that run ends in
`down -v`. Since TICK-166 `compose.yaml` sets the same three flags for the dev
stack too (Keycloak's first boot is fsync-bound), so the overlay is a duplicate
there; production files never set them.

**Time budget: under 5 minutes wall clock on a warm cache.** What keeps it there:
- Each job restores its own Go module and build cache (`actions/cache`, key per
  job and commit, falling back to the newest entry), so `go vet`, `go test` and
  the temporal-server tests compile only what changed. The lint job also caches
  the golangci-lint cache and the pinned tools (`~/go/bin`, keyed on the
  Makefile, which holds the versions).
- `make image-ci` builds the images with `docker buildx bake` and keeps the
  layers in the Actions cache (`IMAGE_CACHE=gha`), while the registry images are
  pulled alongside. The temporal-server image compiles the Temporal dependencies
  in a layer keyed on `go.mod`/`go.sum`, so a source change rebuilds one layer.
  The image no longer runs `go test`; the `temporal-server` job runs the full
  suite.
A change to `go.sum` or a Dockerfile base stage is a cold run (about 8 to 10
minutes); the next run is warm again.

**Job time budgets (TICK-258, re-measured in TICK-400).** The 5-minute figure
above was the target. The stack jobs do not meet it, because `make image-ci`
takes 4 to 13 minutes of every run. `timeout-minutes` is a hang guard, not a
speed target. It must sit above the slowest run in which every step passes, or
a green run is cancelled and blocks the merge. This happened twice:

- PR #165, runs 40443438351 and 40448981983: both cancelled at 931 s under the
  old 15-minute bound (TICK-258).
- PR #344, push run 41278740124: `stack` was cancelled at the 25-minute bound
  on both attempts. On attempt 2 every case had passed; the cancel came 1 s
  over the bound, during report upload and teardown (TICK-400).

Measured on 2026-11-01 (TICK-400) over the last 10 green push runs of `main`
and the 7 green push runs of story branch `TICK-334/workflow-interpreter`
(all it had). Durations are job wall clock, set-up and post steps included.

| Job | `main`, min / max | Story branch, min / max | Image build, min / max | Cases, min / max | `timeout-minutes` |
|---|---|---|---|---|---|
| `stack` | 835 s / 1,399 s | 947 s / 1,474 s | 299 s / 678 s (758 s in cancelled run 41278740124) | 480 s / 719 s | 35 (was 25) |
| `stack-db` | 381 s / 793 s | 409 s / 789 s | 254 s / 652 s | 43 s / 55 s | 25 |
| `stack-internal` | 432 s / 795 s | 356 s / 742 s | 256 s / 684 s | 34 s / 47 s | 25 |
| `lint`, `unit`, `temporal-server`, `integration` | under 5 min | | | | 10 |
| `changes` | under 1 min | | | | 5 |

Run ids: on `main`, 41261072764, 41257268224, 41254956177, 41248729699,
41225258899, 40832394612, 40826427934, 40824849251, 40823401408 and
40821579602. On the story branch, 41277164363, 41274001507, 41269677744,
41267094153, 41262361451, 41254059522 and 41212397695. Slowest: `stack`
1,474 s in 41277164363, `stack-db` 793 s in 41254956177, `stack-internal`
795 s in 41225258899. The cases column counts only runs with a single
`Stack cases` step (TICK-296), so it excludes the older `main` runs that
listed each case as its own step.

Why the image build doubled: the Actions cache was full. GitHub keeps 10 GB
per repository and evicts least recently used first. On 2026-11-01 the
repository held 17.9 GB. Of the image-layer blobs, 6.25 GB belonged to task
branches and 2.99 GB to pull requests; only 0.19 GB belonged to `main`. A
cache entry can be read only by its own ref, by pull requests into that ref,
and by every ref when it is on the default branch. So each task-branch or
pull-request write evicted layers that other refs needed and that nothing
would read again. The per-sha Go build caches, about 0.3 GB per job per run
written from every ref, added to the churn. In the slowest run (41277164363),
even the `useradd` and `go mod download` layers rebuilt, and seven `go build`
steps ran 360 to 440 s each in parallel.

What TICK-400 changed:

- Only `main` writes the Actions cache. `CACHE_WRITE` is `true` on a push,
  manual run or schedule on `main`, and false everywhere else. `make image-ci`
  sets `cache-to` only when `CACHE_WRITE=true`, and the Go caches are split
  into `actions/cache/restore` and `actions/cache/save`, with the save step
  gated on `CACHE_WRITE`. Every other ref reads `main`'s entries, which stay
  warm. A story branch whose `go.sum` differs from `main` misses only the
  dependency layers, as it would have anyway.
- `stack` goes to 35 minutes. Its cold-cache worst case adds the worst steps
  seen: build 760 s, stack up 47 s, cases 720 s, set-up, reports, teardown
  and post steps about 70 s, total about 1,600 s (27 min). 35 minutes leaves
  about 30 % above that. A real hang still ends within 35 minutes.
- `stack-db` and `stack-internal` stay at 25. Their cold-cache worst case is
  about 900 s (15 min), so 25 leaves more than 60 % above it.

Revisit when:

- a green `stack` run exceeds 28 minutes: split the `stack` cases across two
  jobs, or move `make image-ci` into its own job that hands the images to the
  three stack jobs;
- the image build on a `main` run stays under 4 minutes: lower `stack` back
  towards 25;
- the `stack` cases alone pass 15 minutes. They grew from about 600 s to about
  720 s between the 2026-10-29 and 2026-11-01 runs.

Containers: `scripts/ci/test-integration.sh` owns, on a private network, one
shared PostgreSQL with the two PgBouncers of the load test (T06,
`docs/quality/pgbouncer-load.md`) in front of it, plus `DBTEST_SHARDS` (default
3) more clusters for dbtest, and removes them on exit, Ctrl-C or TERM. Every
container a Go test starts itself is removed in `t.Cleanup`.

Leftovers (TICK-185): the integration and isolation scripts never `exec` their
last command, since `exec` drops the EXIT trap; until TICK-185 every
`make test-integration ARGS=...` run left its cluster, PgBouncers and network
running, and 23 such networks on the shared host exhausted Docker's address
pools. They run each long `go` command in the background and wait for it, so
INT or TERM sent to the script alone reaches the trap at once and stops them.
`make test-integration-cleanup` (in `ci-integration` and its own `ci.yml`
step, as TICK-224) checks that an `ARGS` run and a TERM part way leave nothing
named for the run (`NAVA_IT_ID`, default the shell's pid plus a random suffix), that a run beside a TERM-stopped one still passes, and that a cluster removed under a run fails it at once (TICK-182). SIGKILL, or a host killing
the whole process tree, still skips the trap: `make reap-integration`
(`task reap:integration`, `tools/ci/itreap`) removes `nava-it-*` and
`nava-iso-*` containers and networks older than `REAP_OLDER` (default `6h`),
matching only the scripts' exact name forms and skipping anything with a
compose project label, so `nava`, `nava-<id>` and a run in progress are never
touched. `REAP_DRY=1` lists without removing. Image tags come
from `docs/pins.md`: the script exports the `postgres` row as
`NAVA_TEST_PG_IMAGE` and starts the `edoburu/pgbouncer` row. Every cluster runs
with `fsync`, `synchronous_commit` and `full_page_writes` off: they are thrown
away. The dev compose file sets them for its Postgres since TICK-166; the
production files never do (TICK-153 O1).

Three groups run at once (TICK-153 O3): dbtest's top-level tests, dealt
round-robin over the shard clusters, one `go test -run` each; the other
packages that use the shared cluster (`pooltest`, `tenancy`, `-p 1`, since they
create the same role); and the packages that use none (`health`, which starts
its own container). The script lists dbtest's tests first and fails unless the
shards ran every one of them. Extra arguments (`ARGS=-run ...`) run every
package one after another on the shared cluster, as before.

**Integration timeout and the log (TICK-225).** Every `go test` in the script
runs with `-timeout 6m`, against the job's `timeout-minutes: 10`. A green run
starts its tests about 2 minutes into the job (set-up 17 s, clusters and test
compile 85 to 100 s) and they take under a minute, so 6 minutes is several
times the need and still ends, dump printed, before GitHub cancels the step at
10. Without it, Go's default `-timeout` of 10m equalled the job limit, GitHub
cancelled first and the log was empty (PR #113, run 40276586007). A later
`-timeout` in `ARGS` overrides it. Each group's output streams to the log as
it arrives, every line prefixed with the group (`[dbtest-1]`, `[shared]`,
`[own]`, by `tools/ci/testlog`), so even a cancelled step shows it; at the end
suiteran repeats, per group, the package results and the output of every
failed test, then the `dbtest: N of M` check and the failed-group summary.

Reading a timeout dump: search the job log for `panic: test timed out`. The
lines after it, under the same `[group]` prefix, list the test or tests
`running` with how long each ran; that is the stuck test. Then the goroutine
dump: find the goroutine whose stack goes through that test's function, and
the frame above it (a `pgx` query, a channel receive, `time.Sleep`) is where it
waits. The package then fails as `FAIL <package> 6m0.0s`, and the group is
named in `integration: failed groups:`.

Inside dbtest, `migratedCluster` copies each manifest database from a template
taken after the first real db-roles and migrator run in the process
(`CREATE DATABASE ... TEMPLATE`, TICK-153 O2), and gives the copy its grants
again. It copies only while every role, membership and role setting is still
byte for byte what it was when the templates were taken, and falls back to the
full rebuild otherwise; `TestMigratedClusterMatchesFreshRun` proves a copy
matches a real run. Tests of db-roles or of a migrator run from scratch
(`roles_test.go`, `d28_test.go`, and the migrate tests that read the run's own
result) still call `freshCluster`.

**govulncheck once a day per dependency set (TICK-153 D2).** `make
vulncheck-ci` (`scripts/ci/vulncheck.sh`) scans the root module and
`temporal-server/` at the same time. A module is skipped when a scan of its
current `go.mod` and `go.sum`, with the same Go and govulncheck versions,
already passed on this host today (a stamp under `$(go env GOCACHE)/nava-vulncheck`).
So a dependency change is always scanned, and the first CI run of each day
scans both modules for new advisories (ADR-001-A1's R7 trigger). The trade-off,
accepted in D2: a code-only change that starts calling an already-listed
vulnerable symbol is caught by the next day's first run, not the same run.
`VULNCHECK=all make vulncheck-ci` scans regardless; `make vulncheck` and
`make vulncheck-temporal-server` always scan, and the GitHub workflow still
calls those.

**One race-detector run per host (TICK-153 D3).** `make test` and `task test`
run `go test -race` through `tools/ci/racelock`, which holds a listener on
`127.0.0.1:47613` (`NAVA_RACE_LOCK_PORT`) while the command runs. A second run
on the host, from any checkout or agent, waits and says so every 30 s; the OS
releases the lock when the holder exits or is killed, so there is no stale
lock. `NAVA_RACE_LOCK=off` skips it.

## Risks and the layer that covers them

| Risk (by impact) | Layer |
| --- | --- |
| A failing integration test does not fail the PR | Canary `TestCICanary` (`NAVA_CI_CANARY=fail`). Under `CI`, the health integration test fails instead of skipping when Docker is missing |
| A forbidden or unpinned module or image gets in | `check-pins` (fixture tests in `tools/ci/checkpins`) |
| Taskfile and Makefile drift apart, so local runs differ from CI | `check-taskfile` (drift fixture in `tools/ci/checktaskfile`) |
| `api/api.gen.go` is stale against `api/openapi.yaml` (TICK-281: a 422 response in the spec since #130 was never generated) | `check-generate` in the lint job: `go generate ./api/... && git diff --exit-code -- api/`. Fix: `make generate` and commit |
| The acceptance guide is stale against its sources: `api/errors.yaml` gains a code and `bruno/lib/nava.js` `REGISTERED` is not rebuilt, so Bruno passes a body with an unregistered code (TICK-346 review, TICK-391) | `check-guide` in the lint job: `sh scripts/ci/guide-generated.sh` reruns `build.py`, then `git diff --exit-code` on `docs/acceptance/guide/` plus untracked files, and compares the kit zip by member names and bytes (`scripts/ci/zipsame.py`; `build.py` fixes entry times and modes, and deflate output can differ between zlib builds). Fix: `python -I docs/acceptance/guide/build/build.py` and commit |
| A suite or report upload drops out of `ci.yml` and CI stays green because it never ran (TICK-281: an API upload of `ci.yml` lost `make identity-attacks` and the `ci5-report` upload) | `check-ci-steps` in the lint job (`tools/ci/cisteps`): every step in `tools/ci/cisteps/required.yaml`, the one list, must be in its job, not `if: false`, not `continue-on-error`, and a report upload must keep `if-no-files-found: error`. Edit fixtures in `TestEditsFail`; `TestDroppingAnyRequiredStepFails` deletes each listed step from the real `ci.yml` and expects exactly that failure |
| A story's work never reaches `main`: its story branch gets task PRs after the story PR squash-merged (S5: #251, #252, #255), or the story is closed with no PR to `main` at all (S6, TICK-48), and CI on `main` stays green because the work is not there to test (TICK-286) | `story-branches` job, `make check-story-branches` (`tools/ci/storybranches`, TICK-322), below. Rules in `TestMilestone1Replay` and its neighbours; the tree comparison against a real squash merge in `TestLandedSquashMerge` |
| A build-tagged test file stops compiling and CI stays green, because `go vet` and `go test` build untagged files only and the tagged suites run only in the stack jobs (TICK-286 finding 4: a duplicate `onboardPostRaw` behind `identitystack && acceptance` on `main`) | `vet-tags` in the lint job (`tools/ci/buildtags`, TICK-320): finds every module (each `go.mod` outside `testdata`), reads every `//go:build` line, takes for each file every minimal set of custom tags that builds it (one for `a && b`, one per side of `a || b`; GOOS, GOARCH and `go1.N` take their linux/amd64 values), and runs `go vet -tags <set> ./...` in the module once per distinct set, so a new tag or module is covered with no edit. A file with more than 16 custom tags fails the step instead of being skipped. `TestBrokenTaggedFileFails` vets a fixture whose `stacktag && accept` test redeclares a `stacktag` helper and expects exactly that set to fail; `TestRepositoryCoversTheAcceptanceTag` keeps the `acceptance,identitystack` set discovered. Tags are never combined beyond what one file needs, so two tags no file uses together are not built together |
| A security-path change goes without the D-P10 review | `review-paths` prints the summary lines on every PR (fixture: a PR touching `db/`) |
| A registered isolation suite runs zero tests and CI stays green (a `TestMain` exiting before `m.Run`, a step dropping `-tags`) | `tools/ci/suiteran` (TICK-230) wraps `go test -json` in `test`, `temporal-server-test`, `test-integration`, `isolation`, `stack-test-identity` and `stack-test-sdk`: every `cto_signoff.isolation_tests` file whose package ran and whose build constraints hold for the step's `-tags` must have at least one of its tests pass or fail. Fixtures in `tools/ci/suiteran/testdata/fixture`; `TestEveryRegisteredSuiteHasACheckedStep` fails when a registered suite has no step whose `-tags` builds it |
| An integration test hangs and the cancelled job's log is empty, so nobody can tell which test | `go test -timeout 6m` in `test-integration.sh` (below the job's 10 minutes) and the groups' output streamed through `tools/ci/testlog` (TICK-225). Unit tests in `tools/ci/testlog`; the hang itself was proven once on a throwaway PR (not merged) |
| A shallow clone passes `check-openapi` with nothing to compare against | `check-openapi` fails when `BASE` is missing |
| A change to `nava.interpreter` breaks determinism, so runs already in flight fail on their next workflow task after a deploy (Epic 09 AC2) | The replay gate in the `unit` job (`make test`), below: `TestReplayRecordedHistories` replays every history under `internal/workflow/testdata/histories/`; `TestHistoriesAppendOnly` keeps the ones on `main` unchanged; `TestHistoriesHoldReferencesOnly` keeps them references only (ADR-001 R1) |
| A branch's migration merges with a version older than one already on `main`, so fresh CI databases and deployed ones migrate in different orders | `TestNewMigrationsLaterThanBase` in `db/` (TICK-299), in the `unit` job with `BASE=origin/main` for every event, also pull requests into a story branch (TICK-309), and full history: fails naming each added migration, its version and `main`'s newest version. Skips locally unless `BASE` is set. Rule: `db/migrations/README.md` |
| pgx's prepared-statement cache breaks through PgBouncer in transaction mode | `internal/db/pooltest` (T06): 50 clients, 10k statements, zero errors; a second PgBouncer with `max_prepared_statements = 0` must fail |
| nava-core goes ready without PostgreSQL | compose healthcheck on `/readyz`, plus the `dependencies` case (200 with Keycloak stopped) |

## Deliberately not covered

- **A suite dropped by editing the step itself.** suiteran trusts the step's own `-tags` and package list. Removing both `-tags`, narrowing the packages, adding a `-run` filter or removing the suiteran call passes the check, so each of those is an edit to `.github/`, the `Makefile` or `scripts/ci/`, all on the `security` list in `review-paths.yaml` (`TestCIScriptsAreSecurityPaths` in `tools/ci/reviewpaths` pins the two scripts). Extra `ARGS` to `make test-integration` or `make isolation` skip the check, since a `-run` filter leaves suites out on purpose; CI never sets `ARGS`.
- **A hang in the isolation suite.** `scripts/ci/isolation.sh` runs after `test-integration.sh` in the same step and keeps Go's default `-timeout` of 10m, so a hang there is still cancelled by the job before Go's dump. Its output does stream (suiteran prints as it reads), so the log shows how far it got. Follow-up, not in TICK-225's scope.
- **testcontainers-go.** Approved, not adopted (decision log, 2026-10-19). The integration tests start a pinned container through the docker CLI, the way T03 and T04 built them.

## Changing nava.interpreter (replay gate, P3-23)

`nava.interpreter` (`internal/workflow/interp`) is replayed against recorded
histories on every `make test` (PLAN-003 D-3-17, TICK-350). The replayer needs
no server, so the gate is an ordinary unit test in the `unit` job; `ci.yml`
has no step for it. A history-only change runs every stage (`cichanged` has no
rule for `internal/workflow/testdata/`), so the gate always runs.

| Test (`internal/workflow/interp`) | Fails when |
| --- | --- |
| `TestReplayRecordedHistories` | a recorded history no longer replays: the change alters the commands a run already produced (non-determinism) |
| `TestHistorySetRequired` + the set check in every gate test | a history in `wantHistories` is missing, the directory holds fewer, or a file is not listed. An empty glob is never green |
| `TestReplayCatchesNonDeterminism` | a deliberate non-deterministic change (an extra activity before the timer) replays green, so the gate would miss one |
| `TestReplayGetVersionKeepsHistoriesGreen` | the same change behind `workflow.GetVersion` fails replay |
| `TestHistoriesAppendOnly` | a history on `main` (`BASE=origin/main`, as for migrations) was changed, or deleted without a tombstone in `retired-histories.json` that names its blob on `main`, a `DefaultVersion` arm `main` had, the call kept with `minSupported` raised, and a replay failing with that change id's TMPRL1100 (RFC-007 R-1 to R-8, rule (c) below). Skips without `BASE`. `TestAppendOnlyCheck` proves each rule on a scratch repository |
| `TestHistoriesHoldReferencesOnly` | a payload in a history is not the start payload (`outbox.Delivery` with `{definition_version_id, record_id}`, or the continue-as-new `RunInput` carrying it with UUID-only refs), a signal delivery (`{signal}`), a `workerguard.Directive`, a UUID-only object or an empty result; or a header other than a UUID `nava-tenant`. `TestR1WalkCatchesRecordData` plants a record field in each kind of payload |

The histories cover every step type (timer, condition both ways, branch on
both cases and the default, wait_signal, update_record, end), the signal path
(with a foreign signal and a redelivery dropped), the timeout path and a
continue-as-new chain that carries a pending signal into the next run.

**Recording.** `make workflow-record-histories` (`make/workflow.mk`) is a
developer target, never a CI step. It starts a Temporal dev server
(`testsuite.StartDevServer`; `NAVA_TEMPORAL_CLI=<path>` uses a local `temporal`
CLI), runs the scenarios in `record_histories_test.go` (build tag
`recordhistories`) behind the worker guard and the Propagator, and writes the
missing files only. Add a scenario there, record, and add its file names to
`wantHistories` in `replay_test.go`.

**Shipping a change that the gate rejects.** Never re-record or edit a history
that is on `main`. Either:

1. **`workflow.GetVersion`**: put the new command order behind
   `v := workflow.GetVersion(ctx, "<change-id>", workflow.DefaultVersion, 1)`,
   keep the old order for `DefaultVersion`, record a new history that takes the
   new branch, and add it to `wantHistories`. Old histories replay down the old
   branch; the new one pins the new branch. The call is never deleted; retiring
   its old arm is rule (c). Or
2. **a new workflow type** (`nava.interpreter.v2`): register it beside the old
   one through a D9 policy PR (the Temporal authorizer allow-list,
   `temporal-server/authz`), start new runs on it, and keep the old type and
   its histories until its runs have drained.

Worker Versioning is not an option (D-3-17): a versioned poll is denied
(`checkUnversioned`, `temporal-server/authz/authorizer.go`), and the
versioning-rule calls are denied to every platform identity (`policy.go`);
allowing them is an ADR-001 amendment.

**Deploy rules** (Technical Architect, TICK-363):

- **(a)** For any change that needs `GetVersion`, stop the old interpreter
  workers before the new ones start. An old worker that picks up a task of a
  run the new code already advanced would replay it against the old code and
  fail it.
- **(b)** Never roll back across a recorded `GetVersion`. Runs that recorded
  the new version's marker cannot replay on code without the branch; fix
  forward instead.
- **(c)** Retire a `GetVersion` change's old arm **only by raising its `minSupported`** above `workflow.DefaultVersion` and deleting the `DefaultVersion` code. **Never delete the `GetVersion` call**: kept, it makes any old run that is still open fail on TMPRL1100 and block, where a deleted call lets it run the new arm silently. Do this only when an Operator `ListWorkflowExecutions` query (`WorkflowType = 'nava.interpreter' AND ExecutionStatus = 'Running' AND StartTime < '<T>'`, where `T` is the completion of the latest interpreter-worker deploy that introduced the change) returns nothing. Run the query after the Technical Architect approves the PR. Run it again if a `nava.interpreter` reset or an interpreter-worker deploy or rollback happens before the merge.
  **Retiring the histories this breaks.** In the same PR, delete each history that now fails replay with `TMPRL1100 … removed support of version -1 … "<change_id>"` from `histories/` and from `wantHistories`, and append one entry per file (`history`, `blob` on main, `change_id`, `issue`) to `internal/workflow/testdata/retired-histories.json`. `TestHistoriesAppendOnly` checks the tombstone, the kept call and the replay failure (RFC-007, D-3-25). The evidence goes on the `issue`. Only the human owner merges a PR that touches the manifest ("MergePr" label); agents never do.
  If an old run was missed and blocks after the deploy, fix forward: lower `minSupported` back to `DefaultVersion`, restore the arm, and redeploy.

## Run it

```sh
make tools        # once
make ci           # all four stages
make ci-lint      # or one stage
NAVA_CI_CANARY=fail make test-integration ARGS="-run TestCICanary"   # canary, must fail
make ci-isolated ID=pr-57   # all four stages at once, on a stack of its own (below)
make ci-changed ID=pr-57    # only the stages the change against origin/main needs
make reap-integration REAP_DRY=1   # integration leftovers older than 6h; drop REAP_DRY to remove
```

`make ci-changed` (`task ci:changed`, TICK-153 O7) asks `tools/ci/cichanged`
which stages the files changed since the merge base with `BASE` need,
uncommitted and untracked files included, and runs `ci-isolated` with them.
Every change runs `ci-lint`. Documentation (`.md` and `.html` outside a Go
package and outside `testdata`) runs nothing else; any other file under
`docs/` runs all four, since tests read some of them. A Go package's files add
`ci-unit`, plus `ci-integration` when an integration test depends on the
package and `ci-stack` when an image or a stack test does (`go list -deps`).
Compose and `deploy/` files add `ci-unit` (unit tests read the compose and
collector files) and `ci-stack`; the integration script adds `ci-integration`.
`go.mod`, `go.sum`, `docs/pins.md`, the Makefile, and any file no rule knows
run all four. A renamed file counts at both paths (`git diff --no-renames`).
`go run ./tools/ci/cichanged -v` prints the reason for each file. Whether a
`ci-changed` run can replace the full pipeline as a PR's final gate (D5) was
settled by the human on 2026-10-25 (decision log): it can.

### In GitHub

A pull request runs only the jobs its change needs. The `changes` job runs
`make ci-stages`, which prints the same stages `cichanged` picks and writes
`unit`, `integration` and `stack` as `true` or `false` for the job conditions:

| Stage needed | Jobs that run |
| --- | --- |
| always | `changes`, `lint` |
| `ci-unit` | `unit`, `temporal-server` |
| `ci-integration` | `integration` |
| `ci-stack` | `stack`, `stack-db`, `stack-internal` |

A change under `.github/workflows` runs every job, since only a full run proves
a workflow edit. A skipped job shows as skipped, not failed. For a full run on
any branch, start the `ci` workflow by hand from the Actions tab
(`workflow_dispatch`, or `gh workflow run ci --ref <branch>`); it sets
`CI_ALL=1`, so every job runs.

A push to `main` or to any `TICK-**` branch (TICK-289) also sets `CI_ALL=1`:
the merged tree runs every job, since a merge result has no meaningful diff
base. Story branches are `TICK-<story>/<slug>`, so the filter needs `**` to
cross the `/`; it matches task branches too, which then run the full suite on
every push (an accepted cost: a glob cannot tell the two apart). Each push has its own concurrency group (per ref and sha), so a newer
merge neither cancels nor replaces the run of an older one. When any job of a
push to `main` fails, the `main-red` job (`scripts/ci/main-red.sh`, the only
job with `issues: write` besides `story-branches`) opens a GitHub issue `main is red: <short sha>`
labelled `P0`, with the failing jobs and the run URL; while one such issue is
open, a later red push comments on it instead. Close it once `main` is green. Locally, `make ci-stages`
prints the stages for the current branch and `CI_ALL=1 make ci-stages` prints
all four.

### Story branches (TICK-322)

The `story-branches` job runs `make check-story-branches` on every event, and
every 6 hours on `main` (`schedule`; a scheduled run skips every other job). It
reads the remote's branches and every pull request, not the checked-out tree,
so its result is about the repository: it can go red on a pull request that
did not cause it. It is not a branch-protection check and not in `main-red`'s
`needs`.

CI cannot read Multica, so **a merged story PR, or an entry in
`tools/ci/storybranches/closed.yaml`, is the "story closed" signal**
(TICK-379).
A **story branch** is a `TICK-<n>/<slug>` branch that at least one merged pull
request targets (task PRs merge into it). Its **story PR** is the newest merged
pull request out of it, into `main` or, for a nested branch, into another
story branch. An open pull request out of it means the story is still in
flight, and nothing below fires for it. So does a branch with no PR out of
it whose story `closed.yaml` does not list: under
`docs/delivery/branching-strategy.md` the story PR opens only after every task
has merged, so until then the branch is in flight, not orphaned. A done story
normally has a merged story PR. When a story closes without one (usually a
cancellation), the Technical Lead deletes its branch or adds a `closed.yaml`
entry (`branch`, `status: done|cancelled`, `issue`) in the same change. A bad
entry makes the tool exit 2.

| Finding | Fires when | Clear it by |
| --- | --- | --- |
| `story PR #N merged …, but changes since then are not on main` | the branch gained changes after its story PR merged (a PR merged into it, or a push) and they are not on `main` | a PR into `main` that carries them (TICK-316, #267), or a `resolved.yaml` entry |
| `orphaned: story TICK-<n> is <status> (closed.yaml), …` | `closed.yaml` lists its story as done or cancelled, the branch has merged task PRs, no merged or open PR out of it, and its changes are not on `main` | open its story PR (TICK-314, #266), delete the branch if the work is abandoned, or a `resolved.yaml` entry |
| `open PR #N targets it, but its story PR #M merged …` | an open pull request targets a story branch whose story PR has merged | retarget the pull request to `main` |

**"On main" compares trees, not commits.** Squash merges keep the tree and drop
the ancestry, so `git merge-base --is-ancestor` cannot answer. For each
first-parent `main` commit `c` since the story PR merged (newest first, from an
hour before the merge, at most 500), the check runs
`git merge-tree --write-tree --merge-base=<story PR head> c <branch>` (git 2.40
or later). If any result is `c`'s own tree, the branch's changes since its
story PR were all on `main` at `c`, by whichever PR they got there. The search
covers more than `main`'s tip because, once the changes land, `main` may edit
the same files again, and merging the old branch into the tip would then
conflict although nothing is missing. A conflict at `c` means "not on main at
`c`". For an orphan, git picks the merge base and the search starts at the
branch head's commit time. The story PR's head is GitHub's `headRefOid`, which
stays fetchable even when the branch was force-pushed. The cost of searching
history: a change that landed and was later reverted on `main` on purpose
counts as on `main`.

**Accepting a branch.** `tools/ci/storybranches/resolved.yaml` lists a branch
with its full head sha and why (the issue that decided it). It is accepted at
that head only; a later push flags it again. Deleting an abandoned branch is
usually better.

**The issue.** On `main` (push, schedule or manual run), a red check opens one
issue, `Story branches out of step with main`, labelled `story-drift`, with the
report (`scripts/ci/story-branches-issue.sh`, `make story-branches-issue`). A
later red run comments only when the findings differ from the last report the
workflow posted (people's comments do not count); the first green run closes
it. Runs that can touch the issue share the job concurrency group
`story-drift-issue`, so they queue rather than race; a newer queued run can
replace an older pending one, which then shows as cancelled. If two issues
open anyway, a red run keeps the oldest and closes the others as duplicates,
and a green run closes them all. A red check with no `story-branches.md` means
the tool errored (exit 2): the issue step logs that and leaves the issue
alone. Pull requests and other branches never touch the issue.

**Replay.** `-as-of <RFC 3339 time>` judges the pull requests as they stood
then (created later: ignored; merged or closed later: open). Branch heads stay
current. Locally, with `gh` signed in:

```sh
make check-story-branches                                          # now
go run ./tools/ci/storybranches -as-of 2026-10-28T18:00:00Z        # before #266 opened
```

**Not covered.** A story closed in Multica while its story PR is still open
(S6 since #266 opened on 2026-10-28 18:04Z): CI sees an open PR and treats the
story as in flight. Branches outside `TICK-<n>/` (`agent/...`), and pull
requests from forks, are not story branches.

## Parallel runs (TICK-147)

`make ci-isolated ID=<pr-or-sha>` (`task ci:isolated ID=...`) runs the same four
stages as `make ci`, all at once (TICK-153 O4; `PARALLEL=0` runs them in
order), on a stack nothing else shares, so several checkouts run
CI at once on one Docker host. Run it from each PR's own checkout; one run per
checkout at a time (the dev certificates and `ci6-report.md` live in it).

| Shared on the host before | Per run now | Set by |
| --- | --- | --- |
| compose project `nava` (containers, networks, volumes) | `nava-<ID>` | `COMPOSE_PROJECT_NAME` |
| image tags `nava/*:dev` | `nava/*:<ID>` | `NAVA_IMAGE_TAG` (compose.yaml, `TEMPORAL_SERVER_IMAGE`, the stack tests) |
| host ports 18080, 18081, 17233, 18233 | four ports in 20000-21999 from a checksum of ID, skipping ports a running container publishes; printed at the start | `NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `TEMPORAL_TEST_PORT`, `UI_PORT` |

The ID is lowercased; characters other than `a-z 0-9 _ -` become `-`; `dev`
is refused. On exit, pass or fail, the run removes its project
(`down -v --remove-orphans`) and untags its seven `nava/*:<ID>` images (keycloak,
nava-core, nava-db, nava-relay, nava-worker, otel-collector, temporal-server). It never
prunes, so shared layers and the build cache stay. It then fails if any
`nava-<ID>` container, volume, network or `nava/*:<ID>` image tag is left, so
an image added to compose.yaml but not to the script's list fails the run.
`STAGES="ci-unit ci-stack"` runs a subset. Each stage writes its own log
to `$TMPDIR/ci-isolated-<ID>/`, printed whole when the stage ends. A run killed outright (SIGKILL,
a closed session) skips that cleanup; `make ci-isolated-down ID=<ID>` does it.
It first untags every `nava/*:<ID>` tag Docker lists, not only the seven, so a
tag of an image renamed or dropped since the killed run no longer fails every
later run with that ID (TICK-211; `make test-ci-isolated-down` checks it on
throwaway tags).
The `integration` stage takes random host ports and, since TICK-182, names
its `nava-it-*` and `nava-iso-*` containers and networks by PID plus a random
suffix (a PID alone repeated across PID namespaces on one Docker host, and one
run's cleanup removed another's cluster). It labels each with
`nava.ci.run=<name>` and `nava.ci.owner=<COMPOSE_PROJECT_NAME>` (`none`
outside ci-isolated); the teardown removes and then leak-checks the objects
labelled with its own project only. A cluster removed under a run fails its
health gate at once with "container ... vanished", not after 120 s of
`No such object`.

**Caches, shared by every run.** Nothing is set per run and nothing is cleaned.
- Go: the per-user `GOCACHE`, `GOMODCACHE` and golangci-lint cache, safe for
  concurrent use. The `go1.26.10` toolchain downloads once (`GOTOOLCHAIN=auto`).
- Pinned tools: `make tools` runs only when `make check-tools` fails.
- Images: `IMAGE_CACHE=local` makes `ci-stack` build with `make image-ci`
  (`buildx bake`) on the local builder. Its BuildKit cache is shared by every
  tag and safe for concurrent builds, so per-run tags still share layers. There
  is no `type=local` cache directory, because two concurrent exports to one
  directory can corrupt it.
- Registry images: `image-ci` pulls with `--policy missing`, `compose up`
  pulls only missing images, and the integration script's `docker run` uses the
  local copy.

**How many fit.** Measured on this host (16 CPU, about 20 GB for Docker); see
TICK-147 for the runs. One stack holds about 1.2 GB at rest; Keycloak (about
0.8 GB) is the largest part. The limit is memory during `ci-unit`, not the
stacks: `go test -race ./...` across both modules peaks at several GB, and a
fully loaded host has hit `fatal error: runtime: cannot allocate memory`
there. The race lock (D3) now runs those one at a time on the host, so
parallel runs queue in `ci-unit` instead of running out of memory.

**Stack slots (TICK-153 D4).** At most 7 compose stacks on the host at once;
`ci-stack` holds 1 (2 with `STACK_INTERNAL=beside`) through `tools/ci/racelock
-slots 7` on `127.0.0.1:47620-47626`, and waits, saying so every 30 s, when
fewer are free. N = 7 comes from the memory a stack holds while its tests run
(1.34 GB, `docker stats`, two stacks up) against the Docker budget (19.4 GiB),
less about 4 GB for the other agents' containers and the integration
clusters, with half again as headroom for Keycloak's start-up:
(19.4 - 4) / (1.34 x 1.5) = 7.6. `STACK_SLOTS` changes N;
`NAVA_STACK_SLOTS=off` skips the queue.

`make ci` and `make stack-up` with no variables behave as before (project
`nava`, tag `dev`, the old ports). The GitHub workflow is unchanged.
