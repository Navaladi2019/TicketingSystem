package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TICK-690: 503 retry_later advises a Retry-After per route. Record create
// leaves no key row behind (its claim rolls back with the write, TICK-688),
// so it advises ClaimRetryAfter; every other route keeps the key lease, which
// its in_flight row stands for.
func TestRetryLaterRetryAfterIsPerRoute(t *testing.T) {
	t.Parallel()
	secs := func(d time.Duration) string { return strconv.Itoa(int(d / time.Second)) }
	for pattern, want := range map[string]string{
		createRecordPattern:                   secs(idempotency.ClaimRetryAfter),
		"PATCH /api/v1/records/{entity}/{id}": secs(idempotency.Lease),
		"POST /api/v1/entity-types":           secs(idempotency.Lease),
	} {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()
			h := testHandlers(t)
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/records/ticket", nil)
			r.Pattern = pattern
			w := httptest.NewRecorder()
			h.responseError(nil)(w, r, &problemError{problem.RetryLater, nil, errors.New("allocation timed out")})
			if w.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503", w.Code)
			}
			if got := w.Header().Get("Retry-After"); got != want {
				t.Errorf("Retry-After = %q, want %s", got, want)
			}
		})
	}
}

// ClaimRetryAfter is short, on the order of ClaimWait, and a whole number of
// seconds of at least 1, so the header is never 0.
func TestClaimRetryAfterIsShort(t *testing.T) {
	t.Parallel()
	ra := idempotency.ClaimRetryAfter
	if ra < time.Second || ra%time.Second != 0 || ra > 2*idempotency.ClaimWait || ra >= idempotency.Lease {
		t.Errorf("ClaimRetryAfter = %v, want whole seconds, >= 1s, <= 2*ClaimWait (%v) and < Lease", ra, idempotency.ClaimWait)
	}
}
