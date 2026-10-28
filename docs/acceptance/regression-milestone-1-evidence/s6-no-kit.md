# S6 (TICK-48): no acceptance kit on main

Skipped, as instructed: there is no `docs/acceptance/` kit for S6. Its rate-limit and idempotency tasks (T22a, T22b, T24) are known not to be on main. Observed API ground rules on the S5 endpoint `GET /api/v1/me` are in `cross-story.md` section 3c: problem+json errors with `instance = urn:nava:request:<X-Request-Id>`, an `X-Request-Id` on every response, a 405 with `Allow`, no RateLimit headers, and `Idempotency-Key` ignored on GET.
