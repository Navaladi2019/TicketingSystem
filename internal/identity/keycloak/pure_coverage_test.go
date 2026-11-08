package keycloak

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func TestKeyFitsRS256OnlyAcceptsRSASigningKeys(t *testing.T) {
	t.Parallel()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mk := func(raw any, set func(jwk.Key)) jwk.Key {
		k, err := jwk.PublicKeyOf(raw)
		if err != nil {
			t.Fatal(err)
		}
		if set != nil {
			set(k)
		}
		return k
	}
	cases := []struct {
		name string
		key  jwk.Key
		want bool
	}{
		{"rsa", mk(&rsaKey.PublicKey, nil), true},
		{"rsa for RS256 in sig use", mk(&rsaKey.PublicKey, func(k jwk.Key) {
			_ = k.Set(jwk.AlgorithmKey, jwa.RS256())
			_ = k.Set(jwk.KeyUsageKey, "sig")
		}), true},
		{"an EC key is not an RS256 key", mk(&ecKey.PublicKey, nil), false},
		{"rsa advertised for another algorithm", mk(&rsaKey.PublicKey, func(k jwk.Key) { _ = k.Set(jwk.AlgorithmKey, jwa.RS512()) }), false},
		{"rsa for encryption", mk(&rsaKey.PublicKey, func(k jwk.Key) { _ = k.Set(jwk.KeyUsageKey, "enc") }), false},
	}
	for _, c := range cases {
		if got := keyFitsRS256(c.key); got != c.want {
			t.Errorf("%s: keyFitsRS256 = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseReasonNamesTheFailureClass(t *testing.T) {
	t.Parallel()
	for want, err := range map[string]error{
		"expired":       jwt.TokenExpiredError(),
		"not yet valid": jwt.TokenNotYetValidError(),
		"iat":           jwt.InvalidIssuedAtError(),
		"audience":      jwt.InvalidAudienceError(),
		"issuer":        jwt.InvalidIssuerError(),
		"claims":        jwt.ValidateError(),
		"signature":     errors.New("anything else"),
	} {
		if got := parseReason(fmt.Errorf("wrapped: %w", err)); got != want {
			t.Errorf("parseReason(%v) = %q, want %q", err, got, want)
		}
	}
}
