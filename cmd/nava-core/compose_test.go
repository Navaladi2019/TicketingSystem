package main

import (
	"os"
	"strings"
	"testing"
)

// TestComposePublishesNoAdminPort: the admin listener has no auth, so no
// compose file may publish its port. The only mention allowed is the
// container's own healthcheck URL.
func TestComposePublishesNoAdminPort(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"../../compose.yaml", "../../deploy/compose/compose.internal.yaml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			code, _, _ := strings.Cut(line, "#")
			if strings.Contains(code, "9090") && !strings.Contains(code, "http://127.0.0.1:9090/") {
				t.Errorf("%s:%d mentions the admin port: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
