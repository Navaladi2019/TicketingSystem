package server_test

import (
	"strings"
	"testing"
)

// bodyCarries reports whether a response body carries the number.
func bodyCarries(body, number string) bool {
	return strings.Contains(body, number)
}

// TICK-1111: the byte-limit checks failed at random when a request id happened
// to contain the limit's digits.
func TestBodyCarriesIgnoresRandomIDsOnly(t *testing.T) {
	t.Parallel()
	const (
		limit   = "54321"
		request = "urn:nava:request:f87ae328ad54321dbdab6c17145acd8d" // the id of the failing CI run
	)
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"request id", `{"status":405,"instance":"` + request + `"}`, false},
		{"uuid", `{"id":"0b54321b-0000-4000-8000-000000000001"}`, false},
		{"sha-256", `{"sha256":"` + strings.Repeat("ab54321c", 8) + `"}`, false},
		{"time", `{"published_at":"2026-11-23T04:16:23.543211Z"}`, false},
		{"time with an offset", `{"published_at":"2026-11-23T04:16:23.054321+05:30"}`, false},
		{"another limit", `{"detail":"more than 65536 bytes"}`, false},

		{"limit in the detail, id holds the digits", `{"detail":"more than 54321 bytes","instance":"` + request + `"}`, true},
		{"limit in the detail, id does not", `{"detail":"more than 54321 bytes","instance":"urn:nava:request:00000000000000000000000000000000"}`, true},
		{"limit as a number", `{"max_document_bytes":54321}`, true},
		{"limit inside a word", `{"detail":"limit54321B"}`, true},
		{"limit in place of the request id", `{"instance":"urn:nava:request:54321"}`, true},
		{"limit after the request id", `{"instance":"` + request + `54321"}`, true},
		{"limit after a uuid", `{"id":"0b0b0b0b-0000-4000-8000-00000000000154321"}`, true},
		{"limit after a time", `{"at":"2026-11-23T04:16:23Z54321"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := bodyCarries(tc.body, limit); got != tc.want {
				t.Errorf("bodyCarries(%s, %s) = %t, want %t", tc.body, limit, got, tc.want)
			}
		})
	}
}
