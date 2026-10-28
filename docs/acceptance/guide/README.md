# Acceptance guide: stories S1–S7 (TICK-287)

A hands-on test kit for everything stories S1 to S7 delivered on `main`.

- **Open `index.html`** in a browser (offline is fine), or the single-file `guide-standalone.html`.
- **Postman:** import `postman/TicketingSystem.postman_collection.json` and `postman/local.postman_environment.json`.
- **Download:** `ticketing-acceptance-guide.zip` holds the site, the collection, the environment and the two scripts.
- **Helper scripts** (run from the repository root, Git Bash on Windows):
  - `sh docs/acceptance/guide/tools/seed.sh`: seeds tenants A and B and runs one group sync.
  - `sh docs/acceptance/guide/tools/tunnels.sh up|down|status`: local-only forwards to nava-onboard, nava-core's admin listener and the Temporal UI, which publish no port.
- **Last verified run:** [`acceptance-run.md`](acceptance-run.md).

## Changing the guide

Every HTML page, the Postman files and the zip are generated. Edit
`build/content.py` (pages) or `build/collection.py` (requests and tests), then:

```sh
python -I docs/acceptance/guide/build/build.py
```

Commit the sources and the generated files together. The pages use no script
and no network: inline CSS and inline SVG flowcharts only.

Only the documented dev-only seed credentials appear here. Tokens are fetched at
run time and never stored in the files.
