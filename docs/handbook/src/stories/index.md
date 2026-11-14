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

- [S9: An agent logs an incident, updates it, sees its history; a stale update is refused](s9.md)
- [S10: A service-desk agent sees only their group's incidents; masked fields stay hidden](s10.md)
- [S11a: The workflow interpreter runs a definition on Temporal](s11a.md)
- [S13: An auditor verifies an audit export offline, and any tampering is detected](s13.md)
- [S14: Every record and entity-type change lands in one audit log](s14.md)
- [S15: Each tenant's audit entries are sealed into a hash chain](s15.md)
- [S16: An auditor exports a filtered audit log and proves it complete](s16.md)
- [S17: Record numbers keep counting while the counter table moves](s17.md)
- [S25: A developer installs the TypeScript SDK and lists, reads and creates records](s25.md)

S17 and S25 have pages whose verdict and date are pending until the story verdict is posted; S25 is still on its story branch. S11b (in progress) and S12, S18 to S20 (planned) have no page: they are not on `main`. The table above lists them apart from the delivered stories.
