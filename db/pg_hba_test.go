package db

import (
	"os"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/db/hba"
)

func readHBA(t *testing.T) []hba.Line {
	t.Helper()
	b, err := os.ReadFile("pg_hba.conf")
	if err != nil {
		t.Fatal(err)
	}
	lines, err := hba.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	// The embedded copy is the file on disk (the isolation suite reads it).
	if string(HBA()) != string(b) {
		t.Fatal("db.HBA() differs from pg_hba.conf")
	}
	return lines
}

func TestPgHBA(t *testing.T) {
	lines := readHBA(t)
	core, core6, backup := hba.CoreAddr, hba.Core6Addr, hba.BackupAddr
	cases := []hba.Case{
		// T02 security review: the superuser only over the local socket.
		{Name: "pgadmin from nava-core", Conn: hba.Conn{Database: "postgres", User: "pgadmin", Addr: core}, Want: "reject"},
		{Name: "pgadmin from nava-core to nava", Conn: hba.Conn{Database: "nava", User: "pgadmin", Addr: core}, Want: "reject"},
		{Name: "pgadmin over IPv6", Conn: hba.Conn{Database: "postgres", User: "pgadmin", Addr: core6}, Want: "reject"},
		{Name: "pgadmin from inside the postgres container over TCP", Conn: hba.Conn{Database: "postgres", User: "pgadmin", Addr: "127.0.0.1"}, Want: "reject"},
		{Name: "pgadmin physical walsender", Conn: hba.Conn{Database: "replication", Physical: true, User: "pgadmin", Addr: core}, Want: ""},
		{Name: "pgadmin over the socket", Conn: hba.Conn{Local: true, Database: "postgres", User: "pgadmin"}, Want: "scram-sha-256"},

		// Everyone else: SCRAM, and no physical replication.
		{Name: "nava_app from nava-core", Conn: hba.Conn{Database: "nava", User: "nava_app", Addr: core}, Want: "scram-sha-256"},
		{Name: "temporal over IPv6", Conn: hba.Conn{Database: "temporal", User: "temporal", Addr: core6}, Want: "scram-sha-256"},
		{Name: "nava_app physical walsender", Conn: hba.Conn{Database: "replication", Physical: true, User: "nava_app", Addr: backup}, Want: ""},
	}
	// ADR-001 A.3 / CI-9: nava_backup is a physical walsender from the backup
	// host and nothing else. The isolation registry runs the same cases.
	cases = append(cases, hba.BackupCases()...)
	if err := hba.Check(lines, cases); err != nil {
		t.Error(err)
	}
}

// Every reject for a role comes before the first line that would let that
// role in through the catch-all, so a later edit cannot reorder them silently.
func TestPgHBARejectsPrecedeCatchAll(t *testing.T) {
	lines := readHBA(t)
	firstCatchAll := -1
	for i, l := range lines {
		if l.User == "all" {
			firstCatchAll = i
			break
		}
	}
	if firstCatchAll < 0 {
		t.Fatal("no catch-all line")
	}
	for i, l := range lines {
		if l.Method == "reject" && i > firstCatchAll {
			t.Errorf("reject line %d (%s %s %s) is after the catch-all at line %d", i, l.Type, l.Database, l.User, firstCatchAll)
		}
	}
}
