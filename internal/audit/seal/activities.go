package seal

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The registered activity names. The audit.seal workflow schedules them.
const (
	RescheduleActivity = outbox.SealKind + ".Reschedule"
	SealActivity       = outbox.SealKind + ".Seal"
)

// ApplicationError types. Both are non-retryable: a retry would read the
// same header, or the same entry.
const (
	ErrTypeNoRun      = "nava.audit.seal.NoRun"
	ErrTypeUnsealable = "nava.audit.seal.Unsealable"
)

// Rescheduled is A1's result. It holds references only, as workerguard
// requires of every recorded result: NextOutboxID is o′, absent when the
// lineage was stale.
type Rescheduled struct {
	NextOutboxID *uuid.UUID `json:"next_outbox_id,omitempty"`
}

// DefaultMaxBatches is how many batches one A2 call seals at most when
// Activities.MaxBatches is unset: 10,000 entries. A tenant with a longer
// backlog is continued by the next run, 10 s later (TICK-440 review).
const DefaultMaxBatches = 10

// Activities are A1 and A2 on a nava_worker pool. Register them with
// Register. Metrics and Log may be nil.
type Activities struct {
	Pool    *pgxpool.Pool
	Metrics *Metrics
	Log     *slog.Logger
	// MaxBatches caps the batches of one A2 call; 0 is DefaultMaxBatches.
	MaxBatches int
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.Reschedule, activity.RegisterOptions{Name: RescheduleActivity})
	r.RegisterActivityWithOptions(a.Seal, activity.RegisterOptions{Name: SealActivity})
}

// Reschedule is A1 for the calling workflow. Run o is read from the workflow
// id, t:<tenant>:audit.seal:<o>, which the relay built from the start row;
// nothing comes from an argument.
func (a *Activities) Reschedule(ctx context.Context) (Rescheduled, error) {
	o, err := RunOf(activity.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return Rescheduled{}, temporal.NewNonRetryableApplicationError("seal: "+err.Error(), ErrTypeNoRun, nil)
	}
	next, err := Reschedule(ctx, a.Pool, o)
	if err != nil || next == uuid.Nil {
		return Rescheduled{}, err
	}
	return Rescheduled{NextOutboxID: &next}, nil
}

// Seal is A2. It seals at most MaxBatches batches and heartbeats after each
// committed one, so the workflow's HeartbeatTimeout bounds a stuck batch, not
// the whole backlog. It returns no count: workerguard records only
// references, and the run's outcome the next A1 needs is already on the head
// (seal_interval_s).
//
// Every attempt then measures the tenant's oldest unsealed entry for
// audit_unsealed_oldest_seconds, and a failed attempt counts in
// audit_seal_failures_total. A poison entry fails the attempt without a retry
// and is never skipped: skipping it would break the (occurred_at, id) order.
// The next run, which A1 has already committed, meets it again.
func (a *Activities) Seal(ctx context.Context) error {
	limit := a.MaxBatches
	if limit <= 0 {
		limit = DefaultMaxBatches
	}
	n, err := Seal(ctx, a.Pool, WithMaxBatches(limit), WithProgress(func(int) {
		activity.RecordHeartbeat(ctx)
	}))
	a.measure(ctx)
	if err != nil {
		a.Metrics.failed(ctx, err)
		a.logger().WarnContext(ctx, "audit seal failed", a.attrs(ctx, slog.Int("sealed", n), slog.Any("error", err))...)
	}
	if errors.Is(err, ErrUnsealable) {
		return temporal.NewNonRetryableApplicationError("seal: entry cannot be canonicalised", ErrTypeUnsealable, nil)
	}
	return err
}

// measure records the tenant's oldest unsealed entry on the gauge. A failed
// read leaves the last sample in place, so the gauge keeps aging.
func (a *Activities) measure(ctx context.Context) {
	if a.Metrics == nil {
		return
	}
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok {
		return
	}
	oldest, found, err := OldestUnsealed(ctx, a.Pool)
	if err != nil {
		a.logger().WarnContext(ctx, "audit seal lag not measured", a.attrs(ctx, slog.Any("error", err))...)
		return
	}
	a.Metrics.observe(tenant, oldest, found)
}

func (a *Activities) logger() *slog.Logger {
	if a.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return a.Log
}

// attrs are the log attributes of an A2 line: the tenant and the workflow
// id, never an entry's values.
func (a *Activities) attrs(ctx context.Context, extra ...any) []any {
	out := make([]any, 0, 2+len(extra))
	if tenant, ok := workerguard.TenantFromContext(ctx); ok {
		out = append(out, slog.String("tenant_id", tenant.String()))
	}
	if activity.IsActivity(ctx) {
		out = append(out, slog.String("workflow_id", activity.GetInfo(ctx).WorkflowExecution.ID))
	}
	return append(out, extra...)
}

// errNotSealRun is RunOf's error for any id that is not a sealer run's.
var errNotSealRun = errors.New("workflow id is not t:<tenant>:" + outbox.SealKind + ":<outbox id>")

// RunOf returns o from a sealer workflow id t:<tenant>:audit.seal:<o>. o must
// be a canonical UUID.
func RunOf(workflowID string) (uuid.UUID, error) {
	tenant, err := workerguard.TenantOf(workflowID)
	if err != nil {
		return uuid.Nil, errNotSealRun
	}
	s, ok := strings.CutPrefix(workflowID, "t:"+tenant.String()+":"+outbox.SealKind+":")
	if !ok {
		return uuid.Nil, errNotSealRun
	}
	o, err := uuid.Parse(s)
	if err != nil || o == uuid.Nil || o.String() != s {
		return uuid.Nil, errNotSealRun
	}
	return o, nil
}
