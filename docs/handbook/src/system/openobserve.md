---
title: "openobserve"
section: system
status: delivered
sources: [compose.yaml, docs/pins.md]
tests: [TestComposePublishesOnlyAllowedUIs, TestOperatorServicesNeedTheProfile]
---

Local trace and log search: shows what the Collector forwards. Local development only, in the `operator` profile.

| | |
|---|---|
| Image | `public.ecr.aws/zinclabs/openobserve:v1.1.2-debug` (pin in `docs/pins.md`) |
| Networks | `internal` (so the Collector reaches it), `operator` |
| Listener | `:5080` (UI and OTLP/HTTP ingest) |
| Host port | `127.0.0.1:5080` (`OPENOBSERVE_PORT`) |
| Data | volume `openobserve-data` |
| DB role | none |
| Introduced by | TICK-634 (replaced the earlier trace UI) |

## Boundary it enforces

No ADR-001 section 1 invariant. Its boundary is the publishing rule: only allow-listed UIs publish, and only on loopback. `TestComposePublishesOnlyAllowedUIs` proves it.

[Back to the system map](index.md)
