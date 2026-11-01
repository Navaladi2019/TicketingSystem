package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

func baseEnvelope() Envelope {
	return Envelope{
		TenantID:    uuid.MustParse("0190f3a0-0000-7000-8000-000000000001"),
		ID:          uuid.MustParse("0190f3a0-0000-7000-8000-000000000101"),
		OccurredAt:  time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC),
		Domain:      "record",
		Action:      "update",
		SubjectType: "record",
		SubjectID:   uuid.MustParse("0190f3a0-0000-7000-8000-0000000000e1"),
		ActorSystem: ptr("workflow"),
		Source:      "workflow",
	}
}

// TestCanonicalEnvelopeLiteral pins the full canonical form of one envelope,
// written by hand from the spec: sorted keys, no whitespace, every NULL
// present.
func TestCanonicalEnvelopeLiteral(t *testing.T) {
	const want = `{"action":"update","actor_principal_id":null,"actor_system":"workflow","backfilled":false,` +
		`"causation_depth":0,"causation_id":null,"change_set_id":null,"content_sha256":null,"diff_sha256":null,` +
		`"diff_truncated":false,"domain":"record","entity_definition_id":null,` +
		`"id":"0190f3a0-0000-7000-8000-000000000101","occurred_at":"2026-11-02T10:00:00.000000Z",` +
		`"request_id":null,"source":"workflow","source_ref":null,"subject_id":"0190f3a0-0000-7000-8000-0000000000e1",` +
		`"subject_type":"record","tenant_id":"0190f3a0-0000-7000-8000-000000000001","version_after":null,"version_before":null}`
	got, err := CanonicalEnvelope(baseEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	sum, err := EnvelopeSHA256(baseEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if sum != sha256.Sum256([]byte(want)) {
		t.Fatal("EnvelopeSHA256 is not SHA-256 of the canonical bytes")
	}
}

// mutators changes each envelope field in turn. TestEveryFieldIsHashed
// requires one per struct field.
//
//nolint:gochecknoglobals // read-only test table
var mutators = map[string]func(*Envelope){
	"TenantID":           func(e *Envelope) { e.TenantID = uuid.MustParse("0190f3a0-0000-7000-8000-000000000002") },
	"ID":                 func(e *Envelope) { e.ID = uuid.MustParse("0190f3a0-0000-7000-8000-000000000102") },
	"OccurredAt":         func(e *Envelope) { e.OccurredAt = e.OccurredAt.Add(time.Microsecond) },
	"Domain":             func(e *Envelope) { e.Domain = "config" },
	"Action":             func(e *Envelope) { e.Action = "create" },
	"SubjectType":        func(e *Envelope) { e.SubjectType = "entity_definition" },
	"SubjectID":          func(e *Envelope) { e.SubjectID = uuid.MustParse("0190f3a0-0000-7000-8000-0000000000e2") },
	"EntityDefinitionID": func(e *Envelope) { e.EntityDefinitionID = ptr(uuid.MustParse("0190f3a0-0000-7000-8000-0000000000d1")) },
	"VersionBefore":      func(e *Envelope) { e.VersionBefore = ptr[int64](0) },
	"VersionAfter":       func(e *Envelope) { e.VersionAfter = ptr[int64](0) },
	"ActorPrincipalID":   func(e *Envelope) { e.ActorPrincipalID = ptr(uuid.MustParse("0190f3a0-0000-7000-8000-0000000000a1")) },
	"ActorSystem":        func(e *Envelope) { e.ActorSystem = nil },
	"Source":             func(e *Envelope) { e.Source = "system" },
	"SourceRef":          func(e *Envelope) { e.SourceRef = ptr("x") },
	"ChangeSetID":        func(e *Envelope) { e.ChangeSetID = ptr(uuid.Nil) },
	"CausationID":        func(e *Envelope) { e.CausationID = ptr(uuid.Nil) },
	"CausationDepth":     func(e *Envelope) { e.CausationDepth = 1 },
	"RequestID":          func(e *Envelope) { e.RequestID = ptr("r") },
	"Backfilled":         func(e *Envelope) { e.Backfilled = true },
	"ContentSHA256":      func(e *Envelope) { e.ContentSHA256 = &[32]byte{} },
	"DiffSHA256":         func(e *Envelope) { e.DiffSHA256 = &[32]byte{} },
	"DiffTruncated":      func(e *Envelope) { e.DiffTruncated = true },
}

// TestEveryFieldIsHashed: changing any envelope field, including NULL to a
// zero value, changes envelope_sha256, and every struct field is in the
// canonical form.
func TestEveryFieldIsHashed(t *testing.T) {
	typ := reflect.TypeFor[Envelope]()
	if typ.NumField() != len(mutators) || typ.NumField() != len(envelopeFields) {
		t.Fatalf("Envelope has %d fields, %d mutators, %d canonical members; add the new field to all three",
			typ.NumField(), len(mutators), len(envelopeFields))
	}
	base, err := EnvelopeSHA256(baseEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	for i := range typ.NumField() {
		name := typ.Field(i).Name
		mut, ok := mutators[name]
		if !ok {
			t.Errorf("no mutator for %s", name)
			continue
		}
		e := baseEnvelope()
		mut(&e)
		got, err := EnvelopeSHA256(e)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got == base {
			t.Errorf("changing %s did not change envelope_sha256", name)
		}
	}
}

func TestCanonicalKeysAreSortedAndComplete(t *testing.T) {
	b, err := CanonicalEnvelope(baseEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for dec.More() {
		k, _ := dec.Token()
		keys = append(keys, k.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	if len(keys) != len(envelopeFields) {
		t.Fatalf("%d members, want %d (NULL is emitted, never omitted)", len(keys), len(envelopeFields))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Fatalf("keys out of order: %q before %q", keys[i-1], keys[i])
		}
	}
	for _, k := range []string{"txid", "diff", "diff_salt", "seq", "prev_hash", "entry_hash", "sealed_at", "erased_at", "erasure_ref"} {
		if bytes.Contains(b, []byte(`"`+k+`"`)) {
			t.Errorf("%s must not be in the envelope", k)
		}
	}
}

// RFC 8785 §3.2.3: members sort by UTF-16 code units, which differs from
// UTF-8 byte order above U+FFFF.
func TestSortByUTF16(t *testing.T) {
	keys := []string{"�", "\U0001F600", "a", "é", "A", "1"}
	sortByUTF16(keys, func(s string) string { return s })
	want := []string{"1", "A", "a", "é", "\U0001F600", "�"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("got %q, want %q", keys, want)
	}
}

func TestJCSStrings(t *testing.T) {
	cases := []struct{ in, want string }{
		{`plain`, `"plain"`},
		{`q"b\`, `"q\"b\\"`},
		{"\b\f\n\r\t", `"\b\f\n\r\t"`},
		{"\x00\x01\x0b\x1f", `"\u0000\u0001\u000b\u001f"`},
		{"\x7f", "\"\x7f\""}, // DEL is not escaped
		{"</x>&", `"</x>&"`}, // no HTML escaping
		{"  ", "\"  \""},     // raw, unlike encoding/json
		{"é中😀", `"é中😀"`},     // raw UTF-8
		{"/", `"/"`},         // solidus not escaped
	}
	for _, c := range cases {
		var b bytes.Buffer
		writeJCSString(&b, c.in)
		if b.String() != c.want {
			t.Errorf("%q: got %s, want %s", c.in, b.String(), c.want)
		}
		var back string
		if err := json.Unmarshal(b.Bytes(), &back); err != nil || back != c.in {
			t.Errorf("%q: does not round-trip (%v)", c.in, err)
		}
	}
}

func TestNumberForms(t *testing.T) {
	for _, n := range []int64{0, 1, -1, 2147483647, -2147483648, MaxSafeInteger, -MaxSafeInteger} {
		e := baseEnvelope()
		e.VersionAfter = &n
		b, err := CanonicalEnvelope(e)
		if err != nil {
			t.Fatalf("%d: %v", n, err)
		}
		if !bytes.Contains(b, []byte(`"version_after":`+strconv.FormatInt(n, 10)+`,`)) {
			t.Errorf("%d: not written as a plain integer: %s", n, b)
		}
	}
	for _, n := range []int64{MaxSafeInteger + 1, -MaxSafeInteger - 1} {
		e := baseEnvelope()
		e.VersionAfter = &n
		if _, err := CanonicalEnvelope(e); !errors.Is(err, ErrInvalidEnvelope) {
			t.Errorf("%d: err = %v, want ErrInvalidEnvelope", n, err)
		}
	}
	for _, s := range []string{"1.0", "1e0", "-0", "+1", "01", "0x1", `"1"`, "9007199254740992"} {
		if _, err := decInt(json.RawMessage(s)); err == nil {
			t.Errorf("decInt(%s) accepted a non-canonical integer", s)
		}
	}
}

func TestTimestamps(t *testing.T) {
	e := baseEnvelope()
	ist := time.FixedZone("IST", 5*3600+1800)
	e.OccurredAt = time.Date(2026, 11, 2, 15, 30, 0, 123456000, ist)
	b, err := CanonicalEnvelope(e)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"occurred_at":"2026-11-02T10:00:00.123456Z"`)) {
		t.Fatalf("timestamp not normalised to UTC with six digits: %s", b)
	}
	e.OccurredAt = e.OccurredAt.Add(time.Nanosecond)
	if _, err := CanonicalEnvelope(e); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("sub-microsecond timestamp: err = %v", err)
	}
}

func TestTextRules(t *testing.T) {
	for name, mut := range map[string]func(*Envelope){
		"empty required": func(e *Envelope) { e.Domain = "" },
		"empty nullable": func(e *Envelope) { e.RequestID = ptr("") },
		"invalid utf8":   func(e *Envelope) { e.RequestID = ptr("a\xffb") },
		"lone surrogate": func(e *Envelope) { e.RequestID = ptr("\xed\xa0\x80") },
	} {
		e := baseEnvelope()
		mut(&e)
		if _, err := CanonicalEnvelope(e); !errors.Is(err, ErrInvalidEnvelope) {
			t.Errorf("%s: err = %v, want ErrInvalidEnvelope", name, err)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	e := baseEnvelope()
	for _, mut := range mutators {
		mut(&e)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var back Envelope
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	b2, err := CanonicalEnvelope(back)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, b2) {
		t.Fatalf("round trip changed the canonical bytes\n%s\n%s", b, b2)
	}
	// Member order and whitespace in the input do not matter.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	pretty, err := json.MarshalIndent(m, "", "   ")
	if err != nil {
		t.Fatal(err)
	}
	var again Envelope
	if err := json.Unmarshal(pretty, &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(again, back) {
		t.Fatal("reformatted input parsed differently")
	}
}

func TestUnmarshalRejectsTrailingData(t *testing.T) {
	b, err := CanonicalEnvelope(baseEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	var e Envelope
	if err := e.UnmarshalJSON(append(b, []byte(` {}`)...)); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("err = %v", err)
	}
	if err := e.UnmarshalJSON([]byte(`[]`)); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("err = %v", err)
	}
}

func TestGenesisIsZero(t *testing.T) {
	if Genesis != [32]byte{} {
		t.Fatal("Genesis must be 32 zero bytes")
	}
}

func TestEntryHash(t *testing.T) {
	prev := [32]byte{1}
	env := [32]byte{2}
	want := sha256.Sum256(append(prev[:], env[:]...))
	if EntryHash(prev, env) != want {
		t.Fatal("EntryHash is not SHA-256(prev ‖ envelope_sha256)")
	}
	if EntryHash(env, prev) == want {
		t.Fatal("EntryHash must be order-sensitive")
	}
}

// diff_sha256 covers salt ‖ the stored bytes exactly: no re-encoding, so
// whitespace and key order matter.
func TestDiffSHA256(t *testing.T) {
	salt := [16]byte{9, 8, 7}
	diff := []byte(`[ {"op":"add","path":"/a","value":1} ]`)
	want := sha256.Sum256(append(append([]byte{}, salt[:]...), diff...))
	if DiffSHA256(salt, diff) != want {
		t.Fatal("DiffSHA256 is not SHA-256(salt ‖ diff)")
	}
	if DiffSHA256(salt, []byte(`[{"op":"add","path":"/a","value":1}]`)) == want {
		t.Fatal("a whitespace change must change diff_sha256")
	}
	if DiffSHA256([16]byte{}, diff) == want {
		t.Fatal("the salt must be hashed")
	}
}

func TestParseHex(t *testing.T) {
	good := strings.Repeat("ab", 32)
	if _, err := ParseHash32(good); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{strings.Repeat("AB", 32), good[:62], good + "00", strings.Repeat("zz", 32)} {
		if _, err := ParseHash32(s); !errors.Is(err, ErrInvalidEnvelope) {
			t.Errorf("ParseHash32(%q) err = %v", s, err)
		}
	}
	if _, err := ParseSalt16(strings.Repeat("0f", 16)); err != nil {
		t.Fatal(err)
	}
}

// TestImportBoundary keeps the package linkable by the offline verifier
// (PLAN-005 D-5-6): standard library and google/uuid only, no database or
// network packages.
func TestImportBoundary(t *testing.T) {
	allowed := map[string]bool{
		"bytes": true, "crypto/sha256": true, "encoding/hex": true, "encoding/json": true,
		"errors": true, "fmt": true, "io": true, "slices": true, "strconv": true,
		"time": true, "unicode/utf16": true, "unicode/utf8": true,
		"github.com/google/uuid": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !allowed[p] {
				t.Errorf("%s imports %s; the chain package may import only the standard library and google/uuid", name, p)
			}
		}
	}
}
