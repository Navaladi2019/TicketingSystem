---
title: "5. Organisation units"
section: tour
status: delivered
sources: [db/migrations/00003_identity_tables.sql, internal/groupsync/apply.go, deploy/compose/keycloak/import/realm-nava.json]
adrs: [ADR-001]
tables: [org_units]
packages: [internal/groupsync]
---

## In plain words

An organisation unit is a department or team in a company's chart, such as "Engineering" with "Platform" inside it. **This is not a feature yet.** The platform copies the department tree from the customer's identity system, but nothing in the product shows it, searches it or uses it to decide who sees what. Today the groups from step 3 do that job.

## What you do

Nothing. There is no step to follow: no API, no screen and no access rule uses organisation units.

## Try it

There is no request. The only way to see them is the database. On the local stack, after `seed.sh`:

```sh
docker compose exec -T postgres sh -c \
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -U pgadmin -d nava -c \
   "select o.name, coalesce(p.name,'"'"'(root)'"'"') as parent from org_units o
    left join org_units p on p.tenant_id=o.tenant_id and p.id=o.parent_id
    where o.tenant_id='"'"'00000000-0000-4000-8000-0000000000a1'"'"' order by 1"'
```

## What you see

```
eng       (root)
platform  eng
support   (root)
```

These come from Keycloak groups under `/t/<tenant id>/ou/` in the demo realm (`/ou/eng`, `/ou/eng/platform`, `/ou/support`).

## What happens behind it

Group sync (step 3) also reads the groups below `ou/` and writes them to the `org_units` table as a tree. It rejects a change that would make a unit its own ancestor and stops at a depth of 64. After that, no code reads the table: no endpoint in the API names organisation units, and no access rule or mask can refer to one. The table was added with the identity tables (ADR-001) so the tree is ready when a feature needs it.

## Not built yet

All of it, as a feature:

- No API to list or change units, and no screen.
- No access rule such as "everyone under Engineering". Rules name groups and users only (step 7).
- No ADR describes how units would be used. Until one does, this page does not describe behaviour beyond the table above.
