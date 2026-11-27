package schemabackfill

import (
	"encoding/json"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// Defaults of the workflow.
const (
	// DefaultAdmitWait is the first wait of a job whose tenant has another
	// job active; each wait doubles, up to DefaultAdmitWaitMax (P10-06
	// item 2).
	DefaultAdmitWait    = 10 * time.Second
	DefaultAdmitWaitMax = 60 * time.Second
	// DefaultR6After is how long a failed_records job waits before its R6
	// check (D-10-3).
	DefaultR6After = 7 * 24 * time.Hour
)

// Activity bounds. Every activity retries without an attempt cap, backing
// off to a minute; only ErrTypeNoJob is non-retryable. A page is 100
// records of a few milliseconds each; the fill is the heaviest statement.
const (
	shortStartToClose   = 30 * time.Second
	pageStartToClose    = 5 * time.Minute
	fillStartToClose    = 30 * time.Minute
	barrierStartToClose = 24 * time.Hour
	barrierHeartbeat    = 30 * time.Second
)

// Workflow is schema.backfill. Zero fields take the defaults.
type Workflow struct {
	AdmitWait, AdmitWaitMax time.Duration
	R6After                 time.Duration
}

// RegisterWorkflow adds schema.backfill to r with the defaults. It is the
// form the handbook build parses (docs/handbook/build/build.py
// GO_REGISTER_FUNCS).
func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(run, workflow.RegisterOptions{Name: WorkflowType})
}

func run(ctx workflow.Context, in outbox.Delivery) error { return (&Workflow{}).Run(ctx, in) }

// Register adds w.Run to r as WorkflowType.
func (w *Workflow) Register(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(w.Run, workflow.RegisterOptions{Name: WorkflowType})
}

func retryForever() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{InitialInterval: time.Second, BackoffCoefficient: 2, MaximumInterval: time.Minute,
		NonRetryableErrorTypes: []string{ErrTypeNoJob}}
}

func withActivity(ctx workflow.Context, startToClose, heartbeat time.Duration) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: startToClose, HeartbeatTimeout: heartbeat, RetryPolicy: retryForever()})
}

// Run is one job, t:<tenant>:schema.backfill:<entity_definition_id>:<target_version>
// (D2). Its input is the relay's outbox.Delivery, references only, with no
// tenant: the job is the workflow id's, and a start payload naming another
// definition is refused before anything runs. Then:
//
//  1. admission (Admit), waiting on a timer, 10 s backing off to 60 s, while
//     another job of the tenant is active (P10-06);
//  2. the barrier (D9 of ADR-006);
//  3. the fill and the flip to running, one transaction;
//  4. pages until the queue is empty, with the ADR-006 D10 pause between
//     them; a page that finds the job superseded ends the run;
//  5. the end, done or failed_records; after failed_records, a timer of
//     R6After and the R6 check. While it waits the job holds no
//     filling/running state, so it blocks no admission.
func (w *Workflow) Run(ctx workflow.Context, in outbox.Delivery) error {
	j, err := ParseWorkflowID(workflow.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return noJob("workflow id is not a job's")
	}
	if !payloadMatches(in.Payload, j) {
		return noJob("start payload names another entity definition")
	}
	short := withActivity(ctx, shortStartToClose, 0)

	wait := w.admitWait()
	for {
		var a AdmitResult
		if err := workflow.ExecuteActivity(short, AdmitActivity).Get(ctx, &a); err != nil {
			return err
		}
		if a.Ended != nil {
			return nil
		}
		if a.Admitted != nil {
			break
		}
		if err := workflow.Sleep(ctx, wait); err != nil {
			return err
		}
		wait = min(wait*2, w.admitWaitMax())
	}

	bctx := withActivity(ctx, barrierStartToClose, barrierHeartbeat)
	if err := workflow.ExecuteActivity(bctx, BarrierActivity).Get(ctx, nil); err != nil {
		return err
	}
	var f FillResult
	if err := workflow.ExecuteActivity(withActivity(ctx, fillStartToClose, 0), FillActivity).Get(ctx, &f); err != nil {
		return err
	}
	if f.Ended != nil {
		return nil
	}

	pctx := withActivity(ctx, pageStartToClose, 0)
	var p fieldindex.Pacer
	var cursor Cursor
	for {
		start := workflow.Now(ctx)
		var out PageResult
		if err := workflow.ExecuteActivity(pctx, PageActivity, cursor).Get(ctx, &out); err != nil {
			return err
		}
		if out.Ended != nil {
			return nil
		}
		if out.Next == nil {
			break
		}
		cursor = Cursor{After: out.Next}
		if err := workflow.Sleep(ctx, p.Next(workflow.Now(ctx).Sub(start))); err != nil {
			return err
		}
	}

	var e EndResult
	if err := workflow.ExecuteActivity(short, EndActivity).Get(ctx, &e); err != nil {
		return err
	}
	if e.FailedRecords == nil {
		return nil
	}
	if err := workflow.Sleep(ctx, w.r6After()); err != nil {
		return err
	}
	return workflow.ExecuteActivity(short, R6Activity).Get(ctx, nil)
}

// payloadMatches reports whether a start payload is the job's: absent or {}
// (a test start), or {"entity_definition_id": the id's definition}. The
// relay refuses any other row before Temporal (contract C3).
func payloadMatches(p json.RawMessage, j Job) bool {
	if len(p) == 0 || string(p) == "{}" || string(p) == "null" {
		return true
	}
	var v map[string]string
	if json.Unmarshal(p, &v) != nil {
		return false
	}
	d, ok := v["entity_definition_id"]
	return ok && len(v) == 1 && d == j.Definition.String()
}

func (w *Workflow) admitWait() time.Duration {
	if w.AdmitWait > 0 {
		return w.AdmitWait
	}
	return DefaultAdmitWait
}

func (w *Workflow) admitWaitMax() time.Duration {
	if w.AdmitWaitMax > 0 {
		return w.AdmitWaitMax
	}
	return DefaultAdmitWaitMax
}

func (w *Workflow) r6After() time.Duration {
	if w.R6After > 0 {
		return w.R6After
	}
	return DefaultR6After
}
