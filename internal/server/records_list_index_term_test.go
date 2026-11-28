package server

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// placeholders numbers placeholders from base+1, as listPlan.where does.
func placeholders(base int) func(v any, cast string) string {
	n := base
	return func(_ any, cast string) string {
		n++
		return "$" + strconv.Itoa(n) + "::" + cast
	}
}

// TestIndexTermOfReadsBackExistsParts checks the re-parse of the D7 term in
// indexTermOf (TICK-1009, TICK-1039): FieldArg and Cmp read back from d7 must
// be exactly what existsParts renders, for every comparison shape, or a
// WithListIndexTerm arm would bind the wrong placeholder.
func TestIndexTermOfReadsBackExistsParts(t *testing.T) {
	num := &pipeline.IndexField{ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), ValueType: fieldindex.TypeNum}
	text := &pipeline.IndexField{ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a2"), ValueType: fieldindex.TypeText}
	cases := []struct {
		name string
		fs   []listFilter
	}{
		{"one op, num eq", []listFilter{{prop: "pk", op: "eq", values: []any{10.0}, idx: num}}},
		{"one op, text eq (COLLATE)", []listFilter{{prop: "tier", op: "eq", values: []any{"v2"}, idx: text}}},
		{"two ops, num range", []listFilter{
			{prop: "pk", op: "gte", values: []any{3.0}, idx: num},
			{prop: "pk", op: "lt", values: []any{30.0}, idx: num},
		}},
		{"two ops, text in and lt", []listFilter{
			{prop: "tier", op: "in", values: []any{"v2", "v5"}, idx: text},
			{prop: "tier", op: "lt", values: []any{"v9"}, idx: text},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The same placeholder base for both, as the list renders the term once.
			d7 := existsTerm(c.fs, placeholders(4))
			table, fieldArg, cmp := existsParts(c.fs, placeholders(4))
			in := indexTermOf(c.fs, d7)
			if in.FieldArg != fieldArg {
				t.Errorf("FieldArg = %q, want %q (d7 %q)", in.FieldArg, fieldArg, d7)
			}
			if in.Cmp != cmp {
				t.Errorf("Cmp = %q, want %q (d7 %q)", in.Cmp, cmp, d7)
			}
			if in.Table != table {
				t.Errorf("Table = %q, want %q", in.Table, table)
			}
			if in.D7 != d7 || in.FieldID != c.fs[0].idx.ID || in.ValueType != c.fs[0].idx.ValueType {
				t.Errorf("D7, FieldID or ValueType not carried: %+v", in)
			}
			var ops []string
			var values [][]any
			for _, f := range c.fs {
				ops, values = append(ops, f.op), append(values, f.values)
			}
			if !reflect.DeepEqual(in.Ops, ops) || !reflect.DeepEqual(in.Values, values) {
				t.Errorf("Ops, Values = %v, %v, want %v, %v", in.Ops, in.Values, ops, values)
			}
		})
	}
}

// TestWhereIndexTermReplacesExists checks that listPlan.where hands each
// ready field's group to indexTerm once and puts its output in place of the
// D7 EXISTS, with the arguments D7 bound.
func TestWhereIndexTermReplacesExists(t *testing.T) {
	num := &pipeline.IndexField{ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), ValueType: fieldindex.TypeNum}
	pl := &listPlan{filters: []listFilter{
		{prop: "pk", op: "gte", values: []any{3.0}, idx: num},
		{prop: "pk", op: "lt", values: []any{30.0}, idx: num},
	}}
	plain, plainArgs := pl.where(2)
	if !strings.HasPrefix(plain, "EXISTS (SELECT 1 FROM rfv_num v") {
		t.Fatalf("without a hook where = %q, want the D7 EXISTS", plain)
	}
	var calls []IndexTerm
	pl.indexTerm = func(in IndexTerm) string {
		calls = append(calls, in)
		return "stub(" + in.FieldArg + ")"
	}
	got, args := pl.where(2)
	if got != "stub($3::uuid)" {
		t.Errorf("where = %q, want the stub's term in place of the EXISTS", got)
	}
	if len(calls) != 1 || calls[0].D7 != plain || len(calls[0].Ops) != 2 {
		t.Errorf("indexTerm calls = %+v, want one call for the field's two-op group with D7 %q", calls, plain)
	}
	if !reflect.DeepEqual(args, plainArgs) {
		t.Errorf("args = %v, want D7's %v", args, plainArgs)
	}
}
