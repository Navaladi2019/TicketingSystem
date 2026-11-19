package authz

import (
	"slices"
	"testing"
)

// generatedKinds are the node kinds the I5 agreement generator builds. The
// generator is test code (TICK-942 P11-10), so its coverage is listed here, in
// the same change that teaches it a kind.
func generatedKinds() map[string]bool { return map[string]bool{} }

// pendingRenderings are the registry gaps allowed until the agreement suite
// lands: TICK-937 P11-05 rendered every S30 kind in SQL and Go, and the
// generator covers none before TICK-942 P11-10. The list may only shrink.
// TestRegistryKindsAreRendered fails both when a SQLSafe, Available kind is
// missing a rendering that is not listed here and when a listed gap has been
// filled but is still listed, so it cannot hide a new kind or go stale.
//
// TODO(TODO-S30-01): TICK-942 empties the generator entries; S30 is not
// accepted while this list is non-empty.
func pendingRenderings() []RegistryGap {
	var out []RegistryGap
	for _, k := range []string{NodeAll, NodeAny, NodeNot, NodeRef, NodeMemberOf, NodeCmp, NodeShared} {
		out = append(out, RegistryGap{k, "generator"})
	}
	return out
}

// CI-11 (RFC-022 §5, ADR-010 D11): a SQLSafe, Available kind without a SQL
// rendering, a Go rendering and generator coverage fails the build.
func TestRegistryKindsAreRendered(t *testing.T) {
	t.Parallel()
	gaps := registryGaps(Kinds(), sqlRendered(), goRendered(), generatedKinds())
	pending := pendingRenderings()
	for _, g := range gaps {
		if !slices.Contains(pending, g) {
			t.Errorf("node kind %q is SQLSafe and Available but has no %s rendering (ADR-010 D11)", g.Kind, g.Missing)
		}
	}
	for _, p := range pending {
		if !slices.Contains(gaps, p) {
			t.Errorf("node kind %q now has its %s rendering: remove it from pendingRenderings", p.Kind, p.Missing)
		}
	}
}

// The check catches a kind added without renderers: a test-only SQLSafe kind
// turns it red until all three are present, and an unavailable or unsafe kind
// is exempt.
func TestRegistryCheckFailsAKindWithoutRenderers(t *testing.T) {
	t.Parallel()
	ks := append(Kinds(), NodeKind{Name: "test_only", SQLSafe: true, Available: true})
	all := func(extra ...string) map[string]bool {
		m := map[string]bool{}
		for _, k := range Kinds() {
			m[k.Name] = true
		}
		for _, e := range extra {
			m[e] = true
		}
		return m
	}
	full := all("test_only")
	if gaps := registryGaps(ks, full, full, full); len(gaps) != 0 {
		t.Fatalf("gaps with every rendering present = %v, want none", gaps)
	}
	for _, missing := range []string{"sql", "go", "generator"} {
		sql, goR, gen := full, full, full
		switch missing {
		case "sql":
			sql = all()
		case "go":
			goR = all()
		case "generator":
			gen = all()
		}
		got := registryGaps(ks, sql, goR, gen)
		if want := []RegistryGap{{"test_only", missing}}; !slices.Equal(got, want) {
			t.Errorf("missing %s: gaps = %v, want %v", missing, got, want)
		}
	}
	exempt := []NodeKind{{Name: "later", SQLSafe: true, Available: false}, {Name: "go_only", SQLSafe: false, Available: true}}
	if gaps := registryGaps(exempt, nil, nil, nil); len(gaps) != 0 {
		t.Errorf("unavailable and unsafe kinds are exempt, got gaps %v", gaps)
	}
}

// Every kind the decoder knows is in the registry, once, and the registry
// holds nothing the decoder does not know; under is SQLSafe but not
// Available until S31 (PLAN-011 D-11-5).
func TestRegistryMatchesTheGrammar(t *testing.T) {
	t.Parallel()
	grammar := []string{NodeAll, NodeAny, NodeNot, NodeRef, NodeMemberOf, NodeCmp, NodeShared, NodeUnder}
	var names []string
	for _, k := range Kinds() {
		names = append(names, k.Name)
		if !k.SQLSafe {
			t.Errorf("kind %q is not SQLSafe; every D3 kind is", k.Name)
		}
		if k.Available == (k.Name == NodeUnder) {
			t.Errorf("kind %q: Available = %v", k.Name, k.Available)
		}
	}
	if !slices.Equal(names, grammar) {
		t.Fatalf("registry = %v, want %v", names, grammar)
	}
	if k, ok := LookupKind(NodeUnder); !ok || k.Available {
		t.Errorf("LookupKind(under) = %+v, %v", k, ok)
	}
	if _, ok := LookupKind("fn"); ok {
		t.Error("LookupKind(fn) found a kind")
	}
	k := Kinds()
	k[0].SQLSafe = false
	if !Kinds()[0].SQLSafe {
		t.Error("Kinds returns the registry itself, not a copy")
	}
}
