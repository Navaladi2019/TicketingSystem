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
