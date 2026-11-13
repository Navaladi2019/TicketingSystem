---
title: "4. Add users to groups"
section: tour
status: delivered
sources: [docs/acceptance/s5-group-changes-within-10-minutes.md, docs/acceptance/guide/bruno/S5/S5-5.bru, docs/acceptance/guide/bruno/S5/S5-6.bru, internal/groupsync/apply.go]
adrs: [ADR-001]
operations: [getMe]
tables: [groups, group_members]
packages: [internal/groupsync]
---

## In plain words

Adding a person to a group is how you give them a role: someone added to "loan officers" gets what that group is allowed to do. The change reaches the platform on the next refresh cycle, and the person does not need to sign out and in again. Removing a person, or disabling their account, works the same way and takes effect within the same cycle.

## What you do

Add the user to the group in Keycloak. Then ask the platform who the user is.

## Try it

Bruno: `S5.5 Admin removes Alice from service-desk` and `S5.6 Poll: the same token's /me loses /service-desk` show the removal; this page shows the addition. Continues from [step 3](03-create-groups.md), with `$KT`, `$A` and `$ALICE` set:

```sh
curl -sS $CORE/api/v1/me -H "Authorization: Bearer $ALICE"      # before
GID=$(curl -sS -H "Authorization: Bearer $KT" $KC/admin/realms/nava/group-by-path/t/$A/loan-officers \
  | sed -n 's/^[^{]*{"id":"\([^"]*\)".*/\1/p')
UID_=$(curl -sS -H "Authorization: Bearer $KT" "$KC/admin/realms/nava/users?username=alice%40tenant-a.test&exact=true" \
  | sed 's/^\[//' | sed -n 's/^[^{]*{"id":"\([^"]*\)".*/\1/p')
curl -i -X PUT -H "Authorization: Bearer $KT" $KC/admin/realms/nava/users/$UID_/groups/$GID
sleep 15                                                          # one sync cycle on this stack
curl -sS $CORE/api/v1/me -H "Authorization: Bearer $ALICE"      # after, the same token
```

## What you see

Before:

```json
{"group_paths":[{"id":"f0a59f7b-…","path":"/service-desk"}], …}
```

Keycloak answers `204 No Content` to the add. After one cycle, with the same token:

```json
{"group_paths":[{"id":"4a0dff44-…","path":"/loan-officers"},{"id":"f0a59f7b-…","path":"/service-desk"}], …}
```

In this run the new group showed within 2 seconds of the change, because a cycle happened to start just after it. The worst case is one full interval: 15 seconds here, five minutes by default (the acceptance bound is ten).

## What happens behind it

The platform reads a person's groups from its own table on every request, not from the token, so a change applies to a token already issued. The token proves who the person is; the table says what the groups are now. A disabled user is refused with `401 tenant_unresolved` after the next cycle, even with an unexpired token (Bruno `S5.13`). Details: [`nava-groupsync`](../system/nava-groupsync.md), [users and groups](../database/identity.md).

## Not built yet

- No platform API adds or removes members; it is done in the identity system.
- Within the interval a removed member keeps their access. Nothing pushes the change sooner.
