package numbering_test

import (
	"errors"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

func TestParseFormat(t *testing.T) {
	c, err := numbering.ParseFormat([]byte(`{"template":"{prefix}-{yyyy}{MM}{seq:4}","overflow":"refuse","time_zone":"Asia/Kolkata"}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Head != "{prefix}-{yyyy}{MM}" || c.Width != 4 || c.Overflow != numbering.Refuse || c.TimeZone != "Asia/Kolkata" {
		t.Errorf("ParseFormat = %+v", c)
	}
	d, err := numbering.ParseFormat([]byte(`{"template":"{prefix}{seq:7}","overflow":"grow","time_zone":""}`))
	if err != nil || d.CompiledSHA256 != numbering.Default().CompiledSHA256 || d.Epoch != 0 || d.TimeZone != "UTC" {
		t.Errorf("ParseFormat(default) = %+v, %v; want Default()", d, err)
	}
}

func TestParseFormatRefuses(t *testing.T) {
	for doc, field := range map[string]string{
		`not json`:                          "/",
		`[]`:                                "/",
		`{"overflow":"grow"}`:               "/template",
		`{"template":"{prefix}{seq:7}"}`:    "/overflow",
		`{"template":"","overflow":"grow"}`: "/template",
		`{"template":"{prefix} {seq:7}","overflow":"grow"}`:                           "/template",
		`{"template":"{prefix}{seq:7}","overflow":"wrap"}`:                            "/overflow",
		`{"template":"{prefix}{seq:7}","overflow":"grow","tenant_id":"b"}`:            "/tenant_id",
		`{"template":"{prefix}{seq:7}","overflow":"grow","time_zone":"1/x"}`:          "/time_zone",
		`{"template":"{prefix}{seq:7}","overflow":"grow","time_zone":"Mars/Olympus"}`: "time_zone",
		`{"template":"{prefix}X{seq:7}","overflow":"grow"}`:                           "template",
	} {
		_, err := numbering.ParseFormat([]byte(doc))
		var fe *numbering.FormatError
		if !errors.As(err, &fe) || fe.Field != field || fe.Status() != 422 {
			t.Errorf("ParseFormat(%s) = %v, want a 422 FormatError on %s", doc, err, field)
		}
	}
}
