package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

// deniedSpanAttributePrefix and deniedSpanAttributeKeys are the known-bad keys
// the Collector's second layer deletes (deploy/compose/otel-collector/config.yaml).
// The in-process filter is an allow-list and does not read them; this list is
// what the Collector config must mirror. Older semconv names are listed too.
const deniedSpanAttributePrefix = "http.request.header."

//nolint:gochecknoglobals // read-only test table
var deniedSpanAttributeKeys = map[attribute.Key]struct{}{
	"url.path": {}, "url.query": {}, "url.full": {},
	"http.url": {}, "http.target": {},
	"user_agent.original": {}, "http.user_agent": {},
	"client.address": {}, "client.port": {}, "http.client_ip": {},
	"network.peer.address": {}, "network.peer.port": {}, "net.sock.peer.addr": {}, "net.peer.ip": {},
	"server.port":                  {}, // from the Host header when the server name has no port
	"http.request.method_original": {},
	"network.protocol.version":     {}, // the client picks it from HTTP/1.0 to HTTP/1.9
	"db.query.text":                {}, "db.statement": {},
	"exception.message": {},
}

// TestCollectorDenyListMatchesFilter keeps the two layers consistent. Every
// denied key, and a header key, must be rejected by the in-process allow-list.
// The Collector must delete every denied key from span attributes
// (attributes/deny) and from span events (transform/deny), cover the header
// prefix in both, and clear the status description.
func TestCollectorDenyListMatchesFilter(t *testing.T) {
	for k := range deniedSpanAttributeKeys {
		if spanAttributeAllowed(k.String("x")) {
			t.Errorf("spanAttributeAllowed keeps denied key %q", k)
		}
	}
	if k := attribute.Key(deniedSpanAttributePrefix + "x"); spanAttributeAllowed(k.String("x")) {
		t.Errorf("spanAttributeAllowed keeps header key %q", k)
	}

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "compose", "otel-collector", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(raw)

	for k := range deniedSpanAttributeKeys {
		if want := "{key: " + string(k) + ", action: delete}"; !strings.Contains(cfg, want) {
			t.Errorf("attributes/deny lacks %q: add it to the Collector config", want)
		}
		if want := `delete_key(attributes, "` + string(k) + `")`; !strings.Contains(cfg, want) {
			t.Errorf("transform/deny (spanevent) lacks %q: add it to the Collector config", want)
		}
	}

	// The prefix as a regexp: dots escaped.
	re := strings.ReplaceAll(deniedSpanAttributePrefix, ".", `\.`) + ".*"
	if want := "{pattern: ^" + re + ", action: delete}"; !strings.Contains(cfg, want) {
		t.Errorf("attributes/deny lacks the header pattern %q", want)
	}
	if want := `delete_matching_keys(attributes, "^` + strings.ReplaceAll(re, `\`, `\\`) + `")`; !strings.Contains(cfg, want) {
		t.Errorf("transform/deny (spanevent) lacks the header pattern %q", want)
	}

	// Logs go through the same attributes/deny processor, not a copy of it.
	if n := strings.Count(cfg, "attributes/deny:"); n != 1 {
		t.Errorf("want one attributes/deny definition shared by traces and logs, found %d", n)
	}
	i := strings.Index(cfg, "    logs:")
	if i < 0 {
		t.Fatal("the Collector config has no logs: pipeline")
	}
	logsPipeline := cfg[i:]
	if !strings.Contains(logsPipeline, "attributes/deny") {
		t.Error("the logs pipeline must run attributes/deny")
	}

	if !strings.Contains(cfg, `set(status.message, "")`) {
		t.Error(`transform/deny must clear the status description: set(status.message, "")`)
	}
}

// TestLogDenyListMatchesCollector: the in-process log deny-list covers every
// key the Collector deletes, so the two layers cannot drift.
func TestLogDenyListMatchesCollector(t *testing.T) {
	for k := range deniedSpanAttributeKeys {
		if !logAttributeDenied(string(k)) {
			t.Errorf("deniedLogAttributeKeys lacks %q", k)
		}
	}
	if len(deniedLogAttributeKeys) != len(deniedSpanAttributeKeys) {
		t.Errorf("in-process log deny-list has %d keys, Collector list has %d",
			len(deniedLogAttributeKeys), len(deniedSpanAttributeKeys))
	}
	if deniedLogAttributePrefix != deniedSpanAttributePrefix {
		t.Errorf("header prefix %q != %q", deniedLogAttributePrefix, deniedSpanAttributePrefix)
	}
}
