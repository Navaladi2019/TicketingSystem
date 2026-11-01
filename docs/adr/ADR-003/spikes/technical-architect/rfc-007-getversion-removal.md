---
type: spike
title: "Technical Architect spike: deleting a GetVersion call vs raising minSupported (RFC-007 round 1)"
issue: TICK-388
source: TICK-388 comment 01a1950c-5bf9-7362-bf8a-c27afa965b65 (round 1 critique); re-check 01a19510-22ed-7314-86ed-884dea3bbcf9
pin: main 69f3a62, go.temporal.io/sdk v1.49.0, go1.26.10 windows/amd64
---

# Technical Architect spike: deleting a GetVersion call vs raising minSupported

> Archived verbatim from the Technical Architect's critique comments. The spike code was throwaway and not committed; the rerun recipe is below.

**Hypothesis:** on our recorded histories, deleting a `GetVersion` call lets an old history replay green on the new arm, while keeping the call with `minSupported=1` fails replay with TMPRL1100. **Pass:** the deleted form is green and the kept form is red on every timer-bearing history.

**Result:** confirmed on all four timer-bearing histories (table below). Runtime 0.08 s.

### O-1 in detail (blocking)

**Known (SDK v1.49.0 source):**
- `internal/internal_task_handlers.go:1539-1544`: `skipDeterministicCheckForEvent` skips `versionMarkerName` markers. Replay therefore tolerates a history whose `GetVersion` call no longer exists.
- `internal/internal_event_handlers.go:953-957, 966-976`: on replay with no marker, `GetVersion` resolves to `DefaultVersion` and `validateVersion` panics with `[TMPRL1100] Workflow code removed support of version -1 … for "<changeID>" changeID` when `minSupported > DefaultVersion`.
- `internal/workflow.go:2457-2472`: the documented way to drop the `DefaultVersion` arm is `GetVersion(ctx, id, 0|1, max)`. The docs say: "It is recommended to keep the GetVersion() call … it ensures that if there is older version execution still running, it will fail here and not proceed."
- `internal/worker.go:542-548`: the default `WorkflowPanicPolicy` is `BlockWorkflow` (a stuck task retry loop, which recovers after a code fix). The repo does not override it (grep of non-test code finds no `WorkflowPanicPolicy`).

**Known (spike, our 4 timer-bearing recorded histories):** the change is "the timer duration doubles", a realistic arm difference that the command matcher does not see.

| History | Call deleted (what R-5 requires) | Call kept, `minSupported=1` |
|---|---|---|
| `continue-as-new.run1.json` | **GREEN** | `TMPRL1100 … "spike-rfc007-timer"` |
| `else-default.json` | **GREEN** | `TMPRL1100 …` |
| `signal.json` | **GREEN** | `TMPRL1100 …` |
| `timeout.json` | **GREEN** | `TMPRL1100 …` |

**Failure scenario:** branch `X` is deleted per R-5. Rule (c)'s query missed one old run (for example, it was taken after a reset, or with the wrong deploy time). In CI, the "which histories fail" set can be empty, so nothing is retired and the gate proves nothing. In production, the old run replays green and **executes the new arm on a run that started on the old one**, with no error. Under `minSupported=1`, the same run blocks on TMPRL1100. The Operator sees it, restores the `DefaultVersion` arm (`minSupported` back to `DefaultVersion`) and redeploys, and the run resumes. That is a fix forward, and rule (b) still holds.


### Spike: how to rerun

This is throwaway code and has not been committed. Put it at `internal/workflow/interp/zz_spike_rfc007_test.go` on `main` 69f3a62, run `go test ./internal/workflow/interp/ -run TestSpikeRFC007 -v -count=1`, then delete the file. Runtime is 0.08 s.

<details><summary>spike code</summary>

```go
package interp_test

import (
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

const spikeID = "spike-rfc007-timer"

// "deleted": GetVersion call removed, new arm only (timer 2x).
// "min1":    GetVersion(ctx, id, 1, 1) kept, new arm only (timer 2x).
type spikeW struct {
	interceptor.WorkerInterceptorBase
	mode string
}

func (s *spikeW) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &spikeIn{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, mode: s.mode}
}

type spikeIn struct {
	interceptor.WorkflowInboundInterceptorBase
	mode string
}

func (i *spikeIn) Init(o interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&spikeOut{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: o}, mode: i.mode})
}

type spikeOut struct {
	interceptor.WorkflowOutboundInterceptorBase
	mode string
}

func (o *spikeOut) NewTimer(ctx workflow.Context, d time.Duration) workflow.Future {
	if o.mode == "min1" {
		_ = workflow.GetVersion(ctx, spikeID, 1, 1)
	}
	return o.Next.NewTimer(ctx, 2*d)
}

func TestSpikeRFC007(t *testing.T) {
	for _, h := range loadHistories(t) { // existing helpers in replay_test.go
		if !hasTimer(h.h) {
			continue
		}
		for _, mode := range []string{"deleted", "min1"} {
			err := replay(t, h, &spikeW{mode: mode})
			msg := "<nil> (GREEN)"
			if err != nil {
				msg = strings.SplitN(err.Error(), "\n", 2)[0]
			}
			t.Logf("%-22s mode=%-7s err=%s", h.name, mode, msg)
		}
	}
}
```
</details>


## Round 2 re-check (R-8 error string)

| Item | Verdict | Evidence |
|---|---|---|
| **R-8**, including the SA's narrowing to `removed support of version -1` + `"<change_id>"` | **confirmed sound**, and the narrowing is correct | **Known:** `internal_event_handlers.go:953-963` has two TMPRL1100 messages ("removed support" when `version < minSupported`, "too old to support" when `version > maxSupported`). A history recorded before the change has no marker and resolves to `DefaultVersion` = -1 (`:973-976`). So the only error a correctly retired history can produce is exactly `removed support of version -1. for "<change_id>"`. My round-1 spike produced that exact string through the local replayer (`[TMPRL1100] Workflow code removed support of version -1. for "spike-rfc007-timer" changeID. The oldest supported version is 1`). Replaying only newly added entries keeps the rule a property of one commit pair |
