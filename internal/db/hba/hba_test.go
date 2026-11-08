package hba_test

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/hba"
)

// The nava_backup cases hold on db/pg_hba.conf, and each fixture, one edit
// that would let nava_backup in where A.3 says no, makes them fail. The CI-9
// path runs the same cases (internal/tenancy/isolation).
func TestBackupCasesCatchEachWidening(t *testing.T) {
	conf := string(db.HBA())
	lines, err := hba.Parse([]byte(conf))
	if err != nil {
		t.Fatal(err)
	}
	if err := hba.Check(lines, hba.BackupCases()); err != nil {
		t.Fatalf("db/pg_hba.conf: %v", err)
	}
	for _, f := range []struct{ name, old, new string }{
		{"IPv4 reject removed", "host    all          nava_backup  0.0.0.0/0      reject\n", ""},
		{"IPv6 reject removed", "host    all          nava_backup  ::/0           reject\n", ""},
		{"socket reject removed", "local   all          nava_backup                 reject\n", ""},
		{"walsender from anywhere", "nava_backup  127.0.0.1/32", "nava_backup  0.0.0.0/0"},
		{"replication opened to every database", "host    replication  nava_backup", "host    all          nava_backup"},
	} {
		t.Run(f.name, func(t *testing.T) {
			if !strings.Contains(conf, f.old) {
				t.Fatalf("db/pg_hba.conf no longer has %q; update the fixture", f.old)
			}
			lines, err := hba.Parse([]byte(strings.Replace(conf, f.old, f.new, 1)))
			if err != nil {
				t.Fatal(err)
			}
			if hba.Check(lines, hba.BackupCases()) == nil {
				t.Error("the nava_backup cases still pass")
			}
		})
	}
}

// Parse refuses every line outside the subset it can evaluate, naming the
// line, so a new form of rule cannot be read wrongly and pass the checks.
func TestParseRejectsLinesOutsideTheSubset(t *testing.T) {
	t.Parallel()
	ok := "local all pgadmin scram-sha-256\n"
	for name, c := range map[string]struct{ conf, want string }{
		"no lines":          {"# only a comment\n\n", "pg_hba.conf has no lines"},
		"address not CIDR":  {ok + "host all nava_app 10.0.0.1 scram-sha-256\n", `pg_hba.conf:2: address is not a CIDR prefix: "host all nava_app 10.0.0.1 scram-sha-256"`},
		"hostssl":           {ok + "hostssl all all 0.0.0.0/0 scram-sha-256\n", "pg_hba.conf:2: not a local or host line this parser understands"},
		"host with options": {"host all all 0.0.0.0/0 scram-sha-256 clientcert=verify-full\n", "pg_hba.conf:1: not a local or host line"},
		"trust method":      {ok + "local all all trust\n", "pg_hba.conf:2: method is not scram-sha-256 or reject"},
		"md5 method":        {"host all all 10.0.0.0/8 md5\n", "pg_hba.conf:1: method is not scram-sha-256 or reject"},
		"database list":     {"local nava,keycloak all reject\n", "pg_hba.conf:1: lists, groups and include files are not used"},
		"group membership":  {"local all +nava_staff scram-sha-256\n", "pg_hba.conf:1: lists, groups and include files are not used"},
		"include file":      {"local all @users scram-sha-256\n", "pg_hba.conf:1: lists, groups and include files are not used"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			lines, err := hba.Parse([]byte(c.conf))
			if err == nil || !strings.HasPrefix(err.Error(), c.want) {
				t.Fatalf("Parse = %v, %v; want error starting %q", lines, err, c.want)
			}
			if lines != nil {
				t.Fatalf("Parse returned lines %v with an error", lines)
			}
		})
	}
}

func TestParseReadsTheSubset(t *testing.T) {
	t.Parallel()
	lines, err := hba.Parse([]byte("local all pgadmin scram-sha-256 # trailing comment\n\nhost nava nava_app 10.0.0.0/8 reject\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].Type != "local" || lines[0].User != "pgadmin" ||
		lines[1].Addr.String() != "10.0.0.0/8" || lines[1].Method != "reject" || lines[1].Database != "nava" {
		t.Fatalf("Parse = %+v", lines)
	}
}
