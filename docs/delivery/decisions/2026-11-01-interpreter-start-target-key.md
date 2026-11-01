## 2026-11-01: D-3-13 amended: an interpreter start uses the existing outbox start key `nava.interpreter:<outbox_id>`
- **Decided by:** Technical Lead, 2026-11-01, on TICK-367 (P3-22b).
- **Decision:** the relay's `target_key` for a `nava.interpreter` start is the existing outbox start form `nava.interpreter:<outbox_id>` (`outbox.Enqueue`, `relay.Claimed.IsStart`), not `interp:<uuidv5(outbox_id, definition_id)>`. The outbox id is unique per row, so a workflow id never repeats under `REJECT_DUPLICATE`. Relay and outbox code are unchanged. If P3-42 needs a key derived from the definition, that is its own Security-reviewed task.
- **Dissent:** none.
