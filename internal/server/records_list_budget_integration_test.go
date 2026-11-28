//go:build integration

// ADR-006-A2 (TICK-1046): a list with a data filter whose count outruns its
// budget answers 200 with the full page and an unknown total, through the real
// handler. The count is slowed by the test-only WithCountDelay hook.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func TestListCountOutOfBudgetAnswersUnknown(t *testing.T) {
	e := newETEnvWith(t, server.WithCountDelay(400*time.Millisecond))
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "fa", assetTypeBody("asset", "AST")); rec.Code != http.StatusCreated {
		t.Fatalf("seed asset = %d: %s", rec.Code, rec.Body)
	}
	for i := range 25 {
		e.mustCreateRec(t, "admin-a", "asset", fmt.Sprintf("b%d", i), fmt.Sprintf(`{"short_description": "d%d", "name": "same"}`, i))
	}
	e.flagField(t, e.a.id, "asset", "name", "text", "ready", 100)

	// Indexed filter: the page is complete, the total is unknown, a next page exists.
	rec, p := e.list(t, "admin-a", "asset", "limit=10&"+filterQuery("name", "eq", "same"))
	if rec.Code != http.StatusOK {
		t.Fatalf("indexed list = %d: %s", rec.Code, rec.Body)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if v, ok := raw["total"]; !ok || string(v) != "null" {
		t.Errorf(`"total" = %s (present %v), want null`, v, ok)
	}
	if p.State != "unknown" || len(p.Items) != 10 || p.Next == nil {
		t.Errorf("state %q, %d items, next %v; want unknown, 10 items and a next cursor", p.State, len(p.Items), p.Next)
	}

	// No data filter: the count keeps today's behaviour whatever its time.
	rec, p = e.list(t, "admin-a", "asset", "limit=10")
	if rec.Code != http.StatusOK || p.State != "exact" || p.Total != 25 {
		t.Errorf("unfiltered list = %d, total %d %q; want 200, exact 25: %s", rec.Code, p.Total, p.State, rec.Body)
	}

	// The shortcut: a first page that holds every match runs no count, so the
	// injected delay is never paid and the total is exact.
	rec, p = e.list(t, "admin-a", "asset", "limit=50&"+filterQuery("name", "eq", "same"))
	if rec.Code != http.StatusOK || p.State != "exact" || p.Total != 25 || p.Next != nil {
		t.Errorf("shortcut list = %d, total %d %q, next %v; want 200, exact 25, no next: %s", rec.Code, p.Total, p.State, p.Next, rec.Body)
	}
}
