.PHONY: stack-test-worker stack-test-interpreter

# The nava-worker service (T31b, TICK-159): ready with /readyz green, /metrics
# and the probes on its unpublished admin listener without httpobs, and a
# workflow's workflow_ref activity run in the service through the guard, which
# /metrics counts. The interpreter cases (TICK-367) start
# nava-worker-interptest (compose.test.yaml, profile interptest) for their own
# test only and remove it when it ends.
stack-test-worker:
	go test -tags workerstack -count=1 ./internal/workerstack

# The interpreter crash case (P3-24, TICK-351): nava-worker-interptest is
# killed after ApplyStep's write committed and before it completed, then
# restarted; the retry applies the effect once, read from the
# interptest-state volume, outside the killed process. Behind the
# interpcrash tag, so stack-test-worker never compiles it. The recipe is
# explicit: make does not apply stack-test-% to a .PHONY target.
stack-test-interpreter:
	go test -tags workerstack,interpcrash -count=1 -timeout 10m -run '^TestInterpreterCrash' -v ./internal/workerstack
