.PHONY: bench-records-list

## TICK-339 records list benchmark (D25): 800k records, pgbench via PgBouncer, ~140 min; not in CI
bench-records-list:
	sh bench/records-list/run.sh
