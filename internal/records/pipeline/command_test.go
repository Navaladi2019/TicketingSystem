package pipeline_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// nopDefs is a DefinitionReader that finds nothing.
type nopDefs struct{}

func (nopDefs) Version(context.Context, pgx.Tx, string, int) (*metadata.Compiled, error) {
	return nil, entitytypes.ErrNotFound
}

func (nopDefs) Current(context.Context, pgx.Tx, string) (*metadata.Compiled, int, error) {
	return nil, 0, entitytypes.ErrNotFound
}

// Every malformed command is refused before the first statement: the nil tx
// would panic if Write reached the database.
func TestWriteRefusesBadCommands(t *testing.T) {
	p := pipeline.New(authz.TenantMemberAuthorizer{}, nopDefs{}, pipeline.NoopSink{}, pipeline.PoolRunner{}, pipeline.SystemClock{})
	user := identity.Principal{TenantID: uuid.New(), PrincipalID: uuid.New()}
	system := identity.Principal{TenantID: user.TenantID}
	valid := pipeline.Command{Op: pipeline.OpCreate, EntityKey: "incident", Actor: user, Source: pipeline.SourceUser}
	cases := map[string]func(c *pipeline.Command){
		"unknown op":               func(c *pipeline.Command) { c.Op = 0 },
		"create naming a record":   func(c *pipeline.Command) { c.RecordID = uuid.New() },
		"create with a version":    func(c *pipeline.Command) { c.ExpectedVersion = 1 },
		"update without a record":  func(c *pipeline.Command) { c.Op = pipeline.OpUpdate; c.ExpectedVersion = 1 },
		"update without a version": func(c *pipeline.Command) { c.Op = pipeline.OpUpdate; c.RecordID = uuid.New() },
		"no entity key":            func(c *pipeline.Command) { c.EntityKey = "" },
		"unknown source":           func(c *pipeline.Command) { c.Source = "admin" },
		"user write, no principal": func(c *pipeline.Command) { c.Actor = system },
		"user write with a key":    func(c *pipeline.Command) { c.IdempotencyKey = "k" },
		"key too long": func(c *pipeline.Command) {
			c.Source, c.IdempotencyKey = pipeline.SourceWorkflow, strings.Repeat("k", 201)
		},
		"negative causation depth":   func(c *pipeline.Command) { c.Causation = outbox.Causation{Depth: -1} },
		"causation depth, no origin": func(c *pipeline.Command) { c.Causation = outbox.Causation{Depth: 1} },
		// A5-3: a depth-0 write's origin is minted, never given.
		"depth 0 naming an origin": func(c *pipeline.Command) { c.Causation = outbox.Causation{OriginID: uuid.New()} },
		// A5-7: a workflow write is one deeper than its delivery.
		"workflow write at depth 0": func(c *pipeline.Command) { c.Actor, c.Source = system, pipeline.SourceWorkflow },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := valid
			mutate(&c)
			if _, err := p.Write(context.Background(), nil, c); !errors.Is(err, pipeline.ErrInvalidCommand) {
				t.Fatalf("err = %v, want ErrInvalidCommand", err)
			}
		})
	}
}

func TestNewNeedsEveryDependency(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New with a nil sink did not panic")
		}
	}()
	pipeline.New(authz.TenantMemberAuthorizer{}, nopDefs{}, nil, pipeline.PoolRunner{}, pipeline.SystemClock{})
}

// A5-7 names the error a workflow write at depth 0 gets: ErrInvalidEntry,
// as well as the pipeline's own ErrInvalidCommand.
func TestWorkflowWriteAtDepthZeroIsInvalidEntry(t *testing.T) {
	p := pipeline.New(authz.TenantMemberAuthorizer{}, nopDefs{}, pipeline.NoopSink{}, pipeline.PoolRunner{}, pipeline.SystemClock{})
	c := pipeline.Command{Op: pipeline.OpCreate, EntityKey: "incident", Actor: identity.Principal{TenantID: uuid.New()},
		Source: pipeline.SourceWorkflow}
	if _, err := p.Write(context.Background(), nil, c); !errors.Is(err, outbox.ErrInvalidEntry) || !errors.Is(err, pipeline.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidEntry and ErrInvalidCommand", err)
	}
}

// The message names paths and keywords, never values.
func TestValidationErrorMessage(t *testing.T) {
	err := &pipeline.ErrValidation{Fields: []metadata.FieldError{{Path: "/title", Code: "required"}}}
	if got, want := err.Error(), "pipeline: validation failed at /title (required)"; got != want {
		t.Fatalf("message %q, want %q", got, want)
	}
}

func TestOpString(t *testing.T) {
	if pipeline.OpCreate.String() != "create" || pipeline.OpUpdate.String() != "update" || pipeline.Op(9).String() != "unknown" {
		t.Fatal("audit actions changed")
	}
}

// AuditSuppressed audits only an unclaimed start that ClaimStart built; the
// zero Claim names no record and is refused before any statement runs.
func TestAuditSuppressedRefusesZeroClaim(t *testing.T) {
	err := pipeline.AuditSuppressed(context.Background(), nil, cascade.Claim{}, pipeline.Event{}, time.Now())
	if !errors.Is(err, cascade.ErrInvalidKey) {
		t.Fatalf("AuditSuppressed(zero claim) = %v, want cascade.ErrInvalidKey", err)
	}
}
