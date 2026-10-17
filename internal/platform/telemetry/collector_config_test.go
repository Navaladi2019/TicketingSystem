package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollectorDenyListMatchesFilter keeps the Collector's second layer in
// step with the in-process deny-list: every denied key must be deleted from
// span attributes (attributes/deny) and from span events (transform/deny), the
// header prefix must be covered in both, and the status description cleared.
func TestCollectorDenyListMatchesFilter(t *testing.T) {
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

	if !strings.Contains(cfg, `set(status.message, "")`) {
		t.Error(`transform/deny must clear the status description: set(status.message, "")`)
	}
}
