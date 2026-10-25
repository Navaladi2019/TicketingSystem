package workerguard

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// ConfigureClient sets on o what every platform Temporal client needs, the
// relay's and nava-core's as well as the worker's:
//   - Propagator, the only ContextPropagator;
//   - the SDK tracing interceptor first, then any interceptors o already has;
//   - the SDK metrics handler;
//   - no worker heartbeats (TICK-94 item 5): with them on, every process adds
//     a new task_queue series.
func ConfigureClient(o *client.Options, tt *telemetry.Temporal) {
	o.ContextPropagators = []workflow.ContextPropagator{Propagator{}}
	o.Interceptors = append([]interceptor.ClientInterceptor{tt.Tracing}, o.Interceptors...)
	o.MetricsHandler = tt.Metrics
	o.WorkerHeartbeatInterval = -1
}

// WorkerOptions returns o with the guard as the last worker interceptor, so
// it runs inside the tracing interceptor ConfigureClient installs and closest
// to the workflow and activity code.
func WorkerOptions(o worker.Options, g *Guard) worker.Options {
	o.Interceptors = append(append([]interceptor.WorkerInterceptor(nil), o.Interceptors...), g)
	return o
}
