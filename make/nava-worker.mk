.PHONY: stack-test-worker

# The nava-worker service (T31b, TICK-159): ready with /readyz green, /metrics
# and the probes on its unpublished admin listener without httpobs, and a
# workflow's workflow_ref activity run in the service through the guard, which
# /metrics counts.
stack-test-worker:
	go test -tags workerstack -count=1 ./internal/workerstack

