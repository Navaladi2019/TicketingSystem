---
title: "prometheus"
section: system
status: delivered
sources: [compose.yaml, deploy/compose/prometheus/prometheus.yml, deploy/compose/prometheus/rules/records.yml, deploy/compose/prometheus/tests/records_test.yml]
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
| Introduced by | commit `4a9f3788` (names no ticket) |
| Alert rules | every file in `deploy/compose/prometheus/rules/` (`audit.yml`, `authz.yml`, `groupsync.yml`, `numbering.yml`, `records.yml`); tests in `deploy/compose/prometheus/tests/`, run by `lineage.TestAlertRules` |

## Alert rules

The rules and what to do when one fires are in [traces, logs and metrics](../operations/observability.md#metrics-and-alerts). On the S21 story branch, `records.yml` adds `RecordsListTotalUnknown` (ADR-006-A2 A2-7, TICK-1051): more than 5% of field-index lists in the last hour answered `total_state="unknown"`, read from the `total_state` label of `nava_records_list_indexed_duration_seconds`.

## Boundary it enforces

No ADR-001 section 1 invariant. It reads the admin listeners over `internal`, so they never need publishing: `TestComposePublishesNoAdminPort`.

[Back to the system map](index.md)
