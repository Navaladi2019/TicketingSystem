package workerguard

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// Results, side effects and failure text (TICK-245, Security triage rows 3 to
// 7). Like arguments, each is recorded in a cross-tenant, unencrypted history
// (ADR-001 R1), so the guard checks it before the SDK sends it.

// ErrTypeOpaque is the ApplicationError type of a failure the guard withheld:
// its message and details are replaced by fixed text, and the original goes to
// the worker log only.
const ErrTypeOpaque = "nava.Opaque"

const opaqueMessage = "error withheld from the history; the worker log has its shape"

// sideEffectDenied is what a denied SideEffect records in its marker instead
// of the value. It is not a UUID, a time, a number, a bool or an object, so
// no allowed value encodes to it, and a replay reads the deny back from the
// marker.
const sideEffectDenied = "\x00nava.TenantGuard: side effect withheld"

// resultOK reports whether an activity, workflow or update result may be
// recorded: nil (or a nil value, which encodes as null), or a value argOK
// accepts.
func resultOK(v any) bool {
	if v == nil {
		return true
	}
	// Already encoded, as the SDK test environment hands a local activity's
	// result: each payload is checked as recorded, by the generic rule.
	if ps, ok := v.(*commonpb.Payloads); ok {
		for _, p := range ps.GetPayloads() {
			if string(p.GetMetadata()[converter.MetadataEncoding]) != converter.MetadataEncodingNil &&
				string(p.GetData()) != "null" && outbox.ValidatePayload(p.GetData()) != nil {
				return false
			}
		}
		return true
	}
	p, err := DataConverter().ToPayload(v)
	if err != nil {
		return false
	}
	return string(p.GetData()) == "null" || argOK(v)
}

// sideEffectValueOK reports whether a SideEffect value may be recorded: nil,
// a canonical UUID (ChildWorkflowID draws one), an RFC3339 time, a number, a
// bool, or a value argOK accepts.
func sideEffectValueOK(v any) bool {
	if v == nil {
		return true
	}
	p, err := DataConverter().ToPayload(v)
	if err != nil {
		return false
	}
	switch string(p.GetMetadata()[converter.MetadataEncoding]) {
	case converter.MetadataEncodingNil:
		return true
	case converter.MetadataEncodingJSON:
	default:
		return false
	}
	var x any
	if json.Unmarshal(p.GetData(), &x) != nil {
		return false
	}
	switch x := x.(type) {
	case nil, bool, float64:
		return true
	case string:
		if u, err := uuid.Parse(x); err == nil && u.String() == x {
			return true
		}
		_, err := time.Parse(time.RFC3339Nano, x)
		return err == nil
	}
	return argOK(v)
}

// failureTypeAllowed lists the ApplicationError types whose message is
// recorded as is. Their messages are constants or ids (review rule, TICK-245):
// add a type here only with that review.
func failureTypeAllowed(t string) bool {
	return t == ErrTypeTenantGuard || t == ErrTypeWorkflowRef
}

// passesUnchanged: SDK errors whose text the SDK writes (cancel, timeout,
// async completion), and the wrappers of an activity or child failure, whose
// cause the guard of the worker that ran it already checked. Only the top
// level counts: a wrapper built around one has workflow code's text.
func passesUnchanged(err error) bool {
	switch err.(type) { //nolint:errorlint // the top level only, as above
	case *temporal.CanceledError, *temporal.TimeoutError, *temporal.ActivityError,
		*temporal.ChildWorkflowExecutionError, *workflow.ContinueAsNewError:
		return true
	}
	//nolint:errorlint // the top level only
	return err == activity.ErrResultPending || err == context.Canceled
}

// recordable returns whether the guard changed err, and err as it may be
// recorded in a history:
//   - SDK errors pass unchanged (passesUnchanged);
//   - an ApplicationError of an allowed type with no cause keeps its message,
//     type and flags, and only the details argOK accepts;
//   - anything else becomes ErrTypeOpaque with fixed text and the same
//     non-retryable flag.
func recordable(err error) (changed bool, out error) {
	// Reading details uses reflection; a guard never panics, so a failure
	// here withholds the error.
	defer func() {
		if recover() != nil {
			changed, out = true, temporal.NewApplicationError(opaqueMessage, ErrTypeOpaque)
		}
	}()
	if err == nil || passesUnchanged(err) {
		return false, err
	}
	app, isApp := err.(*temporal.ApplicationError) //nolint:errorlint // the top level is what the SDK records
	if isApp && failureTypeAllowed(app.Type()) && app.Unwrap() == nil {
		kept, dropped := checkedDetails(app)
		if !dropped {
			return false, err
		}
		return true, temporal.NewApplicationErrorWithOptions(app.Message(), app.Type(), temporal.ApplicationErrorOptions{
			NonRetryable: app.NonRetryable(), Details: kept, NextRetryDelay: app.NextRetryDelay(), Category: app.Category(),
		})
	}
	return true, temporal.NewApplicationErrorWithOptions(opaqueMessage, ErrTypeOpaque, temporal.ApplicationErrorOptions{
		NonRetryable: isApp && app.NonRetryable(),
	})
}

// maxDetails bounds the details read from one error.
const maxDetails = 16

// checkedDetails returns the details of app that argOK accepts, and whether
// any was dropped. The SDK has no count, so they are read one more at a time
// until it reports too many.
func checkedDetails(app *temporal.ApplicationError) ([]any, bool) {
	if !app.HasDetails() {
		return nil, false
	}
	var all []any
	for n := 1; n <= maxDetails; n++ {
		ptrs := make([]any, n)
		vals := make([]any, n)
		for i := range ptrs {
			ptrs[i] = &vals[i]
		}
		if app.Details(ptrs...) != nil {
			break
		}
		all = vals
	}
	kept := make([]any, 0, len(all))
	for _, d := range all {
		if d != nil && argOK(d) {
			kept = append(kept, d)
		}
	}
	return kept, len(all) == 0 || len(kept) != len(all)
}

// withheld logs the shape of a failure the guard replaced: the Go types of
// its chain and any ApplicationError type, never its text, which is what the
// guard withheld (house rule: log ids and shapes, not contents). The alert is
// the deny line next to it.
func (r *recorder) withheld(guard string, err error, attrs ...slog.Attr) {
	defer func() { _ = recover() }()
	base := []slog.Attr{slog.String("guard", guard), slog.String("error_shape", errorShape(err))}
	r.log.LogAttrs(context.Background(), slog.LevelError, "worker tenant guard withheld a failure from the history", append(base, attrs...)...)
}

// maxShape bounds the chain errorShape walks.
const maxShape = 8

// errorShape is the Go type of each error in err's chain, with the type of an
// ApplicationError in brackets: "*fmt.wrapError / *internal.ApplicationError[nava.TenantGuard]".
// Types are code, not data.
func errorShape(err error) string {
	var b strings.Builder
	for i := 0; err != nil && i < maxShape; i++ {
		if i > 0 {
			b.WriteString(" / ")
		}
		b.WriteString(reflect.TypeOf(err).String())
		if app, ok := err.(*temporal.ApplicationError); ok { //nolint:errorlint // one link at a time
			b.WriteString("[" + app.Type() + "]")
		}
		err = errors.Unwrap(err)
	}
	return b.String()
}

// --- workflow results and failures ----------------------------------------------

// workflowOutcome checks what a workflow run, or an update, ends with: a
// result that is not references only is replaced by the guard error, a
// failure by its recordable form. ContinueAsNewError is checked by the
// caller.
func (r *recorder) workflowOutcome(ctx workflow.Context, resultGuard, failureGuard string, result any, err error, attrs ...slog.Attr) (any, error) {
	if err != nil {
		changed, out := recordable(err)
		if changed && !workflow.IsReplaying(ctx) {
			r.withheld(failureGuard, err, append(workflowAttrs(ctx), attrs...)...)
			r.record(ctx, failureGuard, outcomeRecordData, attrs...)
		}
		return nil, out
	}
	outcome := check(func() string {
		if resultOK(result) {
			return outcomeAllow
		}
		return outcomeRecordData
	})
	r.record(ctx, resultGuard, outcome, attrs...)
	if outcome != outcomeAllow {
		return nil, guardError(resultGuard, outcome)
	}
	return result, nil
}

// --- activity results and failures ----------------------------------------------

// activityOutcome is workflowOutcome for an activity or local activity. A
// panic is recovered by the caller and arrives here as activityPanic, so its
// value and stack never reach the history.
func (r *recorder) activityOutcome(result any, err error, attrs ...slog.Attr) (any, error) {
	if err != nil {
		changed, out := recordable(err)
		if changed {
			r.withheld(guardActivityFailure, err, attrs...)
			r.deny(guardActivityFailure, outcomeRecordData, attrs...)
		}
		return nil, out
	}
	if check(func() string {
		if resultOK(result) {
			return outcomeAllow
		}
		return outcomeRecordData
	}) != outcomeAllow {
		r.deny(guardActivityResult, outcomeRecordData, attrs...)
		return nil, guardError(guardActivityResult, outcomeRecordData)
	}
	r.allow(guardActivityResult)
	return result, nil
}

// activityPanic stands for a recovered panic. The panic value may hold record
// data and is dropped, with the stack.
type activityPanic struct{}

func (activityPanic) Error() string { return "activity panic" }

// --- side effects ---------------------------------------------------------------

// sideEffectValue is the EncodedValue of a SideEffect: the recorded value,
// or, when its marker holds sideEffectDenied, the guard error.
type sideEffectValue struct{ converter.EncodedValue }

func (v sideEffectValue) denied() bool {
	var s string
	return v.EncodedValue != nil && v.EncodedValue.HasValue() && v.EncodedValue.Get(&s) == nil && s == sideEffectDenied
}

func (v sideEffectValue) HasValue() bool { return !v.denied() && v.EncodedValue.HasValue() }

func (v sideEffectValue) Get(valuePtr any) error {
	if v.denied() {
		return guardError(guardSideEffect, outcomeRecordData)
	}
	return v.EncodedValue.Get(valuePtr)
}

// deniedValue is the EncodedValue of a call the guard refused outright.
type deniedValue struct{ err error }

func (deniedValue) HasValue() bool  { return false }
func (d deniedValue) Get(any) error { return d.err }
