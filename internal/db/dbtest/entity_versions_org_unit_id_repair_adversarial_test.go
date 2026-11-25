//go:build integration

package dbtest

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/db/repairtest"
)

// Adversarial tests of the TICK-1125 repair (Security review of PR #877).

// The plan belongs to the inputs that were checked: an input changed between
// the two pastes must not reach a write.
func TestTICK1125PlanIsBoundToItsInputs(t *testing.T) {
	_, super := preS31Cluster(t)
	for _, c := range []struct{ name, key, value string }{
		{"new_name", "new_name", "old_region"},
		{"source_ref", "source_ref", "Jane Doe"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := repairSeed(t, super, versionRows(repairType), repairRecord{data: `{"org_unit_id":"v"}`})
			conn := superNava(t)
			if out := pasteOne(t, conn, f.inputs()); out.Err != nil {
				t.Fatal(out.Err)
			}
			exec(t, conn, "SELECT set_config('nava_repair.' || $1::text, $2, true)", c.key, c.value)
			_ = pasteTwo(t, conn) // refused, or written with the checked inputs
			if got := queryString(t, super, `SELECT count(*)::text FROM records r WHERE r.tenant_id = $1 AND r.data ? 'old_region'`, f.tenant); got != "0" {
				t.Errorf("%s records hold a name the schema does not have", got)
			}
			if got := queryString(t, super, `SELECT count(*)::text FROM audit_log WHERE tenant_id = $1 AND source_ref = 'Jane Doe'`, f.tenant); got != "0" {
				t.Errorf("%s entries hold a source_ref that was not checked", got)
			}
		})
	}
	t.Run("tenant and type", func(t *testing.T) {
		a, b := repairSeed(t, super, versionRows(repairType)), repairSeed(t, super, versionRows(repairClean))
		before := tenantState(t, super, b.tenant)
		conn := superNava(t)
		if out := pasteOne(t, conn, a.inputs()); out.Err != nil {
			t.Fatal(out.Err)
		}
		exec(t, conn, "SELECT set_config('nava_repair.tenant_id', $1, true), set_config('nava_repair.type_id', $2, true), set_config('app.tenant_id', $1, true)", b.tenant, b.typ)
		_ = pasteTwo(t, conn)
		if tenantState(t, super, b.tenant) != before {
			t.Error("tenant A's plan was written to tenant B's type")
		}
	})
	t.Run("paste 1 for B on A's open transaction", func(t *testing.T) {
		a, b := repairSeed(t, super, versionRows(repairType)), repairSeed(t, super, versionRows(repairClean))
		before := tenantState(t, super, b.tenant)
		conn := superNava(t)
		for _, f := range []*repairFixture{a, b} {
			one, err := repairtest.WithInputs(repairBlock(t, repairtest.PasteOne), f.inputs())
			if err != nil {
				t.Fatal(err)
			}
			paste(t, conn, one, true)
		}
		paste(t, conn, repairBlock(t, repairtest.PasteTwo), true)
		if tenantState(t, super, b.tenant) != before {
			t.Error("tenant A's plan was written to tenant B's type")
		}
	})
}

// The audit masker matches the markers as encoding/json does, whatever their
// case, so R3 must too: after a repair the old entries are masked no more.
func TestTICK1125R3MatchesTheMaskersCase(t *testing.T) {
	_, super := preS31Cluster(t)
	for _, marker := range []string{"X-Nava-Secret", "WriteOnly", "x-nava-ſecret"} {
		t.Run(marker, func(t *testing.T) {
			f := repairSeed(t, super, versionRows(withProp(`{"type":"string","`+marker+`":true}`)))
			err := repair(t, superNava(t), f.inputs())
			if pe, ok := errors.AsType[*pgconn.PgError](err); !ok || !strings.HasPrefix(pe.Message, "R3:") {
				t.Errorf("repair = %v, want R3", err)
			}
		})
	}
}
