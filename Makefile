# Tag of the nava/* images compose.yaml builds; `make ci-isolated` sets one per run.
export NAVA_IMAGE_TAG ?= dev

# Shared variables. The targets live in make/*.mk, one fragment per subsystem
# or story suite: a story that adds make targets adds a new make/<name>.mk and
# does not edit this file (docs/delivery/merge-safety.md).
GOLANGCI_LINT ?= golangci-lint
# docs/pins.md, "Toolchain and CI tools".
GOLANGCI_LINT_VERSION := 2.14.0
GOVULNCHECK_VERSION := v1.9.0

# The base ref of the change: check-openapi, ci-changed and review-paths.
BASE ?= origin/main

.DEFAULT_GOAL := help
.PHONY: help

## Lists the targets that have a ## comment above them.
help:
	@awk '/^## /{d=substr($$0,4);next} /^[A-Za-z0-9_.-]+:/{if(d!=""){split($$0,a,":");printf "%-28s %s\n",a[1],d}} {d=""}' $(MAKEFILE_LIST)

include make/*.mk
