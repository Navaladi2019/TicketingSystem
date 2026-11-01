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
