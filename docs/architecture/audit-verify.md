# Offline audit verifier: `nava audit verify`

P5-03 (TICK-425), ADR-004 R20, R22, R24 and R7, PLAN-005 C4 and D-5-6.
Package `internal/audit/verify`, command `cmd/nava`. The file format and the
check list are in [audit-export-v1.md](../api/audit-export-v1.md) §3 and §4.
Hashes come from `internal/audit/chain` (P5-02).

```sh
task build                                   # bin/nava and bin/nava-core
bin/nava audit verify export.jsonl; echo $?  # or - for stdin
bin/nava audit verify export.csv; echo $?    # CSV (R22), TICK-463
```

## Packages

```mermaid
flowchart LR
  CLI["cmd/nava<br/>args, output, exit codes"] --> V["internal/audit/verify<br/>Verify(io.Reader) → Report"]
  V --> C["internal/audit/chain<br/>JCS, envelope parse, hashes"]
  T["verify/verifytest (tests only)<br/>fixtures from the vector generator"] --> G["internal/audit/chain/vectors"]
```

`arch_test.go` fails if `cmd/nava` or `internal/audit/verify` links
`net/http`, `database/sql`, pgx or `internal/db`, or any platform package
other than `chain` and `verify`.

## One pass over the file

`Verify` streams the file once. Memory is bounded by one line (4 MiB) plus
the erasure bookkeeping.

```mermaid
flowchart TD
  H["header: strict, format v1, tenant_id"] --> A{"next line"}
  A -->|anchor| AN["genesis, or stub at first_seq − 1;<br/>after_seq / unbounded window fixes the seq"]
  A -->|stub or sealed entry| X1["tampered: anchor_missing"]
  A -->|unsealed or trailer| E["empty range"]
  AN --> R["range line: seq = prev + 1, prev_hash link,<br/>tenant, envelope_sha256, diff state, entry_hash"]
  R --> R
  R --> U["unsealed lines: tenant; an audit/erase<br/>with a matching diff is recorded as pending"]
  R --> TR
  U --> TR["trailer: counts, anchor_seq, head (step 6),<br/>erasures (R24), flags vs computed verifiability"]
  TR --> O["pass · pending · unverifiable · empty"]
```

The first failing check stops the pass and becomes `Report.Failure`: line
number, seq and reason code.

## Outcomes and exit codes

| Outcome | Exit | When |
|---|---|---|
| `pass` | 0 | every check holds. Redacted lines are `diff_unverified`: printed as "diff not verifiable: redacted". |
| `tampered` | 1 | the first failing check, with its reason code |
| (usage) | 2 | bad arguments, unreadable file |
| `pending` | 3 | an erased entry is listed only by an unsealed `audit/erase` entry (R24) |
| `unverifiable` | 4 | "not verifiable: stubs omitted": a `stubs=false` file, honestly flagged |
| `empty` | 5 | no sealed entry in the window |

Order when several apply: tampered, then unverifiable, then pending.

## Choices worth knowing

- **Seq gaps.** In a `stubs: true` file a gap is `seq_missing` at the line
  after it. In a `stubs: false` file gaps are expected, and the file ends
  `unverifiable` if its flags say so.
- **Anchor position.** When the header filter fixes `first_seq`
  (`after_seq = n` gives anchor `n`; no `from` gives genesis), a different
  anchor is `anchor_mismatch`. That catches a forger who deletes the first
  entry and re-anchors on it. A time-window export (`from` set) cannot be
  checked this way offline; `nava audit verify -h` lists this with the other
  §4 step 9 limits.
- **Erase entries** list an id only if their diff is the one R23 shape and
  their diff hash verifies. A redacted erase diff lists nothing.
- **An unsealed line's** `envelope_sha256` is informational (§3) and is not
  compared.
- **Formats.** JSONL and CSV (§5, R22; TICK-463). See below.

## CSV

`Verify` reads a file as CSV when it starts with the column row (`kind,`);
a JSONL line starts with `{`. `Report.Encoding` says which, and the CLI
names a CSV failure by row (the column row is row 1) instead of by line.

```mermaid
flowchart LR
  R["csvReader: strict RFC 4180,<br/>CRLF rows, quoted bytes verbatim"] --> P["drop one leading ' per cell<br/>(formula prefix)"]
  P --> L["rowLine: rebuild the JSONL line;<br/>empty cell = null, envelope re-nested"]
  L --> V["verifier.line: the JSONL checks"]
```

- **Same checks.** Each row becomes the JSONL line it stands for and goes
  through the same code, so outcomes, reason codes and exit codes match. The
  JSONL tests run on both encodings.
- **Own reader.** `encoding/csv` turns CRLF inside a quoted cell into LF; the
  diff cell's bytes are hashed, and §5 forbids that. The reader also rejects
  LF row ends, a bare CR, a `"` in an unquoted cell and rows over 4 MiB.
- **Strict rows.** The column row must be exact; a row has 37 cells; a cell
  in a column its kind does not use must be empty; integer and boolean cells
  must be literals (no way to inject JSON); `header` and `trailer` carry their
  JSONL line in `meta`, whose `kind` must match the row's.
- **Formula prefix.** The reader drops exactly one leading `'` (§5). It does
  not reject a dangerous cell the writer left unprefixed: the prefix is not
  hashed, and CI-A15 tests the writer.
- **Fixtures** in `internal/audit/verify/testdata/csv` are built from the
  golden export vectors by `verifytest.CSV`; rebuild them with
  `go test ./internal/audit/verify -run TestCSVFixturesAreCurrent -update`.
