//go:build integration

package server

import "github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"

// WithListIndexTerm makes the list route render each ready data filter with
// term instead of the D7 EXISTS (existsTerm). It exists for the TICK-1039
// spike only (ADR-006-A1 A1-6): the adaptive shape and the promoted-column
// arm, measured through the real handler, ACL and RLS. The integration tag
// keeps it out of every binary that ships.
func WithListIndexTerm(term IndexTermFunc) Option {
	return func(o *options) { o.listIndexTerm = term }
}

// WithListQuery lets fn rewrite the list statement's parts just before the
// store runs them. It exists for the TICK-1205 spike only (S-b1): the
// per-group merge page, one ordered UNION ALL arm per visible assignment
// group, measured through the real handler, ACL and RLS. The integration
// tag keeps it out of every binary that ships.
func WithListQuery(fn func(q *pipeline.ListQuery)) Option {
	return func(o *options) { o.listQuery = fn }
}
