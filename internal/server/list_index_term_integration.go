//go:build integration

package server

// WithListIndexTerm makes the list route render each ready data filter with
// term instead of the D7 EXISTS (existsTerm). It exists for the TICK-1039
// spike only (ADR-006-A1 A1-6): the adaptive shape and the promoted-column
// arm, measured through the real handler, ACL and RLS. The integration tag
// keeps it out of every binary that ships.
func WithListIndexTerm(term IndexTermFunc) Option {
	return func(o *options) { o.listIndexTerm = term }
}
