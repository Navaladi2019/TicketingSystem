---
title: "5. Organisation units"
section: tour
status: delivered
sources: [internal/groupsync/apply.go, internal/groupsync/closure.go, deploy/compose/keycloak/import/realm-nava.json, docs/acceptance/S31/README.md, docs/data/org-units.md, docs/data/org-unit-closure.md]
adrs: [ADR-010, ADR-010-A2]
stories: [S31]
tables: [org_units, org_unit_members, org_unit_closure, records]
packages: [internal/groupsync]
---

## In plain words

An organisation unit is a part of a company's chart, such as "EMEA" with
"Germany" inside it. The platform copies the tree, and who belongs where,
from the customer's identity system. Since S31 an access rule can use it:
"agents see the tickets of their region", meaning their unit and every unit
below it. A ticket remembers the unit of the person who raised it.

## What you do

1. In Keycloak, put each user in a group under `/t/<tenant id>/ou/`, for
   example `/ou/EMEA` or `/ou/EMEA/Germany`. A user with several units gets
   a primary one from the user attribute `nava_primary_org_unit`, which only
   an admin can set; a user with one unit has it as their primary.
2. Wait for group sync (at most `NAVA_GROUPSYNC_INTERVAL`, 5 minutes by
   default), or run `docker compose run --rm --no-deps nava-groupsync --once`.
3. Publish an access rule with the `under` test ([step 7](07-access-rules-and-masks.md)):

   ```json
   {"name": "in_my_region", "ops": ["read"], "when": {"under": {"of": "caller"}}}
   ```

   `{"of": ["<unit id>", ...]}` names units instead of the caller's own.

## Try it

The S31 kit walks the whole story on a fresh stack:
`sh docs/acceptance/S31/run.sh` ([S31](../stories/s31.md)). No API lists
units yet, so the kit reads the tree from the database. It prints:

```
     path     | state | closure_rows |           members
--------------+-------+--------------+------------------------------
 APAC         | live  |            1 | requester-apac*
 EMEA         | live  |            2 | agent-emea*, requester-emea*
 EMEA/Germany | live  |            1 | requester-germany*
 eng          | live  |            2 |
 eng/platform | live  |            1 | alice*
 support      | live  |            1 |
```

`*` marks the primary unit. `closure_rows` is how many live units are at or
below the unit, itself included.

## What you see

With `in_my_region` published, `agent-emea` lists the tickets raised by the
EMEA and Germany requesters and gets the missing-record 404 on an APAC one.
A ticket raised by someone with no unit is in nobody's region.

## What happens behind it

- Group sync writes `org_units` (the tree), `org_unit_members` (who is in
  which unit, and the primary) and `org_unit_closure` (for each live unit,
  every live unit at or below it) in one transaction per tenant. It refuses a
  change that would make a unit its own ancestor and stops at a depth of 64.
- A unit removed in Keycloak is tombstoned, never deleted. It and its whole
  subtree drop out of every region at that sync; the tickets keep their unit.
- On create, the write pipeline stores the requester's primary unit in
  `records.org_unit_id`. It is never recomputed: moving a person to another
  unit leaves their old tickets in the old unit.
- The `under` test reads the closure in SQL for lists and in Go for a single
  record, and both give the same answer (I5). See
  [Core: record ACL](../system/core-acl.md#the-under-leaf-s31) and
  [Org units (tables)](../database/org-units.md).

## Not built yet

- No API or screen to list units, show a ticket's unit, or move tickets
  between units. Moving tickets is planned as a workflow step (ADR-010 D8).
- The tenant identity provider's mapper does not fill
  `nava_primary_org_unit` yet; an admin sets it in Keycloak.
