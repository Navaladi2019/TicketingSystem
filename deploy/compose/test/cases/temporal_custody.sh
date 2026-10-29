#!/bin/sh
# stack-case: job=stack order=10 mode=parallel
set -eu
. "$(dirname "$0")/../case-env.sh"

# Each certificate is mounted only into the service that owns it. temporal-system
# is never mounted into a platform service.
t_temporal_custody() {
    bad=""
    for cid in $(compose ps -a -q); do
        svc=$(docker inspect -f '{{index .Config.Labels "com.docker.compose.service"}}' "$cid")
        for src in $(docker inspect -f '{{range .Mounts}}{{.Source}} {{end}}' "$cid"); do
            case $src in
                */certs/temporal-system.*|*/certs/internode*|*/certs/server.*) [ "$svc" = temporal-server ] || bad="$bad $svc:$src" ;;
                */certs/nava-core.*) [ "$svc" = nava-core ] || bad="$bad $svc:$src" ;;
                */certs/nava-operator.*) [ "$svc" = temporal-ui ] || bad="$bad $svc:$src" ;;
                */certs/nava-worker.*) [ "$svc" = nava-worker ] || bad="$bad $svc:$src" ;;
                */certs/nava-relay.*) [ "$svc" = nava-relay ] || bad="$bad $svc:$src" ;;
                */certs/nava-unknown.*) bad="$bad $svc:$src" ;;
            esac
        done
    done
    [ -z "$bad" ] || fail "certificates mounted into the wrong service:$bad"
    pass "every certificate is mounted only into its own service"
}

t_temporal_custody
