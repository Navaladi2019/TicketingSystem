## 2026-10-31: PLAN-003 approved; golang-zod not adopted
- **Decided by:** nava approved PLAN-003 on TICK-332. The Technical Lead decided on the library question nava raised with the approval ("use golang-zod by leandroluk if needed"); see PLAN-003 D-3-24.
- **Decision:** JSON validation stays on `santhosh-tekuri/jsonschema/v6` for tenant entity schemas (D-3-18) and on kin-openapi for request bodies. `github.com/leandroluk/golang-zod` is not added: it has no license, is four days old (v0.13.0, one maintainer), needs Go 1.27.1 against our Go 1.26 pin, and is code-first while Epic 04 validates tenant-supplied JSON Schema 2020-12 documents.
- **Trigger:** an OSI license and a v1 tag, plus the platform on Go 1.27. Then a one-run spike against the JSON-Schema-Test-Suite 2020-12.
- **Released on approval:** P3-01, P3-02, P3-03, P3-04 (S8, branch `TICK-333/entity-types-versioned`) and P3-20, P3-21, P3-26 (S11a, branch `TICK-334/workflow-interpreter`); P3-00 was already with the Solution Architect.
- **Dissent:** none.
