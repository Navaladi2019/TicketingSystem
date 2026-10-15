# Technical Architect spike material (ADR-001)

These spikes were posted **inline in issue comments** on TICK-36, not as attachments. Each file below is the exact text of one code block or one `<details>` section from the comment named. Nothing was edited. Only the file names and folders were added.

| Folder | Source comment on TICK-36 | What it covers |
|---|---|---|
| `round-1/` | `01a13b74-d359-73b0-a697-a982e9e54593` (TA critique, round 1) | A1/A2 Keycloak organization claims and the `kc.org` tamper (`a1-a2/kc_spike.py`). A3 Temporal header propagation, the guard, duplicate starts and the search attribute (`a3/main.go`). Raw output and rerun steps (`spike-report-raw-output.md`). |
| `round-2/` | `01a13d33-402a-7e39-b6d2-07b776dcaf87` (TA re-check, round 2) | A7 Temporal mTLS, the custom Authorizer and worker-command inspection (`a7/server/main.go`, `a7/client/main.go`). A6 Keycloak brokered login into an organization (`a6/a6_spike.py`). Raw output (`raw-output.md`). |
| `zitadel-assessment/` | `01a13ed1-a45e-7c7c-a910-bbbb3dd2889f` (Zitadel vs Keycloak assessment) | Footprint method and raw numbers (`footprint-raw-numbers.md`) |

**These are throwaway spike programs.** The credentials in them are placeholders. Each spike's environment and rerun steps are in its raw-output file.

The Postgres & Data Modeling Specialist's spikes were attachments, and they are committed byte for byte under `../postgres/`:
- round 1, from comment `01a13b56-bdef-77de-a80b-afb08df02b99`;
- round 2, from comment `01a13d24-d15a-7267-b361-5703c14c0352`.
