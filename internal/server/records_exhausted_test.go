package server

import (
	"context"
	"testing"
	"time"
)

// ADR-004-E9 N1: a refusal's append outlives the request's cancellation and
// is bounded by the request's deadline, or by exhaustedAuditTimeout when the
// request has none.
func TestExhaustedAuditContextIsBoundedByTheRequest(t *testing.T) {
	t.Run("no deadline gets the fallback", func(t *testing.T) {
		start := time.Now()
		ctx, cancel := exhaustedAuditContext(context.Background())
		defer cancel()
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("no deadline, want exhaustedAuditTimeout")
		}
		if d := deadline.Sub(start); d < exhaustedAuditTimeout || d > exhaustedAuditTimeout+time.Second {
			t.Errorf("deadline in %v, want about %v", d, exhaustedAuditTimeout)
		}
	})
	for name, in := range map[string]time.Duration{"a shorter deadline wins": time.Second, "a longer deadline wins": time.Minute} {
		t.Run(name, func(t *testing.T) {
			want := time.Now().Add(in)
			req, cancelReq := context.WithDeadline(context.Background(), want)
			defer cancelReq()
			ctx, cancel := exhaustedAuditContext(req)
			defer cancel()
			if got, ok := ctx.Deadline(); !ok || !got.Equal(want) {
				t.Errorf("deadline %v (%v), want the request's %v", got, ok, want)
			}
		})
	}
	t.Run("the request's cancellation does not reach it", func(t *testing.T) {
		type key struct{}
		req, cancelReq := context.WithCancel(context.WithValue(context.Background(), key{}, "v"))
		ctx, cancel := exhaustedAuditContext(req)
		defer cancel()
		cancelReq()
		if err := ctx.Err(); err != nil {
			t.Errorf("err = %v after the request was cancelled, want nil", err)
		}
		if ctx.Value(key{}) != "v" {
			t.Error("the request's values are lost")
		}
	})
}
