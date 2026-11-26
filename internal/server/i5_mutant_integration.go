//go:build integration

package server

// WithListIndexTerm makes the list route pass each ready data filter's
// side-table term through term before it joins the statement. It exists for
// the I5 filter mutants of the isolation suite only (TICK-900, TICK-752): a
// deliberately broken side-table compiler the suite must catch. The
// integration tag keeps it out of every binary that ships.
func WithListIndexTerm(term IndexTerm) Option {
	return func(o *options) { o.listIndexTerm = term }
}
