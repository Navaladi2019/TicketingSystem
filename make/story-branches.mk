.PHONY: check-story-branches story-branches-issue

# Fails when a story branch's work is not on main (TICK-322): changes merged or
# pushed into a story branch after its story PR merged, a story branch with no
# PR out of it whose story closed.yaml lists as done or cancelled (TICK-379),
# or an open PR into a merged story branch. Reads the remote and
# every PR, so it needs gh and GH_TOKEN. Writes story-branches.md for the issue.
# "Story branches" in docs/quality/ci-pipeline.md has the rules. Not in ci-lint:
# the ci workflow runs it as a job of its own (story-branches).
check-story-branches:
	go run ./tools/ci/storybranches -report story-branches.md

# Opens, updates or closes the one "Story branches out of step with main" issue
# from story-branches.md (RESULT=success|failure, RUN_URL; the ci job sets them).
story-branches-issue:
	sh scripts/ci/story-branches-issue.sh
