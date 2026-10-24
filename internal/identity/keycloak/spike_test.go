//go:build spike

package keycloak

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TestSpikeRandomKidFlood is the TICK-94 item 11 spike, in real time:
// "Pass if 1,000 random-kid tokens in 60 s cause ≤ 3 JWKS fetches, and a
// token signed by a newly rotated key is accepted within 30 s."
//
//	go test -tags spike -run TestSpike -v ./internal/identity/keycloak/
func TestSpikeRandomKidFlood(t *testing.T) {
	const (
		tokens   = 1000
		window   = 60 * time.Second
		rotateAt = 40 * time.Second // the worst case: just after a forced fetch at ~30 s
	)
	h := newHarness(t, []string{realmNava})
	h.a.now, h.a.keys.now = time.Now, time.Now
	s := h.idp.signer(realmNava)

	start := time.Now()
	if _, err := h.a.verify(t.Context(), sign(t, s, h.idp.tenantClaims(realmNava, start)), DefaultAudience); err != nil {
		t.Fatal(err)
	}

	var (
		wg       sync.WaitGroup
		accepted time.Duration
	)
	wg.Go(func() {
		time.Sleep(rotateAt - time.Since(start))
		rotated := h.idp.rotate(realmNava, "nava-rotated")
		rotatedAt := time.Now()
		for time.Since(rotatedAt) < 45*time.Second {
			_, err := h.a.verify(t.Context(), sign(t, rotated, h.idp.tenantClaims(realmNava, time.Now())), DefaultAudience)
			if err == nil {
				accepted = time.Since(rotatedAt)
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	})

	interval := window / tokens
	for i := range tokens {
		time.Sleep(time.Until(start.Add(time.Duration(i) * interval)))
		tok := sign(t, signer{kid: fmt.Sprintf("random-%04d", i), priv: s.priv}, h.idp.tenantClaims(realmNava, time.Now()))
		if _, err := h.a.verify(t.Context(), tok, DefaultAudience); !errors.Is(err, identity.ErrInvalidToken) {
			t.Fatalf("token %d: err = %v, want ErrInvalidToken", i, err)
		}
	}
	flood := time.Since(start)
	fetchesInFlood := h.idp.jwksHits.Load()
	wg.Wait()

	t.Logf("spike: %d random-kid tokens in %s caused %d JWKS fetches (first fetch included); criterion ≤ 3", tokens, flood.Round(time.Millisecond), fetchesInFlood)
	if accepted > 0 {
		t.Logf("spike: key rotated at +%s accepted after %s; criterion ≤ 30s", rotateAt, accepted.Round(time.Millisecond))
	}
	t.Logf("spike: %d JWKS fetches in all, the rotated-key fetch included", h.idp.jwksHits.Load())
	if fetchesInFlood > 3 {
		t.Errorf("FAIL: %d JWKS fetches, want ≤ 3", fetchesInFlood)
	}
	if accepted == 0 || accepted > 30*time.Second {
		t.Errorf("FAIL: rotated key accepted after %s, want ≤ 30s", accepted)
	}
}
