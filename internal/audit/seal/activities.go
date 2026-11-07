package seal

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The registered activity names. The sealer workflow (P5-34b) schedules them.
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

// Activities are A1 and A2 on a nava_worker pool. Register them with
// Register.
type Activities struct {
	Pool *pgxpool.Pool
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

// Seal is A2. It returns no count: workerguard records only references, and
// the run's outcome the next A1 needs is already on the head
// (seal_interval_s).
func (a *Activities) Seal(ctx context.Context) error {
	_, err := Seal(ctx, a.Pool)
	if errors.Is(err, ErrUnsealable) {
		return temporal.NewNonRetryableApplicationError("seal: entry cannot be canonicalised", ErrTypeUnsealable, nil)
	}
	return err
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
