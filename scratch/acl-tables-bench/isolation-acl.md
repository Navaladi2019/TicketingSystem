# Isolation report rows for acl_versions and record_shares (P3-30)

| Role | Table | Status | Detail |
|---|---|---|---|
| keycloak | public.acl_versions | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| keycloak | public.record_shares | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| nava_app | public.acl_versions | PASS | SELECT: direct none=0, pgbouncer none=0, direct ''=0, pgbouncer ''=0, direct prior=0, pgbouncer prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 1 |
| nava_app | public.record_shares | PASS | SELECT: direct none=0, pgbouncer none=0, direct ''=0, pgbouncer ''=0, direct prior=0, pgbouncer prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 1 |
| nava_backup | public.acl_versions | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| nava_backup | public.record_shares | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| nava_groupsync | public.acl_versions | PASS | no SELECT, UPDATE or DELETE grant |
| nava_groupsync | public.record_shares | PASS | no SELECT, UPDATE or DELETE grant |
| nava_onboard | public.acl_versions | PASS | no SELECT, UPDATE or DELETE grant |
| nava_onboard | public.record_shares | PASS | no SELECT, UPDATE or DELETE grant |
| nava_purge | public.acl_versions | PASS | no SELECT, UPDATE or DELETE grant |
| nava_purge | public.record_shares | PASS | no SELECT, UPDATE or DELETE grant |
| nava_relay | public.acl_versions | PASS | no SELECT, UPDATE or DELETE grant |
| nava_relay | public.record_shares | PASS | no SELECT, UPDATE or DELETE grant |
| nava_worker | public.acl_versions | PASS | SELECT: direct none=0, pgbouncer none=0, direct ''=0, pgbouncer ''=0, direct prior=0, pgbouncer prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 1 |
| nava_worker | public.record_shares | PASS | SELECT: direct none=0, pgbouncer none=0, direct ''=0, pgbouncer ''=0, direct prior=0, pgbouncer prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 1 |
| pgbouncer_auth | public.acl_versions | PASS | no SELECT, UPDATE or DELETE grant |
| pgbouncer_auth | public.record_shares | PASS | no SELECT, UPDATE or DELETE grant |
| temporal | public.acl_versions | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| temporal | public.record_shares | PASS | cannot connect to the platform database (SQLSTATE 42501) |
| nava_migrator (owner) | public.acl_versions | PASS | SELECT: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; UPDATE: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; DELETE: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 0 |
| nava_migrator (owner) | public.record_shares | PASS | SELECT: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; UPDATE: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; DELETE: direct none=0, direct ''=0, direct prior=0, tenant A, other tenants' rows=0; own rows with tenant A set: 0 |
| acl_versions: read tenant B's ACL by its type | nava_app | PASS | 0 rows, all tenant A's |
| acl_versions: read every type's version 1 | nava_worker | PASS | 1 rows, all tenant A's |
| acl_versions: publish an ACL for tenant B | nava_app | PASS | denied: SQLSTATE 42501 |
| acl_versions: publish an ACL in tenant A for tenant B's type | nava_worker | PASS | denied: SQLSTATE 23503 |
| acl_versions: rewrite tenant B's ACL | nava_app | PASS | denied: SQLSTATE 42501 |
| acl_versions: delete tenant B's ACL | nava_worker | PASS | denied: SQLSTATE 42501 |
| record_shares: list the records shared with a principal id in every tenant | nava_app | PASS | 1 rows, all tenant A's |
| record_shares: read tenant B's shares by record | nava_worker | PASS | 0 rows, all tenant A's |
| record_shares: share a record for tenant B | nava_app | PASS | denied: SQLSTATE 42501 |
| record_shares: share tenant B's record from tenant A | nava_worker | PASS | denied: SQLSTATE 23503 |
| record_shares: rewrite tenant B's shares | nava_app | PASS | denied: SQLSTATE 42501 |
| record_shares: delete tenant B's shares | nava_worker | PASS | denied: SQLSTATE 42501 |
