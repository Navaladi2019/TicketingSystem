package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// materializeCmd is a valid OpMaterialize of record id in tenant.
func materializeCmd(tenant, id uuid.UUID) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpMaterialize, EntityKey: "incident", RecordID: id,
		Actor: identity.Principal{TenantID: tenant}, Source: pipeline.SourceSystem, ActorSystem: pipeline.ActorSchemaBackfill}
}

// CI-3 (4), ADR-009 D3: OpMaterialize is refused, before the first statement,
// unless the patch is empty, the source is system with no principal and
// actor_system is schema-backfill; and no other op names an actor_system.
// The nil tx would panic if Write reached the database.
func TestMaterializeRefusesEverythingButTheBackfill(t *testing.T) {
	p := pipeline.New(fakeAuthz{}, nopDefs{}, pipeline.NoopSink{}, pipeline.PoolRunner{}, pipeline.SystemClock{})
	tenant := uuid.New()
	cases := map[string]func(c *pipeline.Command){
		"a non-empty patch": func(c *pipeline.Command) { c.Data = map[string]any{"channel": "web"} },
		"a user source":     func(c *pipeline.Command) { c.Source = pipeline.SourceUser; c.Actor.PrincipalID = uuid.New() },
		"a workflow source": func(c *pipeline.Command) {
			c.Source = pipeline.SourceWorkflow
			c.Causation = outbox.Causation{OriginID: uuid.New(), Depth: 1}
		},
		"a principal":               func(c *pipeline.Command) { c.Actor.PrincipalID = uuid.New() },
		"another actor_system":      func(c *pipeline.Command) { c.ActorSystem = "rollout" },
		"no actor_system":           func(c *pipeline.Command) { c.ActorSystem = "" },
		"no record":                 func(c *pipeline.Command) { c.RecordID = uuid.Nil },
		"an expected version":       func(c *pipeline.Command) { c.ExpectedVersion = 3 },
		"an idempotency key":        func(c *pipeline.Command) { c.IdempotencyKey = "k" },
		"a workflow ref":            func(c *pipeline.Command) { c.WorkflowRef = uuid.New() },
		"an update with the system": func(c *pipeline.Command) { c.Op = pipeline.OpUpdate; c.ExpectedVersion = 1 },
	}
	for name, mutate := range cases {
		cmd := materializeCmd(tenant, uuid.New())
		mutate(&cmd)
		_, err := p.Write(context.Background(), nil, cmd)
		if !errors.Is(err, pipeline.ErrInvalidCommand) {
			t.Errorf("%s: err = %v, want ErrInvalidCommand", name, err)
		}
		if got := pipeline.ErrorCode(err); got != pipeline.CodeInvalidCommand {
			t.Errorf("%s: code = %q, want %q", name, got, pipeline.CodeInvalidCommand)
		}
	}
	if got := pipeline.OpMaterialize.String(); got != "materialize" {
		t.Errorf("OpMaterialize.String() = %q", got)
	}
}

// ErrorCode gives every typed refusal a stable code that fits
// schema_backfill_failures_code_check, through ErrAtVersion and wrapping.
func TestErrorCode(t *testing.T) {
	check := regexp.MustCompile(`^[a-z_]{1,64}$`)
	at := func(err error) error { return &pipeline.ErrAtVersion{SchemaVersion: 2, Err: err} }
	for err, want := range map[error]string{
		nil:                           "",
		at(&pipeline.ErrValidation{}): pipeline.CodeValidationFailed,
		at(&pipeline.ErrUniqueViolation{Field: "x"}):      pipeline.CodeUniqueViolation,
		at(&pipeline.ErrIllegalTransition{}):              pipeline.CodeIllegalTransition,
		at(pipeline.ErrAssignmentGroupRemoved):            pipeline.CodeAssignmentGroupRemoved,
		at(pipeline.ErrVersionConflict):                   pipeline.CodeVersionConflict,
		pipeline.ErrNotFound:                              pipeline.CodeNotFound,
		fmt.Errorf("%w: x", pipeline.ErrInvalidCommand):   pipeline.CodeInvalidCommand,
		authz.ErrForbidden:                                pipeline.CodeForbidden,
		&authz.ErrFieldNotWritable{Fields: []string{"a"}}: pipeline.CodeForbidden,
		fmt.Errorf("x: %w", outbox.ErrCascadeTooDeep):     pipeline.CodeCascadeTooDeep,
		fmt.Errorf("%w: x", pipeline.ErrRetryable):        pipeline.CodeRetryable,
		errors.New("connection reset"):                    pipeline.CodeDatabaseError,
	} {
		got := pipeline.ErrorCode(err)
		if got != want {
			t.Errorf("ErrorCode(%v) = %q, want %q", err, got, want)
		}
		if err != nil && !check.MatchString(got) {
			t.Errorf("ErrorCode(%v) = %q does not fit the failures code check", err, got)
		}
	}
}
