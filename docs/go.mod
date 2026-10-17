// This file only fences docs/ off from the root module. The ADR spikes under
// docs/adr/ADR-001/spikes are historical evidence with their own Temporal
// imports; they are not platform code and must not join `go build ./...`.
module github.com/Navaladi2019/TicketingSystem/docs

go 1.26
