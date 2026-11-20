.PHONY: bruno-cli

## Installs the pinned Bruno CLI from tools/bruno/package-lock.json (npm ci; a no-op when up to date).
# TICK-1003: the stack job runs it before the stack cases, so no case resolves
# the CLI's dependency tree from the registry; docs/acceptance/guide/tools/
# api-test.sh calls the same script, so a laptop gets the same tree. Needs npm.
bruno-cli:
	@sh tools/bruno/install.sh >/dev/null
