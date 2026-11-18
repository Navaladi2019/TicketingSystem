# RFC-021 round 1: real-pipeline backfill spike (Technical Architect)

Source: TICK-897, comment `01a1df34-8e90-7f3a-8b82-9e2961308cef` (2026-11-15), archived verbatim.

- `zz_spike_rfc021_integration_test.go.txt`: the throwaway spike. It is stored as `.txt` so `go build ./...` and the linters ignore it. To rerun it, copy it to `internal/records/pipeline/zz_spike_rfc021_integration_test.go` (never commit it there).
- `raw-results.txt`: the raw output.

**Setup:** `main` `ecedb2e9` plus PR #711 `5aa2973f`. `postgres:16.10`, stock configuration. Docker Desktop/WSL2 with 16 vCPU. Real `pipeline.Write` (`OpUpdate`, empty patch, `SourceSystem`, `NoopSink`) as `nava_worker`. 4 live writers at a 20 ms cadence.

**How to rerun:** `CGO_ENABLED=0 GOOS=linux go test -c -tags integration -o spike.test ./internal/records/pipeline/`. Run the binary in a container on the same Docker network as Postgres, with `NAVA_TEST_PG_URL` set. Options:
- `SPIKE_N` (default 100000);
- `SPIKE_BATCH` (default 100);
- `SPIKE_PAUSE_MS`;
- `SPIKE_GRANT=1`;
- `SPIKE_NOSP=1`;
- `SPIKE_PERREC=1`;
- `SPIKE_NOLIVE=1`;
- `SPIKE_PAG=1`.

Then run `-test.run TestSpikeRFC021 -test.v`.

**Findings it backs (ADR-009):**

| Finding | Result | Backs |
|---|---|---|
| D8 `FOR SHARE` as `nava_worker` | `42501` (with a grant but no policy: 0 rows, no error) | D8 |
| Batch row lock, live p95 | +416% at 100k, +28% at 20k | rejected option |
| One transaction per record, live p95 | +1.7% at 20k | D7 |
| Rewriting `updated_at` | a paging client missed 10% of records | D5 |
| Savepoints | subxact cache overflow at a batch of 100 | D7 |
