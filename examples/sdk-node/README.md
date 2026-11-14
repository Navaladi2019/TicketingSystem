# @nava/sdk Node example (S25)

A Node 20+ project that installs `@nava/sdk` from the packed tarball and talks to the local stack.

```sh
task sdk:build                 # builds sdk/ts and packs sdk/ts/nava-sdk-<version>.tgz
cd examples/sdk-node && npm install
node list-read-create.mjs      # list every incident (all pages), create one, read it back
node criteria.mjs              # the S25 acceptance checks, as tenants A and B
```

`dev-token.mjs` signs in seeded users to the local Keycloak with the realm's dev-only passwords; it is for the local stack only. The full walk and its evidence are in `docs/acceptance/S25/`.
