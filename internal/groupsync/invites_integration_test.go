//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-252: pending invites in the tenant transaction, as the real
// nava_groupsync role under the real pending_invite policies.

// placerDir is a directory that also places invitees. place decides, and calls
// consume the way keycloak.Directory.PlaceInvitee does after its checks.
type placerDir struct {
	*fakeDir
	failSnapshot atomic.Bool
	mu           sync.Mutex
	offered      []string
	bound, orgs  []string
	place        func(ctx context.Context, email string, consume func(context.Context) (bool, error)) (bool, error)
}

func (p *placerDir) ChangesSince(ctx context.Context, id uuid.UUID, cursor string) (identity.DirectoryChanges, string, error) {
	if p.failSnapshot.Load() {
		return identity.DirectoryChanges{}, "", errors.New("keycloak unavailable")
	}
	return p.fakeDir.ChangesSince(ctx, id, cursor)
}

func (p *placerDir) PlaceInvitee(ctx context.Context, _ uuid.UUID, email string, bound, orgIDs []string,
	consume func(context.Context) (bool, error)) (bool, error) {
	p.mu.Lock()
	p.offered = append(p.offered, email)
	p.bound, p.orgs = bound, orgIDs
	p.mu.Unlock()
	return p.place(ctx, email, consume)
}

func (p *placerDir) offeredEmails() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.offered...)
}

func newPlacer(place func(context.Context, string, func(context.Context) (bool, error)) (bool, error)) *placerDir {
	return &placerDir{fakeDir: &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}, place: place}
}

func seedInvite(t *testing.T, super *pgx.Conn, tid uuid.UUID, email, expires string) {
	t.Helper()
	if _, err := super.Exec(context.Background(), `INSERT INTO pending_invite (tenant_id, email, expires_at, created_at)
		VALUES ($1, $2, now() + $3::interval, now() - interval '30 days')`, tid, email, expires); err != nil {
		t.Fatal(err)
	}
}

func consumed(t *testing.T, super *pgx.Conn, tid uuid.UUID, email string) bool {
	t.Helper()
	return count(t, super, "SELECT count(*) FROM pending_invite WHERE tenant_id = $1 AND email = $2 AND consumed_at IS NOT NULL", tid, email) == 1
}

// An eligible invitee is consumed once; the placer is offered only unexpired,
// unconsumed invites of this tenant, with the tenant's bound org id.
func TestPendingInviteConsumedWhenPlaced(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	other := uuid.New()
	seedTenant(t, super, tid, 1)
	seedTenant(t, super, other, 2)
	seedInvite(t, super, tid, "admin@a.test", "7 days")
	seedInvite(t, super, tid, "expired@a.test", "-1 day")
	seedInvite(t, super, tid, "done@a.test", "7 days")
	seedInvite(t, super, other, "admin@b.test", "7 days")
	if _, err := super.Exec(context.Background(), "UPDATE pending_invite SET consumed_at = now() WHERE email = 'done@a.test'"); err != nil {
		t.Fatal(err)
	}
	dir := newPlacer(func(ctx context.Context, email string, consume func(context.Context) (bool, error)) (bool, error) {
		return consume(ctx)
	})
	s := newSyncer(t, pool, dir)
	rs, err := s.Once(context.Background())
	if err != nil || len(rs) != 2 {
		t.Fatalf("Once = %v, %v", rs, err)
	}
	for _, r := range rs {
		if r.Err != nil {
			t.Fatalf("tenant %s: %v", r.TenantID, r.Err)
		}
	}
	if got := dir.offeredEmails(); len(got) != 2 {
		t.Errorf("offered %v, want admin@a.test and admin@b.test only", got)
	}
	if !consumed(t, super, tid, "admin@a.test") || !consumed(t, super, other, "admin@b.test") {
		t.Error("an eligible invite was not consumed")
	}
	if consumed(t, super, tid, "expired@a.test") {
		t.Error("an expired invite was consumed")
	}
	// Once consumed, it is not offered again.
	dir.mu.Lock()
	dir.offered = nil
	dir.mu.Unlock()
	if _, err := s.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := dir.offeredEmails(); len(got) != 0 {
		t.Errorf("second cycle offered %v, want nothing", got)
	}
	if len(dir.orgs) != 1 || dir.orgs[0] != "org-1" && dir.orgs[0] != "org-2" {
		t.Errorf("placer got org ids %v, want the tenant's binding org id", dir.orgs)
	}
}

// A placer that refuses, or fails, leaves the invite open, and the tenant
// still syncs.
func TestPendingInviteStaysOpenWhenNotPlaced(t *testing.T) {
	for name, place := range map[string]func(context.Context, string, func(context.Context) (bool, error)) (bool, error){
		"refused": func(context.Context, string, func(context.Context) (bool, error)) (bool, error) { return false, nil },
		// The placer consumed and then failed (a failed group write): the
		// savepoint takes the consume back.
		"consumed then failed": func(ctx context.Context, _ string, consume func(context.Context) (bool, error)) (bool, error) {
			if _, err := consume(ctx); err != nil {
				return false, err
			}
			return false, errors.New("group write failed")
		},
	} {
		t.Run(name, func(t *testing.T) {
			super, pool := cluster(t)
			tid := uuid.New()
			seedTenant(t, super, tid, 1)
			seedInvite(t, super, tid, "admin@a.test", "7 days")
			s := newSyncer(t, pool, newPlacer(place))
			if r := once(t, s); r.Err != nil {
				t.Fatalf("tenant did not sync: %v", r.Err)
			}
			if consumed(t, super, tid, "admin@a.test") {
				t.Error("the invite was consumed")
			}
		})
	}
}

// Finding 4: a cycle places the user (group written, invite consumed), then the
// snapshot fails and the tenant's transaction rolls back, so the invite is open
// again. The next cycle offers it again; Keycloak already has the user in the
// group, so the placer consumes without writing, and the invite is closed.
func TestPendingInviteSurvivesSnapshotFailure(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	seedInvite(t, super, tid, "admin@a.test", "7 days")
	var groupWrites atomic.Int32
	var inGroup atomic.Bool
	dir := newPlacer(func(ctx context.Context, _ string, consume func(context.Context) (bool, error)) (bool, error) {
		ok, err := consume(ctx)
		if err != nil || !ok {
			return false, err
		}
		if !inGroup.Load() {
			groupWrites.Add(1)
			inGroup.Store(true)
		}
		return true, nil
	})
	s := newSyncer(t, pool, dir)

	dir.failSnapshot.Store(true)
	if r := once(t, s); r.Err == nil {
		t.Fatal("cycle succeeded with the snapshot failing")
	}
	if consumed(t, super, tid, "admin@a.test") {
		t.Fatal("the invite stayed consumed after the tenant rolled back")
	}
	if groupWrites.Load() != 1 {
		t.Fatalf("group writes = %d, want 1", groupWrites.Load())
	}

	dir.failSnapshot.Store(false)
	if r := once(t, s); r.Err != nil {
		t.Fatalf("next cycle: %v", r.Err)
	}
	if !consumed(t, super, tid, "admin@a.test") {
		t.Error("the next cycle did not consume the invite")
	}
	if groupWrites.Load() != 1 {
		t.Errorf("group writes = %d, want still 1", groupWrites.Load())
	}
}
