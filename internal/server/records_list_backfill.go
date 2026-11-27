package server

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// unionRemainingMax is the most queued and failed records for which the
// UNION ALL shape is used; above it, and at the cap of 1,001 that means
// "more than 1,000", the OR shape is (ADR-009 D12).
const unionRemainingMax = 1000

// resolveWindow decides, once per list request, in the request's transaction
// and as nava_app (SELECT only), whether a backfill window touches the data
// filters of pl and in which shape (ADR-009 D12, D9).
//
// The window is open from a publish that adds a default until the type's
// newest schema_backfills row (the highest target_version) is done. While it
// is open, a record may still lack the default in storage, so a filter on a
// field in that job's default_keys, whose predicate is true for the default,
// also returns the records that lack the field:
//
//   - pending or filling: the queue may not be filled yet, so no count of it
//     can be trusted: the OR shape;
//   - running or failed_records with more than 1,000 remaining: the OR shape;
//   - running or failed_records with 1,000 or fewer remaining: the UNION ALL
//     shape, which walks only the queued and the failed records;
//   - two or more filtered fields in the window: the OR shape for all of
//     them, never a UNION ALL (each field would need its own arm).
//
// A request with no data filter on the side tables reads nothing here. The
// legacy JSONB scan keeps its own absent-matches rule (I5).
func (h *handlers) resolveWindow(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, def *metadata.Compiled, pl *listPlan) error {
	if !pl.hasIndexFilter() {
		return nil
	}
	b, err := h.store.NewestBackfillOf(ctx, tx, tenantID, defID)
	if err != nil || b == nil || !b.Open() {
		return err
	}
	defaults := pipeline.Defaults(def)
	fields := map[string]any{}
	for _, f := range pl.filters {
		if f.col != "" || f.idx == nil {
			continue
		}
		d, has := defaults[f.prop]
		if has && slices.Contains(b.DefaultKeys, f.prop) {
			fields[f.prop] = d
		}
	}
	if len(fields) == 0 {
		return nil
	}
	pl.win = &filterWindow{fields: fields, union: len(fields) == 1 && unionShape(b), target: b.TargetVersion}
	return nil
}

// unionShape reports whether the job's state and remaining count call for the
// UNION ALL shape for a single filtered field.
func unionShape(b *entitytypes.BackfillStatus) bool {
	return (b.State == "running" || b.State == "failed_records") && b.Remaining <= unionRemainingMax
}

// queued renders "the record is in the newest job's pending queue or its
// failures": the join the second arm of the UNION ALL shape is driven by
// (ADR-009 D12). The store binds $1 to the tenant and $2 to the entity
// definition of every list statement; the job's key is (tenant, definition,
// target version) in both tables, which have the (tenant_id, record_id)
// indexes this reads through.
func queued(target int, next func(v any, cast string) string) string {
	v := next(target, "integer")
	return "r.id IN (SELECT p.record_id FROM schema_backfill_pending p WHERE p.tenant_id = $1 AND p.entity_definition_id = $2 AND p.target_version = " + v +
		" UNION ALL SELECT f.record_id FROM schema_backfill_failures f WHERE f.tenant_id = $1 AND f.entity_definition_id = $2 AND f.target_version = " + v + ")"
}
