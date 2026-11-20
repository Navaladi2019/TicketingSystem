// Package fieldindex names the field-index registry's value types and states
// (ADR-006 D2, P7-01). They are the values of record_index_fields'
// value_type and state CHECKs (db/migrations/20261120140000_field_index_tables.sql);
// TestP701RegistryChecksMatchFieldindex in internal/db/dbtest fails when the
// two drift apart.
package fieldindex

// ValueType is record_index_fields.value_type: which typed side table holds a
// field's values.
type ValueType string

// The four value types (ADR-006 D3, CTO ruling 1).
const (
	// TypeText is a JSON string, stored in rfv_text as text COLLATE "C".
	TypeText ValueType = "text"
	// TypeNum is a finite JSON number, stored in rfv_num as double precision.
	TypeNum ValueType = "num"
	// TypeTS is an RFC 3339 timestamp or a format: date day, stored in rfv_ts
	// as timestamptz.
	TypeTS ValueType = "ts"
	// TypeBool is a JSON boolean, stored in rfv_bool.
	TypeBool ValueType = "bool"
)

// ValueTypes returns every value type, in the order of the CHECK.
func ValueTypes() []ValueType { return []ValueType{TypeText, TypeNum, TypeTS, TypeBool} }

// SideTable returns the side table that holds values of type v, and false
// for a value that is not one of the four types. The name comes from this
// fixed set only, never from v itself, so a caller may splice it into SQL.
func (v ValueType) SideTable() (string, bool) {
	switch v {
	case TypeText:
		return "rfv_text", true
	case TypeNum:
		return "rfv_num", true
	case TypeTS:
		return "rfv_ts", true
	case TypeBool:
		return "rfv_bool", true
	}
	return "", false
}

// State is record_index_fields.state (ADR-006 D2, D6, D10, D11).
type State string

// The three states.
const (
	// StateBackfilling: the field is flagged and its existing values are
	// being copied (D10). The trigger keeps new writes; a filter answers 422
	// field_not_ready.
	StateBackfilling State = "backfilling"
	// StateReady: the side table holds every value; filters use it.
	StateReady State = "ready"
	// StateRemoving: the flag is gone and its side rows are being deleted
	// (D11). Filters answer 400 field_not_filterable; the trigger skips it.
	StateRemoving State = "removing"
)

// States returns every state, in the order of the CHECK.
func States() []State { return []State{StateBackfilling, StateReady, StateRemoving} }

// LegacyField names one data field of a seed template type.
type LegacyField struct {
	// Type is the seed template type key (incident, request, change).
	Type string
	// Field is the top-level property key.
	Field string
}

// LegacyScanFields is the fixed allow-list of seed-type data fields that a
// list may still filter on through the JSONB scan while they are not flagged
// x-nava-filterable (PLAN-007 D-7-6). It is keyed by (seed template type,
// field key), never by field key alone, so a tenant type with its own
// "priority" never reaches the scan (D-7-14). S22 P7-12 flags these fields and
// deletes this list and store.DataFilterTimeout. Nothing is added to it.
//
//nolint:gochecknoglobals // constant allow-list
var LegacyScanFields = map[LegacyField]struct{}{
	{Type: "incident", Field: "priority"}: {},
	{Type: "request", Field: "priority"}:  {},
	{Type: "change", Field: "risk"}:       {},
}

// LegacyScan reports whether a filter on field of the entity type typeKey may
// use the legacy JSONB scan.
func LegacyScan(typeKey, field string) bool {
	_, ok := LegacyScanFields[LegacyField{Type: typeKey, Field: field}]
	return ok
}
