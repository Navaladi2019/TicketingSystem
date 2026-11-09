package authz_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

type maskFixture struct {
	tenant, desk, admins uuid.UUID
	agent, requester     identity.Principal
	pol                  *authz.Policy
}

// newMaskFixture: "cost" is readable and writable by the desk group only,
// "notes" is readable by everyone but writable only by one named user.
func newMaskFixture() maskFixture {
	f := maskFixture{tenant: uuid.New(), desk: uuid.New(), admins: uuid.New()}
	f.agent = identity.Principal{TenantID: f.tenant, PrincipalID: uuid.New(), Groups: []uuid.UUID{f.desk}}
	f.requester = identity.Principal{TenantID: f.tenant, PrincipalID: uuid.New()}
	f.pol = &authz.Policy{Masks: []authz.Mask{
		{Field: "cost", Read: authz.Subject{Groups: []uuid.UUID{f.desk}}, Write: authz.Subject{Groups: []uuid.UUID{f.desk}}},
		{Field: "notes", Read: authz.Subject{Groups: []uuid.UUID{f.desk}, Users: []uuid.UUID{f.requester.PrincipalID}}, Write: authz.Subject{Users: []uuid.UUID{f.agent.PrincipalID}}},
	}}
	return f
}

func TestReadableAndWritableFollowGroupsAndUsers(t *testing.T) {
	f := newMaskFixture()
	cases := []struct {
		name            string
		p               identity.Principal
		field           string
		read, writeable bool
	}{
		{"agent in the group reads and writes cost", f.agent, "cost", true, true},
		{"requester outside the group neither reads nor writes cost", f.requester, "cost", false, false},
		{"a named user reads notes", f.requester, "notes", true, false},
		{"a named writer writes notes", f.agent, "notes", true, true},
		{"an unmasked field is open to everyone", f.requester, "title", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := authz.Readable(f.pol, c.p, c.field); got != c.read {
				t.Errorf("Readable = %v, want %v", got, c.read)
			}
			if got := authz.Writable(f.pol, c.p, c.field); got != c.writeable {
				t.Errorf("Writable = %v, want %v", got, c.writeable)
			}
		})
	}
}

// A2: a mask subject matches on Groups and Users only. A mask that names only
// Requester, Assignee or Shared hides the field from that requester, assignee
// or share holder, on read and on write: there is no row to match against.
func TestRowDependentMaskSubjectsNeverMatch(t *testing.T) {
	f := newMaskFixture()
	for _, s := range []authz.Subject{{Requester: true}, {Assignee: true}, {Shared: true}, {Requester: true, Assignee: true, Shared: true}} {
		pol := &authz.Policy{Masks: []authz.Mask{{Field: "secret_note", Read: s, Write: s}}}
		// The caller is the record's requester, assignee and share holder: a
		// rule would match it on all three, a mask must not.
		for _, p := range []identity.Principal{f.requester, f.agent} {
			if authz.Readable(pol, p, "secret_note") || authz.Writable(pol, p, "secret_note") {
				t.Fatalf("mask %+v admitted %v", s, p.PrincipalID)
			}
			if err := authz.CheckQueryFields(pol, p, []string{"secret_note"}); err == nil {
				t.Fatalf("mask %+v: a query on the field passed", s)
			}
			if got := authz.Shape(pol, p, map[string]any{"secret_note": "x"}); len(got) != 0 {
				t.Fatalf("mask %+v: Shape kept %v", s, got)
			}
		}
	}
}

// Two masks on one field combine: each must admit the principal.
func TestEveryMaskOnAFieldMustAdmit(t *testing.T) {
	f := newMaskFixture()
	pol := &authz.Policy{Masks: []authz.Mask{
		{Field: "cost", Read: authz.Subject{Groups: []uuid.UUID{f.desk}}},
		{Field: "cost", Read: authz.Subject{Groups: []uuid.UUID{f.admins}}},
	}}
	if authz.Readable(pol, f.agent, "cost") {
		t.Fatal("a desk agent outside the admins group read a field both masks guard")
	}
	f.agent.Groups = append(f.agent.Groups, f.admins)
	if !authz.Readable(pol, f.agent, "cost") {
		t.Fatal("a member of both groups could not read the field")
	}
}

func TestShapeDropsMaskedFieldsAndNeverMutates(t *testing.T) {
	f := newMaskFixture()
	data := map[string]any{"title": "t", "cost": 12, "notes": "n"}
	got := authz.Shape(f.pol, f.requester, data)
	if want := map[string]any{"title": "t", "notes": "n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Shape = %v, want %v", got, want)
	}
	if len(data) != 3 || data["cost"] != 12 {
		t.Fatalf("Shape changed its input: %v", data)
	}
	if got := authz.Shape(f.pol, f.agent, data); !reflect.DeepEqual(got, data) {
		t.Fatalf("Shape for an allowed caller = %v, want %v", got, data)
	}
	if authz.Shape(f.pol, f.agent, nil) != nil {
		t.Fatal("Shape(nil) is not nil")
	}
	if got := authz.MaskedFields(f.pol, f.requester); !reflect.DeepEqual(got, []string{"cost"}) {
		t.Fatalf("MaskedFields = %v, want [cost]", got)
	}
}

func TestCheckQueryFields(t *testing.T) {
	f := newMaskFixture()
	err := authz.CheckQueryFields(f.pol, f.requester, []string{"title", "cost", "notes"})
	var mf *authz.ErrMaskedField
	if !errors.As(err, &mf) || mf.Field != "cost" {
		t.Fatalf("CheckQueryFields on a masked field = %v, want *ErrMaskedField{cost}", err)
	}
	if err := authz.CheckQueryFields(f.pol, f.agent, []string{"title", "cost", "notes"}); err != nil {
		t.Fatalf("CheckQueryFields for an allowed caller = %v, want nil", err)
	}
	if err := authz.CheckQueryFields(f.pol, f.requester, []string{"title", "notes"}); err != nil {
		t.Fatalf("CheckQueryFields on unmasked fields = %v, want nil", err)
	}
	if err := authz.CheckQueryFields(f.pol, f.requester, nil); err != nil {
		t.Fatalf("CheckQueryFields with no field = %v, want nil", err)
	}
}

func TestWrittenFields(t *testing.T) {
	before := map[string]any{"title": "t", "cost": 1, "obj": map[string]any{"a": 1}}
	cases := []struct {
		name          string
		before, after map[string]any
		want          []string
	}{
		{"a create writes every sent field", nil, map[string]any{"title": "t", "cost": 1}, []string{"cost", "title"}},
		{"an unchanged value is no write", before, map[string]any{"title": "t", "cost": 1, "obj": map[string]any{"a": 1}}, nil},
		{"a changed value", before, map[string]any{"title": "t", "cost": 2, "obj": map[string]any{"a": 1}}, []string{"cost"}},
		{"a changed nested member writes its top-level field", before, map[string]any{"title": "t", "cost": 1, "obj": map[string]any{"a": 2}}, []string{"obj"}},
		{"a removed field", before, map[string]any{"title": "t", "obj": map[string]any{"a": 1}}, []string{"cost"}},
		{"an added field", before, map[string]any{"title": "t", "cost": 1, "obj": map[string]any{"a": 1}, "notes": "n"}, []string{"notes"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := authz.WrittenFields(c.before, c.after); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("WrittenFields = %v, want %v", got, c.want)
			}
		})
	}
}

func TestCheckWriteFields(t *testing.T) {
	f := newMaskFixture()
	err := authz.CheckWriteFields(f.pol, f.requester, nil, map[string]any{"title": "t", "notes": "n", "cost": 1}, nil)
	var nw *authz.ErrFieldNotWritable
	if !errors.As(err, &nw) || !reflect.DeepEqual(nw.Fields, []string{"cost", "notes"}) {
		t.Fatalf("CheckWriteFields = %v, want not writable [cost notes]", err)
	}
	if err := authz.CheckWriteFields(f.pol, f.agent, nil, map[string]any{"title": "t", "notes": "n", "cost": 1}, nil); err != nil {
		t.Fatalf("CheckWriteFields for an allowed writer = %v", err)
	}
	// An update that leaves a masked field as stored writes nothing to it.
	before := map[string]any{"title": "t", "cost": 1}
	if err := authz.CheckWriteFields(f.pol, f.requester, before, map[string]any{"title": "u", "cost": 1}, map[string]any{"title": "u"}); err != nil {
		t.Fatalf("an update that keeps cost = %v, want nil", err)
	}
	if err := authz.CheckWriteFields(nil, f.requester, nil, map[string]any{"cost": 1}, nil); err != nil {
		t.Fatalf("a nil policy has no masks: %v", err)
	}
}

// TICK-633 B1: on an update, a key of the caller's patch that the caller
// cannot read is a write whatever its value, so the answer never tells the
// caller whether it guessed the stored value, or (with null) whether the
// field is set. For a field the caller can read, an unchanged value is still
// no write.
func TestUnreadablePatchKeyIsAlwaysAWrite(t *testing.T) {
	f := newMaskFixture()
	before := map[string]any{"title": "t", "cost": 500}
	cases := []struct {
		name         string
		after, patch map[string]any
	}{
		{"the stored value, guessed", map[string]any{"title": "t", "cost": 500}, map[string]any{"cost": 500}},
		{"a wrong guess", map[string]any{"title": "t", "cost": 7}, map[string]any{"cost": 7}},
		{"null on a set field", map[string]any{"title": "t"}, map[string]any{"cost": nil}},
		{"null on an absent field", before, map[string]any{"absent_masked": nil}},
	}
	pol := &authz.Policy{Masks: append(f.pol.Masks, authz.Mask{Field: "absent_masked"})}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := authz.CheckWriteFields(pol, f.requester, before, c.after, c.patch)
			var nw *authz.ErrFieldNotWritable
			if !errors.As(err, &nw) || len(nw.Fields) != 1 {
				t.Fatalf("CheckWriteFields = %v, want one field not writable", err)
			}
		})
	}
	// notes is readable but not writable by the requester: sending back its
	// stored value is no write.
	stored := map[string]any{"notes": "n"}
	if err := authz.CheckWriteFields(f.pol, f.requester, stored, stored, map[string]any{"notes": "n"}); err != nil {
		t.Fatalf("a readable field sent back unchanged = %v, want nil", err)
	}
	// The agent may write cost: its unchanged value passes.
	if err := authz.CheckWriteFields(f.pol, f.agent, before, before, map[string]any{"cost": 500}); err != nil {
		t.Fatalf("the agent's unchanged cost = %v, want nil", err)
	}
}
