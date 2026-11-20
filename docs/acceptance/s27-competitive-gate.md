# Acceptance kit: S27, CI refuses an ADR without a Competitive Gate and a change that removes tenant configuration

Story TICK-878, kit TICK-882. Story branch `TICK-878/competitive-gate-checks`.
ADR: ADR-008 (`docs/adr/ADR-008-competitive-gate.md`). Plan: PLAN-009 (`docs/delivery/plan-adr-008.md`).

## What you can now do

- `task lint` refuses a design record (ADR, amendment, erratum) that has no `## Competitive Gate` section. The message names the file.
- `task lint` refuses a design record that cites a competitor pain point (`CP-…`) or a measurement method (`M-X…`) that is not on our lists. The message names the unknown id.
- The tests refuse a change that removes a workflow step type tenants can use, such as `timer`. The message names the removed item.

## Try it (about 10 minutes, no stack)

Nothing here touches the running service: no Docker, no seed, no Bruno.

**Prerequisites.** A clone of the repo on the story branch, with a clean working tree, and the tools the repo already needs for `task lint`: Go (the version in `go.mod`), go-task v3, golangci-lint. Run every command from the repository root.

```sh
git checkout TICK-878/competitive-gate-checks
git status --short      # prints nothing
```

Two commands do the checking:

- `task lint` runs all the lint gates. The Competitive Gate check is the `go run ./tools/adrgate` step in it. `go run ./tools/adrgate` on its own prints the same lines in a few seconds.
- `go test ./internal/configsurface/...` is the upgrade-compatibility suite (M-X2). `task test` runs it too, among every other package with the race detector, which takes more than 10 minutes on a laptop. The suite prints the same lines either way.

### Step 1 (AC1): an ADR without a Competitive Gate fails, and passes once the section is back

1. Copy the template:

   ```sh
   cp docs/adr/_template-adr.md docs/adr/ADR-099-scratch.md
   ```

2. In `docs/adr/ADR-099-scratch.md`, change the front-matter line `date: YYYY-MM-DD` to `date: 2026-11-15`.
3. Delete everything from the line `## Competitive Gate (ADR-008)` to the end of the file. It is the last section of the template.
4. Run `task lint`. Expected: it fails at the `adrgate` step with exactly this line:

   ```text
   docs/adr/ADR-099-scratch.md: CI-1: no ## Competitive Gate section (ADR-008 D6)
   ```

5. Put the section back: copy the lines from `## Competitive Gate (ADR-008)` to the end of `docs/adr/_template-adr.md` and paste them at the end of the scratch file.
6. Run `task lint`. Expected: it passes (exit 0). The template's `**Result:** PASS | …` line starts with `PASS`, so the unfilled section already counts.

### Step 2 (AC2): an unknown pain point or method fails, naming the id

1. Keep the scratch file from step 1, with its section restored. Add this line at the end:

   ```text
   This scratch ADR avoids CP-SF-99 and is measured by M-X9.
   ```

2. Run `task lint`. Expected: it fails at the `adrgate` step with exactly these two lines:

   ```text
   docs/adr/ADR-099-scratch.md: CI-2: CP-SF-99 is not a row in docs/adr/competitive/pain-points.md
   docs/adr/ADR-099-scratch.md: CI-4: M-X9 is not a method in ADR-008 "The axes"
   ```

3. Revert: delete the line you added. `task lint` passes again.

### Step 3 (AC3): removing the `timer` step type fails, naming `timer`

1. In `internal/workflow/def/schema.json`, find the step `type` enum (around line 58):

   ```json
   "type": { "enum": ["condition", "branch", "timer", "wait_signal", "update_record", "end"] }
   ```

   Remove `"timer", ` so it reads:

   ```json
   "type": { "enum": ["condition", "branch", "wait_signal", "update_record", "end"] }
   ```

2. Run `go test ./internal/configsurface/...`. Expected: `FAIL`, and `TestRemoval` prints, among its lines:

   ```text
   configsurface_test.go:65: removed: workflow.step_types/timer
   ```

   It also lists the four `timer` step fields as removed, and `TestFixturesStillAccepted` reports `not accepted: workflow/timer.json: …` (the stored `timer` fixture no longer parses). All of them name `timer`.
3. Revert: `git checkout internal/workflow/def/schema.json`.

### Step 4 (AC4): on the story branch head, both checks pass

1. Clean up:

   ```sh
   git checkout .
   rm docs/adr/ADR-099-scratch.md
   git status --short      # prints nothing again
   ```

2. Run `task lint`. Expected: exit 0. Its last line is:

   ```text
   compat: no internal/configsurface/testdata/inventory.json on origin/main: nothing to compare
   ```

   That line is normal until the story merges into `main` (see Known limits).
3. Run `go test ./internal/configsurface/...`. Expected:

   ```text
   ok  	github.com/Navaladi2019/TicketingSystem/internal/configsurface	…
   ok  	github.com/Navaladi2019/TicketingSystem/internal/configsurface/cmd/compat	…
   ```

No existing ADR and no row of `docs/adr/competitive/pain-points.md` was changed to get here: `git status` is clean.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| AC1 | A copy of the template saved as `docs/adr/ADR-099-scratch.md`, with `## Competitive Gate` deleted, makes `task lint` fail naming that file. Putting the section back makes it pass. | ✅ | Step 1: `docs/adr/ADR-099-scratch.md: CI-1: no ## Competitive Gate section (ADR-008 D6)`, then exit 0 after the restore. |
| AC2 | Citing `CP-SF-99` or `M-X9` in that file makes `task lint` fail, naming the unknown id. | ✅ | Step 2: the `CI-2: CP-SF-99 …` and `CI-4: M-X9 …` lines, then exit 0 after the revert. |
| AC3 | Removing `timer` from the step types in `internal/workflow/def/schema.json` makes `task test` fail, naming `timer` as removed. | ✅ | Step 3: `removed: workflow.step_types/timer`, from `go test ./internal/configsurface/...` and from a `task test` run with the same edit. |
| AC4 | On the story branch head, both checks pass with no change to any existing ADR or to the register. | ✅ | Step 4: `task lint` exit 0 and both configsurface packages `ok`, with a clean working tree. |

**Status: ready.**

## Not included

- The other gate rules are not walked here: CI-1 for amendments and errata dated on or after ADR-008's acceptance, CI-3 (register rows need an `https://` Evidence link and a grade), the `**Result:**` value check, and the in-flight exemption. They are covered by `tools/adrgate`'s tests (`main_test.go`, `run_test.go`), which `task test` runs.
- Checks 2 to 4 of the suite (unlisted additions, deprecated items) and type-definition keywords are covered by `internal/configsurface`'s own tests, not by a kit step.
- No CI workflow change: both checks ride the existing `lint` and `unit` jobs.

## Known limits

- **The baseline check (`cmd/compat`) compares nothing until S27 merges.** It reads the inventory on `origin/main`, which does not have one yet, so it prints `nothing to compare` and passes. It starts catching an item removed from both the code and the inventory on the first PR after the merge.
- **Step 3's edit also fails tests outside the suite.** `internal/workflow/def`'s own tests use `timer` (for example `TestParseRejects/timer_without_duration`), so `task test` reports more failures than the configsurface ones. That is expected.
- Some configsurface helper tests (`TestRemovalReportsDroppedExtension`, `TestStillAcceptedReportsWorkflowFixtureThatNoLongerParses`, `TestDeprecatedItem`) also fail in step 3, because they build on the real schema. They still name `timer`.

## Last verified run

On the story branch head `69c1f116` (TICK-880, PR #690, and TICK-881, PR #691, merged), Windows 11, Go 1.26.11, 2026-11-15. The terminal output of every step is attached to the TICK-882 `RESULT: DONE` comment (`s27-kit-run.txt`).

| Step | Command | Result |
|---|---|---|
| 0 baseline | `go run ./tools/adrgate`, `go test ./internal/configsurface/...` | exit 0, both `ok` |
| 1 | `task lint` | the CI-1 line, exit 201 (task's code for a failed command); exit 0 after the restore |
| 2 | `task lint` | the CI-2 and CI-4 lines, exit 201; exit 0 after the revert |
| 3 | `go test ./internal/configsurface/...` and `task test` | `removed: workflow.step_types/timer`, FAIL. The `task test` run was stopped after the configsurface package had reported (the full run is longer than 10 minutes). |
| 4 | `task lint`, `go test -count=1 ./internal/configsurface/...` | `golangci-lint` 0 issues, every lint step passed, exit 0; both packages `ok` |
