# Acceptance guide: stories S1–S7 and the API test suite (TICK-287, TICK-331)

A hands-on test kit for everything stories S1 to S7 delivered on `main`, plus a
Bruno test suite for every operation in `api/openapi.bundled.yaml`.

- **Open `index.html`** in a browser (offline is fine), or the single-file `guide-standalone.html`.
- **Bruno:** Bruno › **Open Collection** › the `bruno/` folder; choose **Developer Mode** when asked for the sandbox; select the `local` environment. Folders: S1–S7 (the stories), X1–X3 (the API suite: `getMe`, `onboardTenant`, every admin listener) and Z (expired tokens, runs last).
- **Headless run** (from the repository root, Git Bash on Windows; needs Node.js, verified with 22):
  - `sh docs/acceptance/guide/tools/api-test.sh`: the whole collection with the pinned Bruno CLI (`@usebruno/cli@4.3.0` through `npx`).
  - `sh docs/acceptance/guide/tools/api-test.sh S4`: one folder. Extra arguments go to `bru run` (for example `--env-var core_url=http://127.0.0.1:28080`).
  - Exits non-zero on any failed check. Reports (JUnit, HTML, JSON) go to `docs/acceptance/guide/reports/`, which is git-ignored.
- **Download:** `ticketing-acceptance-guide.zip` holds the site, the Bruno collection and the helper scripts.
- **Helper scripts** (run from the repository root):
  - `sh docs/acceptance/guide/tools/seed.sh`: seeds tenants A and B and runs one group sync.
  - `sh docs/acceptance/guide/tools/tunnels.sh up|down|status`: local-only forwards to nava-onboard, every service's admin listener and the Temporal UI, which publish no port.
- **Last verified run:** [`acceptance-run.md`](acceptance-run.md).

## Changing the guide or the suite

Every HTML page, everything under `bruno/` and the zip are generated (one
source). Edit `build/content.py` (pages) or `build/collection.py` (requests,
tests and the shared helpers in `bruno/lib/nava.js`), then:

```sh
python -I docs/acceptance/guide/build/build.py
```

Commit the sources and the generated files together. The pages use no script
and no network: inline CSS and inline SVG flowcharts only.

**Coverage check.** `build.py` and the Go test `TestBrunoCoversEveryOperation`
(`api/bruno_coverage_test.go`, part of `task test` and CI) fail when an
operation in `api/openapi.bundled.yaml` has no Bruno request, or when a status
it documents is asserted by no request and has no reason in
`build/coverage-exempt.json`. A new endpoint therefore needs requests in
`build/collection.py` before CI passes.

Only the documented dev-only seed credentials appear here. Tokens are fetched at
run time, kept in Bruno runtime variables (`bru.setVar`) and never written to a
file. The run reports do hold the run's short-lived tokens in response bodies:
keep them local.
