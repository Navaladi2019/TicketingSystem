# S10 fresh-stack run: what ran (TICK-623)

Date: 2026-11-11. Host: a developer laptop (Windows, Docker Desktop, Git Bash).
Branch `TICK-623/s10-acceptance-kit`, on the story branch `TICK-336/group-acl-masking` at `e431e8b5`.

## Steps

```sh
docker compose down -v                  # no volume left from an earlier run
docker compose up -d --wait --no-build  # every service healthy
sh docs/acceptance/guide/tools/seed.sh  # tenants A and B, admins groups, default incident ACL v1 in both
sh docs/acceptance/guide/tools/api-test.sh S10
```

Result: **66 of 66 requests, 220 of 220 tests**. The bru output is in `s10-fresh-stack-run.txt` (no tokens, passwords or host paths). The Bruno HTML/JSON reports stayed on the host because they hold short-lived tokens.

## Images

| Image | Source |
|---|---|
| `nava/nava-core`, `nava-onboard`, `nava-groupsync`, `nava-worker`, `nava-relay`, `nava-purge`, `nava-db` (migrations), `keycloak`, `otel-collector` | Built from this branch (`docker compose build`) |
| `nava/temporal-server:dev` | **Prebuilt from main, toolchain 1.26.10 (TICK-699 not in image).** Pulled `ghcr.io/navaladi2019/nava-temporal-server:51fcaaa7e553e75f45e2dbf37695a4d65541c8e2` (main's `temporal-server/` tree hash), digest `sha256:5dfb894190123a1076ac60c797acab967ef8b3c316a804ed7f81bd69df8ab388`, and tagged it as `nava/temporal-server:dev`. The story branch's own key is not published. The difference is only the Go toolchain (TICK-699). S10 tests nava-core's ACL, not Temporal. |

## Database after the run (superuser, read only)

```
 tenant (incident ACL)                | version | default_acl | masks
 00000000-0000-4000-8000-0000000000a1 |       1 | t           |     0   seed: default ACL
 00000000-0000-4000-8000-0000000000a1 |       2 | f           |     0   S10.15 republish
 00000000-0000-4000-8000-0000000000a1 |       3 | f           |     2   S10.31 masks on description, assignee
 00000000-0000-4000-8000-0000000000a1 |       4 | f           |     0   S10.66 masks removed
 00000000-0000-4000-8000-0000000000b1 |       1 | t           |     0   seed: default ACL

 audit_log, subject_type acl_policy: one `publish` row per version above (A: 1, 2, 3, 4; B: 1)
 records: tenant A 6 incidents (S10.17, S10.33 to S10.36, S10.38); tenant B none
```

The refused publishes (S10.13, S10.14, S10.16, S10.22, S10.24 to S10.26) wrote no version and no audit row.

## Windows host fix

On the first attempt `tools/seed.sh` stopped with `curl: (23)`. It exported `MSYS_NO_PATHCONV=1` for the whole script, so Git Bash passed curl's `-o /dev/null` through unmapped. The variable is now set only on the `docker compose` calls. On Linux (CI) nothing changes.
