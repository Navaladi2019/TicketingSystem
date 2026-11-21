## 2026-11-21: S32, the missing-policy audit entry stores only a hash of the name; adding the name is a product follow-up

- **Decided by:** the Technical Lead, recorded on TICK-959 (S32 handbook), from the TICK-958 kit. Story S32 (TICK-955), ADR-010 D9; built by TICK-957 and TICK-1025.
- **Decision:** accepted as a known limit. A `policy_ref_refused` audit entry (domain `config`, subject the entity type's `acl_policy`, `actor_system: workflow`) records the refused reference as `content_sha256` = SHA-256 of `<entity_type>/<name>`, not the name. An admin matches an entry by hashing the name their workflow uses, or reads the nava-worker warning `authz: workflow policy reference names no policy, it denies`, which carries the name.
- **Why:** this is how TICK-957 built it; the name comes from a tenant's definition and the entry has no field for free text today. Criterion 3 ("the audit log shows a missing-policy entry") is met.
- **Follow-up:** whether to store the name in the entry is a product decision for the story owner. Trigger: an admin reports they cannot find the refusal, or a UI for audit entries is planned.
- **Dissent:** none.
