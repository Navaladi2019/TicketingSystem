---
title: "Product tour at a glance"
section: tour
status: delivered
sources: [docs/product/vision.md, docs/handbook/examples/README.md, docs/acceptance/guide/prerequisites.html, docs/acceptance/guide/tools/seed.sh, docs/acceptance/guide/tools/tunnels.sh]
adrs: [ADR-001]
---

This tour follows the product in the order a customer meets it: a tenant is created, people sign in and are put in groups, an administrator defines a type of record, sets who may see it, and people then create and work on records. Everything is recorded in an audit log that can be checked without trusting the platform.

Every step has the same parts. **In plain words** is written for support and business readers and can be read on its own. The rest is for engineers: the request to send, the real response, and what happens behind it. Stop after "In plain words" if you do not need the detail.

## The ten steps in plain words

1. **[Onboard a tenant](01-onboard-a-tenant.md).** A platform operator sets up a new customer organisation (a "tenant") with one call. The customer gets its own private space, its own sign-in through its own company login, and a first administrator. Built.
2. **[Sign in](02-sign-in.md).** People sign in with their company login. The platform works out which customer they belong to from the sign-in itself, never from anything the person types, so nobody can look into another customer's data. Built.
3. **[Create groups](03-create-groups.md).** Groups such as "loan officers" or "service desk" are created where the customer manages its people. The platform picks them up on a schedule. Built; a customer creates groups in the identity system, not in the platform.
4. **[Add users to groups](04-add-users-to-groups.md).** Putting a person in a group changes what they can do. The change reaches the platform within one refresh cycle, without the person signing in again. Built.
5. **[Organisation units](05-organisation-units.md).** Departments and teams. The platform copies the department tree today but nothing uses it yet. **Not built yet** as a feature.
6. **[Create an entity type](06-create-an-entity-type.md).** An administrator defines a kind of record, for example a loan application: its fields, its states and its number format. Changing it later creates a new version and keeps the old one. Built.
7. **[Set access rules and field masks](07-access-rules-and-masks.md).** The administrator decides who may see or change each kind of record and hides sensitive fields, such as income, from most people. A "why can this person see it" check explains any decision. Built.
8. **[Workflows](08-workflows.md).** Automatic steps that follow a record, such as waiting for an approval. The engine that runs them exists but is not switched on in the running product, so a new record does not start one yet. **Not built yet** as a feature.
9. **[Create and work with records](09-create-and-work-with-records.md).** People create records, list and filter them, change their state, and see the history. Every record gets a number such as `LOAN0000001`. Built.
10. **[Audit](10-audit.md).** Every change is written to a log whose entries are chained together. An auditor can download the log and check offline that nothing was altered or removed. Built.

## Built and not built

| Step | Status on `main` |
|---|---|
| 1 Onboard a tenant, 2 Sign in, 3 Create groups, 4 Add users to groups | Built |
| 5 Organisation units | Table filled from the identity system; no API, no screens, no rule uses it |
| 6 Entity types, 7 Access rules and masks, 9 Records, 10 Audit | Built |
| 8 Workflows | Engine on `main`, registered only in a test build; nothing starts it from a record write |

There is no user interface in the product yet: everything in this tour is the REST API, called with `curl` or Bruno.

## Before you start

You need Docker, Go, Git Bash or any POSIX shell, and about 10 minutes for the first image build. From a clone of `main`:

```sh
cat > .env <<'EOF'
NAVA_GROUPSYNC_INTERVAL=15s
EOF
make dev-ca
docker compose build
docker compose up --wait
sh docs/acceptance/guide/tools/seed.sh
sh docs/acceptance/guide/tools/tunnels.sh up
```

`NAVA_GROUPSYNC_INTERVAL=15s` makes group changes show up in seconds instead of the default five minutes. `seed.sh` creates two demo tenants, A (`00000000-0000-4000-8000-0000000000a1`) and B, and their users. `tunnels.sh` opens the operator ports on `127.0.0.1`. Do not run `docs/handbook/examples/seed-examples.sh`: step 6 creates `loan_application` itself, and that script would create it first.

Then set up the shell once. A token lasts five minutes, so run the `token` lines again when a call answers `401`.

```sh
export CORE=http://127.0.0.1:18080 KC=http://127.0.0.1:18081 ONBOARD=http://127.0.0.1:18082
token() { curl -fsS -X POST $KC/realms/nava/protocol/openid-connect/token \
  -d grant_type=password -d client_id=$1 -d scope=openid \
  --data-urlencode "username=$2" --data-urlencode "password=$3" \
  | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p'; }
OPERATOR=$(token nava-operator-cli operator@nava.test dev-only-operator)
ADMIN=$(token nava-web admin@tenant-a.test dev-only-admin-a)
AUDITOR=$(token nava-web auditor@tenant-a.test dev-only-auditor-a)
ALICE=$(token nava-web alice@tenant-a.test dev-only-alice)
```

These are development-only users and passwords from the local realm; they exist nowhere else. The commands run from the repository root. The Bruno collection with the same requests is `docs/acceptance/guide/bruno/`; each step names the closest request.

Every request and response in this tour comes from one run of that stack, built from `main`, in order, as written. Ids, timestamps and hashes differ in your run. The page footer shows the commit.
