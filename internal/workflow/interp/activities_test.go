package interp_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp/interptest"
)

// versionStore is a DefinitionStore that returns one in-memory version, so a
// test can hand the activities a version Parse would never produce.
type versionStore struct{ v *interp.Version }

func (s versionStore) Load(context.Context, uuid.UUID, uuid.UUID) (*interp.Version, error) {
	return s.v, nil
}

// staticRecords is a RecordReader returning one record.
type staticRecords map[string]any

func (r staticRecords) Read(context.Context, uuid.UUID, uuid.UUID) (map[string]any, int64, error) {
	return r, 1, nil
}

// noWrites fails the test if ApplyStep reaches the writer.
type noWrites struct{ t *testing.T }

func (w noWrites) Write(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, map[string]any) error {
	w.t.Error("RecordWriter.Write called on a refused step")
	return nil
}

// fixtureVersion parses fixture-v1 into a fresh Version the test may mutate.
func fixtureVersion(t *testing.T) (*interp.Version, fixture) {
	t.Helper()
	f := fixtureV1(t)
	d, err := def.Parse(f.def)
	if err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]uuid.UUID, len(f.stepIDs))
	for k, id := range f.stepIDs {
		ids[k] = id
	}
	return &interp.Version{Definition: d, StepIDs: ids}, f
}

// mutStep returns the parsed step key of v for in-place change.
func mutStep(t *testing.T, v *interp.Version, key string) *def.Step {
	t.Helper()
	s, ok := v.Definition.Step(key)
	if !ok {
		t.Fatalf("fixture has no step %q", key)
	}
	return s
}

func tenantCtx() context.Context {
	return workerguard.ContextWithTenant(context.Background(), tenantA)
}

func wantNoTenant(t *testing.T, err error) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) {
		t.Fatalf("error %v, want an ApplicationError", err)
	}
	if app.Type() != "nava.interp.NoTenant" || !app.NonRetryable() || app.Message() != "interp: no verified tenant" {
		t.Errorf("error type %q non-retryable %v message %q, want nava.interp.NoTenant, non-retryable, fixed text",
			app.Type(), app.NonRetryable(), app.Message())
	}
}

// TestNextStepRefusals: every NextStep refusal is a fixed rejection (or the
// no-tenant error), never a Directive built from a broken version.
func TestNextStepRefusals(t *testing.T) {
	high := staticRecords{"priority": "high"}
	cases := []struct {
		name   string
		ctx    context.Context
		mutate func(t *testing.T, v *interp.Version, f fixture, ref *interp.StepRef)
		nilVer bool
		want   workerguard.Rejection
	}{
		{
			name: "nil definition version id",
			mutate: func(_ *testing.T, _ *interp.Version, _ fixture, ref *interp.StepRef) {
				ref.DefinitionVersionID = uuid.Nil
			},
			want: workerguard.RejectInput,
		},
		{
			name: "nil record id",
			mutate: func(_ *testing.T, _ *interp.Version, _ fixture, ref *interp.StepRef) {
				ref.RecordID = uuid.Nil
			},
			want: workerguard.RejectInput,
		},
		{name: "store returns no version", nilVer: true, want: workerguard.RejectNotFound},
		{
			name: "signal is a step id, not a signal id",
			mutate: func(t *testing.T, _ *interp.Version, f fixture, ref *interp.StepRef) {
				s := f.id(t, "await_approval")
				ref.Signal = &s
			},
			want: workerguard.RejectNotFound,
		},
		{
			name: "signal id unknown to the version",
			mutate: func(_ *testing.T, _ *interp.Version, _ fixture, ref *interp.StepRef) {
				s := uuid.New()
				ref.Signal = &s
			},
			want: workerguard.RejectNotFound,
		},
		{
			name: "step map misses the action step's own id",
			mutate: func(_ *testing.T, v *interp.Version, _ fixture, _ *interp.StepRef) {
				delete(v.StepIDs, "cool_off")
			},
			want: workerguard.RejectNotFound,
		},
		{
			name: "step map misses the action step's next",
			mutate: func(_ *testing.T, v *interp.Version, _ fixture, _ *interp.StepRef) {
				delete(v.StepIDs, "is_high")
			},
			want: workerguard.RejectNotFound,
		},
		{
			name: "condition leads to a step the definition lacks",
			mutate: func(t *testing.T, v *interp.Version, f fixture, ref *interp.StepRef) {
				mutStep(t, v, "is_high").Then = "ghost"
				ref.Step = f.id(t, "is_high")
			},
			want: workerguard.RejectNotFound,
		},
		{
			name: "step of an unknown type",
			mutate: func(t *testing.T, v *interp.Version, f fixture, ref *interp.StepRef) {
				mutStep(t, v, "is_high").Type = def.StepType("loop")
				ref.Step = f.id(t, "is_high")
			},
			want: workerguard.RejectStep,
		},
		{
			name: "condition cycle stops at the walk bound",
			mutate: func(t *testing.T, v *interp.Version, f fixture, ref *interp.StepRef) {
				s := mutStep(t, v, "is_high")
				s.Then, s.Else = "is_high", "is_high"
				ref.Step = f.id(t, "is_high")
			},
			want: workerguard.RejectStep,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, f := fixtureVersion(t)
			ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New()}
			if tc.mutate != nil {
				tc.mutate(t, v, f, &ref)
			}
			store := versionStore{v: v}
			if tc.nilVer {
				store.v = nil
			}
			a := &interp.Activities{Definitions: store, Records: high, Writer: noWrites{t}}
			d, err := a.NextStep(tenantCtx(), ref)
			wantRejected(t, err, tc.want)
			if !reflect.DeepEqual(d, workerguard.Directive{}) {
				t.Errorf("Directive %+v returned with the refusal, want zero", d)
			}
		})
	}

	t.Run("version without a definition", func(t *testing.T) {
		a := &interp.Activities{Definitions: versionStore{v: &interp.Version{}}, Records: high, Writer: noWrites{t}}
		_, err := a.NextStep(tenantCtx(), interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New()})
		wantRejected(t, err, workerguard.RejectNotFound)
	})

	t.Run("no verified tenant", func(t *testing.T) {
		v, _ := fixtureVersion(t)
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: high, Writer: noWrites{t}}
		_, err := a.NextStep(context.Background(), interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New()})
		wantNoTenant(t, err)
	})
}

// TestNextStepIsDeterministic: NextStep for the same ref over the same
// version and record returns the same Directive every time, so a retried or
// re-run activity (a redelivered signal's NextStep included) never steers the
// workflow somewhere else.
func TestNextStepIsDeterministic(t *testing.T) {
	v, f := fixtureVersion(t)
	a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{"priority": "high"}, Writer: noWrites{t}}
	sig := f.id(t, def.SignalKey("approve"))
	for name, ref := range map[string]interp.StepRef{
		"start":        {DefinitionVersionID: uuid.New(), RecordID: uuid.New()},
		"after signal": {DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "mark_approved"), Signal: &sig},
	} {
		t.Run(name, func(t *testing.T) {
			first, err := a.NextStep(tenantCtx(), ref)
			if err != nil {
				t.Fatal(err)
			}
			for range 3 {
				again, err := a.NextStep(tenantCtx(), ref)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(again, first) {
					t.Fatalf("NextStep again = %+v, first %+v", again, first)
				}
			}
		})
	}
}

// TestApplyStepRefusals: ApplyStep refuses anything but an update_record step
// of the loaded version, before it reaches the writer.
func TestApplyStepRefusals(t *testing.T) {
	cases := []struct {
		name   string
		step   func(t *testing.T, f fixture) uuid.UUID
		mutate func(t *testing.T, v *interp.Version)
		want   workerguard.Rejection
	}{
		{name: "step id not in the version", step: func(*testing.T, fixture) uuid.UUID { return uuid.New() }, want: workerguard.RejectNotFound},
		{name: "start (nil) step id", step: func(*testing.T, fixture) uuid.UUID { return uuid.Nil }, want: workerguard.RejectNotFound},
		{name: "a timer step", step: func(t *testing.T, f fixture) uuid.UUID { return f.id(t, "cool_off") }, want: workerguard.RejectStep},
		{name: "a signal id", step: func(t *testing.T, f fixture) uuid.UUID { return f.id(t, def.SignalKey("approve")) }, want: workerguard.RejectStep},
		{
			name: "a set literal that does not decode",
			step: func(t *testing.T, f fixture) uuid.UUID { return f.id(t, "mark_approved") },
			mutate: func(t *testing.T, v *interp.Version) {
				mutStep(t, v, "mark_approved").Set["approved"] = json.RawMessage(`{`)
			},
			want: workerguard.RejectStep,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, f := fixtureVersion(t)
			if tc.mutate != nil {
				tc.mutate(t, v)
			}
			a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: noWrites{t}}
			err := a.ApplyStep(tenantCtx(), interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: tc.step(t, f)})
			wantRejected(t, err, tc.want)
		})
	}

	t.Run("no verified tenant", func(t *testing.T) {
		v, f := fixtureVersion(t)
		a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: noWrites{t}}
		err := a.ApplyStep(context.Background(), interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "mark_approved")})
		wantNoTenant(t, err)
	})
}

// TestApplyStepDeliveredTwice: the same ApplyStep delivered twice (a retry
// after a lost completion) writes with the same idempotency key both times,
// so the effect is applied once.
func TestApplyStepDeliveredTwice(t *testing.T) {
	v, f := fixtureVersion(t)
	w := &interptest.FileWriter{Root: t.TempDir()}
	a := &interp.Activities{Definitions: versionStore{v: v}, Records: staticRecords{}, Writer: w}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: tenantCtx()})
	env.RegisterActivityWithOptions(a.ApplyStep, activity.RegisterOptions{Name: interp.ApplyStepActivity})

	step, record := f.id(t, "mark_approved"), uuid.New()
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: record, Step: step}
	for i := range 2 {
		if _, err := env.ExecuteActivity(interp.ApplyStepActivity, ref); err != nil {
			t.Fatalf("delivery %d: %v", i+1, err)
		}
	}
	calls, err := w.Calls()
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("writes %+v, want 2 calls", calls)
	}
	if calls[0].Key != calls[1].Key || calls[0].Key == "" {
		t.Errorf("keys %q and %q, want one key for both deliveries", calls[0].Key, calls[1].Key)
	}
	for i, c := range calls {
		if c.Tenant != tenantA || c.Record != record || c.Step != step || !reflect.DeepEqual(c.Set, map[string]any{"approved": true}) {
			t.Errorf("call %d %+v, want tenant A, the record, mark_approved, {approved: true}", i, c)
		}
	}
	if !calls[0].Applied || calls[1].Applied {
		t.Errorf("applied %v then %v, want true then false", calls[0].Applied, calls[1].Applied)
	}
	if n, err := w.Applications(calls[0].Key); err != nil || n != 1 {
		t.Errorf("applications %d (%v), want 1", n, err)
	}
}
