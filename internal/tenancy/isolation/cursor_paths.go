package isolation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
)

// cursorPaths are the CI-8 cursor cases of T23 (TICK-58, PR #66): a cursor
// issued to tenant A's caller decodes for nobody else. The unit tests of
// internal/platform/cursor hold the edge cases; these keep the five
// isolation properties in the registry and the report.
func cursorPaths() []Path {
	check := func(name string, fn func(*cursor.Keyring, cursor.Claims, string) error) Path {
		return Path{Name: "cursor: " + name, Case: "CI-8", Check: func(context.Context) error {
			kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: []byte(strings.Repeat("a", 32))})
			if err != nil {
				return err
			}
			own := cursorClaims()
			tok, err := kr.Encode(own, []byte(`{"after":"`+RecordB1+`"}`))
			if err != nil {
				return err
			}
			return fn(kr, own, tok)
		}}
	}
	rejected := func(kr *cursor.Keyring, claims cursor.Claims, tok, what string) error {
		_, err := kr.Decode(claims, tok)
		switch {
		case err == nil:
			return fmt.Errorf("%s: Decode accepted the token, want ErrInvalid", what)
		case !errors.Is(err, cursor.ErrInvalid):
			return fmt.Errorf("%s: Decode returned %w, want ErrInvalid", what, err)
		}
		return nil
	}
	with := func(c cursor.Claims, k, v string) cursor.Claims {
		out := cursor.Claims{}
		for kk, vv := range c {
			out[kk] = vv
		}
		out[k] = v
		return out
	}
	return []Path{
		// Same principal id in both tenants: only the tenant claim differs.
		check("replayed by tenant B", func(kr *cursor.Keyring, own cursor.Claims, tok string) error {
			return rejected(kr, with(own, cursor.ClaimTenant, TenantB), tok, "tenant B")
		}),
		check("replayed by another principal of the same tenant", func(kr *cursor.Keyring, own cursor.Claims, tok string) error {
			return rejected(kr, with(own, cursor.ClaimPrincipal, "principal-2"), tok, "other principal")
		}),
		check("replayed with a changed filter", func(kr *cursor.Keyring, own cursor.Claims, tok string) error {
			return rejected(kr, with(own, cursor.ClaimQuery, cursor.QueryHash(url.Values{"status": {"closed"}})), tok, "changed filter")
		}),
		check("one byte tampered", func(kr *cursor.Keyring, own cursor.Claims, tok string) error {
			for i := range len(tok) {
				b := []byte(tok)
				b[i] ^= 1
				if err := rejected(kr, own, string(b), fmt.Sprintf("byte %d flipped", i)); err != nil {
					return err
				}
			}
			return nil
		}),
		check("key rotation", func(kr *cursor.Keyring, own cursor.Claims, tok string) error {
			next := cursor.Key{ID: "k2", Secret: []byte(strings.Repeat("b", 32))}
			retired, err := cursor.NewKeyring("k2", next)
			if err != nil {
				return err
			}
			if err := rejected(retired, own, tok, "token of a retired key"); err != nil {
				return err
			}
			// The key id swapped onto a token signed by another key.
			return rejected(retired, own, "k2."+strings.SplitN(tok, ".", 2)[1], "swapped key id")
		}),
	}
}

func cursorClaims() cursor.Claims {
	return cursor.Claims{
		cursor.ClaimTenant:    TenantA,
		cursor.ClaimPrincipal: "principal-1",
		cursor.ClaimQuery:     cursor.QueryHash(url.Values{"status": {"open"}}),
	}
}
