package server

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const listTestSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object", "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string"},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "group": {"type": "string", "format": "uuid"},
    "assignee": {"type": "string", "format": "uuid"},
    "cost": {"type": "number"},
    "count": {"type": "integer"},
    "urgent": {"type": "boolean"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "token": {"type": "string", "x-nava-secret": true}
  }
}`

func listTestDef(t *testing.T) *metadata.Compiled {
	t.Helper()
	c, err := metadata.Compile(metadata.Definition{
		Key: "job", NumberPrefix: "JOB", Schema: json.RawMessage(listTestSchema),
		FixedColumns: map[string]string{"state": metadata.ColumnState, "group": metadata.ColumnAssignmentGroupID, "assignee": metadata.ColumnAssigneeID},
		States:       metadata.StateModel{Initial: "new", States: []string{"new", "closed"}, Transitions: []metadata.Transition{{From: "new", To: "closed"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func qs(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func strp(s string) *string { return &s }

func wantProblemCode(t *testing.T, err error, code problem.Code) {
	t.Helper()
	var pe *problemError
	if !errors.As(err, &pe) || pe.code != code {
		t.Fatalf("error = %v, want problem %s", err, code)
	}
}

func TestPlanListValidation(t *testing.T) {
	t.Parallel()
	def := listTestDef(t)
	bad := map[string]struct{ q, sort string }{
		"unknown filter field":        {q: "filter[nope]=1"},
		"secret field":                {q: "filter[token]=x"},
		"nested path":                 {q: "filter[a.b]=1"},
		"unknown operator":            {q: "filter[cost][gt]=1"},
		"bare filter":                 {q: "filter=1"},
		"repeated value":              {q: "filter[cost]=1&filter[cost]=2"},
		"non-uuid group":              {q: "filter[group]=x"},
		"integer given text":          {q: "filter[count]=1.5"},
		"number given hex":            {q: "filter[cost]=0x10"},
		"boolean given yes":           {q: "filter[urgent]=yes"},
		"array property":              {q: "filter[tags]=a"},
		"empty state":                 {q: "filter[state]="},
		"in with one bad value":       {q: "filter[count][in]=1,x"},
		"in with too many values":     {q: "filter[short_description][in]=" + strings.Repeat("a,", maxInValues) + "a"},
		"sort on a data property":     {sort: "cost"},
		"sort on an unknown property": {sort: "-nope"},
	}
	for name, c := range bad {
		var sort *string
		if c.sort != "" {
			sort = strp(c.sort)
		}
		if _, perr := planList(def, qs(t, c.q), sort, nil); perr == nil || perr.code != problem.ValidationFailed {
			t.Errorf("%s: planList = %v, want 400 validation_failed", name, perr)
		}
	}
}

func TestPlanListAccepts(t *testing.T) {
	t.Parallel()
	def := listTestDef(t)
	pl, perr := planList(def, qs(t, "filter[state]=new&filter[count][in]=1,2&filter[urgent]=true&filter[cost]=1.50&filter[group]="+uuid.NewString()), strp("-state"), nil)
	if perr != nil {
		t.Fatal(perr)
	}
	if pl.sortProp != "state" || pl.sortCol != metadata.ColumnState || !pl.desc || pl.limit != defaultRecordsLimit || len(pl.filters) != 5 {
		t.Fatalf("plan = %+v", pl)
	}
	sql, args := pl.where(4)
	if want := "$5::"; !strings.Contains(sql, want) || strings.Contains(sql, "$4") {
		t.Errorf("placeholders must start after the base: %s", sql)
	}
	if len(args) != 6 {
		t.Errorf("args = %d, want 6 (cost, count x2, group, state, urgent)", len(args))
	}
	// The default sort is -updated_at.
	pl, _ = planList(def, qs(t, ""), nil, nil)
	if pl.sortProp != "updated_at" || !pl.desc {
		t.Fatalf("default plan = %+v", pl)
	}
	// A value is data, never SQL.
	pl, _ = planList(def, qs(t, "filter[short_description]="+url.QueryEscape("x' OR 1=1 --")), nil, nil)
	if sql, _ := pl.where(0); strings.Contains(sql, "OR 1=1") {
		t.Errorf("filter value reached the SQL text: %s", sql)
	}
}

// maskPolicy masks the named fields from everyone: a mask naming nobody.
func maskPolicy(fields ...string) *authz.Policy {
	pol := &authz.Policy{}
	for _, f := range fields {
		pol.Masks = append(pol.Masks, authz.Mask{Field: f})
	}
	return pol
}

// One test per entry point that names a field: each answers 400
// masked_field from the pure planning step, which has no database to reach.
func TestMaskedFieldInFilterIsRefusedBeforeSQL(t *testing.T) {
	t.Parallel()
	pl, perr := planList(listTestDef(t), qs(t, "filter[cost]=1"), nil, nil)
	if perr != nil {
		t.Fatal(perr)
	}
	wantProblemCode(t, checkPlan(maskPolicy("cost"), identity.Principal{}, pl), problem.MaskedField)
}

func TestMaskedFieldInFilterInIsRefusedBeforeSQL(t *testing.T) {
	t.Parallel()
	pl, perr := planList(listTestDef(t), qs(t, "filter[count][in]=1,2"), nil, nil)
	if perr != nil {
		t.Fatal(perr)
	}
	wantProblemCode(t, checkPlan(maskPolicy("count"), identity.Principal{}, pl), problem.MaskedField)
}

func TestMaskedFixedColumnInFilterIsRefusedBeforeSQL(t *testing.T) {
	t.Parallel()
	// assignment_group_id is checked under its property name, group.
	pl, perr := planList(listTestDef(t), qs(t, "filter[group]="+uuid.NewString()), nil, nil)
	if perr != nil {
		t.Fatal(perr)
	}
	wantProblemCode(t, checkPlan(maskPolicy("group"), identity.Principal{}, pl), problem.MaskedField)
}

func TestMaskedFieldInSortIsRefusedBeforeSQL(t *testing.T) {
	t.Parallel()
	pl, perr := planList(listTestDef(t), qs(t, ""), strp("-assignee"), nil)
	if perr != nil {
		t.Fatal(perr)
	}
	wantProblemCode(t, checkPlan(maskPolicy("assignee"), identity.Principal{}, pl), problem.MaskedField)
}

func TestMaskedFieldInDefaultSortIsRefusedBeforeSQL(t *testing.T) {
	t.Parallel()
	pl, perr := planList(listTestDef(t), qs(t, ""), nil, nil)
	if perr != nil {
		t.Fatal(perr)
	}
	if got := pl.fields(); len(got) != 1 || got[0] != "updated_at" {
		t.Fatalf("fields = %v: the default sort must be checked", got)
	}
	wantProblemCode(t, checkPlan(maskPolicy("updated_at"), identity.Principal{}, pl), problem.MaskedField)
}

func TestUnmaskedPlanPasses(t *testing.T) {
	t.Parallel()
	pl, _ := planList(listTestDef(t), qs(t, "filter[cost]=1"), strp("created_at"), nil)
	if err := checkPlan(maskPolicy("other"), identity.Principal{}, pl); err != nil {
		t.Fatal(err)
	}
}

func TestKeysetPredicate(t *testing.T) {
	t.Parallel()
	pl, _ := planList(listTestDef(t), qs(t, ""), strp("-created_at"), nil)
	id := uuid.NewString()
	v := "2026-11-09 03:39:54.145534+00"
	if !pl.validPos(listPos{Sort: "created_at", Desc: true, Val: &v, ID: id}) {
		t.Error("a cursor for this sort must be valid")
	}
	for name, pos := range map[string]listPos{
		"other sort":      {Sort: "updated_at", Desc: true, Val: &v, ID: id},
		"other direction": {Sort: "created_at", Desc: false, Val: &v, ID: id},
		"bad id":          {Sort: "created_at", Desc: true, Val: &v, ID: "x"},
		"bad time":        {Sort: "created_at", Desc: true, Val: strp("yesterday"), ID: id},
	} {
		if pl.validPos(pos) {
			t.Errorf("%s: cursor accepted", name)
		}
	}
	sql, args := pl.keyset(listPos{Sort: "created_at", Desc: true, Val: &v, ID: id}, 7)
	if !strings.Contains(sql, "$8::timestamptz") || !strings.Contains(sql, "$9::uuid") || !strings.Contains(sql, "(r.created_at, r.id) < ($8::timestamptz, $9::uuid)") || len(args) != 2 {
		t.Errorf("keyset = %s %v", sql, args)
	}
}

func TestKeysetNullSegments(t *testing.T) {
	t.Parallel()
	id := uuid.NewString()
	v := uuid.NewString()
	for _, c := range []struct {
		sort string
		pos  listPos
		want string
	}{
		{"-assignee", listPos{Sort: "assignee", Desc: true, ID: id}, "(r.assignee_id IS NOT NULL OR r.id < $4::uuid)"},
		{"assignee", listPos{Sort: "assignee", ID: id}, "(r.assignee_id IS NULL AND r.id > $4::uuid)"},
		{"-assignee", listPos{Sort: "assignee", Desc: true, Val: &v, ID: id}, "((r.assignee_id, r.id) < ($4::uuid, $5::uuid))"},
		{"assignee", listPos{Sort: "assignee", Val: &v, ID: id}, "(((r.assignee_id, r.id) > ($4::uuid, $5::uuid)) OR r.assignee_id IS NULL)"},
	} {
		pl, perr := planList(listTestDef(t), qs(t, ""), strp(c.sort), nil)
		if perr != nil {
			t.Fatal(perr)
		}
		if got, _ := pl.keyset(c.pos, 3); got != c.want {
			t.Errorf("%s %+v:\n got %s\nwant %s", c.sort, c.pos, got, c.want)
		}
	}
}
