//go:build integration

// TICK-674 (PLAN-006 P6-09, ADR-005 §2.2 A7, §3.3, §3.5, §3.6, §4.1): a
// create of a type linked to a number format takes its number from the
// counter of the format's scope (e1|d:202611), which PostgreSQL computes,
// with period_end, from now() in the numbering time zone, and stores that
// now() as created_at, so the number's date and created_at agree, midnight
// included. On both numbering paths, with no added statement.

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/numberformats"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// frozenClock shadows now() for nava_app in the platform database: a
// function test_clock.now() first on nava_app's search_path, reading the
// instant from test_clock.frozen. Every unqualified now() of the create path
// then returns it, while Go's clock keeps the real time, so a date taken from
// Go instead of the statement's now() fails the tests. Column defaults were
// bound to pg_catalog.now() when created and are unaffected. The pools are
// reopened so every connection has the setting.
type frozenClock struct {
	e *env
}

func freezeClock(t *testing.T, e *env, at time.Time) frozenClock {
	t.Helper()
	for _, sql := range []string{
		`CREATE SCHEMA test_clock`,
		`CREATE TABLE test_clock.frozen (at timestamptz NOT NULL)`,
		`CREATE FUNCTION test_clock.now() RETURNS timestamptz LANGUAGE sql STABLE AS 'SELECT at FROM test_clock.frozen'`,
		`GRANT USAGE ON SCHEMA test_clock TO nava_app`,
		`GRANT SELECT ON test_clock.frozen TO nava_app`,
		`ALTER ROLE nava_app IN DATABASE ` + pgx.Identifier{e.platform}.Sanitize() + ` SET search_path = test_clock, pg_catalog, "$user", public`,
	} {
		mustExec(t, e.super, sql)
	}
	t.Cleanup(func() {
		_, _ = e.super.Exec(context.Background(), `ALTER ROLE nava_app IN DATABASE `+pgx.Identifier{e.platform}.Sanitize()+` RESET search_path`)
	})
	mustExec(t, e.super, `INSERT INTO test_clock.frozen VALUES ($1)`, at)
	dsn := appDSN(t, e.platform)
	e.app, e.alloc = appPool(t, dsn), allocPool(t, dsn)
	return frozenClock{e: e}
}

// set moves the frozen instant.
func (c frozenClock) set(t *testing.T, at time.Time) {
	t.Helper()
	mustExec(t, c.e.super, `UPDATE test_clock.frozen SET at = $1`, at)
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// linkFormat publishes a number format with template, overflow and zone
// (""= the tenant's), and links its version 1 to the incident type, in one
// committed transaction. It returns the format's id.
func linkFormat(t *testing.T, e env, f tenantFixture, template string, overflow numbering.Overflow, zone string) uuid.UUID {
	t.Helper()
	ff := numbering.Format{Template: template, Overflow: overflow, TimeZone: zone}
	c, err := numbering.CompileFormat(ff)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(ff)
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		v, err := numberformats.Store{}.Create(context.Background(), tx, f.id, numberformats.NewInput{
			Key: fmt.Sprintf("fmt_%d", time.Now().UnixNano()), Document: doc, Compiled: c, CreatedBy: f.principal.PrincipalID,
		})
		if err != nil {
			return err
		}
		id = v.ID
		return nil
	})
	relink(t, e, f, &entitytypes.FormatRef{ID: id, Version: 1})
	return id
}

// relink publishes the incident type again with ref pinned (nil: the
// default numbering).
func relink(t *testing.T, e env, f tenantFixture, ref *entitytypes.FormatRef) {
	t.Helper()
	inTx(t, e, f.id, func(tx pgx.Tx) error {
		_, err := entitytypes.Store{}.Publish(context.Background(), tx, f.id, entitytypes.PublishInput{
			Key: entityKey, PublishedBy: f.principal.PrincipalID, Definition: definition(schemaV1),
			NumberFormat: entitytypes.NumberFormatChange{Set: true, Ref: ref},
		})
		return err
	})
}

func setTenantZone(t *testing.T, e env, f tenantFixture, zone string) {
	t.Helper()
	mustExec(t, e.super, `UPDATE tenants SET time_zone = $2 WHERE id = $1`, f.id, zone)
}

// counter is one record_counters row of the incident type.
type counter struct {
	key       string
	next      int
	periodEnd *time.Time
}

func counters(t *testing.T, e env, f tenantFixture) []counter {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT c.scope_key, c.next, c.period_end FROM record_counters c
  JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
  WHERE c.tenant_id = $1 AND d.key = $2 ORDER BY c.scope_key`, f.id, entityKey)
	if err != nil {
		t.Fatal(err)
	}
	var out []counter
	for rows.Next() {
		var c counter
		if err := rows.Scan(&c.key, &c.next, &c.periodEnd); err != nil {
			t.Fatal(err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// createdAt is the stored created_at of record id.
func createdAt(t *testing.T, e env, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := e.super.QueryRow(context.Background(), `SELECT created_at FROM records WHERE id = $1`, id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// seedFor is a fresh tenant whose incident type numbers in mode.
func seedFor(t *testing.T, e env, mode string) tenantFixture {
	t.Helper()
	if mode == "gapless" {
		return seedTenantWith(t, e, gaplessDefinition(definition(schemaV1)))
	}
	return seedTenant(t, e)
}

// modes are the two numbering paths.
var modes = []string{"gaps", "gapless"}

// AC (midnight): tenant zone Asia/Kolkata, a create at 23:59:59.9 local and
// one at 00:00:00.1 the next local day. Each number's date is created_at's
// local date, each day starts its own counter at 1, and period_end is the
// next local midnight. The second instant is still 31 March in UTC, so a date
// read in UTC, or from Go's clock, fails.
func TestFormatMidnightDateMatchesCreatedAt(t *testing.T) {
	kolkata := mustZone(t, "Asia/Kolkata")
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			setTenantZone(t, e, f, "Asia/Kolkata")
			linkFormat(t, e, f, "{prefix}-{yyyy}{MM}{dd}-{seq:3}", numbering.Grow, "")
			before := time.Date(2031, 3, 31, 23, 59, 59, 900_000_000, kolkata)
			after := time.Date(2031, 4, 1, 0, 0, 0, 100_000_000, kolkata)
			clock := freezeClock(t, &e, before)
			p := newPipeline(e, nil)
			first := mustCreate(t, e, p, f, map[string]any{"title": "before midnight"})
			clock.set(t, after)
			second := mustCreate(t, e, p, f, map[string]any{"title": "after midnight"})

			for _, c := range []struct {
				res  pipeline.Result
				at   time.Time
				want string
			}{{first, before, "INC-20310331-001"}, {second, after, "INC-20310401-001"}} {
				if c.res.Number != c.want {
					t.Errorf("number %q, want %q", c.res.Number, c.want)
				}
				got := createdAt(t, e, c.res.RecordID)
				if !got.Equal(c.at) {
					t.Errorf("%s: created_at %v, want the allocation's now() %v", c.want, got, c.at)
				}
				if d := got.In(kolkata).Format("20060102"); !strings.Contains(c.res.Number, d) {
					t.Errorf("%s: created_at's local date %s is not the number's", c.res.Number, d)
				}
			}
			got := counters(t, e, f)
			want := []counter{
				{"e1|d:20310331", 2, ptr(time.Date(2031, 4, 1, 0, 0, 0, 0, kolkata))},
				{"e1|d:20310401", 2, ptr(time.Date(2031, 4, 2, 0, 0, 0, 0, kolkata))},
			}
			if !equalCounters(got, want) {
				t.Errorf("counters %s, want %s", fmtCounters(got), fmtCounters(want))
			}
		})
	}
}

// AC: a monthly format, two fixture months: two scope rows, each from 1, each
// ending at its next month in the tenant zone (UTC); the UPDATE arm leaves
// period_end as the INSERT arm set it.
func TestFormatMonthlyScopesStartAtOne(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			linkFormat(t, e, f, "{prefix}-{yyyy}{MM}-{seq:4}", numbering.Grow, "")
			clock := freezeClock(t, &e, time.Date(2031, 1, 15, 10, 0, 0, 0, time.UTC))
			p := newPipeline(e, nil)
			var got []string
			for i := range 2 {
				got = append(got, mustCreate(t, e, p, f, map[string]any{"title": fmt.Sprint("jan", i)}).Number)
			}
			clock.set(t, time.Date(2031, 2, 3, 10, 0, 0, 0, time.UTC))
			got = append(got, mustCreate(t, e, p, f, map[string]any{"title": "feb"}).Number)
			if want := []string{"INC-203101-0001", "INC-203101-0002", "INC-203102-0001"}; !slices.Equal(got, want) {
				t.Errorf("numbers %v, want %v", got, want)
			}
			rows := counters(t, e, f)
			want := []counter{
				{"e1|d:203101", 3, ptr(time.Date(2031, 2, 1, 0, 0, 0, 0, time.UTC))},
				{"e1|d:203102", 2, ptr(time.Date(2031, 3, 1, 0, 0, 0, 0, time.UTC))},
			}
			if !equalCounters(rows, want) {
				t.Errorf("counters %s, want %s", fmtCounters(rows), fmtCounters(want))
			}
		})
	}
}

// AC (§3.6): changing the tenant time zone changes only future scope keys.
// The format's own zone, when set, wins over the tenant's.
func TestFormatTenantTimeZoneChangeMovesOnlyFutureKeys(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			linkFormat(t, e, f, "{prefix}-{yyyy}{MM}{dd}-{seq:3}", numbering.Grow, "")
			// 20:00 UTC on 31 January is 01:30 on 1 February in Kolkata.
			freezeClock(t, &e, time.Date(2031, 1, 31, 20, 0, 0, 0, time.UTC))
			p := newPipeline(e, nil)
			first := mustCreate(t, e, p, f, map[string]any{"title": "utc"}).Number
			before := counters(t, e, f)
			setTenantZone(t, e, f, "Asia/Kolkata")
			second := mustCreate(t, e, p, f, map[string]any{"title": "kolkata"}).Number
			if first != "INC-20310131-001" || second != "INC-20310201-001" {
				t.Errorf("numbers %s, %s; want INC-20310131-001, INC-20310201-001", first, second)
			}
			after := counters(t, e, f)
			if len(before) != 1 || len(after) != 2 || !equalCounters(after[:1], before) {
				t.Errorf("counters before %s, after %s: the earlier scope row changed", fmtCounters(before), fmtCounters(after))
			}

			// A format with its own zone ignores the tenant's.
			linkFormat(t, e, f, "{prefix}/{yyyy}{MM}{dd}/{seq:3}", numbering.Grow, "America/New_York")
			if n := mustCreate(t, e, p, f, map[string]any{"title": "new york"}).Number; n != "INC/20310131/001" {
				t.Errorf("number with the format's zone %s, want INC/20310131/001", n)
			}
		})
	}
}

// AC (§3.3): re-adopting an earlier template continues that epoch's counter;
// a date-free format numbers on its own epoch's row, never on epoch 0's ”
// (Security, P6-03 review).
func TestFormatReadoptedTemplateContinuesItsEpoch(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			freezeClock(t, &e, time.Date(2031, 1, 15, 10, 0, 0, 0, time.UTC))
			p := newPipeline(e, nil)
			create := func(title string) string { return mustCreate(t, e, p, f, map[string]any{"title": title}).Number }
			var got []string
			got = append(got, create("default"))
			monthly := linkFormat(t, e, f, "{prefix}-{yyyy}{MM}-{seq:4}", numbering.Grow, "")
			got = append(got, create("a1"), create("a2"))
			linkFormat(t, e, f, "{prefix}/{seq:5}", numbering.Grow, "")
			got = append(got, create("b1"))
			relink(t, e, f, &entitytypes.FormatRef{ID: monthly, Version: 1})
			got = append(got, create("a3"))
			relink(t, e, f, nil)
			got = append(got, create("default again"))
			want := []string{"INC0000001", "INC-203101-0001", "INC-203101-0002", "INC/00001", "INC-203101-0003", "INC0000002"}
			if !slices.Equal(got, want) {
				t.Errorf("numbers %v, want %v", got, want)
			}
			rows := counters(t, e, f)
			wantRows := []counter{{"", 3, nil}, {"e1|d:203101", 4, ptr(time.Date(2031, 2, 1, 0, 0, 0, 0, time.UTC))}, {"e2", 2, nil}}
			if !equalCounters(rows, wantRows) {
				t.Errorf("counters %s, want %s", fmtCounters(rows), fmtCounters(wantRows))
			}
		})
	}
}

// AC (§3.5): refuse at the width fails the create with
// ErrNumberSpaceExhausted naming the type and scope and writes no record;
// the next create in a new scope succeeds from 1. A `gapless` refusal gives
// its value back (the counter stays at the width). Every number taken is a
// fill-ratio observation, on both paths: 8 of the first 10 at or under 0.9
// of {seq:1}'s 9, the 9th and the refused 10th above it, which is what
// NumberSpaceNearlyExhausted alerts on.
func TestFormatRefuseAtTheWidth(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			linkFormat(t, e, f, "{prefix}-{yyyy}{MM}-{seq:1}", numbering.Refuse, "")
			clock := freezeClock(t, &e, time.Date(2031, 1, 15, 10, 0, 0, 0, time.UTC))
			reader := sdkmetric.NewManualReader()
			m, err := pipeline.NewAllocMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
			if err != nil {
				t.Fatal(err)
			}
			runner := e.runner()
			runner.Metrics = m
			p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
			for i := 1; i <= 9; i++ {
				if n := mustCreate(t, e, p, f, map[string]any{"title": fmt.Sprint(i)}).Number; n != fmt.Sprintf("INC-203101-%d", i) {
					t.Fatalf("create %d: number %s", i, n)
				}
			}
			_, err = write(context.Background(), e, p, f.id, createCmd(f, map[string]any{"title": "tenth"}))
			var ex *pipeline.ErrNumberSpaceExhausted
			if !errors.As(err, &ex) || ex.EntityKey != entityKey || ex.ScopeKey != "e1|d:203101" || ex.Width != 1 {
				t.Fatalf("tenth create: %v, want ErrNumberSpaceExhausted for incident in e1|d:203101", err)
			}
			if n := len(numbers(t, e, f.id, entityKey)); n != 9 {
				t.Errorf("%d records, want 9", n)
			}
			wantNext := map[string]int{"gaps": 11, "gapless": 10}[mode]
			if rows := counters(t, e, f); len(rows) != 1 || rows[0].next != wantNext {
				t.Errorf("counters %s, want next %d", fmtCounters(rows), wantNext)
			}
			if count, under := fillRatio(t, reader); count != 10 || under != 8 {
				t.Errorf("fill ratio: %d observations, %d at or under 0.9; want 10 and 8", count, under)
			}
			clock.set(t, time.Date(2031, 2, 1, 0, 0, 1, 0, time.UTC))
			if n := mustCreate(t, e, p, f, map[string]any{"title": "february"}).Number; n != "INC-203102-1" {
				t.Errorf("first create in the new month %s, want INC-203102-1", n)
			}
		})
	}
}

// AC (§2.1.1 on a monthly format): the pinned mode-switch test passes on a
// type linked to a monthly format.
func TestModeSwitchNeverReusesANumberOnAMonthlyFormat(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	linkFormat(t, e, f, "{prefix}-{yyyy}{MM}-{seq:5}", numbering.Grow, "")
	modeSwitchNeverReusesANumber(t, e, f)
	for _, c := range counters(t, e, f) {
		if !strings.HasPrefix(c.key, "e1|d:") || c.periodEnd == nil {
			t.Errorf("counter %s: want the monthly scope with a period_end", fmtCounters([]counter{c}))
		}
	}
}

// fillRatio is the nava_numbering_fill_ratio histogram's count and its
// count at or under 0.9.
func fillRatio(t *testing.T, reader *sdkmetric.ManualReader) (count, under uint64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			h, ok := mt.Data.(metricdata.Histogram[float64])
			if !ok || mt.Name != "nava_numbering_fill_ratio" {
				continue
			}
			for _, dp := range h.DataPoints {
				count += dp.Count
				for i, b := range dp.Bounds {
					if b <= 0.9 {
						under += dp.BucketCounts[i]
					}
				}
			}
		}
	}
	return count, under
}

func equalCounters(a, b []counter) bool {
	return slices.EqualFunc(a, b, func(x, y counter) bool {
		if x.key != y.key || x.next != y.next || (x.periodEnd == nil) != (y.periodEnd == nil) {
			return false
		}
		return x.periodEnd == nil || x.periodEnd.Equal(*y.periodEnd)
	})
}

func fmtCounters(cs []counter) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		end := "NULL"
		if c.periodEnd != nil {
			end = c.periodEnd.UTC().Format(time.RFC3339)
		}
		parts[i] = fmt.Sprintf("%q=%d until %s", c.key, c.next, end)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
