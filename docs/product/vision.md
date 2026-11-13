# Product vision

Status: draft for the product owner's confirmation (TICK-787). Product direction
is the product owner's decision; this page records it and adds nothing of its own.

## What it is

A generic ticket and case platform. It is not an IT service management (ITSM)
product. Each tenant defines its own record types, the fields on them, who may
see and change them, and the workflow each record follows.

The platform keeps tenants apart, records every change in an audit log and runs
workflows reliably. The tenant decides what a "ticket" is.

## Example domains

These three show the range. None of them is built into the platform as code;
each is configuration a tenant would publish.

### IT service management

- **A ticket is** an incident (something broke) or a request (someone needs something).
- **Fields:** summary, priority, affected service, assignment group, resolution notes.
- **Workflow:** new → assigned → in progress → resolved → closed. An unresolved
  high-priority incident is escalated after a set time.

### Loan applications

- **A ticket is** one application for credit from one applicant.
- **Fields:** applicant, amount, term, purpose, income, credit decision. Income
  and identity documents are visible to underwriters only.
- **Workflow:** submitted → documents checked → underwriting → approved or
  declined → disbursed. A decision above a set amount needs a second approver.

### Visa applications

- **A ticket is** one application by one traveller for one visa category.
- **Fields:** applicant, passport number, nationality, visa category, travel
  dates, decision. Passport details are hidden from front-desk staff.
- **Workflow:** received → biometrics booked → under review → decision →
  passport returned. A missing document pauses the case until it arrives.

## Embeddable

Other applications can embed the platform in two ways (decision: both):

- **A ready-made UI widget** the host application drops into its own pages.
- **A headless API and SDK** the host application builds its own screens on.

How a host application authenticates, themes the widget and scopes it to a
tenant is for the embedding RFC (TICK-786), not this page.

## Who uses it

| Who | What they do |
|---|---|
| Tenant end users | Create, update and follow records: agents, underwriters, case officers, applicants. |
| Tenant administrators | Define record types, fields, access rules and workflows for their tenant. |
| Host applications | Embed the platform through the widget or the API and SDK. |
| Platform operators | Onboard tenants, run the platform, answer audits. |

## Goals

- A tenant can model a new kind of case through configuration alone, with no
  code change and no schema change.
- One tenant can never read or change another tenant's data.
- Every change is audited, and an auditor can prove the export is complete and untampered.
- A saved change starts its workflow exactly once, even after retries and failures.
- Other applications can embed the platform through a widget or an API and SDK.

## Non-goals

- Shipping a fixed ITSM, lending or visa product. Those are examples of tenant
  configuration, not features.
- Tenant code running inside the platform with direct database or workflow access.
  Extensions go through the public API with a scoped token (ADR-001, I8).
- AI that changes data on its own below an explicit confidence threshold (ADR-001, I8).

## Delivered on `main` today vs intended

Checked against `main` on 2026-11-13 (stories in
`docs/handbook/src/stories/stories.yaml`).

| Capability | Today on `main` | Intended |
|---|---|---|
| Tenant onboarding | Delivered: an operator onboards a tenant with one API call; its admin signs in through the tenant's identity provider (S3, S4). | Same. |
| Tenant isolation | Delivered: a query without the right tenant returns nothing (S2). | Same. |
| Groups, users, organisation units | Delivered as a sync from the identity provider (Keycloak); changes apply within 10 minutes (S5). No platform API to manage them. | Managed by tenant administrators. |
| Record types and fields | Delivered: a tenant admin publishes versioned types; breaking changes are refused (S8). New tenants start with `incident`, `request` and `change` only. | Example types for ITSM, loan and visa (TICK-786). |
| Records | Delivered: create, update, history, stale update refused (S9). | Same, plus comments, work notes and links (S12, planned). |
| Access rules and masked fields | Delivered: users see only their group's records; masked fields stay hidden (S10). | Same. |
| Workflows | Partly: the interpreter runs a definition (S11a). Creating a record does not yet start a workflow (S11b, in progress). | Every record follows its type's workflow. |
| Audit log and export | Delivered: one audit log, hash-chained per tenant, filtered export verifiable offline (S13 to S16). | Same. |
| Record numbering | Delivered: per-tenant counters (S17). | Tenant-defined number formats (S18, S19, planned). |
| User interface | Not built yet. The platform is API only (`/api/v1`). | Embeddable UI widget. |
| Embedding SDK | Not built yet. | Headless API and SDK; RFC to follow (TICK-786). |
