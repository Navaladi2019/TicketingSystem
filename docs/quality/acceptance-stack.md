# Acceptance kits: one image tag per checkout (TICK-1021)

## Scope

How every acceptance kit builds and starts its fresh stack, so that two
checkouts on one host cannot run each other's images.

## The risk

`compose.yaml` tags every `nava/*` image `${NAVA_IMAGE_TAG:-dev}`. With the
default, every checkout on the host builds and runs the same tags. On
2026-11-20 the `nava-s21` stack (TICK-749 checkout) rebuilt `nava/nava-db:dev`
from its branch; the S30 fresh stack then ran S21's migrations (latest
`20261119013700`) without S30's format-2 CHECK, and the S30 folder failed 32 of
47 requests. Nothing in the walk said the images came from another branch.

A second trap: `docker compose build ... | tail` exits with `tail`'s status, so
a failed or killed build left the previous images in place and the kit went on.

## The rule

Every kit's README starts the stack with, from the repository root:

```sh
export NAVA_IMAGE_TAG=$(sh scripts/acceptance/image-tag.sh)
sh scripts/acceptance/fresh-stack.sh
```

and every kit's `run.sh` runs `sh scripts/acceptance/check-images.sh` as its
first step.

| Script | What it does |
|---|---|
| `scripts/acceptance/image-tag.sh` | Prints the commit (12 hex); with tracked changes or untracked files, `-dirty-` and a hash of their content, so an edit gives a new tag |
| `scripts/acceptance/fresh-stack.sh` | Refuses `NAVA_IMAGE_TAG=dev`; `docker compose build` (never piped, exit status checked, the stack is untouched if it fails); `down -v`; `up -d --wait --no-build`; then `check-images.sh` |
| `scripts/acceptance/check-images.sh` | Fails unless every container of the compose project with a `nava/*` image, exited one-shots such as `db-migrate` included, runs it under `NAVA_IMAGE_TAG` (default: this checkout's tag), and at least one does |

A second stack on the same host still needs its own `COMPOSE_PROJECT_NAME` and
host ports (`compose.yaml`, header): the project name is `nava` in every
checkout, so `docker compose down -v` in one removes the other's volumes.

## Applied to

S25 and S30 (`docs/acceptance/S25`, `docs/acceptance/S30`): the kits with a
`run.sh` that a fresh stack is rebuilt for. Older kits (S13 to S18,
`tools/run-kit.sh`) are accepted and keep their own tags (`s17pre`, `s18pre`)
where they build two versions; a re-run of one should start its stack the same
way.

## Tests

- `check-images.sh` against a fake `docker` (right tag; one `:dev`
  container; a tag that only ends with the wanted one, `:xabc` for `abc`; no
  container): exit 0, 1, 1, 1. Done by hand in the TICK-1021 pull request; no
  CI step, since these scripts change rarely and a kit run exercises them.
- `docker compose ps -a --format '{{.Service}} {{.Image}}'` prints the tagged
  reference (`nava/nava-core:s30-55902ac0`), checked on a running stack.

Deliberately not tested: that an image's content matches its tag. Two
checkouts at the same clean commit build the same tag from the same sources,
which is the point.
