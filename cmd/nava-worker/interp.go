package main

import (
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// registerInterpreter registers nava.interpreter and its NextStep and
// ApplyStep activities (regular activities, never local) on w, which polls
// nava-tq (D-3-11), behind the guard workerOptions installs. acts nil
// registers nothing: interpreterActivities returns nil in a production build
// (interp_default.go), and the worker stays activity-only.
func registerInterpreter(w worker.Registry, acts *interp.Activities) {
	if acts == nil {
		return
	}
	interp.RegisterWorkflow(w)
	acts.Register(w)
}
