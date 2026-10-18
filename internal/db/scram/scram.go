// Package scram builds and checks PostgreSQL SCRAM-SHA-256 password
// verifiers (the rolpassword format), so db-roles sends PostgreSQL a verifier
// instead of a plaintext password and leaves a role alone when its password
// already matches.
package scram

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Iterations matches PostgreSQL's default scram_iterations.
const Iterations = 4096

const saltLen = 16

// CheckPassword rejects passwords this package cannot hash the way
// PostgreSQL does: SASLprep is the identity only on printable ASCII.
func CheckPassword(pw string) error {
	if pw == "" {
		return errors.New("empty password")
	}
	for i := 0; i < len(pw); i++ {
		if pw[i] < 0x20 || pw[i] > 0x7e {
			return errors.New("password must be printable ASCII")
		}
	}
	return nil
}

// Encode returns a new verifier for pw with a random salt.
func Encode(pw string) (string, error) {
	if err := CheckPassword(pw); err != nil {
		return "", err
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	stored, server, err := keys(pw, salt, Iterations)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s", Iterations, b64(salt), b64(stored), b64(server)), nil
}

// Matches reports whether verifier (a rolpassword value) accepts pw. Any
// value it cannot parse, NULL or MD5 included, does not match.
func Matches(verifier, pw string) bool {
	if CheckPassword(pw) != nil {
		return false
	}
	rest, ok := strings.CutPrefix(verifier, "SCRAM-SHA-256$")
	if !ok {
		return false
	}
	iterSalt, storedServer, ok := strings.Cut(rest, "$")
	if !ok {
		return false
	}
	iterStr, saltB64, ok1 := strings.Cut(iterSalt, ":")
	storedB64, serverB64, ok2 := strings.Cut(storedServer, ":")
	if !ok1 || !ok2 {
		return false
	}
	iter, err := strconv.Atoi(iterStr)
	if err != nil || iter < 1 {
		return false
	}
	dec := base64.StdEncoding.DecodeString
	salt, err1 := dec(saltB64)
	wantStored, err2 := dec(storedB64)
	wantServer, err3 := dec(serverB64)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	stored, server, err := keys(pw, salt, iter)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(stored, wantStored) == 1 && subtle.ConstantTimeCompare(server, wantServer) == 1
}

// keys derives StoredKey and ServerKey (RFC 5802, RFC 7677).
func keys(pw string, salt []byte, iter int) (stored, server []byte, err error) {
	salted, err := pbkdf2.Key(sha256.New, pw, salt, iter, sha256.Size)
	if err != nil {
		return nil, nil, err
	}
	clientKey := mac(salted, "Client Key")
	sum := sha256.Sum256(clientKey)
	return sum[:], mac(salted, "Server Key"), nil
}

func mac(key []byte, msg string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return h.Sum(nil)
}
