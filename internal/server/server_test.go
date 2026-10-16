package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func TestMetricsEndpointExposesHTTPMetrics(t *testing.T) {
	t.Parallel()
	h, err := server.NewHandler(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// A scrape is recorded after it is served, so the second sees the first.
	get(t, srv.URL+"/metrics")
	body := get(t, srv.URL+"/metrics")
	for _, want := range []string{
		"# TYPE http_requests_total counter",
		`http_requests_total{method="GET",route="GET /metrics",status_class="2xx"}`,
		"# TYPE http_request_duration_seconds histogram",
		`http_request_duration_seconds_bucket{method="GET",route="GET /metrics",status_class="2xx",le="0.005"}`,
		"go_goroutines",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/metrics is missing %q", want)
		}
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

func get(t *testing.T, url string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
