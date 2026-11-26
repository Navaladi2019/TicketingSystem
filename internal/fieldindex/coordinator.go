package fieldindex

// The per-tenant field-index coordinator (ADR-006 D10.5, PLAN-007 D-7-12,
// TICK-745). The names live here, in the import-free leaf, so the relay can
// signal the coordinator without importing the workflow package (and its
// database driver). The workflow itself is internal/fieldindex/backfill.

// The Temporal names. The Authorizer allow-lists the three workflow types
// (temporal-server/authz/policy.go): CoordinatorWorkflowType in
// platformWorkflowTypes, BackfillWorkflowType and RemoveWorkflowType in
// childOnlyWorkflowTypes too, so only a coordinator may start a backfill or a
// removal.
const (
	// CoordinatorWorkflowType runs one tenant's backfills and removals, one
	// at a time, in arrival order.
	CoordinatorWorkflowType = "field_index.coordinator"
	// BackfillWorkflowType is FieldIndexBackfill: one field's barrier,
	// batches, flip to ready and barrier (D9, D10). A coordinator child only.
	BackfillWorkflowType = "field_index.backfill.run"
	// RemoveWorkflowType is FieldIndexRemove (D11, TICK-750): one field's
	// barrier, side-row deletes and registry DELETE. A coordinator child only.
	RemoveWorkflowType = "field_index.remove.run"
	// SignalEnqueue is the one signal the coordinator listens on. Its body is
	// an EnqueueItem.
	SignalEnqueue = "field_index.enqueue"
)

// The operations an EnqueueItem asks for.
const (
	OpBackfill = "backfill"
	OpRemove   = "remove"
)

// EnqueueItem is the body of SignalEnqueue. The relay builds it from a
// field_index.backfill outbox row: FieldID from the payload, RequestedVersion
// from target_seq (outbox_payload_check admits only UUID payload values) and
// OutboxID from the row's id. A field_index.remove row gives an OpRemove
// item the same way (TICK-750).
//
// The coordinator dedups on (FieldID, RequestedVersion) within a run and on
// OutboxID across continue-as-new. It never trusts Op on its own: what a
// queued field needs is re-read from the registry row before its child
// starts (internal/fieldindex/backfill, Classify).
//
// IDs are canonical UUID strings: this package imports nothing, so
// internal/metadata may import it (internal/metadata import_test).
type EnqueueItem struct {
	Op               string `json:"op"`
	FieldID          string `json:"field_id"`
	RequestedVersion int    `json:"requested_version"`
	OutboxID         string `json:"outbox_id"`
}

// CoordinatorWorkflowID is the coordinator's workflow id for tenant, a
// canonical UUID string: t:<tenant>:field_index.coordinator, one per tenant.
func CoordinatorWorkflowID(tenant string) string {
	return "t:" + tenant + ":" + CoordinatorWorkflowType
}
