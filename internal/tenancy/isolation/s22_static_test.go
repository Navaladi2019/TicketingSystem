package isolation

import "testing"

// TestS22JSONBOnDataForms pins what S22 attack 4's scan counts as a JSONB
// read of records.data (TICK-753): every operator form, the cast form and
// the jsonb_* function forms, and nothing that merely names data.
func TestS22JSONBOnDataForms(t *testing.T) {
	for _, s := range []string{
		"r.data @> $1", "data ? 'prio'", "NOT (r.data ? $2)", "r.data ->> 'prio' = $1", "data -> 'a'",
		"data #>> '{a}'", "data #> '{a}'", "data ?| $1", "data ?& $1", "data @? '$.a'", "data @@ '$.a == 1'",
		"r.data::jsonb ->> 'prio'", "data :: jsonb @> $1",
		"jsonb_extract_path_text(data, 'prio')", "jsonb_path_exists(r.data, '$.prio')", "jsonb_typeof( x.data )",
	} {
		if !s22JSONBOnData.MatchString(s) {
			t.Errorf("%q is a JSONB read of data, not matched", s)
		}
	}
	for _, s := range []string{
		"r.data, r.causation_origin_id", "x.data AS data", "metadata ->> 'a'", "jsonb_build_object('note', $2::text)",
		"SELECT data FROM records", "jsonb_extract_path_text(metadata, 'a')",
	} {
		if s22JSONBOnData.MatchString(s) {
			t.Errorf("%q is not a JSONB read of data, matched", s)
		}
	}
}

// TestS22AbsentSiteAllowlist: the window's absent term is allowed in
// fieldAbsent of records_list_index.go and nowhere else; any other JSONB
// operator on data in that function stays red.
func TestS22AbsentSiteAllowlist(t *testing.T) {
	src := func(fn, lit string) string {
		return "package p\nfunc " + fn + "() string { return `" + lit + "` }\n"
	}
	const frag = "NOT (r.data ? $1::text)"
	for _, c := range []struct {
		name, rel, src string
		want           int
	}{
		{"allowed site", s22AbsentFile, src("fieldAbsent", frag), 0},
		{"same fragment, other function", s22AbsentFile, src("other", frag), 1},
		{"same fragment, other file", "internal/server/records_list.go", src("fieldAbsent", frag), 1},
		{"other operator in the allowed function", s22AbsentFile, src("fieldAbsent", "r.data @> $1"), 1},
		{"second operator beside the fragment", s22AbsentFile, src("fieldAbsent", frag+" AND r.data ->> 'a' = 'b'"), 1},
	} {
		got, err := s22ScanSource(c.rel, c.rel, []byte(c.src))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(got) != c.want {
			t.Errorf("%s: %d findings %v, want %d", c.name, len(got), got, c.want)
		}
	}
}
