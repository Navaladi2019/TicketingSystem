package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// newServers starts the public and admin handlers on one telemetry, as run
// does.
func newServers(t *testing.T) (public, internal *httptest.Server, tel *telemetry.Telemetry) {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err = telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	log := slog.New(slog.DiscardHandler)
	public = httptest.NewServer(server.NewHandler(log, tel))
	t.Cleanup(public.Close)
	internal = httptest.NewServer(telemetry.NewAdminHandler(log, tel))
	t.Cleanup(internal.Close)
	return public, internal, tel
}

// TestOperationalEndpointsOnlyOnAdminListener is TODO-001: the public listener must not
// expose /metrics, /healthz or /readyz, in any format a scraper might ask for.
func TestOperationalEndpointsOnlyOnAdminListener(t *testing.T) {
	t.Parallel()
	public, internal, _ := newServers(t)
	const openMetrics = "application/openmetrics-text; version=1.0.0"

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		for _, accept := range []string{"", openMetrics} {
			code, _, body := request(t, method, public.URL+"/metrics", accept)
			if code != http.StatusNotFound {
				t.Errorf("public %s /metrics (Accept %q) = %d, want 404", method, accept, code)
			}
			if strings.Contains(body, "# TYPE") {
				t.Errorf("public %s /metrics (Accept %q) leaks metrics:\n%s", method, accept, body)
			}
		}
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			if code, _, _ := request(t, method, public.URL+path, ""); code != http.StatusNotFound {
				t.Errorf("public %s %s = %d, want 404", method, path, code)
			}
		}
	}

	code, ctype, body := request(t, http.MethodGet, internal.URL+"/metrics", openMetrics)
	if code != http.StatusOK {
		t.Fatalf("internal GET /metrics = %d, want 200", code)
	}
	if !strings.HasPrefix(ctype, "application/openmetrics-text") {
		t.Errorf("internal /metrics Content-Type = %q, want OpenMetrics", ctype)
	}
	if !strings.HasSuffix(body, "# EOF\n") || !strings.Contains(body, "# TYPE go_goroutine_count") {
		t.Errorf("internal /metrics is not an OpenMetrics exposition:\n%s", body)
	}
}

func TestMetricsEndpointExposesOTelHTTPAndRuntimeMetrics(t *testing.T) {
	t.Parallel()
	public, internal, _ := newServers(t)

	do(t, http.MethodGet, public.URL+"/nope", "")
	body := do(t, http.MethodGet, internal.URL+"/metrics", "")
	for _, want := range []string{
		"# TYPE http_server_request_duration_seconds histogram",
		"http_server_request_duration_seconds_count{",
		`http_request_method="GET"`,
		`http_response_status_code="404"`,
		"go_goroutine_count", // runtime instrumentation
		"go_memory_used_bytes",
		"process_cpu_seconds_total", // process collector
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/metrics is missing %q", want)
		}
	}
	for _, gone := range []string{"http_requests_total", "go_goroutines "} {
		if strings.Contains(body, gone) {
			t.Errorf("/metrics still exposes the pre-OTel %q", gone)
		}
	}
	// Scrapes are untraced: they add no HTTP series of their own.
	if strings.Contains(body, `http_route="/metrics"`) {
		t.Error("/metrics scrapes are recorded as HTTP requests")
	}
}

// TestMetricsCarryTraceExemplars checks the OpenMetrics scrape links latency
// buckets to the trace of a sampled request.
func TestMetricsCarryTraceExemplars(t *testing.T) {
	t.Parallel()
	public, internal, _ := newServers(t)

	do(t, http.MethodGet, public.URL+"/nope", "")
	body := doAccept(t, internal.URL+"/metrics", "application/openmetrics-text; version=1.0.0")
	var bucket string
	for l := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(l, "http_server_request_duration_seconds_bucket{") && strings.Contains(l, "# {") {
			bucket = l
			break
		}
	}
	if bucket == "" || !strings.Contains(bucket, "trace_id=") {
		t.Errorf("no duration bucket carries a trace_id exemplar:\n%s", body)
	}
	// The exemplar carries only trace and span id: attributes the View dropped
	// (server.port from the Host header, the protocol version) must not come
	// back as exemplar labels.
	if _, ex, _ := strings.Cut(bucket, "# {"); strings.Contains(ex, "server_") || strings.Contains(ex, "network_protocol") {
		t.Errorf("exemplar carries dropped attributes: %s", ex)
	}
}

// TestHTTPMetricCardinalityIsBounded sends what a hostile client controls:
// made-up methods, Host headers and paths. None of it may add a series per
// value or appear in a label.
func TestHTTPMetricCardinalityIsBounded(t *testing.T) {
	t.Parallel()
	srv, _, tel := newServers(t)

	hostile := []string{"FOO1", "FOO2", "XYZ"}
	for _, m := range hostile {
		do(t, m, srv.URL+"/nope/method", "")
	}
	hosts := []string{"tenant-a.evil.test:1111", "tenant-b.evil.test:2222", "tenant-c.evil.test"}
	for _, h := range hosts {
		do(t, http.MethodGet, srv.URL+"/nope/host", h)
	}
	paths := []string{"/nope/9f1c", "/nope/77ab", "/users/alice@example.test"}
	for _, p := range paths {
		do(t, http.MethodGet, srv.URL+p, "")
	}

	families, err := tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	series := 0
	for _, f := range families {
		if !strings.HasPrefix(f.GetName(), "http_server_") {
			continue
		}
		for _, m := range f.GetMetric() {
			if f.GetName() == "http_server_request_duration_seconds" {
				series++
			}
			for _, l := range m.GetLabel() {
				v := l.GetValue()
				if l.GetName() == "http_request_method" && v != "GET" && v != "_OTHER" {
					t.Errorf("%s: method label %q, want GET or _OTHER", f.GetName(), v)
				}
				for _, bad := range append(append(append([]string{}, hostile...), hosts...), paths...) {
					if strings.Contains(v, strings.Split(bad, ":")[0]) {
						t.Errorf("%s: label %s=%q carries client value %q", f.GetName(), l.GetName(), v, bad)
					}
				}
				if l.GetName() == "server_address" || l.GetName() == "server_port" {
					t.Errorf("%s: has the Host-derived label %s", f.GetName(), l.GetName())
				}
			}
		}
	}
	// _OTHER 404 and GET 404, both with no route.
	if series != 2 {
		t.Errorf("http_server_request_duration_seconds has %d series, want 2", series)
	}
}

func TestNewServerSetsTimeouts(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New(cfg.HTTP.AdminAddr, cfg.HTTP, http.NotFoundHandler(), slog.New(slog.DiscardHandler))
	if srv.Addr != ":9090" {
		t.Errorf("Addr = %q, want the addr passed in", srv.Addr)
	}
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Errorf("server has an unset timeout: %+v", srv)
	}
}

func do(t *testing.T, method, url, host string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	return send(t, req)
}

func doAccept(t *testing.T, url, accept string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", accept)
	return send(t, req)
}

// request returns the status, Content-Type and body of method url.
func request(t *testing.T, method, url, accept string) (code int, ctype, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
}

func send(t *testing.T, req *http.Request) string {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
