.PHONY: workflow-record-histories

# Records the nava.interpreter histories the replay gate replays (P3-23,
# TICK-350) into internal/workflow/testdata/histories/. A developer target,
# never a CI step: it starts a Temporal dev server (testsuite.StartDevServer
# downloads the temporal CLI once; NAVA_TEMPORAL_CLI=<path> uses one you have)
# and runs the scenarios in internal/workflow/interp/record_histories_test.go.
# It writes only the missing files: committed histories are append-only
# (TestHistoriesAppendOnly). NAVA_RECORD_OVERWRITE=1 overwrites, for a history
# main does not have yet. The replay itself is an ordinary unit test under
# make test.
workflow-record-histories:
	go test -tags recordhistories -run '^TestRecordHistories$$' -count=1 -v -timeout 15m ./internal/workflow/interp

# The S11a demo (TICK-352, docs/acceptance/s11a-workflow-interpreter.md): one
# nava.interpreter run of fixture-v1 on the stack, started and signalled
# through outbox rows (cmd/demo-interpreter, as nava_app) and run by
# nava-worker-interptest with the S11a fakes, while the always-on nava-worker
# is stopped. Needs the stack up (make stack-up). Every target is one action of
# scripts/dev/demo-interpreter.sh; its header lists the variables.
#   make demo-interpreter-up [PAUSE=1]      fixtures, swap the workers
#   make demo-interpreter TENANT=a KEY=k PRIORITY=high   start a run
#   make demo-interpreter-approve TENANT=a KEY=k         send `approve`
#   make demo-interpreter-publish-v2 TENANT=a            publish fixture v2
#   make demo-interpreter-pinning TENANT=a KEY=k         start, publish v2 during its timer
#   make demo-interpreter-crash TENANT=a KEY=k           approve, kill, restart (PAUSE=1)
#   make demo-interpreter-writes / -status / -down
.PHONY: demo-interpreter demo-interpreter-pinning demo-interpreter-up demo-interpreter-approve demo-interpreter-publish-v2 \
	demo-interpreter-crash demo-interpreter-writes demo-interpreter-status demo-interpreter-down
DEMO_INTERPRETER = TENANT=$(TENANT) KEY=$(KEY) PRIORITY=$(PRIORITY) RECORD_ID=$(RECORD_ID) PAUSE=$(PAUSE) sh scripts/dev/demo-interpreter.sh
TENANT ?= a
PRIORITY ?= high

## S11a: starts a nava.interpreter run of the tenant's current definition (TENANT KEY PRIORITY).
demo-interpreter:
	$(DEMO_INTERPRETER) start
## S11a: installs fixture v1 and puts nava-worker-interptest in place of nava-worker (PAUSE=1 for the crash drill).
demo-interpreter-up:
	$(DEMO_INTERPRETER) up
## S11a: sends the approve signal to the newest open run on the record (TENANT KEY).
demo-interpreter-approve:
	$(DEMO_INTERPRETER) approve
## S11a: publishes fixture v2 into the tenant's fixture store; runs in flight keep v1.
demo-interpreter-publish-v2:
	$(DEMO_INTERPRETER) publish-v2
## S11a: approve, then SIGKILL the worker after update_record committed, and restart it.
demo-interpreter-crash:
	$(DEMO_INTERPRETER) crash
## S11a: prints the fake writer's calls and applied effects.
demo-interpreter-writes:
	$(DEMO_INTERPRETER) writes
## S11a: prints the record's outbox rows and workflow_ref (TENANT KEY).
demo-interpreter-status:
	$(DEMO_INTERPRETER) status
## S11a: removes nava-worker-interptest and starts nava-worker again.
demo-interpreter-down:
	$(DEMO_INTERPRETER) down
## S11a: starts a run and publishes fixture v2 while it waits on its timer (TENANT KEY PRIORITY).
demo-interpreter-pinning:
	$(DEMO_INTERPRETER) pinning
