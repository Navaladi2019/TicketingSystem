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

func newServer(t *testing.T) (*httptest.Server, *telemetry.Telemetry) {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	srv := httptest.NewServer(server.NewHandler(slog.New(slog.DiscardHandler), tel))
	t.Cleanup(srv.Close)
	return srv, tel
}

func TestMetricsEndpointExposesOTelHTTPAndRuntimeMetrics(t *testing.T) {
	t.Parallel()
	srv, _ := newServer(t)

	// A request is recorded after it is served, so the second scrape sees the first.
	do(t, http.MethodGet, srv.URL+"/metrics", "")
	body := do(t, http.MethodGet, srv.URL+"/metrics", "")
	for _, want := range []string{
		"# TYPE http_server_request_duration_seconds histogram",
		"http_server_request_duration_seconds_count{",
		`http_route="/metrics"`,
		`http_request_method="GET"`,
		`http_response_status_code="200"`,
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
}

// TestHTTPMetricCardinalityIsBounded sends what a hostile client controls:
// made-up methods, Host headers and paths. None of it may add a series per
// value or appear in a label.
func TestHTTPMetricCardinalityIsBounded(t *testing.T) {
	t.Parallel()
	srv, tel := newServer(t)

	do(t, http.MethodGet, srv.URL+"/metrics", "")
	hostile := []string{"FOO1", "FOO2", "XYZ"}
	for _, m := range hostile {
		do(t, m, srv.URL+"/metrics", "")
	}
	hosts := []string{"tenant-a.evil.test:1111", "tenant-b.evil.test:2222", "tenant-c.evil.test"}
	for _, h := range hosts {
		do(t, http.MethodGet, srv.URL+"/metrics", h)
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
	// GET 200 /metrics, _OTHER 405 (no route), GET 404 (no route).
	if series != 3 {
		t.Errorf("http_server_request_duration_seconds has %d series, want 3", series)
	}
}

func TestNewServerSetsTimeouts(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	srv := server.New(cfg.HTTP, http.NotFoundHandler(), slog.New(slog.DiscardHandler))
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
