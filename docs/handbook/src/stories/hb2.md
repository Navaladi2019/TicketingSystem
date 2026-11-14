---
title: "HB2: A first-time reader learns the product from the handbook"
section: stories
status: delivered
sources: [docs/product/vision.md, docs/handbook/src/tour/index.md, docs/handbook/src/nav.yaml, docs/handbook/examples/README.md, docs/acceptance/guide/tools/seed.sh]
adrs: [ADR-007]
stories: [HB2]
evidence: [docs/product/vision.md, docs/handbook/src/tour/index.md, docs/handbook/examples/README.md, docs/handbook/examples/seed-examples.sh]
---

Story TICK-786. Verdict: **READY** (all tasks merged on `main`). Date: **2026-11-14**. The story verdict (ACCEPTED or REJECTED) is posted by the Technical Lead on TICK-786. This is a docs-only story: it adds no product code, no tests and no Bruno folder.

## What a reader can now do

- Read in two minutes what the product is: a generic ticket and case platform where each tenant defines its own record types, fields, access rules and workflows. The examples are IT service management, loan applications and visa applications. Source: [`docs/product/vision.md`](../../../product/vision.md), confirmed by the product owner on TICK-787.
- See what is **delivered on `main`** and what is only **intended**, side by side, on the [vision page](../vision/index.md).
- Follow the [product tour](../tour/index.md) in ten numbered steps, from onboarding a tenant to exporting and verifying the audit log. Each step opens with an "In plain words" section for support and business readers, then the request to send, a real response, what happens behind it, and what is not built yet.
- Load three example record types (`incident`, `loan_application`, `visa_application`) into tenant A with one script, and see a field mask hide `income` and `passport_number` from non-admins.
- Find every page through a sidebar ordered for a reader: Start here, Product tour, Concepts, How it works, Reference, Operations. The order comes from `docs/handbook/src/nav.yaml`, not from file names.
- Read how other applications will embed the platform (a UI widget and a headless SDK): [ADR-007](../adr/adr-007.md). It is accepted but **not implemented**.

## What delivered it

| Task | What | PR |
|---|---|---|
| TICK-787 | Product vision document `docs/product/vision.md`; the README links to it | #632 |
| TICK-789 | Example record types and `seed-examples.sh` under `docs/handbook/examples/` | #634 |
| TICK-788 | Sidebar and navigation redesign (`nav.yaml`, collapsible groups, previous/next links) | #635 |
| TICK-790 | Product tour, ten steps | #637 |
| TICK-791 | Vision page rebuilt from the vision document (Intent vs Delivered) | #638 |
| TICK-794 | `seed.sh` creates the audit chain head for seeded tenants, so tour step 10 needs no SQL workaround | #641 |
| TICK-792 | ADR-007: embedding widget and SDK (RFC, critique, ADR with `.md` and `.html`) | #644 |
| TICK-850 | This page and the `stories.yaml` entry | this page's PR |

## How to try it

1. Build the handbook from the repository root: `python -I docs/handbook/build/build.py`, then open `docs/handbook/index.html`. It works offline.
2. Start at "Start here", then follow "Product tour" from step 1. The tour's [Before you start](../tour/index.md) section lists the commands for a fresh local stack (`docker compose up --wait`, `seed.sh`, `tunnels.sh up`).
3. To load the example types without the tour, run `sh docs/handbook/examples/seed-examples.sh` after `seed.sh`. Do not run it before the tour: step 6 creates `loan_application` itself.

## Not built yet

These gaps are stated on the pages themselves; this story documents them, it does not close them.

- **Organisation units** (tour step 5): the table is filled from the identity system, but there is no API and no rule uses it.
- **Workflows on records** (tour step 8): the interpreter is on `main` but registered only in a test build; creating a record starts no workflow. That is story S11b.
- **Embedding** (ADR-007): accepted, nothing implemented.
- **No user interface**: the tour uses the REST API through `curl` and Bruno.
- **Domain needs** the examples leave out: document attachments, custom number formats per type, masks that depend on the requester, approval rules. See the [examples README](../../../handbook/examples/README.md).

## Verdict and evidence

- Verdict: READY, 2026-11-14: every task above is merged on `main`. There is no acceptance kit; the check is that the handbook build runs clean and the pages open offline.
- Evidence: the vision document, the tour overview, the examples README and the seed script (listed in `stories.yaml`).
- The tour's requests and responses come from one run on a fresh stack built from `main`; the tour pages say so and their footer shows the commit.
