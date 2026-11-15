// Command bench measures the p95 of the ADR-007 D2 lookup inside InTenantTx,
// through PgBouncer, as nava_app.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func main() {
	dsn := flag.String("dsn", "", "nava_app DSN through PgBouncer")
	n := flag.Int("n", 20000, "samples per run")
	conc := flag.Int("c", 8, "second run's concurrency")
	flag.Parse()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatal(err)
	}
	one := func(i int) time.Duration {
		t := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i%1000+1))
		c := fmt.Sprintf("host-app-%d", i%20+1)
		s := time.Now()
		err := tenancy.InTenantTx(ctx, pool, t, func(tx pgx.Tx) error {
			var o []string
			var st string
			return tx.QueryRow(ctx, `SELECT allowed_origins, state FROM embedding_client WHERE tenant_id=$1 AND client_id=$2`, t, c).Scan(&o, &st)
		})
		if err != nil {
			log.Fatal(err)
		}
		return time.Since(s)
	}
	for i := 0; i < 2000; i++ { // warm-up
		one(i)
	}
	run := func(c int) []time.Duration {
		d := make([]time.Duration, *n)
		var wg sync.WaitGroup
		jobs := make(chan int)
		for w := 0; w < c; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					d[i] = one(i)
				}
			}()
		}
		for i := 0; i < *n; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
		return d
	}
	for _, c := range []int{1, *conc} {
		d := run(c)
		fmt.Printf("concurrency=%d n=%d p50=%v p95=%v p99=%v max=%v\n", c, len(d), d[len(d)/2], d[len(d)*95/100], d[len(d)*99/100], d[len(d)-1])
	}
}
