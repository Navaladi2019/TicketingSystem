# ADR-007 spike S1 (TICK-806)

Host-client token through the unchanged `keycloak.Adapter.Resolve`, plus a top-level sign-in with no prompt. The verdict and the evidence are in [`docs/adr/ADR-007/spikes/s1-host-client-token.md`](../../docs/adr/ADR-007/spikes/s1-host-client-token.md).

| File | What |
| --- | --- |
| `realm/spike-s1.json` | Scratch realm: public PKCE client `spike-host`, scopes `organization` (optional), `nava-api-audience` and `nava-idp`, orgs org-a, org-b and org-c, a single-org user and a two-org user |
| `realm/spike-upstream.json` | The brokered IdP realm (dev-only scratch credentials) |
| `run-keycloak.sh` | Runs the pinned `keycloak:26.8.1` in `start-dev` on `localhost:18181`. Not the compose stack |
| `s1.mjs` | Playwright driver (Chromium with third-party cookies blocked, and WebKit): host page, PKCE, prompt detection; writes `out/` |
| `resolve_test.go` | `-tags spike`: feeds every captured token to `Resolve` |

Needs Docker, Node 22 and Go. Not in CI. `out/` holds raw tokens and is gitignored.
