---
title: "Stories"
section: stories
status: delivered
---

Every story, with its status on `main`, the verdict its acceptance kit
records, and the date. The table is generated from
`docs/handbook/src/stories/stories.yaml`. Only `delivered` stories are on
`main`; `in-progress` stories live on a story branch and `planned` ones have
no branch yet.

<!-- gen:stories -->

## Story pages

Each page says what a user can now do, shows the flow, lists the API
operations, says how to try it (Bruno requests, `seed.sh`, `tunnels.sh`) and
links the verdict and evidence. The step-by-step kit stays in the
[acceptance guide](../../../acceptance/guide/index.html).

- [S1: Local stack starts healthy and CI gates every PR](s1.md)
- [S2: A query without the right tenant returns nothing](s2.md)
- [S3: A tenant user signs in and is scoped to their tenant only](s3.md)
- [S4: An operator onboards a tenant and its admin signs in](s4.md)
- [S5: Group changes and deprovisioning take effect within 10 minutes](s5.md)
- [S6: API ground rules on the first endpoints](s6.md)
- [S7: A committed outbox row runs exactly one tenant-scoped workflow](s7.md)
- [S8: A tenant admin publishes an incident type and reads every version](s8.md)

Pages for S9 and later are not written yet.
