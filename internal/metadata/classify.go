package metadata

import (
	"reflect"
	"slices"
	"strconv"
)

// annotation reports whether a keyword only annotates and never changes what
// validates, so a change to it is Compatible.
func annotation(kw string) bool {
	switch kw {
	case "title", "description", "$comment", "examples":
		return true
	default:
		return false
	}
}

// classifier accumulates the highest class and the paths that raised it.
type classifier struct {
	breaking []string
	backfill []string
}

func (c *classifier) breaks(tokens ...string)    { c.breaking = append(c.breaking, pointer(tokens...)) }
func (c *classifier) backfills(tokens ...string) { c.backfill = append(c.backfill, pointer(tokens...)) }

func (c *classifier) result() (Compat, []string) {
	sorted := func(p []string) []string {
		p = slices.Clone(p)
		slices.Sort(p)
		return slices.Compact(p)
	}
	switch {
	case len(c.breaking) > 0:
		return Breaking, sorted(c.breaking)
	case len(c.backfill) > 0:
		return NeedsBackfill, sorted(c.backfill)
	default:
		return Compatible, nil
	}
}

// Classify compares next with prev, two versions of the same entity type,
// and returns the compatibility class (D-3-5). It fails closed: a change it
// cannot prove compatible is Breaking. The paths are the offending JSON
// pointers when Breaking, the properties that need a backfill when
// NeedsBackfill, and nil when Compatible. A removed property or state is
// named by its pointer in prev.
//
// Compatible: adding an optional property (with or without a default),
// widening an enum, adding a state or transition, changing an annotation
// (title, description, $comment, examples).
//
// NeedsBackfill: adding a required property that has a default; adding a
// default to an existing property.
//
// Breaking: everything else, including changing a type, narrowing an enum,
// adding a required property with no default, removing a property, changing
// FixedColumns, NumberPrefix or an existing default, removing a default,
// adding or removing x-nava-unique, making a property required or optional,
// removing a state or transition, changing the initial state, widening
// maxLength, dropping a pattern, and changing any unknown keyword.
func Classify(prev, next *Compiled) (Compat, []string, error) {
	if prev == nil || next == nil {
		return Breaking, nil, ErrNilCompiled
	}
	if prev.key != next.key {
		return Breaking, nil, ErrKeyMismatch
	}
	var c classifier
	if prev.numberPrefix != next.numberPrefix {
		c.breaks("number_prefix")
	}
	for _, name := range unionKeys(prev.fixedColumns, next.fixedColumns) {
		if prev.fixedColumns[name] != next.fixedColumns[name] {
			c.breaks("fixed_columns", name)
		}
	}
	classifyStates(prev.states, next.states, &c)
	classifySchema(prev.doc, next.doc, &c)
	class, paths := c.result()
	return class, paths, nil
}

// classifyStates: adding states or transitions is Compatible; removing
// either, or moving Initial, is Breaking.
func classifyStates(prev, next StateModel, c *classifier) {
	if prev.Initial != next.Initial {
		c.breaks("states", "initial")
	}
	for i, s := range prev.States {
		if !slices.Contains(next.States, s) {
			c.breaks("states", "states", strconv.Itoa(i))
		}
	}
	for i, t := range prev.Transitions {
		if !slices.Contains(next.Transitions, t) {
			c.breaks("states", "transitions", strconv.Itoa(i))
		}
	}
}

// classifySchema compares the root keyword by keyword. "properties" and
// "required" have their own rules; every other root keyword must be equal.
func classifySchema(prev, next map[string]any, c *classifier) {
	for _, kw := range unionKeys(prev, next) {
		if kw == "properties" || kw == "required" || annotation(kw) {
			continue
		}
		if !reflect.DeepEqual(prev[kw], next[kw]) {
			c.breaks("schema", kw)
		}
	}
	prevProps, _ := prev["properties"].(map[string]any)
	nextProps, _ := next["properties"].(map[string]any)
	prevReq, nextReq := stringSet(prev["required"]), stringSet(next["required"])

	for _, name := range unionKeys(prevReq, nextReq) {
		_, inPrev := prevProps[name]
		_, inNext := nextProps[name]
		if !inPrev && !inNext && prevReq[name] != nextReq[name] {
			c.breaks("schema", "required") // a required name with no property schema
		}
	}
	for _, name := range unionKeys(prevProps, nextProps) {
		p, inPrev := prevProps[name].(map[string]any)
		n, inNext := nextProps[name].(map[string]any)
		switch {
		case !inNext:
			c.breaks("schema", "properties", name) // removed
		case !inPrev:
			classifyNewProperty(name, n, nextReq[name], c)
		default:
			if prevReq[name] != nextReq[name] {
				c.breaks("schema", "required")
			}
			classifyProperty(name, p, n, c)
		}
	}
}

// classifyNewProperty: optional is Compatible, required with a default is
// NeedsBackfill, required without one is Breaking. Marking a new property
// x-nava-unique adds the marker, so it is Breaking.
func classifyNewProperty(name string, n map[string]any, required bool, c *classifier) {
	if isUnique(n) {
		c.breaks("schema", "properties", name, UniqueKeyword)
	}
	if !required {
		return
	}
	if _, ok := n["default"]; ok {
		c.backfills("schema", "properties", name)
	} else {
		c.breaks("schema", "properties", name)
	}
}

// classifyProperty compares one existing top-level property keyword by
// keyword.
func classifyProperty(name string, prev, next map[string]any, c *classifier) {
	at := func(kw string) []string { return []string{"schema", "properties", name, kw} }
	if isUnique(prev) != isUnique(next) {
		c.breaks(at(UniqueKeyword)...)
	}
	for _, kw := range unionKeys(prev, next) {
		pv, inPrev := prev[kw]
		nv, inNext := next[kw]
		switch {
		case annotation(kw) || kw == UniqueKeyword:
			// annotations never break; x-nava-unique compared above by value
		case kw == "default":
			switch {
			case !inPrev:
				c.backfills(at(kw)...) // old rows start reading it (TICK-360)
			case !inNext || !reflect.DeepEqual(pv, nv):
				c.breaks(at(kw)...)
			}
		case kw == "enum" && inPrev && inNext:
			if !isSuperset(nv, pv) {
				c.breaks(at(kw)...)
			}
		default:
			if !reflect.DeepEqual(pv, nv) {
				c.breaks(at(kw)...)
			}
		}
	}
}

func isUnique(p map[string]any) bool {
	v, _ := p[UniqueKeyword].(bool)
	return v
}

// isSuperset reports whether every element of the sub array is in the super
// array. Elements compare as decoded JSON, so 1 and 1.0 differ: fail closed.
func isSuperset(super, sub any) bool {
	sup, ok1 := super.([]any)
	sb, ok2 := sub.([]any)
	if !ok1 || !ok2 {
		return false
	}
	for _, v := range sb {
		if !slices.ContainsFunc(sup, func(w any) bool { return reflect.DeepEqual(v, w) }) {
			return false
		}
	}
	return true
}

func stringSet(v any) map[string]bool {
	set := map[string]bool{}
	arr, _ := v.([]any)
	for _, e := range arr {
		if s, ok := e.(string); ok {
			set[s] = true
		}
	}
	return set
}

// unionKeys returns the keys of both maps, sorted, so paths come out in a
// stable order.
func unionKeys[V any](a, b map[string]V) []string {
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		keys = append(keys, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
}
