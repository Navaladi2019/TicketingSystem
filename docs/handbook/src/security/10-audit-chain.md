---
title: "10. Audit chain and offline verify"
section: security
status: delivered
normative: docs/architecture/audit-verify.md
sources: [docs/architecture/audit-verify.md, docs/architecture/audit-sealer.md, internal/audit, cmd/nava, docs/acceptance/S13/README.md, docs/adr/ADR-004-audit-log.md, docs/security/accepted-risks.md]
evidence: [docs/acceptance/S13/README.md]
adrs: [ADR-004]
stories: [S13, S15, S16]
tables: [audit_log, audit_chain_head]
packages: [internal/audit, cmd/nava]
tests: [TestAdversarialTenantRelabelBreaksChain, TestSealTenantIsolation, TestConcurrentSealersNeverDuplicateSeq, TestChainBreaks, TestEveryFieldIsHashed, TestGoldenVectorsTampered, TestTamperedExportsFail, TestFilteredExportsVerify, TestUnfilteredExportVerifies, TestSeqDuplicateAndMissing, TestAfterSeqAnchorsOnThatSeq, TestStubsOmitted, TestExitCodes, TestHelpDocumentsExitCodes, TestImportBoundary, TestOfflineImports, TestAppendTenantFromTransaction, TestAppendNoTenant, TestUnsealedTenantMismatch]
---

## Boundary

Each tenant's audit entries are sealed into **its own** hash chain. Anyone
holding an export can check it **offline**, with no access to the platform,
and any edit, deletion, reordering, relabelling to another tenant, or
unrecorded erasure is reported as tampered.

## Enforcement point

- **Append.** `internal/audit` takes the tenant from the transaction
  (`TenantOf`), never from a parameter; with no tenant it writes nothing.
- **Seal.** The sealer assigns each tenant's entries a gapless `seq` and links
  each entry hash to the previous one, per tenant, under that tenant's head
  lock. The tenant id is part of the hashed envelope, so moving an entry to
  another tenant breaks the chain.
- **Verify.** `nava audit verify <file>` (`cmd/nava`, package
  `internal/audit/verify`) streams the export once and recomputes every hash.
  It links no network, database or platform package other than the chain and
  verify packages, so it runs on an auditor's machine.

| Outcome | Exit | When |
|---|---|---|
| `pass` | 0 | every check holds |
| `tampered` | 1 | the first failing check, with its reason code |
| (usage) | 2 | bad arguments, unreadable file |
| `pending` | 3 | an erased entry is listed only by an unsealed erase entry |
| `unverifiable` | 4 | a `stubs=false` file, honestly flagged |
| `empty` | 5 | no sealed entry in the window |

The table is from the normative
[audit-verify.md](../../../architecture/audit-verify.md#outcomes-and-exit-codes).

```mermaid
sequenceDiagram
    participant F as Forger (edits an export, or the table)
    participant X as Export file (JSONL / CSV)
    participant V as nava audit verify (offline)
    F->>X: change a byte, delete a line, or relabel an entry to tenant B
    X->>V: stream once
    V->>V: recompute envelope and entry hashes, seq, anchor
    V-->>F: tampered (exit 1), with the reason code of the first failure
    Note over V: an erasure without a sealed erase entry is also tampered, an unsealed one is pending
```

## Proving tests

| Attack | Test |
|---|---|
| Relabel an entry to another tenant | `TestAdversarialTenantRelabelBreaksChain`, `TestUnsealedTenantMismatch` |
| Sealers of two tenants, or two sealers of one tenant | `TestSealTenantIsolation`, `TestConcurrentSealersNeverDuplicateSeq` |
| Change any field | `TestEveryFieldIsHashed`, `TestChainBreaks` |
| Byte edits, deleted or duplicated lines, re-anchoring | `TestGoldenVectorsTampered`, `TestTamperedExportsFail`, `TestSeqDuplicateAndMissing`, `TestAfterSeqAnchorsOnThatSeq` |
| Honest exports still verify | `TestFilteredExportsVerify`, `TestUnfilteredExportVerifies`, `TestStubsOmitted` |
| Exit codes are stable and documented | `TestExitCodes`, `TestHelpDocumentsExitCodes` |
| The verifier stays offline | `TestImportBoundary`, `TestOfflineImports` |
| Append takes the tenant only from the transaction | `TestAppendTenantFromTransaction`, `TestAppendNoTenant` |

## Fixture that turns it red

The golden tampered vectors (`internal/audit/chain/vectors`) are committed
export files that must each fail with a fixed reason code; an honest vector
set must pass. The S13 acceptance run shows the same by hand with the built
binary: an untouched export exits 0, a byte-edited one and a line-deleted one
exit 1 (see [S13 acceptance](../../../acceptance/S13/README.md)).

## Accepted risks

AR-TICK-409-01 is the main one: the chain has no external anchor in M1, so the
database owner or migrator could rewrite a whole tenant's chain consistently.

<!-- gen:accepted-risks ids=AR-TICK-409-01,AR-TICK-409-02,AR-TICK-422-01 -->

## Related

- [Exports and erasure](08-exports-erasure.md).
- [Audit tables](../database/audit.md).
