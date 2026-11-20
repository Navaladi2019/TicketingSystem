---
title: "Access document size limit (NAVA_ACL_MAX_DOCUMENT_BYTES)"
section: operations
status: delivered
sources: [docs/operator/acl-document-limit.md, internal/platform/config/acl.go, internal/server/access.go, .env.example, docs/adr/ADR-010-A1-sp2-caps-size-cap-app-only-publish.md, db/migrations/20261119100000_acl_rules_valid_format2.sql]
adrs: [ADR-010-A1, ADR-010]
stories: [S30]
packages: [internal/platform/config]
tables: [acl_versions]
tests: [TestLoadACLMaxDocumentBytes, TestPrepareByteLimitIsOnStoredTextAndOnMasks, TestTICK940ByteCeilings]
---

`nava-core` limits the size of an entity type's access rules and of its masks.
It is one setting for the whole installation, set by a platform operator in
the deployment environment. **No tenant, admin or API call can change it, and
it is never per tenant.** The operator note is
[docs/operator/acl-document-limit.md](../../../operator/acl-document-limit.md).

## The setting

| | |
|---|---|
| Variable | `NAVA_ACL_MAX_DOCUMENT_BYTES`, read by `nava-core` only |
| Default (unset or empty) | 32,768 bytes (32 KiB) |
| Allowed range | 16,384 to 131,072, a plain integer |
| Outside the range, or not an integer (`abc`, `32768 `, `3.2e4`) | `nava-core` refuses to start; the config error names the variable, not the value |
| What it measures | `rules` and `masks` separately, each as the bytes of its stored PostgreSQL `jsonb` text (`octet_length(rules::text)`), the same number the database counts |
| Database backstop | `acl_versions_bytes_rules_check` and `acl_versions_bytes_masks_check` refuse more than 131,072 bytes whatever the setting says |

The floor, 16 KiB, keeps a document at every fixed cap publishable (about
10.8 kB with one id per leaf). The ceiling, 128 KiB, is the measured cost
bound of a publish (ADR-010-A1, accepted risk `AR-TICK-971-01`). The fixed
caps (depth 8, 16 children, 64 policies, 128 leaves per policy and per
document) are not settings; see the [S30 page](../stories/s30.md#published-limits).

## What a tenant admin sees

A publish over the limit is refused with `422 invalid_access_policy`, code
`document_too_large`, pointer `/rules` or `/masks`, and nothing is written.
The effective limit is in that error's `detail` and nowhere else
([Access API](../system/access-api.md#problem-codes)).

## How to change it safely

1. **Pick a value in range.** Raising it lets larger documents publish and
   makes the worst publish slower (about 90 ms p95 at 32 KiB, about 340 ms
   p95 at 128 KiB, in the admin's own tenant). Lowering it refuses large
   documents on their next publish.
2. **Lowering never touches stored versions.** Rows already in
   `acl_versions` keep being read and enforced. A tenant whose current
   document is larger than the new limit can still use it, but its next
   publish must fit. Before lowering, check the largest stored documents:

   ```sql
   SELECT tenant_id, entity_definition_id, version,
          octet_length(rules::text) AS rules_bytes, octet_length(masks::text) AS masks_bytes
   FROM acl_versions ORDER BY greatest(octet_length(rules::text), octet_length(masks::text)) DESC LIMIT 20;
   ```

3. **Set it on every `nava-core` replica to the same value**, then restart
   them. A value outside the range stops the process at start-up, so a typo
   fails the rollout instead of running with a wrong limit.
4. **Onboarding is unaffected.** `nava-onboard` publishes the default incident
   ACL with the default limit; that document is far under 16,384 bytes.

Locally, the variable is documented in `.env.example`. `compose.yaml` does
not pass it to `nava-core`, so the local stack always runs with the default;
to try another value, add it to `nava-core`'s `environment` in a Compose
override file.

## Proving tests

| Behaviour | Test |
|---|---|
| Default, range and refusal at start-up | `TestLoadACLMaxDocumentBytes` |
| The route measures the stored text, for rules and for masks | `TestPrepareByteLimitIsOnStoredTextAndOnMasks` |
| The database ceilings | `TestTICK940ByteCeilings` |
