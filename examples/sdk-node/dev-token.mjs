// Dev helper for the S25 acceptance kit: a password-grant access token for a user seeded by
// docs/acceptance/guide/tools/seed.sh, from the local stack's Keycloak (realm nava, client
// nava-web), the same way the acceptance guide signs in. Local stack only: the passwords are
// the realm's dev-only seed values (also in sdk/ts/test/integration/stack.ts), never real ones.
// The token is cached until shortly before it expires; it is never printed.

const KEYCLOAK = process.env.NAVA_KEYCLOAK_URL ?? "http://127.0.0.1:18081";
const DEV_PASSWORDS = {
  "alice@tenant-a.test": process.env.NAVA_ALICE_PASSWORD ?? "dev-only-alice",
  "admin@tenant-a.test": process.env.NAVA_ADMIN_A_PASSWORD ?? "dev-only-admin-a",
  "bob@tenant-b.test": process.env.NAVA_BOB_PASSWORD ?? "dev-only-bob",
};
const cache = new Map();

export async function devToken(username) {
  const hit = cache.get(username);
  if (hit && hit.until > Date.now()) return hit.token;
  const res = await fetch(`${KEYCLOAK}/realms/nava/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ grant_type: "password", client_id: "nava-web", scope: "openid", username, password: DEV_PASSWORDS[username] ?? "" }),
  });
  if (res.status !== 200) throw new Error(`dev token for ${username}: Keycloak answered HTTP ${res.status}`);
  const { access_token: token, expires_in: ttl } = await res.json();
  cache.set(username, { token, until: Date.now() + Math.max(0, (ttl ?? 60) - 15) * 1000 });
  return token;
}
