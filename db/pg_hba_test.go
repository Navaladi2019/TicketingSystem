package db

import (
	"net/netip"
	"os"
	"strings"
	"testing"
)

// hbaLine is one record of pg_hba.conf, in the subset of the format the file
// uses: local and host lines, single keywords or names, CIDR addresses.
type hbaLine struct {
	typ, database, user string
	addr                netip.Prefix // zero for local
	method              string
}

func readHBA(t *testing.T) []hbaLine {
	t.Helper()
	b, err := os.ReadFile("pg_hba.conf")
	if err != nil {
		t.Fatal(err)
	}
	var out []hbaLine
	for n, raw := range strings.Split(string(b), "\n") {
		line, _, _ := strings.Cut(raw, "#")
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		bad := func(why string) { t.Fatalf("pg_hba.conf:%d: %s: %q", n+1, why, raw) }
		var l hbaLine
		switch {
		case f[0] == "local" && len(f) == 4:
			l = hbaLine{typ: f[0], database: f[1], user: f[2], method: f[3]}
		case f[0] == "host" && len(f) == 5:
			p, err := netip.ParsePrefix(f[3])
			if err != nil {
				bad("address is not a CIDR prefix")
			}
			l = hbaLine{typ: f[0], database: f[1], user: f[2], addr: p, method: f[4]}
		default:
			bad("not a local or host line this test understands")
		}
		if l.method != "scram-sha-256" && l.method != "reject" {
			bad("method is not scram-sha-256 or reject")
		}
		if strings.ContainsAny(l.database+l.user, ",+@") {
			bad("lists, groups and include files are not used")
		}
		out = append(out, l)
	}
	return out
}

// conn is a connection attempt. database "replication" with physical true is
// a physical walsender; a logical walsender names its database.
type conn struct {
	local    bool
	database string
	physical bool
	user     string
	addr     string
}

// decide returns the method of the first matching line, or "" when none
// matches (PostgreSQL then refuses the connection).
func decide(lines []hbaLine, c conn) string {
	for _, l := range lines {
		if (l.typ == "local") != c.local {
			continue
		}
		switch {
		case c.physical && l.database != "replication":
			continue
		case !c.physical && l.database != "all" && l.database != c.database:
			continue
		}
		if l.user != "all" && l.user != c.user {
			continue
		}
		if !c.local && !l.addr.Contains(netip.MustParseAddr(c.addr)) {
			continue
		}
		return l.method
	}
	return ""
}

func TestPgHBA(t *testing.T) {
	lines := readHBA(t)
	const (
		core   = "172.20.0.7" // any container on the compose networks, nava-core included
		core6  = "fd00::7"
		backup = "127.0.0.1"
	)
	cases := []struct {
		name string
		c    conn
		want string // "" = no line matches = refused
	}{
		// T02 security review: the superuser only over the local socket.
		{"pgadmin from nava-core", conn{database: "postgres", user: "pgadmin", addr: core}, "reject"},
		{"pgadmin from nava-core to nava", conn{database: "nava", user: "pgadmin", addr: core}, "reject"},
		{"pgadmin over IPv6", conn{database: "postgres", user: "pgadmin", addr: core6}, "reject"},
		{"pgadmin from inside the postgres container over TCP", conn{database: "postgres", user: "pgadmin", addr: "127.0.0.1"}, "reject"},
		{"pgadmin physical walsender", conn{database: "replication", physical: true, user: "pgadmin", addr: core}, ""},
		{"pgadmin over the socket", conn{local: true, database: "postgres", user: "pgadmin"}, "scram-sha-256"},

		// ADR-001 A.3 / CI-9: nava_backup is a physical walsender from the backup host and nothing else.
		{"nava_backup physical walsender from the backup host", conn{database: "replication", physical: true, user: "nava_backup", addr: backup}, "scram-sha-256"},
		{"nava_backup physical walsender from elsewhere", conn{database: "replication", physical: true, user: "nava_backup", addr: core}, ""},
		{"nava_backup SQL session to nava", conn{database: "nava", user: "nava_backup", addr: core}, "reject"},
		{"nava_backup SQL session to postgres", conn{database: "postgres", user: "nava_backup", addr: core}, "reject"},
		{"nava_backup SQL session from the backup host", conn{database: "nava", user: "nava_backup", addr: backup}, "reject"},
		{"nava_backup logical walsender (replication=database)", conn{database: "nava", user: "nava_backup", addr: backup}, "reject"},
		{"nava_backup over IPv6", conn{database: "nava", user: "nava_backup", addr: core6}, "reject"},
		{"nava_backup over the socket", conn{local: true, database: "nava", user: "nava_backup"}, "reject"},

		// Everyone else: SCRAM, and no physical replication.
		{"nava_app from nava-core", conn{database: "nava", user: "nava_app", addr: core}, "scram-sha-256"},
		{"temporal over IPv6", conn{database: "temporal", user: "temporal", addr: core6}, "scram-sha-256"},
		{"nava_app physical walsender", conn{database: "replication", physical: true, user: "nava_app", addr: backup}, ""},
	}
	for _, tc := range cases {
		if got := decide(lines, tc.c); got != tc.want {
			t.Errorf("%s: method %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Every reject for a role comes before the first line that would let that
// role in through the catch-all, so a later edit cannot reorder them silently.
func TestPgHBARejectsPrecedeCatchAll(t *testing.T) {
	lines := readHBA(t)
	firstCatchAll := -1
	for i, l := range lines {
		if l.user == "all" {
			firstCatchAll = i
			break
		}
	}
	if firstCatchAll < 0 {
		t.Fatal("no catch-all line")
	}
	for i, l := range lines {
		if l.method == "reject" && i > firstCatchAll {
			t.Errorf("reject line %d (%s %s %s) is after the catch-all at line %d", i, l.typ, l.database, l.user, firstCatchAll)
		}
	}
}
