## 2026-11-11: D-3-29, S11b contract: causation is copied durably into workflow_cause, and the waiting state is tracked on workflow_ref

**Decided by:** Technical Lead, when grooming S11b (TICK-337, tasks TICK-709 to TICK-713). The full contract is in each task issue.

**Why:** the TICK-88 security review requires the P3-42 interpreter to take `Causation` only from the outbox row the relay delivered, never from workflow input. Dispatched outbox rows are purged after 7 days (`config.MinOutboxDispatchedRetention`), but a `wait_signal` or `timer` can last up to 30 days. A write that comes after a long wait would therefore find no outbox row to read.

**Decision:**
- `workflow_cause(tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth)` is a tenant table (P3-40). The relay writes one row per start or signal it dispatches to a `nava.interpreter` run, copied from that outbox row (P3-42).
- `interp.StepRef` gains `Cause`, the outbox id of the delivery that last resumed the run. It is a reference only (R1). The `RecordWriter` adapter (P3-43) reads origin and depth from that row under RLS, and refuses the row unless `ref_outbox_id` is the run's own ref, which it takes from the workflow id.
- `workflow_ref.record_changed_signal_id` is set by `NextStep` while the run waits on `record_changed`, and cleared when it stops waiting. The sink signals only refs where it is set (D-3-14, R9).
- `pipeline.Command.WorkflowRef` and `pipeline.Event.WorkflowRef` name the writing run, so the sink never signals the run that caused the change.
- The worker pipeline's `TxRunner` gets its own pool, or the worker pool is sized for it (P3-43, TICK-88 availability note).

**Dissent:** none recorded. P3-42 (Principal Software Engineer) may propose a narrower shape before P3-40 merges.

**Deferred:** the `cascade_start` DELETE policy and its `(tenant_id, record_id)` index. Trigger: the first task that deletes records.
