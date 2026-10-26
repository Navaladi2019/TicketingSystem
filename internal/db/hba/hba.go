// Package hba evaluates db/pg_hba.conf the way PostgreSQL picks a line, for
// the subset of the format the file uses: local and host lines, single
// keywords or names, CIDR addresses. db/pg_hba_test.go (T08) and the
// isolation registry (T12, CI-9) run the same nava_backup cases through it.
package hba

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// Line is one record of pg_hba.conf.
type Line struct {
	Type, Database, User string
	Addr                 netip.Prefix // zero for local
	Method               string
}

// Parse reads pg_hba.conf. Anything outside the subset is an error, so a new
// form of line cannot be evaluated wrongly.
func Parse(b []byte) ([]Line, error) {
	var out []Line
	for n, raw := range strings.Split(string(b), "\n") {
		line, _, _ := strings.Cut(raw, "#")
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		bad := func(why string) error { return fmt.Errorf("pg_hba.conf:%d: %s: %q", n+1, why, raw) }
		var l Line
		switch {
		case f[0] == "local" && len(f) == 4:
			l = Line{Type: f[0], Database: f[1], User: f[2], Method: f[3]}
		case f[0] == "host" && len(f) == 5:
			p, err := netip.ParsePrefix(f[3])
			if err != nil {
				return nil, bad("address is not a CIDR prefix")
			}
			l = Line{Type: f[0], Database: f[1], User: f[2], Addr: p, Method: f[4]}
		default:
			return nil, bad("not a local or host line this parser understands")
		}
		if l.Method != "scram-sha-256" && l.Method != "reject" {
			return nil, bad("method is not scram-sha-256 or reject")
		}
		if strings.ContainsAny(l.Database+l.User, ",+@") {
			return nil, bad("lists, groups and include files are not used")
		}
		out = append(out, l)
	}
	if len(out) == 0 {
		return nil, errors.New("pg_hba.conf has no lines")
	}
	return out, nil
}

// Conn is a connection attempt. Database "replication" with Physical true is
// a physical walsender; a logical walsender names its database.
type Conn struct {
	Local    bool
	Database string
	Physical bool
	User     string
	Addr     string
}

// Decide returns the method of the first matching line, or "" when none
// matches (PostgreSQL then refuses the connection).
func Decide(lines []Line, c Conn) string {
	for _, l := range lines {
		if (l.Type == "local") != c.Local {
			continue
		}
		switch {
		case c.Physical && l.Database != "replication":
			continue
		case !c.Physical && l.Database != "all" && l.Database != c.Database:
			continue
		}
		if l.User != "all" && l.User != c.User {
			continue
		}
		if !c.Local && !l.Addr.Contains(netip.MustParseAddr(c.Addr)) {
			continue
		}
		return l.Method
	}
	return ""
}

// Case is a connection attempt and the method it must get ("" = refused).
type Case struct {
	Name string
	Conn Conn
	Want string
}

// Addresses the cases use.
const (
	CoreAddr   = "172.20.0.7" // any container on the compose networks, nava-core included
	Core6Addr  = "fd00::7"
	BackupAddr = "127.0.0.1"
)

// BackupCases are T08's nava_backup cases (ADR-001 A.3, CI-9): a physical
// walsender from the backup host and nothing else. No SQL session on nava or
// postgres, no logical walsender (replication=database), from any address.
func BackupCases() []Case {
	return []Case{
		{"nava_backup physical walsender from the backup host", Conn{Database: "replication", Physical: true, User: "nava_backup", Addr: BackupAddr}, "scram-sha-256"},
		{"nava_backup physical walsender from elsewhere", Conn{Database: "replication", Physical: true, User: "nava_backup", Addr: CoreAddr}, ""},
		{"nava_backup SQL session to nava", Conn{Database: "nava", User: "nava_backup", Addr: CoreAddr}, "reject"},
		{"nava_backup SQL session to postgres", Conn{Database: "postgres", User: "nava_backup", Addr: CoreAddr}, "reject"},
		{"nava_backup SQL session from the backup host", Conn{Database: "nava", User: "nava_backup", Addr: BackupAddr}, "reject"},
		{"nava_backup logical walsender (replication=database)", Conn{Database: "nava", User: "nava_backup", Addr: BackupAddr}, "reject"},
		{"nava_backup over IPv6", Conn{Database: "nava", User: "nava_backup", Addr: Core6Addr}, "reject"},
		{"nava_backup over the socket", Conn{Local: true, Database: "nava", User: "nava_backup"}, "reject"},
	}
}

// Check runs cases against lines and joins every mismatch.
func Check(lines []Line, cases []Case) error {
	var errs []error
	for _, c := range cases {
		if got := Decide(lines, c.Conn); got != c.Want {
			errs = append(errs, fmt.Errorf("%s: method %q, want %q", c.Name, got, c.Want))
		}
	}
	return errors.Join(errs...)
}
