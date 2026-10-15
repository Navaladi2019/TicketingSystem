**Setup.** Windows 11, Docker 28.0.1, 19.4 GB VM. Each IdP had its own `postgres:16`.
- Keycloak: `quay.io/keycloak/keycloak:26.8.1 start` with `KC_DB=postgres` and HTTP enabled, no prebuilt optimized image.
- Zitadel: `ghcr.io/zitadel/zitadel:v4.19.4 start-from-init --tlsMode disabled`.

**Raw results:**
```
keycloak ready after 114s (first boot, uncapped, incl. schema creation)
zitadel ready after 20s (first boot, incl. init+setup)
fp-kc   3.033GiB (uncapped; JVM sizes heap from host RAM)     fp-z   79.54MiB
fp-kcpg 86.67MiB                                               fp-zpg 91.27MiB
fp-kc warm restart ready in 29s        fp-z warm restart ready in 7s
zitadel schemas: public,system,projections,eventstore,adminapi,auth,logstore,cache,queue
zitadel eventstore.events2 rows after init: 995      keycloak public tables: 101
kc(-m 1g) ready in 58s; after 50 password grants: 567.3MiB / 1GiB, OOMKilled=false
```

**Caveats:**
- Readiness means the discovery document returns 200.
- Zitadel was measured without its login container and without load.
- Keycloak's boot time includes Quarkus auto-build; a `kc.sh build` image is faster (Inferred).
- Image sizes: Keycloak 474 MB; Zitadel 181 MB plus 208 MB for the login image.
