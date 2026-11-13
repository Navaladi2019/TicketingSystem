# Example record types

Three record types that show the platform is generic: the same engine, three
domains. Each uses only what is on `main`: fields, states, the access policy,
field masks and the default record number (`<prefix><7 digits>`).

| Domain | Type | Fields | States | Masked field |
|---|---|---|---|---|
| ITSM | `incident` | title (`short_description`), priority, `affected_service`, assignment group | new → in_progress → resolved → closed | none |
| Lending | `loan_application` | applicant name, amount, term (months), income | submitted → under_review → approved / rejected | `income`: only the admins group reads or writes it |
| Immigration | `visa_application` | applicant, passport number, visa type, travel date | received → documents_check → decided | `passport_number`: only the admins group reads or writes it |

Files in this folder:

- `<type>.json`: the body of `POST /api/v1/entity-types` (`key`, `name`, `number_prefix`, `definition`).
- `loan_application.access.json`, `visa_application.access.json`: the body of `PUT /api/v1/entity-types/{key}/access`. `__ADMINS__` and `__AUDITORS__` are replaced with tenant A's group ids by the script. The rules: admins read and write, auditors read, the requester reads its own record and edits it while it is still in the first state.
- `seed-examples.sh`: publishes all three into tenant A and creates one record of each.

## Load them

On a local stack that has been seeded with `docs/acceptance/guide/tools/seed.sh`:

```sh
sh docs/handbook/examples/seed-examples.sh
```

The script is idempotent. `incident` already exists in tenant A (the seeded
template), so it is kept and gets `affected_service` as its next version; it
keeps its default ACL. A re-run replays the records under fixed
`Idempotency-Key`s, so it adds no second record.

## In the product tour

Use `loan_application` for the masking step, not only `incident`:

1. Sign in as `admin@tenant-a.test` and `GET /api/v1/records/loan_application`. The record shows `income`.
2. Sign in as `auditor@tenant-a.test` and read the same record: it is readable (the auditors rule), and `income` is absent from the body. A list filter on `income` answers `400 masked_field`.
3. As the auditor, `PATCH` the record: the write is refused (no write rule). As any caller outside the admins group, a write that touches `income` answers `422 field_not_writable`.
4. `visa_application` does the same for `passport_number`, with a three-step state model (`received` → `documents_check` → `decided`).

## Not built yet

What these domains need that the platform lacks today. The example types leave it out.

- **Document attachments** (passport scan, payslips, supporting documents): there is no file storage or attachment field. The visa check state `documents_check` is a state only.
- **Custom number formats** such as `LOAN-2027-00042` or `VISA-UK-0001`: the format compiler exists, but types are not linked to a format yet; every type numbers as `<prefix><7 digits>`.
- **Masks that depend on the record or the caller's role** (for example, the applicant sees their own `income`): a mask matches groups and users only, never the requester, assignee or a share (A2). The applicant therefore cannot see or submit a masked field; intake is by the admins group.
- **Domain groups** (loan officers, case officers): tenant A has only `admins` and `auditors` in the seeded realm, so the example policies use those two. Add groups in Keycloak and publish a new access policy to use others.
- **Decision rules and approvals** (credit scoring, approve/reject with a reason, a four-eyes check): transitions are plain state changes, with no conditions on a transition and no reason field enforced.
- **Typed validation** beyond JSON Schema (passport number checksum, IBAN, cross-field rules such as amount against income).
