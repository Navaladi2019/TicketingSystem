#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# A container on the platform network reaches 7233 through the gateway and none of
# the internode (7234, 7235, 7239) or membership (6933-6939) ports, and cannot
# even resolve the server.
t_temporal_isolation() {
    nc_ok() { compose exec -T nava-core nc -w 3 "$1" "$2" </dev/null >/dev/null 2>&1; }
    nc_ok temporal 7233 || fail "the gateway does not answer on 7233"
    for port in 7234 7235 7239 6933 6934 6935 6936 6937 6938 6939 9090; do
        nc_ok temporal "$port" && fail "port $port is reachable from the platform network through the gateway"
    done
    # Judge by the answer, not the exit code: BusyBox nslookup exits 0 when one of
    # its A/AAAA queries is NXDOMAIN and the other NODATA, with no address (TICK-206).
    # "Server:" proves the lookup ran, so a failed exec cannot pass. RESOLVE_NAME
    # points the check at a name that does resolve, to prove it can fail (TICK-197).
    name=${RESOLVE_NAME:-temporal-server}
    ns=$(compose exec -T nava-core nslookup "$name" 2>&1) || true
    case $ns in *Server:*) ;; *) fail "nslookup did not run in nava-core: $ns" ;; esac
    if printf '%s\n' "$ns" | grep -q '^Name:'; then
        fail "the platform network can resolve $name: $ns"
    fi
    t_temporal_confinement
    pass "7233 reachable; 7234, 7235, 7239, 6933-6939 and 9090 are not; temporal-server does not resolve"
}

t_temporal_isolation
