package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// separators is the text Go 1.27's encoding/json escapes inside MarshalJSON
// output and Go 1.26 did not, plus the HTML characters, all raw in JCS.
const separators = "x\u2028y\u2029z<&>"

func envelopeWithSeparators(t *testing.T) (Envelope, []byte) {
	t.Helper()
	e := baseEnvelope()
	s := separators
	e.RequestID = &s
	canon, err := CanonicalEnvelope(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(canon, []byte(separators)) {
		t.Fatalf("canonical form lost the raw separators: %s", canon)
	}
	return e, canon
}

type framed struct {
	Kind     string    `json:"kind"`
	Seq      *int64    `json:"seq"`
	Envelope Envelope  `json:"envelope"`
	Ptr      *Envelope `json:"ptr"`
	Note     string    `json:"note"`
	ID       uuid.UUID `json:"id"`
	Tags     []string  `json:"tags"`
	Raw      []byte    `json:"raw"`
	Untagged int
	Skipped  int `json:"-"`
	hidden   int
}

func TestMarshalJSONLWritesEnvelopeVerbatim(t *testing.T) {
	t.Parallel()
	e, canon := envelopeWithSeparators(t)
	seq := int64(7)
	v := framed{Kind: "entry", Seq: &seq, Envelope: e, Ptr: &e, Note: separators,
		ID: uuid.MustParse("0190f3a0-0000-7000-8000-000000000001"), Raw: []byte{1, 2}, Untagged: 3, Skipped: 4, hidden: 5}

	got, err := MarshalJSONL(v)
	if err != nil {
		t.Fatal(err)
	}
	// note is a plain string: encoding/json escapes U+2028/U+2029 there on
	// every Go release, so it keeps that encoding.
	note := `"x\u2028y\u2029z<&>"`
	want := `{"kind":"entry","seq":7,"envelope":` + string(canon) + `,"ptr":` + string(canon) +
		`,"note":` + note + `,"id":"0190f3a0-0000-7000-8000-000000000001","tags":null,"raw":"AQI=","Untagged":3}` + "\n"
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestMarshalJSONLMapsAndLists(t *testing.T) {
	t.Parallel()
	e, canon := envelopeWithSeparators(t)
	v := map[string]any{"b": []any{e, nil, 1.5}, "a": map[string]any{"envelope": &e}, "c": (*Envelope)(nil)}
	got, err := MarshalJSONL(v)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"a":{"envelope":` + string(canon) + `},"b":[` + string(canon) + `,null,1.5],"c":null}` + "\n"
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// Without envelopes the helper writes exactly what json.Encoder writes.
func TestMarshalJSONLMatchesEncoderWithoutEnvelopes(t *testing.T) {
	t.Parallel()
	type inner struct {
		A *string `json:"a"`
		B []int   `json:"b"`
	}
	s := "q\"b\\<&>é\u2028"
	v := map[string]any{"z": inner{A: &s, B: []int{1, 2}}, "y": []inner{{}}, "x": true, "w": json.Number("12")}
	for _, indent := range []bool{false, true} {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		var got []byte
		var err error
		if indent {
			enc.SetIndent("", "  ")
			got, err = MarshalIndentJSONL(v, "", "  ")
		} else {
			got, err = MarshalJSONL(v)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := enc.Encode(v); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, b.Bytes()) {
			t.Fatalf("indent %v\ngot  %s\nwant %s", indent, got, b.Bytes())
		}
	}
}

func TestMarshalIndentJSONLKeepsEnvelopeRaw(t *testing.T) {
	t.Parallel()
	e, _ := envelopeWithSeparators(t)
	got, err := MarshalIndentJSONL(struct {
		Cases []Envelope `json:"cases"`
	}{[]Envelope{e}}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte(`"request_id": "`+separators+`"`)) {
		t.Fatalf("indented envelope was re-escaped:\n%s", got)
	}
}

func TestMarshalJSONLRefusesWhatItCannotMirror(t *testing.T) {
	t.Parallel()
	type withOption struct {
		A int `json:"a,omitempty"`
	}
	type Embedded struct{ A int }
	type withEmbedded struct{ Embedded }
	for name, v := range map[string]any{
		"tag option": withOption{},
		"embedded":   withEmbedded{},
		"int keys":   map[int]string{1: "a"},
	} {
		if _, err := MarshalJSONL(v); !errors.Is(err, errUnsupportedJSON) {
			t.Errorf("%s: err = %v, want errUnsupportedJSON", name, err)
		}
	}
}

func TestMarshalJSONLRejectsInvalidEnvelope(t *testing.T) {
	t.Parallel()
	e := baseEnvelope()
	nul := "a\x00b"
	e.RequestID = &nul
	if _, err := MarshalJSONL(struct {
		E Envelope `json:"envelope"`
	}{e}); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("err = %v, want ErrInvalidEnvelope", err)
	}
}
