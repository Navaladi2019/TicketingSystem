package workerguard

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// replayHistory is the history of one run of collect in tenantA: started with
// tenantA's header, then "add" with tenantB's header (a deny when it ran),
// "add" and "done" with tenantA's, then completed.
func replayHistory(t *testing.T) *historypb.History {
	t.Helper()
	dc := converter.GetDefaultDataConverter()
	payloads := func(v ...any) *commonpb.Payloads {
		p, err := dc.ToPayloads(v...)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	now := timestamppb.New(time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC))
	var events []*historypb.HistoryEvent
	add := func(typ enumspb.EventType, attrs func(*historypb.HistoryEvent)) {
		e := &historypb.HistoryEvent{EventId: int64(len(events) + 1), EventTime: now, EventType: typ}
		attrs(e)
		events = append(events, e)
	}
	task := func() {
		scheduled := int64(len(events) + 1)
		add(enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED, func(e *historypb.HistoryEvent) {
			e.Attributes = &historypb.HistoryEvent_WorkflowTaskScheduledEventAttributes{WorkflowTaskScheduledEventAttributes: &historypb.WorkflowTaskScheduledEventAttributes{
				TaskQueue: &taskqueuepb.TaskQueue{Name: "tq"},
			}}
		})
		add(enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED, func(e *historypb.HistoryEvent) {
			e.Attributes = &historypb.HistoryEvent_WorkflowTaskStartedEventAttributes{WorkflowTaskStartedEventAttributes: &historypb.WorkflowTaskStartedEventAttributes{
				ScheduledEventId: scheduled,
			}}
		})
		add(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED, func(e *historypb.HistoryEvent) {
			e.Attributes = &historypb.HistoryEvent_WorkflowTaskCompletedEventAttributes{WorkflowTaskCompletedEventAttributes: &historypb.WorkflowTaskCompletedEventAttributes{
				ScheduledEventId: scheduled, StartedEventId: scheduled + 1,
			}}
		})
	}
	signal := func(name string, arg any, tenant string) {
		add(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED, func(e *historypb.HistoryEvent) {
			e.Attributes = &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
				SignalName: name, Input: payloads(arg),
				Header: &commonpb.Header{Fields: map[string]*commonpb.Payload{TenantHeader: stringPayload(t, tenant)}},
			}}
		})
	}

	add(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, func(e *historypb.HistoryEvent) {
		e.Attributes = &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
			WorkflowType: &commonpb.WorkflowType{Name: "collect"},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: "tq"},
			Header:       headerFor(t, tenantA),
		}}
	})
	task()
	signal("add", "other-tenant", tenantB.String())
	signal("add", "own", tenantA.String())
	signal("done", nil, tenantA.String())
	task()
	add(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, func(e *historypb.HistoryEvent) {
		e.Attributes = &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{
			Result: payloads([]string{"own"}), WorkflowTaskCompletedEventId: int64(len(events)),
		}}
	})
	return &historypb.History{Events: events}
}

// Replaying a history with an allowed start, a denied signal and allowed
// signals counts nothing and logs nothing: a cache eviction must not re-alert.
// The guard still runs: the wrong-tenant signal is dropped again on replay.
func TestReplayRecordsNothing(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	logs := &syncBuffer{}
	g, err := New(slog.New(slog.NewJSONHandler(logs, nil)), sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	r, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		Interceptors:       []interceptor.WorkerInterceptor{g},
		ContextPropagators: []workflow.ContextPropagator{Propagator{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var applied []string
	r.RegisterWorkflowWithOptions(func(ctx workflow.Context) ([]string, error) {
		got, err := collect(ctx)
		applied = got
		return got, err
	}, workflow.RegisterOptions{Name: "collect"})

	err = r.ReplayWorkflowHistoryWithOptions(nil, replayHistory(t), worker.ReplayWorkflowHistoryOptions{
		OriginalExecution: workflow.Execution{ID: idOf(tenantA), RunID: "run-1"},
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if strings.Join(applied, ",") != "own" {
		t.Errorf("replay applied %v, want [own]: the guard must still drop the wrong-tenant signal", applied)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == DecisionsMetric {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok && len(sum.DataPoints) > 0 {
					t.Errorf("replay recorded %d %s points, want none", len(sum.DataPoints), DecisionsMetric)
				}
			}
		}
	}
	if s := logs.String(); s != "" {
		t.Errorf("replay logged:\n%s", s)
	}
}
