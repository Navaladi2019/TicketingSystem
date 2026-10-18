package telemetry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// Adversarial guards for the outbound client (D-P10 path, TICK-55 review).
// The marker sits in every slot a caller or a remote server controls; it must
// reach neither an exported span nor an error the caller will log.

const outboundMarker = "alice@tenant-a.test"

// otelhttp sets url.full to the whole request URL, query included. The
// export-time filter must drop it, and the span must carry nothing else that
// quotes the URL (status description, error events).
func TestOutboundSpanCarriesNoQueryOrEmail(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c, _, spans := outbound(t, telemetry.OutboundOptions{})

	ok, _ := http.NewRequestWithContext(telemetry.WithRoute(t.Context(), "/users/{id}"),
		http.MethodGet, srv.URL+"/users/42?email="+outboundMarker, nil)
	if resp, err := c.Do(ok); err == nil {
		_ = resp.Body.Close()
	}
	dead, _ := http.NewRequestWithContext(telemetry.WithRoute(t.Context(), "/users/{id}"),
		http.MethodGet, "http://"+closedPort(t)+"/users/42?email="+outboundMarker, nil)
	if resp, err := c.Do(dead); err == nil {
		_ = resp.Body.Close()
	}

	got := spans.GetSpans()
	if len(got) == 0 {
		t.Fatal("no spans exported")
	}
	for _, s := range got {
		dump := fmt.Sprintf("%s %v %v %v", s.Name, s.Attributes, s.Status, s.Events)
		for _, leak := range []string{"alice", "tenant-a", "/users/42", "?email"} {
			if strings.Contains(dump, leak) {
				t.Errorf("span %q leaks %q: %s", s.Name, leak, dump)
			}
		}
	}
}

// http.Client reports an unparsable Location header as a *url.Error whose Err
// quotes the header verbatim. The remote server chooses that header, and a
// webhook target or a misbehaving IdP can echo our query back in it.
func TestOutboundMalformedRedirectDoesNotEchoLocation(t *testing.T) {
	t.Parallel()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://%zz/next?email="+outboundMarker)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(redir.Close)
	c, _, _ := outbound(t, telemetry.OutboundOptions{})
	req, _ := http.NewRequestWithContext(telemetry.WithRoute(t.Context(), "/start"),
		http.MethodGet, redir.URL+"/start", nil)
	resp, err := c.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("malformed Location was followed")
	}
	if strings.Contains(err.Error(), "alice") || strings.Contains(err.Error(), "?email") {
		t.Errorf("error echoes the Location header: %v", err)
	}
}
