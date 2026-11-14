#!/bin/sh
# ADR-007 S4 harness. Needs docker, go, node. Run from the repo root:
#   sh scratch/adr-007-s4/run.sh
# Writes result-outer.md, result-inner.md, bench-result.txt, explain.txt here.
set -eu
D=scratch/adr-007-s4
docker network create s4net >/dev/null 2>&1 || true
docker rm -f s4pg s4pgb >/dev/null 2>&1 || true
docker run -d --name s4pg --network s4net -e POSTGRES_PASSWORD=x postgres:16.10 >/dev/null
until docker exec s4pg pg_isready -U postgres -q; do sleep 1; done; sleep 3
docker exec s4pg psql -U postgres -c 'create database nava' >/dev/null
docker exec -i s4pg psql -U postgres -d nava -v ON_ERROR_STOP=1 -q < $D/seed.sql
docker run -d --name s4pgb --network s4net -p 127.0.0.1:65432:5432 \
  -e DATABASE_URL=postgres://nava_app:scratch-app@s4pg:5432/nava -e POOL_MODE=transaction \
  -e AUTH_TYPE=scram-sha-256 -e MAX_PREPARED_STATEMENTS=100 -e DEFAULT_POOL_SIZE=20 \
  -e MAX_CLIENT_CONN=200 -e IGNORE_STARTUP_PARAMETERS=extra_float_digits edoburu/pgbouncer:v1.24.1-p1 >/dev/null
sleep 4
trap 'docker rm -f s4pg s4pgb >/dev/null 2>&1; kill $P1 $P2 2>/dev/null || true' EXIT
DSN='postgres://nava_app:scratch-app@127.0.0.1:65432/nava?sslmode=disable'
# D2 p95 from inside the docker network (no host port-forward in the path)
mkdir -p "$D/.bin"; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$D/.bin/bench" ./$D/bench
docker run --rm --network s4net -v "$(pwd)/$D/.bin:/b" alpine:3 /b/bench \
  -dsn 'postgres://nava_app:scratch-app@s4pgb:5432/nava?sslmode=disable' -n 20000 -c 8 | tee $D/bench-result.txt
go build -o "$D/.bin/server" ./$D/server
"$D/.bin/server" -dsn "$DSN" -cors outer -addr 127.0.0.1:8087 & P1=$!
"$D/.bin/server" -dsn "$DSN" -cors inner -addr 127.0.0.1:8088 & P2=$!
sleep 2
cd $D/pw && npm install --silent && npx playwright install chromium firefox webkit
node run.mjs http://127.0.0.1:8087 cors-outer > ../result-outer.md || true
node run.mjs http://127.0.0.1:8088 cors-inner > ../result-inner.md || true
tail -1 ../result-outer.md ../result-inner.md
