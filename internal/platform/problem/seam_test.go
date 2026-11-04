package problem

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Not parallel: it swaps the package-level marshal.
func TestWriteFallsBackWhenMarshalFails(t *testing.T) {
	orig := marshal
	marshal = func(any) ([]byte, error) { return nil, errors.New("boom") }
	t.Cleanup(func() { marshal = orig })

	rec := httptest.NewRecorder()
	write(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), Body{Status: http.StatusTeapot, Code: InternalError})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), http.StatusText(http.StatusInternalServerError)) || strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("body = %q, want the plain status text without the error", rec.Body)
	}
}
