---
title: "prometheus"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/prometheus/prometheus.yml]
tests: [TestComposePublishesNoAdminPort, TestComposePublishesOnlyAllowedUIs]
---

Local metrics and alert rules. Local development only, in the `operator` profile; no Alertmanager, so a firing alert shows on its Alerts page.

| | |
|---|---|
| Image | `prom/prometheus:v3.7.3` |
| Networks | `internal`, `operator` |
| Listener | `:9091` |
| Host port | `127.0.0.1:9091` (`PROMETHEUS_PORT`) |
| Scrapes | `/metrics` on `:9090` of nava-core, nava-worker, nava-relay, nava-groupsync, nava-purge and nava-onboard, every 15 s; one-shot jobs are not scraped |
| DB role | none |
| Introduced by | TICK-330 |

## Boundary it enforces

No ADR-001 section 1 invariant. It reads the admin listeners over `internal`, so they never need publishing: `TestComposePublishesNoAdminPort`.

[Back to the system map](index.md)
