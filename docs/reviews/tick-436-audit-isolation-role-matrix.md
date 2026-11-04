# Review: TICK-436 (P5-19) audit isolation and role-matrix tests

- **PR:** #445, `TICK-436/audit-isolation-role-matrix` into `TICK-419/one-audit-log`
- **Reviewed at:** `e350d8a`
- **Reviewer:** Quality, Testing & CI Advisor
- **Contract:** ADR-004 CI-A4 (read path), CI-A5 (table bytes), CI-A10; TICK-449 grooming; the TICK-435 update in the issue body.

## Verdict

**APPROVE.** There are no Bug, Spec-drift or Missing-test findings against the acceptance criteria. The two "Breaks under" findings are about how the harness behaves if things change later. They are not gaps today, so they can go in a follow-up.

Before merge:
- The `stack` job has to finish green.
- The CTO has to sign off. `registry.go` is on the `cto_signoff.paths` list in `docs/delivery/review-paths.yaml`. The change only adds entries: one `auditPaths()` append and three Coverage rows. Nothing is removed or skipped.

## Acceptance criteria against the code

| Criterion | Where | Guard proven by a red run |
|---|---|---|
| CI-A4: tenant B reads none of tenant A's entries by direct SQL as every runtime role, and none with no tenant | `auditDirectSQLEveryRole` (`audit_paths.go:346`) | Yes: RLS `USING (true)` |
| The `nava_app` probe must give 0 rows, not `42501` (TICK-435 update) | Control at `audit_paths.go:382-393` checks `has_column_privilege` on `diff` and `diff_salt` first | Yes, through the same RLS red run |
| DEFAULT partition: `42501` for every runtime role | `auditDefaultPartitionRefused` (`:227`), with a seeded-count control | Yes: GRANT SELECT |
| `nava_onboard` pinned policies | `auditOnboardPinned` (`:444`): tested from a provisioning tenant and from an active one, both reading and appending | Yes |
| Reader: A's `subject_id`, `change_set_id`, `after_seq` and `page_token`, sent as B, give the same response as an unknown value | `auditReaderOtherTenant` (`:620`), for B's admin, auditor and agent | Yes: RLS plus the reader filter, and the cursor binding |
| History: 404, not 403, and `domain = record` only | `auditHistory` (`:688`). Tenant B gets its own `iso_et_aud` type, so only the record guard can answer | Yes: `pipeline.Get`, and the domain filter |
| CI-A5: no canary in the heap or TOAST of `audit_log` (DEFAULT included) or `entity_definition_versions` after `CHECKPOINT`, with positive markers | `auditCanaryBytes` (`:790`), `auditTableBytes` (`:746`) | Yes: `secrets.Check` and `CheckSchemaSecrets` disabled |
| CI-A10: the pools from `compose.yaml` log in as `nava_app` and `nava_worker`, and the P5-10b matrix holds through them | `auditServicePoolMatrix` (`:967`) | Yes: the head grant, and the compose role swap |
| Registry names the tables (TICK-449). `audit_log_v0` is left out (TICK-435) | `audit_paths.go:91-135`, `registry.go` | n/a |

Every path has a positive control, so a probe that sees nothing fails instead of passing. The byte scan reads the dead tuples that rolled-back writes leave in the heap. That is the right target for CI-A5.

## Findings (most severe first)

### 1. Breaks under: a fixture row compressed past the TOAST threshold makes the CI-A5 scan blind

`audit_paths.go:746-780` and `:866-881`. The scan looks for canaries as plain bytes. PostgreSQL compresses a `jsonb` or `text` value inline (pglz or lz4) once the tuple passes about 2 kB. A compressed canary does not appear as a plain byte string. Today the definitions are small, and the positive marker sits in the same field (`description`) of a row about the same size, so the control holds. If `etDefTail` or the schema later grows past the threshold, the canary row and the control row can end up compressed differently. A missed refusal would then pass green.

**Fix (test code):** assert that the control rows are stored uncompressed, for example with `pg_column_compression(definition) IS NULL` on the control version and `pg_column_compression(diff) IS NULL` on the marker entry. Growing fixtures then fail the control loudly instead of hiding a leak.

### 2. Breaks under: `compose.yaml` with an unterminated `${` panics instead of failing

`audit_paths.go:913-916`. `v[i+2:i+j]` is evaluated before `j < 0` is checked. When `j == -1`, the slice is out of range and the whole isolation binary panics, instead of reporting a failure on this one path. Move the `j < 0` check above the `strings.Cut`.

### 3. Preference: name the tasks that own the rest of CI-A4 and CI-A5

`registry.go`, Coverage entries `CI-A4` and `CI-A5`. They say "Export, stubs and reseed: their own tasks" and "export bytes: the export task". The other partial entries use `→ P3-37 (S10)` form. Naming the P5 task ids keeps the partial coverage traceable in the report.

### 4. Preference: `Role` on the multi-role DB paths

`audit_paths.go:137-147`. `auditDefaultPartitionRefused`, `auditDirectSQLEveryRole` and `auditServicePoolMatrix` run as every login or as two roles, but they are registered as `Role: "nava_app"`. The report would read more accurately with a label that says so, such as `every login`, if the registry allows one.

## Not reviewed here

- Masking and the diff-read guard: TICK-541, as agreed.
- The tenant-isolation judgement itself: the Security & Multi-Tenancy Reviewer authored it.

## Verification

- CI on `e350d8a`: `unit`, `lint`, `integration` (both runs), `temporal-server`, `commit-check-budget` and `story-branches` pass. `stack` is still running.
- The local `task test` and `task test:short` gaps in the PR body are host contention and CRLF on Windows (`db` `TestE5GroupsTombstoneMigrationsSplit`, which this PR does not touch). Linux CI is the authority, and `integration` covers the isolation package there.
