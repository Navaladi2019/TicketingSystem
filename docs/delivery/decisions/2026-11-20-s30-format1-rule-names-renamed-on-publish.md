## 2026-11-20: S30, a format-1 rule id that is not a valid name is renamed when published; no legacy_id is stored

- **Decided by:** the Technical Lead, recorded on TICK-945 (S30 handbook), from the TICK-939 review (PR #763). Story S30 (TICK-935), ADR-010 D12.
- **Decision:** a format-1 body sent to `PUT /api/v1/entity-types/{key}/access` is converted and stored as format 2. A rule id that is not a valid policy name (`^[a-z][a-z0-9_]{0,62}$`) gets the derived name, for example `r-desk` becomes `r_desk`, and the stored document has **no `legacy_id`**: format-2 input refuses that key, so the stored policy is renamed in place. The explainer's `rule_id` then shows `r_desk`. Only a format-1 row stored before S30, converted on read, keeps its original id as `legacy_id` and in `rule_id`.
- **Why:** stored rows are format 2 from S30 on (D12), and accepting `legacy_id` on input would let an admin set a `rule_id` unrelated to the policy name.
- **Effect:** clients that matched `rule_id` against a hyphenated format-1 id must match the underscored name after their next publish. The S10 Bruno kit was updated for this (TICK-998). `rule_id` is kept for one release (D10); `policy` is the field to read.
- **Also found:** ADR-010's CI table (and RFC-022) says a stale expected version gives 409; the route gives `412 version_conflict`, as every `If-Match` route does. PLAN-011 does not carry the 409. The ADR text is corrected by an erratum task for the Solution Architect.
- **Dissent:** none.
