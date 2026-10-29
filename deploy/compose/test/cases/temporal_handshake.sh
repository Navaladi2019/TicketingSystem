#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# Rogue-CA certificate fails the handshake: a nava-relay certificate from another
# CA is refused, as is no certificate, while the real nava-core certificate works.
t_temporal_handshake() {
    [ -f "$CERTS/ca.pem" ] || fail "no certificates in $CERTS: run make dev-ca"
    dial_as nava-core "$CERTS" | grep -q SERVING || fail "the nava-core certificate could not dial Temporal"
    rogue=$(mktemp -d "$CERTS/rogue.XXXXXX") # under certs/ so Docker Desktop can mount it
    trap 'rm -rf "${rogue:-}" "$ROOT/temporal-server/.rogue.$$"' EXIT # also on fail, which exits
    (cd temporal-server && go run ./cmd/devcerts -rogue -out ".rogue.$$" && cp ".rogue.$$"/* "$rogue"/ && rm -rf ".rogue.$$") >/dev/null || fail "cannot generate the rogue certificate" # relative -out: portable to Git Bash on Windows
    cp "$CERTS/ca.pem" "$rogue/ca.pem"
    chmod -R a+rX "$rogue"
    # A TLS rejection, not just any failure (a timeout or a missing image must not pass).
    if out=$(dial_as rogue-nava-relay "$rogue"); then rm -rf "$rogue"; fail "a rogue-CA certificate completed the handshake: $out"; fi
    rm -rf "$rogue"
    case $out in *"tls:"*|*certificate*) ;; *) fail "the rogue-CA certificate failed, but not with a TLS rejection: $out" ;; esac
    if out=$(dial_as none "$CERTS"); then fail "a client without a certificate was served: $out"; fi
    case $out in *"tls:"*|*certificate*) ;; *) fail "no client certificate failed, but not with a TLS rejection: $out" ;; esac
    pass "nava-core dials; a rogue-CA certificate and no certificate fail the handshake"
}

t_temporal_handshake
