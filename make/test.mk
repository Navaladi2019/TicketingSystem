.PHONY: test

ci-unit: test

# The navalint analyzers are their own module (tools/navalint), so ./...
# does not reach them.
# -trimpath keeps the checkout path out of the build and test cache keys, so a
# new checkout reuses the cached results (TICK-153 O5). Not for navalint: it
# makes its analysistest runs slower. racelock: one go test -race at a time on
# the host, across every checkout and agent (TICK-153 D3); NAVA_RACE_LOCK=off
# skips it.
# suiteran (TICK-230) fails a go test -json run in which a registered
# isolation suite (review-paths.yaml, cto_signoff.isolation_tests) ran zero
# tests; its -tags are the tags the step means to test with, so keep them
# beside go test's own.
test:
	go run ./tools/ci/racelock -- go run ./tools/ci/suiteran -- go test -json -race -trimpath ./...
	go run ./tools/ci/racelock -- go -C tools/navalint test -race ./...

