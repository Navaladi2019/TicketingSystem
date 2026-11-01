package entitytypes

import "testing"

// TICK-378: a nil fixed-columns map is stored as {}, not null, which
// entity_definition_versions_fixed_columns_check would refuse.
func TestEncodeFixedColumnsIsAlwaysAnObject(t *testing.T) {
	for _, c := range []struct {
		name string
		in   map[string]string
		want string
	}{
		{"nil", nil, `{}`},
		{"empty", map[string]string{}, `{}`},
		{"mapped", map[string]string{"state": "state"}, `{"state":"state"}`},
	} {
		got, err := encodeFixedColumns(c.in)
		if err != nil || string(got) != c.want {
			t.Errorf("%s: encodeFixedColumns = %s, %v; want %s, nil", c.name, got, err, c.want)
		}
	}
}
