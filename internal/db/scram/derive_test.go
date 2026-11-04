package scram

import (
	"errors"
	"testing"
)

func TestKeyDerivationFailure(t *testing.T) {
	t.Parallel()
	errFIPS := errors.New("crypto/pbkdf2: use of salts shorter than 128 bits is not allowed in FIPS 140-only mode")
	failing := func(string, []byte, int) ([]byte, error) { return nil, errFIPS }

	// encode returns the error instead of a verifier.
	if v, err := encode("s3cret-pass", failing); !errors.Is(err, errFIPS) || v != "" {
		t.Fatalf("encode = %q, %v; want no verifier and the derivation error", v, err)
	}
	// matches fails closed: a verifier it cannot check does not match, so
	// db-roles rewrites the password rather than trusting it.
	if matches(pgVerifier, "dev-only-app", failing) {
		t.Fatal("matches = true although the key could not be derived")
	}
	// The same verifier and password match with the real derivation.
	if !matches(pgVerifier, "dev-only-app", saltedPassword) {
		t.Fatal("matches = false with the real derivation")
	}
}

func TestMatchesRejectsMalformedShapes(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct{ verifier, pw string }{
		"password not printable ASCII": {pgVerifier, "naïve"},
		"empty password":               {pgVerifier, ""},
		"no salt/key separator":        {"SCRAM-SHA-256$4096:YQ==", "pw"},
		"no iteration separator":       {"SCRAM-SHA-256$4096$YQ==:YQ==", "pw"},
		"no stored/server separator":   {"SCRAM-SHA-256$4096:YQ==$YQ==", "pw"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if Matches(c.verifier, c.pw) {
				t.Fatalf("Matches(%q, %q) = true", c.verifier, c.pw)
			}
		})
	}
}
