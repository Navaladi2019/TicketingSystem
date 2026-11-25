# S31 acceptance kit: a policy can say "tickets in my region"

Story TICK-946 · kit task TICK-953 (PLAN-011 P11-20) · ADR-010 D7, D8 as amended by ADR-010-A2 (closure design, C3/W0 list shape) · verified 2026-11-25 on a fresh stack from a new clone, story branch `TICK-946/in-my-region` at `719058d4` (all eight S31 fixes and TICK-1133) plus this kit. First verified 2026-11-22 at `8528ea4f`, before the fixes; every request and test of the two runs has the same name and result.

## What you can now do

An admin can publish an access rule "agents see tickets from their region". A ticket remembers the organisation unit of the person who raised it. Anyone in a unit sees the tickets of that unit and of every unit below it. A ticket raised by someone with no unit is in nobody's region. When a unit is removed from the company directory, its tickets drop out of everyone's region after the next sync. Moving a person to another unit does not move their old tickets.

## Try it (about 3 minutes, plus the first image build)

From the repository root, on the story branch, with Docker, Node.js 20 or later and Go:

```sh
make dev-ca                                  # once per checkout: Temporal's dev certificates
docker compose down -v                       # no volume from an earlier run
docker compose up -d --build --wait          # every service healthy, images from this branch
sh docs/acceptance/S31/run.sh                # seed, both Bruno folders, one group sync, the evidence
```

If another stack holds the default ports, put other ones in `.env` (`NAVA_CORE_PORT`, `KEYCLOAK_PORT`, `KEYCLOAK_PUBLIC_URL`, `PROMETHEUS_PORT`). The verified run used the defaults.

`run.sh` does the following, in order, and stops with exit 1 on any failure:

1. Seeds tenants A and B and runs one group sync (`tools/seed.sh`).
2. Checks that the stack is fresh: `EMEA/Germany` is live and no S31 ticket exists. The kit deletes `EMEA/Germany`, so it runs once per stack.
3. Prints tenant A's org-unit tree: each unit, whether it is live, its closure rows, and its members (`*` = primary unit).
4. Reads two org-unit ids from the database for AC5: tenant A's `EMEA` and tenant B's `eng/platform`. No API lists org units, and `GET /me` lists groups, not units.
5. Runs the Bruno folder **`S31 region`** (38 requests): AC1, AC2, AC5, isolation, then the two directory changes in Keycloak (delete `EMEA/Germany`; move the APAC requester to `EMEA`).
6. Runs one group sync (`nava-groupsync --once`). It prints tenant A's `group sync applied` line and the cycle result, and fails unless the run exits 0 with `failed: 0`. Then it prints the tree again.
7. Runs the Bruno folder **`S31 region after sync`** (16 requests): AC3 and AC4.
8. Prints, from the database, the unit each S31 ticket kept. It fails unless the APAC requester's tickets are `APAC` (old) and `EMEA` (new).
9. Prints the metrics behind the three S31 alerts, starts Prometheus (profile `operator`), and fails unless each of the three alerts is loaded once.

The output of the verified run is `fresh-stack-run.txt`. It holds no tokens or passwords. Two things were removed from it: the local directory before `docs/` in the report paths, and the `npm ci` lines of the Bruno CLI's first install.

To click through by hand, open the collection `docs/acceptance/guide/bruno/` in Bruno and select the `local` environment. Fill `s31_a_unit` and `s31_b_unit` with the ids from step 4. Run `S31 region`, then `docker compose run --rm --no-deps nava-groupsync --once`, then `S31 region after sync`. Without the manual sync, S31B.9 waits for the service's next cycle (up to `NAVA_GROUPSYNC_INTERVAL`, 5 minutes by default). Each request's Docs tab says what it shows.

### The seeded directory (realm fixture `deploy/compose/keycloak/import/realm-nava.json`)

| Story user | Seeded user | Keycloak groups in tenant A |
|---|---|---|
| `admin@tenant-a` | `admin@tenant-a.test` | tenant admin (`/admins`, from the seed) |
| `agent-emea@tenant-a` | `agent-emea@tenant-a.test` | `/ou/EMEA` |
| requester in `EMEA` | `requester-emea@tenant-a.test` | `/ou/EMEA`, `/requesters` |
| requester in `EMEA/Germany` | `requester-germany@tenant-a.test` | `/ou/EMEA/Germany`, `/requesters` |
| requester in `APAC` | `requester-apac@tenant-a.test` | `/ou/APAC`, `/requesters` |
| requester with no unit | `requester-none@tenant-a.test` | `/requesters` only |
| tenant B (isolation) | `bob@tenant-b.test` | tenant B's `/service-desk`, `/ou/eng/platform` |

Passwords are the realm's dev-only pattern, listed in `docs/acceptance/guide/bruno/environments/local.bru`. Each requester has one unit, so groupsync makes it their primary unit with no `nava_primary_org_unit` attribute. `/requesters` is a plain group: groupsync only syncs users who are in some group of the tenant, and the no-unit requester needs one.

### The access policy the folder publishes (S31.16)

```json
{ "format": 2, "conditions": [],
  "policies": [
    { "name": "admins",       "ops": ["read", "write"], "when": { "member_of": { "groups": ["<tenant A /admins>"] } } },
    { "name": "in_my_region", "ops": ["read"],          "when": { "under": { "of": "caller" } } } ] }
```

Nobody else has incident access, so everything agent-emea and the requesters see comes through `in_my_region`. The admin creates every ticket, with `requester` set. The platform stores the requester's primary unit on the ticket. A body that names the unit is refused.

## Acceptance criteria

| # | Criterion | Result | Evidence (`fresh-stack-run.txt`) |
|---|---|---|---|
| 1 | `agent-emea` lists exactly the tickets raised by requesters in `EMEA` and `EMEA/Germany`; `total` matches; an `APAC` ticket by id is the missing-record 404 | ✅ | S31.21: the list holds the EMEA and Germany tickets and not the APAC or no-unit ones, every item's requester is the EMEA or Germany requester, `total` 2 = items, one page. S31.23: the APAC ticket by id is 404, byte-identical to the 404 of an id that does not exist (S31.22). S31.24: the Germany ticket by id is 200, so the single read agrees with the list (I5) |
| 2 | A ticket raised by a requester with no primary unit is not visible through `in_my_region` to anyone | ✅ | DB step 8: the ticket's unit is `(none)`. Not visible to agent-emea by id (S31.25, 404), nor in the lists of the EMEA (S31.26, total 2), Germany (S31.27, total 1) and APAC (S31.28, total 1) requesters. The no-unit requester sees nothing, not even their own ticket (S31.29, total 0; S31.30, 404). Only the admin reads it, through `admins` |
| 3 | After `EMEA/Germany` is removed in the directory and groupsync runs, the Germany tickets drop out of `agent-emea`'s list, and the sync run succeeds | ✅ | S31.36 deletes the group in Keycloak. Step 6: `org_units_deleted: 1`, `org_members_removed: 2`, `org_closure_rebuilt: true`, `org_closure_unrooted: 0`, then `nava-groupsync --once done`, `failed: 0`, exit 0. The tree before: `EMEA/Germany` live, 1 closure row; after: tombstoned, 0 closure rows, no members, and `EMEA`'s closure rows went from 2 to 1. S31B.9: agent-emea's list no longer has the Germany ticket (`total` 1, the EMEA ticket). S31B.11: the Germany ticket by id is 404. S31B.12: the Germany requester, now in no unit, sees nothing |
| 4 | Moving a requester from `APAC` to `EMEA` leaves their existing tickets in `APAC` | ✅ | S31.37 and S31.38 move them in Keycloak; after the sync the tree shows them primary in `EMEA`, and `APAC` has no members. DB step 8: their old ticket INC0000003 is still `APAC`, their new one INC0000005 (S31B.14) is `EMEA`: `AC4: PASS`. S31B.13: agent-emea still gets 404 on the old ticket. S31B.15: agent-emea lists the new one, not the old one. S31B.16: the moved requester's own region has the new ticket, not their old APAC one |
| 5 | A policy naming another tenant's org unit is refused with 422 | ✅ | S31.31: `in_my_region` with `{"under": {"of": ["<tenant B eng/platform>"]}}` answers 422 `invalid_access_policy`, `foreign_id` at `/rules/policies/1/when/under/of/0`, and the body does not echo the id. Control S31.32: the same document naming tenant A's `EMEA` is accepted (200), so only the unit's tenant made the difference. The test then publishes `under caller` back |

Result: `S31 region` 38/38 requests, 130/130 tests. `S31 region after sync` 16/16 requests, 47/47 tests. `run.sh` exit 0.

S31 has no UI. Every criterion is an API call, so the evidence is the recorded Bruno run and the database reads, not screenshots.

## Isolation check

| Tried | Result |
|---|---|
| Tenant A publishes a region rule naming tenant B's unit (S31.31) | 422 `foreign_id`, nothing published. The answer is the same as for an id of no tenant, and the body does not echo the id |
| Bob (tenant B) lists incidents (S31.33) | none of tenant A's four S31 tickets; the body names neither tenant A's `EMEA` id nor the run tag |
| Bob opens tenant A's EMEA ticket by id (S31.34) | the missing-record 404, byte-identical to an id that does not exist |

The adversarial isolation tests (tenant B's units and closure rows under RLS, a foreign unit in `under`, a forged `org_unit_id`) are TICK-952's.

## Alerts

Three alerts watch the region design. Step 9 shows all three loaded and `inactive`:

- `GroupsyncOrgUnitsAboveCertified`: `groupsync_org_units_live_max > 10111` for 10 minutes (TICK-1072, `deploy/compose/prometheus/rules/groupsync.yml`). The tree is larger than ADR-010-A2 is certified for. It never stops a sync.
- `GroupsyncOrgUnitsUnrooted`: `increase(groupsync_org_closure_unrooted_units_total[15m]) > 0` (TICK-1116, `deploy/compose/prometheus/rules/groupsync.yml`). A live unit has no closure self row, so it is in nobody's region. It never stops a sync. The walk shows only that it is loaded: the counter has no sample until a cycle adds to it, and no step makes an unrooted unit. `deploy/compose/prometheus/tests/groupsync_test.yml` covers when it fires.
- `EntityRecordEstimatesStale`: `authz_under_estimate_age_seconds > 10800` for 15 minutes (TICK-1071, `deploy/compose/prometheus/rules/authz.yml`). Stale estimates change the list's cost, never its rows.

## Not included

- **The eight S31 fixes are in the verified build, but the kit has no step of its own for them** (for example a workflow policy test with `under`, or the masked unit in history and the audit export). Their tests are in their PRs: #869, #871 to #877.
- No UI to pick or show org units. An admin writes the policy JSON through `PUT /api/v1/entity-types/incident/access`.
- No API lists org units or shows a ticket's unit: `GET /me` lists groups, and a record's body has no `org_unit_id`. The kit reads both from the database (`run.sh` steps 3, 4 and 8).
- A requester who raises their own ticket is not walked. The admin creates every ticket with `requester` set, which takes the "another requester" path of `resolveUnit`. The pipeline's own tests cover the unit on create (`internal/records/pipeline/org_unit_integration_test.go`, for example `TestCreateSetsOrgUnitFromRequestersPrimaryUnit` and `TestOrgUnitOnEveryInsertPath`).
- `nava_primary_org_unit` (choosing a primary among several units) is not walked: every seeded user has one unit. `internal/groupsync/orgmembers_integration_test.go` (`TestOrgMembersAndPrimaryWritten`) covers it.
- `under` with a listed unit is shown only as AC5's accepted control (S31.32), not walked against records.

## Known limits

- **One pass per fresh stack.** S31.36 deletes `EMEA/Germany` in Keycloak, and the folder leaves tenant A's incident access as S31.16 published it (`admins` and `in_my_region`). `run.sh` step 2 stops on a used stack. The folders after it in the full collection (`Z`) do not read incident access.
- **`groupsync_org_units_live_max` has no sample right after the stack starts.** The long-running `nava-groupsync` sets it on its first cycle, `NAVA_GROUPSYNC_INTERVAL` after start (5 minutes by default). The `--once` run that applies AC3 exits with its own metrics. Its log line carries the same tree numbers (`org_subtree_max`, `org_closure_rows`).
- **The realm fixture gained five tenant A users, three units and the `/requesters` group.** `internal/identitystack/directory_test.go` now pins tenant A at eight users and six units. Kits that count tenant A's users or groups would see them; none of the current folders does.
