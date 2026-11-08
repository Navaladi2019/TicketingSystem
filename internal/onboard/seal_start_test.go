package onboard

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// insertColumns returns the column list of the first "INSERT INTO <table> (...)" in sql.
func insertColumns(t *testing.T, sql, table string) []string {
	t.Helper()
	m := regexp.MustCompile(`INSERT INTO ` + table + ` \(([^)]*)\)`).FindStringSubmatch(sql)
	if m == nil {
		t.Fatalf("no INSERT INTO %s in %q", table, sql)
	}
	var cols []string
	for _, c := range strings.Split(m[1], ",") {
		cols = append(cols, strings.TrimSpace(c))
	}
	return cols
}

// The start row's insert column list equals outbox.insertSealSQL's, so the
// two cannot drift (TICK-439 review N2). insertSealSQL is unexported, so the
// test reads it from the outbox source.
func TestSealInsertMatchesOutbox(t *testing.T) {
	src, err := os.ReadFile("../outbox/outbox.go")
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(src), "insertSealSQL = `")
	if i < 0 {
		t.Fatal("insertSealSQL not found in internal/outbox/outbox.go")
	}
	want := insertColumns(t, string(src)[i:], "outbox")
	got := insertColumns(t, insertSealStartSQL, "outbox")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("onboard seal insert columns %v, outbox %v", got, want)
	}
	for _, banned := range []string{"RETURNING", "DO UPDATE", "ON CONFLICT"} {
		if strings.Contains(insertSealStartSQL, banned) {
			t.Errorf("seal start insert contains %q: nava_onboard has INSERT only", banned)
		}
	}
	if strings.Contains(insertHeadSQL, "DO UPDATE") || strings.Contains(insertHeadSQL, "RETURNING") {
		t.Errorf("head insert must be plain INSERT ... ON CONFLICT DO NOTHING: %s", insertHeadSQL)
	}
}

// An existing head (a retry, or migration B) writes no second start row: the
// scripted server answers the head insert with 0 rows and fails any other
// statement, so an outbox insert would error.
func TestPGStoreSeedAuditChainExistingHeadInsertsNoSeal(t *testing.T) {
	t.Parallel()
	st := scriptedStore(t, map[string]scripted{"INSERT INTO audit_chain_head": {}}, noBindings{})
	if err := st.SeedAuditChain(t.Context(), uuid.New()); err != nil {
		t.Fatalf("SeedAuditChain: %v", err)
	}
}

func TestPGStoreSeedAuditChainDatabaseError(t *testing.T) {
	t.Parallel()
	st := scriptedStore(t, map[string]scripted{"INSERT INTO audit_chain_head": {err: "boom", code: "42501"}}, noBindings{})
	err := st.SeedAuditChain(t.Context(), uuid.New())
	if err == nil || errors.Is(err, ErrNotProvisioning) {
		t.Fatalf("err = %v, want a wrapped database error", err)
	}
}
