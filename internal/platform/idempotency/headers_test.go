package idempotency

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEncodeHeadersKeepsOnlyTheAllowList(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("ETag", `W/"3"`)
	h.Add("Location", "/api/v1/records/incident/a")
	h.Add("Location", "/ignored-second-value")
	h.Set("Retry-After", "60")
	h.Set("Set-Cookie", "s=1")
	got, err := encodeHeaders(h)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"ETag":"W/\"3\"","Location":"/api/v1/records/incident/a"}`; string(got) != want {
		t.Errorf("encodeHeaders = %s, want %s", got, want)
	}
	back, err := decodeHeaders(got)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 || back["ETag"] != `W/"3"` || back["Location"] != "/api/v1/records/incident/a" {
		t.Errorf("decodeHeaders round trip = %v", back)
	}
}

func TestEncodeHeadersWithoutAllowListedHeadersIsNull(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("X-Other", "1")
	got, err := encodeHeaders(h)
	if err != nil || got != nil {
		t.Errorf("encodeHeaders = %q, %v; want nil (SQL NULL)", got, err)
	}
	m, err := decodeHeaders(nil)
	if err != nil || len(m) != 0 {
		t.Errorf("decodeHeaders(NULL) = %v, %v; want empty", m, err)
	}
}

func TestDecodeHeadersRejectsANonObject(t *testing.T) {
	t.Parallel()
	if _, err := decodeHeaders([]byte(`["ETag"]`)); err == nil {
		t.Error("decodeHeaders of an array: want an error")
	}
}

func TestWriteStoredRestoresTheStoredHeaders(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeStored(rec, http.StatusCreated, []byte(`{}`), map[string]string{"ETag": `W/"1"`, "Location": "/x", "X-Smuggled": "1"})
	if rec.Header().Get("ETag") != `W/"1"` || rec.Header().Get("Location") != "/x" {
		t.Errorf("replay headers = %v, want ETag and Location", rec.Header())
	}
	if rec.Header().Get("X-Smuggled") != "" {
		t.Error("a stored header outside ReplayHeaders must not be restored")
	}
	if rec.Header().Get(ReplayedHeader) != "true" || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("replay headers = %v", rec.Header())
	}
}
