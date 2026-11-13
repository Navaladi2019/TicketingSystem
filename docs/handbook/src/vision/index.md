---
title: "What the product is"
section: vision
sources: [docs/product/vision.md, docs/handbook/examples/README.md, docs/handbook/src/stories/stories.yaml]
adrs: [ADR-001]
stories: [S2, S4, S5, S8, S9, S10, S11a, S11b, S13, S16, S17]
status: delivered
---

## What it is for

A generic ticket and case platform. It is not an IT service management (ITSM) product. Each tenant, meaning each customer organisation, defines its own record types, the fields on them, who may see and change them, and the workflow each record follows. The platform keeps tenants apart, records every change in an audit log and runs workflows reliably. The tenant decides what a "ticket" is.

The product direction is the product owner's decision, recorded in [the product vision](../../../product/vision.md). This page follows it and says, for each part, whether it is built.

## Three example domains

None of these is built into the platform as code. Each is configuration a tenant would publish.

| Domain | A ticket is | Fields it needs | Workflow |
|---|---|---|---|
| IT service management | An incident (something broke) or a request (someone needs something). | Summary, priority, affected service, assignment group, resolution notes. | new, assigned, in progress, resolved, closed. An unresolved high-priority incident is escalated after a set time. |
| Loan applications | One application for credit from one applicant. | Applicant, amount, term, purpose, income, credit decision. Income and identity documents are visible to underwriters only. | submitted, documents checked, underwriting, approved or declined, disbursed. A decision above a set amount needs a second approver. |
| Visa applications | One application by one traveller for one visa category. | Applicant, passport number, nationality, visa category, travel dates, decision. Passport details are hidden from front-desk staff. | received, biometrics booked, under review, decision, passport returned. A missing document pauses the case until it arrives. |

This table is the intent. What is built is in the next section. The three record types you can load today are in [the examples folder](../../examples/README.md) and are smaller than the table above.

## Intent and delivered

Checked against `main` on 2026-11-13. "Delivered" means it works on `main`. Nothing is described as working unless it is in the "Delivered on `main`" column.

| Capability | Delivered on `main` | Not built yet |
|---|---|---|
| Generic record types | A tenant admin publishes versioned record types with their fields; a breaking change is refused. See [entity types](../database/entity-definitions.md) and [S8](../stories/s8.md). New tenants start with `incident`, `request` and `change`. | |
| Per-type access rules, masks and numbering | Users see only their group's records, and masked fields stay hidden ([S10](../stories/s10.md), [access rules and masks](../database/acl.md)). Every type numbers its records `<prefix><7 digits>` ([S17](../stories/s17.md)). | Tenant-defined number formats (S18, S19, planned). |
| ITSM, loan and visa examples | Three example types, `incident`, `loan_application` and `visa_application`, with a seed script: [examples](../../examples/README.md). Each uses only the fields, states, access rules and masks that exist today. | Documents, approvals, decision rules and typed checks, listed on the examples page. |
| Records | Create, update, history, and a stale update is refused ([S9](../stories/s9.md), [records](../database/records.md)). | Comments, work notes and links (S12, planned). |
| Workflows | The interpreter runs a workflow definition ([S11a](../stories/s11a.md), [workflows](../workflows/index.md)). | Creating a record does not start a workflow yet (S11b, in progress). |
| Tenants, groups and users | An operator onboards a tenant with one API call ([S4](../stories/s4.md)). Groups, users and organisation units arrive as a sync from the identity provider and apply within 10 minutes ([S5](../stories/s5.md)). | A platform API to manage groups and users. |
| Tenant isolation | A query without the right tenant returns nothing ([S2](../stories/s2.md)). | |
| Audit log and export | One audit log, hash-chained per tenant, and a filtered export an auditor can verify offline ([S13](../stories/s13.md), [S16](../stories/s16.md), [audit](../database/audit.md)). | |
| User interface | | The platform is API only, at `/api/v1`. The embeddable UI widget is not built. |
| Embedding SDK | | A headless API and SDK. The embedding RFC (TICK-786) comes first. |

## Embedding

Other applications will embed the platform in two ways, and both are decided: a ready-made UI widget the host application drops into its own pages, and a headless API and SDK the host builds its own screens on. Neither exists yet. How a host authenticates, themes the widget and scopes it to a tenant is for the embedding RFC (TICK-786).

## Who uses it

| Who | What they do |
|---|---|
| Tenant end users | Create, update and follow records: agents, underwriters, case officers, applicants. |
| Tenant administrators | Define record types, fields, access rules and workflows for their tenant. |
| Host applications | Embed the platform through the widget or the API and SDK. |
| Platform operators | Onboard tenants, run the platform, answer audits. |

## Read next

- [Goals and non-goals](goals-and-non-goals.md)
- [Glossary](../glossary/index.md)
- How the platform keeps tenants apart is in [the nine invariants](invariants.md), one page each, under "How it works".
- [Known technical risks](../../../cto/risks.md), kept by the CTO outside the handbook.
