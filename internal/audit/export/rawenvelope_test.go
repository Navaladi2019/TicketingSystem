package export_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/export"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

// trickyText holds every character whose escaping encoding/json has changed
// between releases or that HTML escaping would touch: U+2028, U+2029, < > &.
// RFC 8785 JCS keeps all of them raw.
const trickyText = "a\u2028b\u2029c<d>e&f"

// TestEntryLineCarriesCanonicalEnvelopeBytes pins that each entry line holds
// chain.Envelope.MarshalJSON's bytes verbatim, whatever the Go release's
// encoding/json does to Marshaler output (TICK-970: Go 1.27 escapes U+2028
// and U+2029 there, which the verifier then rejects as non-canonical).
func TestEntryLineCarriesCanonicalEnvelopeBytes(t *testing.T) {
	t.Parallel()
	l := newFakeLogWith(t, 3, func(e *read.Entry) { e.ActorSystem = new(trickyText) })
	b, err := run(t, l, 2, export.Request{Stubs: true})
	if err != nil {
		t.Fatal(err)
	}
	mustPass(t, b)

	entries := 0
	for _, line := range lines(b) {
		if !strings.HasPrefix(line, `{"kind":"entry"`) {
			continue
		}
		seq := entries + 1
		canon, err := envOf(l.rows[seq-1]).MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(canon, []byte(trickyText)) {
			t.Fatalf("canonical envelope lost the raw text: %s", canon)
		}
		// The envelope member sits between sealed_at and diff (section 3).
		start := strings.Index(line, `,"envelope":`) + len(`,"envelope":`)
		end := strings.Index(line, `,"diff":`)
		if start < len(`,"envelope":`) || end < start {
			t.Fatalf("entry line %d has no envelope member: %s", seq, line)
		}
		if got := line[start:end]; got != string(canon) {
			t.Errorf("entry %d envelope bytes\n got %q\nwant %q", seq, got, canon)
		}
		entries++
	}
	if entries != 3 {
		t.Fatalf("found %d entry lines, want 3", entries)
	}
	if !bytes.HasSuffix(b, []byte("}\n")) || bytes.Contains(b, []byte("\r")) {
		t.Fatalf("lines must end in a bare \n")
	}
}

// TestEnvelopeLineEqualsMarshalJSON pins the shared helper on its own: an
// Envelope encodes to exactly its MarshalJSON bytes.
func TestEnvelopeLineEqualsMarshalJSON(t *testing.T) {
	t.Parallel()
	l := newFakeLogWith(t, 1, func(e *read.Entry) { e.ActorSystem = new(trickyText) })
	env := envOf(l.rows[0])
	canon, err := env.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	got, err := chain.MarshalJSONL(env)
	if err != nil {
		t.Fatal(err)
	}
	if want := append(canon, '\n'); !bytes.Equal(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}
