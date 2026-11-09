.PHONY: bench-records-list

## TICK-339 records list benchmark (D25): 800k records, pgbench via PgBouncer, ~140 min; not in CI
bench-records-list:
	sh bench/records-list/run.sh

.PHONY: bench-acl-perf

## TICK-621 P3-36: list and single-read p95 at 800k through the real handlers (compiled ACL SQL), ~20 min; not in CI
bench-acl-perf:
	sh bench/acl-perf/run.sh
