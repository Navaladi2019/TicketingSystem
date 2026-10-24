// navalint holds the platform's go/analysis analyzers, run by `make lint` as
// go vet -vettool. It is its own module so golang.org/x/tools v0.52.0 stays
// out of the production go.mod (TICK-94 item 15).
module github.com/Navaladi2019/TicketingSystem/tools/navalint

go 1.26.0

toolchain go1.26.10

require golang.org/x/tools v0.52.0

require (
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
)
