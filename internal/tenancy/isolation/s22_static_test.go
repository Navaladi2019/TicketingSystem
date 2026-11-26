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
