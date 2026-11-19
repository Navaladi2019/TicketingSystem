package authz

import "slices"

// The node registry (ADR-010 D11, RFC-022 D10 and CI-11; TICK-936 P11-04).
// Every node kind of the expression tree is listed here once. Access rules
// accept only SQLSafe kinds. A kind that is SQLSafe and Available must have a
// SQL rendering, a Go rendering and coverage in the I5 agreement generator:
// TestRegistryKindsAreRendered fails the build otherwise, so one wrong "true"
// cannot admit a kind with only one proven rendering.

// NodeKind is one entry of the registry. (Named NodeKind, not Kind as PLAN-011
// C2 writes it, because Kind already names a TenantRef's id kind.)
type NodeKind struct {
	// Name is the node's JSON key.
	Name string
	// SQLSafe kinds may appear in an access rule (D11).
	SQLSafe bool
	// Available is false for a kind the grammar decodes but the publish route
	// refuses with not_available, until its renderers ship.
	Available bool
}

// kinds is the registry, in D3's order.
func kinds() []NodeKind {
	return []NodeKind{
		{NodeAll, true, true},
		{NodeAny, true, true},
		{NodeNot, true, true},
		{NodeRef, true, true},
		{NodeMemberOf, true, true},
		{NodeCmp, true, true},
		{NodeShared, true, true},
		// under ships in S31 (PLAN-011 D-11-5): decoded, refused at publish.
		{NodeUnder, true, false},
	}
}

// Kinds returns the registry, in D3's order.
func Kinds() []NodeKind { return kinds() }

// LookupKind returns the registry entry of a node kind name.
func LookupKind(name string) (NodeKind, bool) {
	ks := kinds()
	i := slices.IndexFunc(ks, func(k NodeKind) bool { return k.Name == name })
	if i < 0 {
		return NodeKind{}, false
	}
	return ks[i], true
}

// sqlRendered and goRendered are the kinds the format-2 tree renders in SQL
// and in Go. The tree planner (TICK-937 P11-05) adds a kind to both in the
// same change that renders it; the agreement generator (TICK-942 P11-10) lists
// the kinds it builds in the test-only generatedKinds.
func sqlRendered() map[string]bool { return map[string]bool{} }

func goRendered() map[string]bool { return map[string]bool{} }

// RegistryGap is a SQLSafe, Available kind missing one rendering.
type RegistryGap struct {
	Kind    string
	Missing string // "sql", "go" or "generator"
}

// registryGaps lists every SQLSafe, Available kind of ks that one of the
// three sets lacks. Unavailable and unsafe kinds are exempt.
func registryGaps(ks []NodeKind, sql, goR, gen map[string]bool) []RegistryGap {
	var out []RegistryGap
	for _, k := range ks {
		if !k.SQLSafe || !k.Available {
			continue
		}
		for _, m := range []struct {
			set  map[string]bool
			name string
		}{{sql, "sql"}, {goR, "go"}, {gen, "generator"}} {
			if !m.set[k.Name] {
				out = append(out, RegistryGap{k.Name, m.name})
			}
		}
	}
	return out
}
