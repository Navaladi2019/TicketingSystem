# S13 acceptance kit: verify an audit export offline

Story S13 (TICK-418): an auditor takes an audit export file and checks, on
their own laptop with no access to the platform, that it is complete and
unchanged. This kit (TICK-426, P5-04) gives every acceptance step a sample
file, the command to run, the exact output and the exact exit code.

Every sample here is built by the P5-02 golden-vector generator
(`internal/audit/chain/vectors`). No hash was written by hand: each tampered
file is the clean file with one byte, one line or one flag changed.
`go test ./tools/acceptance/s13kit` fails if a sample differs from what the
generator builds, or if any step's output or exit code differs from this page.

## Setup

Needs Go (the version in `go.mod`) and go-task. No database, no Docker, no
network.

```sh
git clone https://github.com/Navaladi2019/TicketingSystem.git
cd TicketingSystem
git checkout TICK-418/audit-verify-offline
task build                  # writes bin/nava (bin/nava.exe on Windows)
cd docs/acceptance/S13
```

Every command below runs from `docs/acceptance/S13`, in a POSIX shell (Git
Bash on Windows). In PowerShell, write `..\..\..\bin\nava.exe audit verify
<file>; "exit $LASTEXITCODE"` instead.

## Exit codes

`nava audit verify` exits with one code per verdict, so a script can tell
them apart. "Not verifiable: stubs omitted" (4) is not tampering (1).

| Exit | Verdict | Meaning |
|---|---|---|
| 0 | `ok` | every line verified; a redacted diff is noted, not attested |
| 1 | `tampered` | a check failed; the first failing line, its seq and a reason code are printed |
| 2 | usage | bad arguments, or the file could not be read |
| 3 | `pending` | nothing tampered, but an erasure's audit/erase entry is not sealed yet |
| 4 | `not verifiable: stubs omitted` | an honest `stubs=false` export: gaps in seq, range not proven complete |
| 5 | `empty` | no sealed entry in the export window |

## The samples

| File | Step | What it is | Exit |
|---|---|---|---|
| `export-ok.jsonl` | 1 | seq 1 to 7 from genesis, all in full | 0 |
| `export-byte-edited.jsonl` | 2 | export-ok with one byte of seq 2's `request_id` changed (`Z` to `Y`) | 1 |
| `export-line-deleted.jsonl` | 2 | export-ok with the seq 3 line removed | 1 |
| `export-first-entry-deleted.jsonl` | 2 | seq 1 removed, trailer recounted | 1 |
| `export-first-entry-deleted-reanchored.jsonl` | 2 | seq 1 removed and turned into the anchor, so every link holds | 1 |
| `export-no-anchor.jsonl` | 3 | export-ok without its anchor line | 1 |
| `export-filtered.jsonl` | 4 | `subject_type=entity_definition`: 2 entries in full, 5 stubs | 0 |
| `export-redacted.jsonl` | 5 | seq 2's diff replaced by the redaction policy (`diff_redacted: true`) | 0 |
| `export-erased-without-erase-entry.jsonl` | 5 | `subject_type=record`: seq 6 is erased, its audit/erase entry (seq 7) is only a stub | 1 |
| `export-erased-pending.jsonl` | extra | seq 6 erased; its audit/erase entry is written but not sealed | 3 |
| `export-truncated-honest.jsonl` | extra | seq 5's diff was cut at 64 KB and says so (`diff_truncated: true`, ADR-004-E1) | 0 |
| `export-truncated-flag-cleared.jsonl` | extra | the same file with `diff_truncated` set back to `false` to hide the cut | 1 |
| `export-stubs-omitted.jsonl` | extra | `subject_type=entity_definition` exported with `stubs=false`, honestly flagged | 4 |
| `export-empty.jsonl` | extra | a time window with no sealed entry | 5 |

## Step 1: a clean export verifies

Prints `ok` with the anchor (genesis), the last seq (7) and the head hash.

```console
$ ../../../bin/nava audit verify export-ok.jsonl; echo "exit $?"
ok: seq 1..7 from genesis to head_hash aa38d9343733e5216513605b244b0a796fab0da11abc743eca96b7ee9040a551; 7 entries, 0 stubs, 0 unsealed
exit 0
```

## Step 2: one byte changed, a line deleted, the first entry deleted

Each fails with the line that broke (its line number in the file and its
seq) and exits 1.

One byte changed in seq 2's envelope: its `envelope_sha256` no longer matches.

```console
$ ../../../bin/nava audit verify export-byte-edited.jsonl; echo "exit $?"
tampered: line 4 (seq 2): envelope_sha256_mismatch: envelope_sha256 is not SHA-256(JCS(envelope))
exit 1
```

The seq 3 line deleted: the chain jumps from seq 2 to seq 4.

```console
$ ../../../bin/nava audit verify export-line-deleted.jsonl; echo "exit $?"
tampered: line 5 (seq 4): seq_missing: seq 3..3 missing from a stubs=true file
exit 1
```

The first entry of the range (seq 1) deleted and the trailer recounted: the
line after the anchor is seq 2.

```console
$ ../../../bin/nava audit verify export-first-entry-deleted.jsonl; echo "exit $?"
tampered: line 3 (seq 2): seq_missing: seq 1..1 missing from a stubs=true file
exit 1
```

The careful forgery: seq 1 deleted and turned into the anchor, so every
remaining link and count holds. The header filter (no `from`, no
`after_seq`) says the range starts at genesis, so the anchor is in the wrong
place.

```console
$ ../../../bin/nava audit verify export-first-entry-deleted-reanchored.jsonl; echo "exit $?"
tampered: line 2 (seq 1): anchor_mismatch: the header filter puts the anchor at seq 0 (R20: first_seq - 1)
exit 1
```

## Step 3: a file whose first chain line is not the anchor is rejected

```console
$ ../../../bin/nava audit verify export-no-anchor.jsonl; echo "exit $?"
tampered: line 2 (seq 1): anchor_missing: the first chain line must be the anchor at first_seq - 1 (R20)
exit 1
```

## Step 4: a filtered file verifies

One subject type in full, every other entry as a stub line. The chain is
checked through the stubs, so the result is `ok` over the whole range.

```console
$ ../../../bin/nava audit verify export-filtered.jsonl; echo "exit $?"
ok: seq 1..7 from genesis to head_hash aa38d9343733e5216513605b244b0a796fab0da11abc743eca96b7ee9040a551; 2 entries, 5 stubs, 0 unsealed
exit 0
```

## Step 5: redacted and erased entries

A redacted entry verifies (its line is in the chain), but its diff text is
not attested, and the output says so.

```console
$ ../../../bin/nava audit verify export-redacted.jsonl; echo "exit $?"
ok: seq 1..7 from genesis to head_hash aa38d9343733e5216513605b244b0a796fab0da11abc743eca96b7ee9040a551; 7 entries, 0 stubs, 0 unsealed
  seq 2: diff not verifiable: redacted
exit 0
```

An erased entry whose audit/erase entry is not in the file in full: nothing
proves the erasure was authorised, so it is `tampered` (ADR-004 R24).

```console
$ ../../../bin/nava audit verify export-erased-without-erase-entry.jsonl; echo "exit $?"
tampered: line 8 (seq 6): erasure_unlisted: erased, but no chain-valid audit/erase entry for its erasure_ref lists its id (R24)
exit 1
```

## Beyond the five steps

The other verdicts a human can meet, each with its own exit code.

An erasure whose audit/erase entry is written but not sealed yet. Nothing is
tampered; export again after the sealer runs.

```console
$ ../../../bin/nava audit verify export-erased-pending.jsonl; echo "exit $?"
pending: seq 1..6 from genesis to head_hash 1e046654c186772c3adf39fcbdcb8c6429929d768560c808c2ecc70f1f973811; 6 entries, 0 stubs, 1 unsealed
  seq 6: pending: erased; its audit/erase entry is not sealed yet
exit 3
```

The `diff_truncated` pair (ADR-004-E1). A diff over 64 KB is stored cut at
whole operations, with `diff_truncated: true` in the hashed envelope. The
honest file verifies:

```console
$ ../../../bin/nava audit verify export-truncated-honest.jsonl; echo "exit $?"
ok: seq 1..7 from genesis to head_hash aa38d9343733e5216513605b244b0a796fab0da11abc743eca96b7ee9040a551; 1 entries, 6 stubs, 0 unsealed
exit 0
```

The same file with `diff_truncated` set back to `false`, to pass the cut
diff off as complete. The flag is inside the hashed envelope, so the line
breaks:

```console
$ ../../../bin/nava audit verify export-truncated-flag-cleared.jsonl; echo "exit $?"
tampered: line 7 (seq 5): envelope_sha256_mismatch: envelope_sha256 is not SHA-256(JCS(envelope))
exit 1
```

An export made with `stubs=false`: the filtered-out entries are missing, not
stubbed, and the file says so. The lines present are checked, but the range
cannot be proven complete. This is exit 4, not exit 1: nothing is tampered.

```console
$ ../../../bin/nava audit verify export-stubs-omitted.jsonl; echo "exit $?"
not verifiable: stubs omitted (1 gaps in seq; lines present checked: 2 entries, 0 unsealed)
exit 4
```

A time window with no sealed entry. Nothing to verify, nothing wrong.

```console
$ ../../../bin/nava audit verify export-empty.jsonl; echo "exit $?"
empty: no sealed entry in the export window (0 unsealed)
exit 5
```

## Run every step at once

```sh
sh scripts/acceptance/s13.sh           # from the repository root, after task build
go test ./tools/acceptance/s13kit      # the same, plus "samples match the generator"
```

`scripts/acceptance/s13.sh` runs `bin/nava` on every sample and compares
stdout and the exit code with `expected/<sample>.txt` (first line
`exit N`, then the exact stdout). It prints `PASS` or `FAIL` per sample and
exits non-zero on any mismatch.

To rebuild the samples after a change to the generator (never edit one by
hand):

```sh
go test ./tools/acceptance/s13kit -run TestSamplesAreCurrent -update
```

## Known limitations

These are the limits `nava audit verify -h` lists (audit-export-v1.md §4
step 9, ADR-004 R-A1, AR-TICK-409-01). The kit does not claim to catch them:

- A full entry replaced by its own stub still verifies: filter completeness is
  not attested.
- A file rewritten consistently as a whole, entries cut off the end with
  `head_hash` rewritten, or a time-window export whose first entries were cut
  and re-anchored still verifies. Nothing authenticates the file itself.
  Compare an overlapping seq's `entry_hash` with an export you already hold.
- JSONL only. The CSV export (R22) is not read yet.
