.PHONY: demo-outbox stack-test-demo_outbox

# The S7 demo (TICK-275): a record of tenant-a created and updated twice through
# the write pipeline, one outbox row per write, then handed to Temporal by the
# relay. Needs the stack up (make stack-up). DEDUPE_KEY names the rows.
DEDUPE_KEY ?= demo-outbox
demo-outbox:
	DEDUPE_KEY=$(DEDUPE_KEY) sh scripts/dev/demo-outbox.sh

# Runs make demo-outbox twice with one key and asserts one workflow whose
# history holds the two signals in commit order (internal/temporalstack,
# TestDemoOutbox). Behind the demooutbox tag, so stack-test-sdk's whole-package run (and its report gate) never compiles it.
stack-test-demo_outbox:
	go test -tags temporalstack,demooutbox -count=1 -timeout 5m -run '^TestDemoOutbox' ./internal/temporalstack

