// workflowcheck pins Temporal's determinism checker for `make lint` (D-3-23).
// It is its own module so the checker's golang.org/x/tools stays out of the
// production go.mod, as navalint's does.
module github.com/Navaladi2019/TicketingSystem/tools/workflowcheck

go 1.26.0

toolchain go1.26.10

tool go.temporal.io/sdk/contrib/tools/workflowcheck

require (
	go.temporal.io/sdk/contrib/tools/workflowcheck v0.6.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
