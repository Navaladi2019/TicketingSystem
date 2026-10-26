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
