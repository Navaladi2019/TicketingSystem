## 2026-11-25: TICK-1128, CI pulls OpenObserve by one digest from ECR Public, then Docker Hub; our own copy is deferred (`AR-TICK-1128-01`)

- **Decided by:** the Technical Lead on TICK-1128, after the Quality, Testing & CI Advisor found that a copy of our own would publish AGPL software. Shipped in PR #878.
- **Decision:** a full CI run pulls OpenObserve from `public.ecr.aws/zinclabs/openobserve` and, when that fails, from `docker.io/openobserve/openobserve`. Both are held to the one index digest in `compose.yaml`. Steps and tests: [CI pipeline, "OpenObserve image"](../../quality/ci-pipeline.md).
- **Why not our own registry:** the image is AGPL-3.0, so whoever publishes a copy must keep its source available. We expect a copy on ghcr.io to be public, as our other packages there are, and GitHub does not let a public package become private again. Docker Hub already serves the other images of the stack, so it adds no vendor.
- **Accepted risk (Low):** a full run fails if both registries refuse (`AR-TICK-1128-01`).
- **Limit of the digest:** it fixes the bytes we pinned. It does not prove who built them; nobody looked for an upstream signature. This is from the Security & Multi-Tenancy Reviewer's review of PR #878 (TICK-1128, 2026-11-25 11:20 UTC).
- **Deferred:** our own copy. **Owner:** the Technical Lead. **Trigger:** one full run red because both registries refused, or Docker Hub limits turn a full run red on any image. Publishing a copy needs the human first.
- **Dissent:** none.
