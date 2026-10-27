package telemetry_test

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func outbound(t *testing.T, opts telemetry.OutboundOptions) (*telemetry.OutboundClient, *telemetry.Telemetry, *tracetest.InMemoryExporter) {
	t.Helper()
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	c, err := telemetry.NewOutboundClient(tel, opts)
	if err != nil {
		t.Fatal(err)
	}
	return c, tel, spans
}

func closedPort(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func TestOutboundErrorHasNoQueryOrEmail(t *testing.T) {
	t.Parallel()
	c, _, _ := outbound(t, telemetry.OutboundOptions{})
	addr := closedPort(t)
	ctx := telemetry.WithRoute(t.Context(), "/users/{id}")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://bob:s3cret@"+addr+"/users/42?email=alice@tenant-a.test&token=abc", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("request to a closed port succeeded")
	}
	text := err.Error()
	for _, leak := range []string{"email", "alice", "tenant-a", "token", "abc", "?", "s3cret", "bob", "/users/42"} {
		if strings.Contains(text, leak) {
			t.Errorf("error leaks %q: %s", leak, text)
		}
	}
	want := "http://" + addr + "/users/{id}"
	var ue *url.Error
	if !errors.As(err, &ue) || ue.URL != want {
		t.Errorf("error = %#v, want a *url.Error with URL %q", err, want)
	}
	if !errors.As(err, new(*net.OpError)) {
		t.Errorf("error = %#v, want the *net.OpError cause kept for retry logic", err)
	}
}

func TestOutboundErrorWithoutRouteKeepsOnlyHost(t *testing.T) {
	t.Parallel()
	c, _, _ := outbound(t, telemetry.OutboundOptions{})
	addr := closedPort(t)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/a/b?x=1", nil)
	err := doErr(c, req)
	var ue *url.Error
	if !errors.As(err, &ue) || ue.URL != "http://"+addr {
		t.Errorf("error = %#v, want URL of scheme and host only", err)
	}
}

func TestOutboundTimeoutAndRedirectDoNotLeak(t *testing.T) {
	t.Parallel()
	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)
	c, _, _ := outbound(t, telemetry.OutboundOptions{Timeout: 100 * time.Millisecond})
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, slow.URL+"/slow?email=alice@tenant-a.test", nil)
	start := time.Now()
	err := doErr(c, req)
	if err == nil {
		t.Fatal("slow server did not time out")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("timeout took %v", time.Since(start))
	}
	if strings.Contains(err.Error(), "alice") || strings.Contains(err.Error(), "?") {
		t.Errorf("timeout error leaks the query: %v", err)
	}
	var ue *url.Error
	if !errors.As(err, &ue) || !ue.Timeout() {
		t.Errorf("error = %#v, want a *url.Error whose Timeout() is true", err)
	}

	// A redirect to a query-carrying URL that then fails must not echo it.
	dead := closedPort(t)
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://"+dead+"/next?email=alice@tenant-a.test", http.StatusFound)
	}))
	t.Cleanup(redir.Close)
	req, _ = http.NewRequestWithContext(telemetry.WithRedirects(t.Context()), http.MethodGet, redir.URL, nil)
	if err = doErr(c, req); err == nil || strings.Contains(err.Error(), "alice") {
		t.Errorf("redirect error = %v, want a failure without the email", err)
	}
}

func TestOutboundTracesAndPropagates(t *testing.T) {
	t.Parallel()
	var traceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("Traceparent")
	}))
	t.Cleanup(srv.Close)
	c, _, spans := outbound(t, telemetry.OutboundOptions{})
	req, _ := http.NewRequestWithContext(telemetry.WithRoute(t.Context(), "/ping"), http.MethodGet, srv.URL+"/ping?email=a@b.test", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if traceparent == "" {
		t.Error("no traceparent header sent")
	}
	found := false
	for _, s := range spans.GetSpans() {
		if s.Name == "GET /ping" {
			found = true
		}
	}
	if !found {
		t.Errorf("no span named %q among %d spans", "GET /ping", len(spans.GetSpans()))
	}
}

func TestOutboundWithoutTraceMakesNoSpan(t *testing.T) {
	t.Parallel()
	var traceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("Traceparent")
	}))
	t.Cleanup(srv.Close)
	c, _, spans := outbound(t, telemetry.OutboundOptions{})
	ctx := telemetry.WithoutTrace(telemetry.WithRoute(t.Context(), "/ping"))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/ping", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if traceparent != "" {
		t.Errorf("traceparent %q sent on an untraced call", traceparent)
	}
	if n := len(spans.GetSpans()); n != 0 {
		t.Errorf("untraced call made %d spans, want 0", n)
	}
}

func TestOutboundRejectsNegativeTimeout(t *testing.T) {
	t.Parallel()
	_, tel, _ := outbound(t, telemetry.OutboundOptions{})
	for _, o := range []telemetry.OutboundOptions{{Timeout: -1}, {DialTimeout: -1}, {TLSHandshakeTimeout: -1}} {
		if _, err := telemetry.NewOutboundClient(tel, o); err == nil {
			t.Errorf("%+v accepted", o)
		}
	}
}

// doErr sends req and returns only the error, closing any body.
func doErr(c *telemetry.OutboundClient, req *http.Request) error {
	resp, err := c.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
	return err
}
