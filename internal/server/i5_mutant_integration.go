//go:build integration

package server

// WithListDataTerm makes the list route render each data-filter value with
// term instead of containment (r.data @> arg). It exists for the I5 mutants
// of the isolation suite only (TICK-900): a deliberately broken filter
// compiler the suite must catch. The integration tag keeps it out of every
// binary that ships.
func WithListDataTerm(term func(arg string) string) Option {
	return func(o *options) { o.listDataTerm = term }
}

// WithLegacyScan adds (typeKey, field) pairs that an unflagged list filter
// may read through the legacy JSONB scan, on top of
// fieldindex.LegacyScanFields (ADR-006 D6). It exists for the I5 data-filter
// oracle of the isolation suite only (TICK-900, TICK-1023): the oracle checks
// the JSONB-scan compiler on a generated type whose fields no registry flags.
// The integration tag keeps it out of every binary that ships.
func WithLegacyScan(allowed func(typeKey, field string) bool) Option {
	return func(o *options) { o.legacyScan = allowed }
}
