---
type: register
id: CP
title: Competitor pain-point register — Salesforce Service Cloud, ServiceNow, Freshworks
status: ACTIVE (ADR-008 accepted 2026-11-15)
date: 2026-11-14
owner: Solution Architect
review: every 6 months, or when a vendor changes pricing, packaging or limits
---

# Competitor pain-point register

ADRs cite entries by ID (`CP-SF-03`) in their **Competitive Gate** section (ADR-008). An ID is never reused; a retired entry stays with status `killed` and the reason.

**Evidence grade** — how far you can lean on the source:

| Grade | Source kind | What it may support |
|---|---|---|
| **A** | Vendor's own documentation, pricing page, status/post-mortem, or reputable press reporting of a public incident | "avoids" and "repeats" findings |
| **B** | Aggregated reviews (G2, Gartner Peer Insights, Capterra, PeerSpot), vendor community forum threads with several practitioners | "avoids" and "repeats" findings |
| **C** | Consultancy, reseller, recruiter or competitor blog; a single anecdote | "avoids" only. A send-back for **repeating** a C-only entry needs a B or A source first. |

Prices are list prices seen on 2026-11-14 and move often; re-check before quoting. Reviewer quotes on Gartner Peer Insights "should not be construed as statements of fact" (Gartner's own caveat).

## Salesforce Service Cloud (CP-SF)

| ID | Pain point | Hypothesis result | Evidence | Grade |
|---|---|---|---|---|
| **CP-SF-01** | **Implementation takes months and usually a partner.** Standard deployments 2–6 months, enterprise 6–18 months; implementation spend often quoted at 1.5x–5x the annual licence. | Confirmed | [HDWebSoft implementation guide](https://www.hdwebsoft.com/blog/salesforce-implementation-guide); [Cynoteck cost guide](https://cynoteck.com/blog-post/salesforce-implementation-cost); [Unthread pricing analysis](https://unthread.io/blog/salesforce-service-cloud-pricing/); G2 time-to-implement range "under 1 day to more than 12 months" ([G2 Service Cloud](https://g2.com/products/salesforce-service-cloud/details)) | C (B for the G2 range) |
| **CP-SF-02** | **Governor limits.** Per-transaction ceilings that tenant logic must be written around: 100 SOQL queries synchronous / 200 async, 10,000 ms CPU synchronous / 60,000 ms async; each parent-child subquery counts as an extra query. | Confirmed | [Apex Developer Guide — Execution Governors and Limits](https://developer.salesforce.com/docs/atlas.en-us.apexcode.meta/apexref/apex_gov_limits.htm) | A |
| **CP-SF-03** | **Three releases a year that customers must regression-test.** Spring/Summer/Winter releases; Salesforce itself recommends regression testing in a preview sandbox each release; Lightning page and Flow metadata changes between releases cause deployment problems. | Confirmed | [Salesforce Admins — Be Release Ready](https://admin.salesforce.com/blog/2026/be-release-ready-the-admin-habit-that-saves-you-every-time); [Salesforce Ben — preview sandboxes](https://www.salesforceben.com/why-admins-should-beware-of-salesforce-sandbox-preview-orgs/) | A / C |
| **CP-SF-04** | **Per-seat pricing plus add-ons.** Service Cloud Enterprise $175, Unlimited $350 per user per month; messaging (Digital Engagement) +$75/user; AI tier (Agentforce 1 Service) $550/user. | Confirmed | [Salesforce pricing page (APAC)](https://www.salesforce.com/ap/service/pricing/); [Capterra pricing](https://www.capterra.com/p/136189/Salesforce/pricing/); [eesel add-on pricing](https://www.eesel.ai/blog/salesforce-service-cloud-ai-add-on-pricing) | A / C |
| **CP-SF-05** | **Slow pages under customisation.** Salesforce's own engineering blog says some standard components are slow against industry peers and that Reports exceed P75 load targets; admins report Lightning slower than Classic, worse with 100–200 fields on a page. | Confirmed (slow). "Cluttered UI" **unconfirmed**: no source measured it; do not cite. | [Salesforce Developers — Our plan to improve Lightning Experience performance (2023)](https://developer.salesforce.com/blogs/2023/09/our-detailed-plan-to-improve-lightning-experience-performance); [Salesforce Ben — why is Salesforce so slow](https://www.salesforceben.com/why-is-salesforce-so-slow/) | A / C |
| **CP-SF-06** | **A deployment broadened permissions across tenants' users.** May 2019: a database script gave Pardot-connected users read/write on all data in their orgs; Salesforce cut access to 100+ instances; admins repaired permissions by hand; fix completed after ~12 days. | Confirmed (public incident) | [The Register, 2019-05-20](https://www.theregister.com/2019/05/20/salesforce_outage_continues/); [TechTarget](https://www.techtarget.com/searchcustomerexperience/news/252463772/Salesforce-outage-exposes-Pardot-marketing-automation-data); [Computer Weekly](https://www.computerweekly.com/news/252464214/Salesforce-completes-Pardot-fix-after-12-days-disruption) | A |

## ServiceNow (CP-SN)

| ID | Pain point | Hypothesis result | Evidence | Grade |
|---|---|---|---|---|
| **CP-SN-01** | **High total cost and licence complexity.** No public list price; quoted per fulfiller by tier (Standard/Pro/Enterprise); third-party estimates ITSM Pro ~$90–150 per fulfiller per month, AI (Now Assist) extra; reviewers name TCO and licensing model as the main challenge. | Confirmed (price figures are estimates, not list) | [Gartner Peer Insights — ServiceNow](https://www.gartner.com/reviews/market/software-asset-management-tools/vendor/servicenow); [G2 ServiceNow ITSM pricing](https://www.g2.com/products/servicenow-it-service-management/pricing); [ServiceNow Community — Confused by ServiceNow licensing](https://www.servicenow.com/community/developer-articles/confused-by-servicenow-licensing/ta-p/3383194); [Redress Compliance ITSM pricing](https://redresscompliance.com/servicenow-itsm-pricing-2026) | B / C |
| **CP-SN-02** | **Long implementations.** G2 users report ~5 months to implement (page dated 2023); third parties put first go-live (incident, request, knowledge) at 3–6 months and full ITIL coverage at 9–18 months. | Confirmed | [G2 ServiceNow ITSM pricing (time to implement)](https://www.g2.com/products/servicenow-it-service-management/pricing); [desk365 review roundup](https://www.desk365.io/blog/servicenow-reviews) | B / C |
| **CP-SN-03** | **Upgrades cost rework because of customisation.** Modified out-of-box records are "skipped" on upgrade and each needs an admin decision (retain or revert); heavily modified instances see thousands of skipped records; one case: 400+ modified records turned a 2-week upgrade into 4 months. | Confirmed (mechanism A; size of effect C) | [ServiceNow docs — Upgrade Summary Report](https://www.servicenow.com/docs/r/UrSRFFKWBbfQBgoRlt~ltw/L5MWl9TOnHeZiCeM3oZqig); [ServiceNow Community — Reducing skipped records](https://www.servicenow.com/community/developer-blog/reducing-skipped-records-in-servicenow-upgrades/ba-p/3102307); [Kumoco technical-debt guide](https://kumoco.pages.dev/servicenow-technical-debt-guide/) | A / B / C |
| **CP-SN-04** | **Needs specialist admins who are scarce.** ~116k certified professionals worldwide; 1–2 years for a certified admin to be confident; ServiceNow itself writes about the admin talent gap. | Confirmed (sources are recruiters and the vendor) | [ServiceNow blog — tech talent gap for system administrators](https://www.servicenow.com/blogs/2023/tech-talent-gap-system-administrators); [Inclusion Cloud](https://inclusioncloud.com/insights/blog/why-is-servicenow-talent-shortage/) | A / C |
| **CP-SN-05** | **Steep learning curve; heavy for small and mid-size teams.** Reviewers cite setup complexity and learning curve. "Slow for SMB" as a latency claim is **unconfirmed** — the evidence is about effort, not speed. | Partly confirmed | [Gartner Peer Insights — ServiceNow](https://www.gartner.com/reviews/market/software-asset-management-tools/vendor/servicenow); [ServiceNow Community — starting a ServiceNow team](https://www.servicenow.com/community/member-feedback-forum/starting-a-servicenow-team-what-should-we-get-right-early/td-p/3555625) | B |

## Freshworks — Freshdesk / Freshservice (CP-FW)

| ID | Pain point | Hypothesis result | Evidence | Grade |
|---|---|---|---|---|
| **CP-FW-01** | **Limits as complexity and volume grow.** Reviewers report automation and customisation limits when workflows get complex or volume is high, and difficulty scaling across many teams; G2 summary lists slow loading at high ticket volume. | Confirmed (reviews) | [G2 Freshdesk reviews](https://g2.com/products/freshdesk/reviews); [Software Advice — Freshdesk](https://www.softwareadvice.com/project-management/freshdesk-profile/) | B |
| **CP-FW-02** | **Shallower customisation and workflow depth.** UI, portal and workflow customisation described as limited. | Confirmed (reviews) | [G2 Freshdesk reviews p.2](https://g2.com/products/freshdesk/reviews?page=2); [HappyFox analysis (competitor)](https://blog.happyfox.com/freshdesk-ticketing-system-analysis) | B / C |
| **CP-FW-03** | **Reporting gaps on lower tiers.** Most frequent complaint; custom reporting only from Pro. | Confirmed | [G2 Freshdesk reviews](https://g2.com/products/freshdesk/reviews); [Freshdesk pricing](https://www.freshworks.com/freshdesk/trial/) | A / B |
| **CP-FW-04** | **Capabilities locked to higher tiers.** Custom objects, custom reporting and custom portals at Pro (~$55/agent/month); audit logs, sandbox, approval workflows and skill-based routing at Enterprise (~$89). Hypothesis "custom objects Enterprise-only" **killed** — they are at Pro. | Confirmed (reworded) | [Freshdesk pricing](https://www.freshworks.com/freshdesk/trial/); [Freshdesk Omni pricing](https://www.freshworks.com/freshdesk/omni/pricing/) | A |
| **CP-FW-05** | **Multi-product data silos.** Freshdesk and Freshservice keep separate user identity and ticket history; the native connector is basic one-way creation; field structures differ; sync hits API rate limits. | Confirmed | [Freshworks Community idea — unified user identity across Freshdesk and Freshservice](https://community.freshworks.com/ideas/unified-user-identity-and-ticket-overview-across-freshdesk-and-freshservice-42693); [Freshworks integration page](https://freshworks.com/freshdesk/collaboration/freshdesk-freshservice-integration); [Tray.ai connector notes](https://tray.ai/connectors/freshdesk-freshservice-integrations/) | A / C |
| **CP-FW-06** | **Account-wide API rate limits by plan.** Per-account per-minute limits regardless of agent count (trial 50/min; paid tiers commonly cited 100 / 400 / 700 per minute for Growth / Pro / Enterprise — paid figures not verified on the official table); 429 on breach. | Confirmed (mechanism A; paid numbers C) | [Freshdesk support — API rate limits](https://support.freshdesk.com/support/solutions/articles/225439-what-are-the-rate-limits-for-the-api-calls-to-freshdesk); [eesel guide](https://www.eesel.ai/blog/freshdesk-api-rate-limits) | A / C |

## Killed or unconfirmed hypotheses (do not cite)

| Hypothesis | Result | Why |
|---|---|---|
| Salesforce UI is cluttered | Unconfirmed | No measured source; only slowness (CP-SF-05) is sourced. |
| ServiceNow is slow (latency) for SMB | Unconfirmed | Evidence is about setup effort and learning curve (CP-SN-05), not response time. |
| Freshworks custom objects are Enterprise-only | Killed | Listed at Pro on the vendor pricing page (CP-FW-04). |
| Freshworks is slow to start | Killed | Reviewers consistently praise fast setup; this is Freshworks' strength, so time-to-value is **parity**, not 10x, against Freshworks. |
