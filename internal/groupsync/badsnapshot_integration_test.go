//go:build integration

// ErrBadSnapshot tests for T25 (TICK-78), from the QA and Principal Software
// Engineer reviews of PR #113 (TICK-221). A snapshot that contradicts itself
// is refused, nothing is written for the tenant, and the failure is logged
// with the tenant id.

package groupsync_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) records(t *testing.T) []map[string]any {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(l.b.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestBadSnapshotIsRefusedLoggedAndWritesNothing(t *testing.T) {
	cases := map[string]func(tid uuid.UUID) identity.DirectoryChanges{
		"org unit parent missing from the snapshot": func(tid uuid.UUID) identity.DirectoryChanges {
			return identity.DirectoryChanges{
				Users:    []identity.DirectoryUser{{KeycloakUserID: "u-new", Email: "new@example.test", Enabled: true}},
				OrgUnits: []identity.DirectoryGroup{unit("ou-a", "ou-nowhere", path(tid, "/ou/a"))},
			}
		},
		"user id twice": func(tid uuid.UUID) identity.DirectoryChanges {
			return identity.DirectoryChanges{Users: []identity.DirectoryUser{
				{KeycloakUserID: "u-new", Email: "new@example.test", Enabled: true},
				{KeycloakUserID: "u-new", Email: "again@example.test", Enabled: true},
			}}
		},
		"group id twice": func(tid uuid.UUID) identity.DirectoryChanges {
			g := identity.DirectoryGroup{KeycloakGroupID: "g-new", Path: path(tid, "/new"), Name: "new"}
			return identity.DirectoryChanges{Groups: []identity.DirectoryGroup{g, g}}
		},
		"org unit id twice": func(tid uuid.UUID) identity.DirectoryChanges {
			return identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
				unit("ou-new", "", path(tid, "/ou/x")), unit("ou-new", "", path(tid, "/ou/y")),
			}}
		},
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			super, pool := cluster(t)
			tid := uuid.New()
			seedTenant(t, super, tid, 1)
			dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
			buf := &lockedBuf{}
			s, err := groupsync.New(pool, dir, noop.NewMeterProvider(), slog.New(slog.NewJSONHandler(buf, nil)), groupsync.Options{MaxOrgUnitDepth: 8})
			if err != nil {
				t.Fatal(err)
			}

			dir.set(tid, victimSnapshot(tid))
			if r := once(t, s); r.Err != nil {
				t.Fatalf("first sync: %v", r.Err)
			}
			before := tenantRows(t, super, tid)
			if before == "" {
				t.Fatal("tenant was not seeded")
			}

			dir.set(tid, bad(tid))
			r := once(t, s)
			if !errors.Is(r.Err, groupsync.ErrBadSnapshot) {
				t.Fatalf("err = %v, want ErrBadSnapshot", r.Err)
			}
			if got := tenantRows(t, super, tid); got != before {
				t.Errorf("a refused snapshot changed the tenant:\n before %s\n after  %s", before, got)
			}

			var logged bool
			for _, rec := range buf.records(t) {
				if rec["level"] == "ERROR" && rec["tenant_id"] == tid.String() &&
					strings.Contains(rec["error"].(string), groupsync.ErrBadSnapshot.Error()) {
					logged = true
				}
			}
			if !logged {
				t.Errorf("no ERROR log with tenant_id %s and ErrBadSnapshot: %s", tid, buf.b.String())
			}
		})
	}
}
