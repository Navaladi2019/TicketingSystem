package authz

import (
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Field masks (PLAN-003 S10, P3-33; ADR-001 I5: masks apply on write and on
// read). A Mask names one top-level schema property: it is hidden from every
// principal its Read subject does not name, and unwritable for every
// principal its Write subject does not name.
//
// Masks are row-independent (contract amendment A2): a mask subject matches
// on Groups and Users only. Requester, Assignee and Shared in a mask never
// match, so such a mask hides the field from that requester, assignee or
// share holder too (fail closed). So one answer holds per principal and field
// everywhere it is asked: on a single read, on history, on the audit
// timeline and on a query (CheckQueryFields). Several masks on one field
// combine: the field is readable (writable) only when every one of them
// admits the principal.
//
// A nil policy has no masks; it already denies every read and write
// (PolicyReader), so nothing reaches the shaping layer under it.

// ErrMaskedField is a query that filters, sorts, counts or aggregates on a
// field masked for the caller (400 masked_field). Field is the first such
// field, in the order the query named them.
type ErrMaskedField struct {
	Field string
}

func (e *ErrMaskedField) Error() string { return "authz: field " + e.Field + " is masked" }

// ErrFieldNotWritable is a write that sets or changes a field the caller may
// not write (422 field_not_writable). Fields are the top-level properties,
// sorted.
type ErrFieldNotWritable struct {
	Fields []string
}

func (e *ErrFieldNotWritable) Error() string {
	return "authz: fields not writable: " + strings.Join(e.Fields, ", ")
}

// Readable reports whether p may read field under pol.
func Readable(pol *Policy, p identity.Principal, field string) bool {
	return admits(pol, p, field, func(m Mask) Subject { return m.Read })
}

// Writable reports whether p may write field under pol.
func Writable(pol *Policy, p identity.Principal, field string) bool {
	return admits(pol, p, field, func(m Mask) Subject { return m.Write })
}

// Shape returns data without the top-level properties p may not read. It
// never mutates data: the result is a new map that shares the values data
// holds. A nil data is nil.
func Shape(pol *Policy, p identity.Principal, data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	out := maps.Clone(data)
	for _, f := range MaskedFields(pol, p) {
		delete(out, f)
	}
	return out
}

// MaskedFields lists, sorted and without repeats, the fields p may not read
// under pol.
func MaskedFields(pol *Policy, p identity.Principal) []string {
	if pol == nil {
		return nil
	}
	var out []string
	for _, m := range pol.Masks {
		if !Readable(pol, p, m.Field) {
			out = append(out, m.Field)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// CheckQueryFields returns *ErrMaskedField for the first of fields p may not
// read under pol, and nil when it may read every one. A query calls it with
// every top-level property it filters, sorts, counts or aggregates on, before
// any SQL runs (P3-34). A field is a top-level property name, not a pointer.
func CheckQueryFields(pol *Policy, p identity.Principal, fields []string) error {
	for _, f := range fields {
		if !Readable(pol, p, f) {
			return &ErrMaskedField{Field: f}
		}
	}
	return nil
}

// WrittenFields lists, sorted, the top-level properties a write sets or
// changes: on a create (before nil) every property of after, on an update
// every property whose presence or value differs between before and after.
// after must be the caller's data only (on an update, before merged with the
// caller's patch), never the payload with defaults applied: a server-applied
// default is not a caller write.
func WrittenFields(before, after map[string]any) []string {
	var out []string
	for k, a := range after {
		if b, ok := before[k]; !ok || !reflect.DeepEqual(a, b) {
			out = append(out, k)
		}
	}
	for k := range before { // a nil before (a create) removes nothing
		if _, ok := after[k]; !ok {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// CheckWriteFields returns *ErrFieldNotWritable naming, sorted, every field
// p writes and may not write under pol, and nil when there is none. The
// fields p writes are WrittenFields(before, after) plus every top-level key
// of patch (the caller's merge patch, nil on a create) that p may not read,
// whatever its value (TICK-633 B1): if an unchanged value of a field p cannot
// read were no write, the answer to a patch would tell p whether it guessed
// the stored value (or, with null, whether the field is set). "Unchanged is
// no write" holds only for the fields p can read, so a client can send back
// the body it was given.
func CheckWriteFields(pol *Policy, p identity.Principal, before, after, patch map[string]any) error {
	written := WrittenFields(before, after)
	for k := range patch {
		if !Readable(pol, p, k) && !slices.Contains(written, k) {
			written = append(written, k)
		}
	}
	slices.Sort(written)
	var bad []string
	for _, f := range written {
		if !Writable(pol, p, f) {
			bad = append(bad, f)
		}
	}
	if len(bad) > 0 {
		return &ErrFieldNotWritable{Fields: bad}
	}
	return nil
}

// admits reports whether every mask on field names p in the subject sel
// picks. A field with no mask admits everyone.
func admits(pol *Policy, p identity.Principal, field string, sel func(Mask) Subject) bool {
	if pol == nil {
		return true
	}
	for _, m := range pol.Masks {
		if m.Field == field && !maskSubjectNames(sel(m), p) {
			return false
		}
	}
	return true
}

// maskSubjectNames is the row-independent subject match of A2: Groups and
// Users only. Requester, Assignee and Shared are deliberately ignored.
func maskSubjectNames(s Subject, p identity.Principal) bool {
	return intersects(s.Groups, p.Groups) || (p.PrincipalID != uuid.Nil && slices.Contains(s.Users, p.PrincipalID))
}
