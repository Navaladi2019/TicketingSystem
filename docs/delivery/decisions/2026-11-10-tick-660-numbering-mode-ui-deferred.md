## 2026-11-10: TICK-660 ships the numbering_mode API only; the type admin UI control is deferred

- **Decided by:** the Technical Lead, on TICK-660 (P3-36g, PLAN-003 D-3-9).
- **Decision:** a type's `numbering_mode` (`gaps` default, `gapless`) is set on create, and changed through `POST /api/v1/entity-types/{key}/versions`. That body takes `numbering_mode` with or instead of `definition`. A mode-only call publishes no version and returns the current one. There is no PATCH route.
- **Cut:** the type admin UI control (help text "Gaps allowed (faster under load): a failed create skips a number" / "Gapless: every number is used; creates of this type queue under load") and its UI test. This repo has no frontend.
- **Trigger:** the type admin UI is built (Epic 26, TICK-29, platform console). That work adds the control, the help text and a UI test against the API above.
- **Follow-up:** TICK-687 fixes the publish description text, the comments and the rollback test from the PR #560 review.
- **Dissent:** none.
