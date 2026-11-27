package pipeline

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// Effective is the value of a record every surface that returns a record body
// shows (ADR-009 D11): the current version's defaults applied to the stored
// payload,
//
//	cur.ApplyDefaults(rec.Payload(fixed))
//
// where fixed is the fixed-column mapping of the version the row was written
// under (rec.SchemaVersion), not the current one. Field masks are applied
// after it, never before, so a role that cannot see a field never sees its
// default either. Audit diffs and snapshots keep the stored bytes.
//
// It is the one place outside the pipeline's own write path that may call
// metadata.Compiled.ApplyDefaults (TestApplyDefaultsOnlyInHelper), so a
// future surface (GraphQL, the embed SDK, an export) cannot show stored bytes
// by accident. The result is a new map; rec is not changed.
func Effective(cur *metadata.Compiled, rec *Record, fixed map[string]string) map[string]any {
	return cur.ApplyDefaults(rec.Payload(fixed))
}

// EffectiveUnder is Effective for a caller that holds only the entity key: it
// reads the fixed-column mapping of rec.SchemaVersion (cur when that is the
// current version, curVersion) and returns it with the effective value, as
// masks and ACL shaping need the mapping too.
func EffectiveUnder(ctx context.Context, tx pgx.Tx, defs entitytypes.DefinitionReader, entity string, rec *Record, curVersion int, cur *metadata.Compiled) (map[string]any, map[string]string, error) {
	fixed := cur.FixedColumns()
	if rec.SchemaVersion != curVersion {
		stored, err := defs.Version(ctx, tx, entity, rec.SchemaVersion)
		if err != nil {
			return nil, nil, fmt.Errorf("read stored definition: %w", err)
		}
		fixed = stored.FixedColumns()
	}
	return Effective(cur, rec, fixed), fixed, nil
}

// Defaults is the current version's top-level defaults as ApplyDefaults gives
// them for a record with no data (ADR-009 D12 reads them to decide whether a
// filter matches the default). Each call returns fresh values.
func Defaults(cur *metadata.Compiled) map[string]any {
	return cur.ApplyDefaults(nil)
}
