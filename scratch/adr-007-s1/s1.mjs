// ADR-007 spike S1, browser half (RFC-017 §7 S1). Needs run-keycloak.sh.
//
// A browser-only host app (public client spike-host, PKCE S256) at
// http://127.0.0.1:18282 signs users in with a top-level redirect to the
// scratch Keycloak at http://localhost:18181 and exchanges the code in the
// page (CORS via webOrigins). 127.0.0.1 and localhost are different sites,
// so Keycloak's cookies are third-party to the host page.
//
// A "prompt" is any document the browser commits on a Keycloak origin: a
// sign-in that is pure redirects commits only /start and /callback on the
// host. Every scenario records the committed URLs.
//
// Writes out/tokens.json (raw tokens, gitignored) for resolve_test.go and
// out/report.json (decoded claims, prompts) for the evidence page.
//
//   node s1.mjs
import { chromium, webkit } from 'playwright';
import http from 'node:http';
import fs from 'node:fs';

const HOST = 'http://127.0.0.1:18282';
const XSITE = 'http://localhost:18283';
const KC = 'http://localhost:18181';
const REALM = `${KC}/realms/spike-s1`;
const PASSWORD = 'dev-only-spike';

// The host page. /start builds the PKCE pair in the browser and leaves with
// location.assign (top-level navigation, no iframe, no popup). /callback
// redeems the code with fetch and publishes the token in the DOM.
const startPage = `<!doctype html><meta charset=utf-8><title>host</title><script type=module>
const q = new URLSearchParams(location.search);
const b64 = (a) => btoa(String.fromCharCode(...new Uint8Array(a))).replace(/\\+/g,'-').replace(/\\//g,'_').replace(/=+$/,'');
const verifier = b64(crypto.getRandomValues(new Uint8Array(32)));
const challenge = b64(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier)));
const state = b64(crypto.getRandomValues(new Uint8Array(16)));
sessionStorage.setItem('pkce', JSON.stringify({verifier, state}));
const u = new URL('${REALM}/protocol/openid-connect/auth');
u.search = new URLSearchParams({client_id:'spike-host', response_type:'code', redirect_uri:'${HOST}/callback',
  scope:q.get('scope'), state, code_challenge:challenge, code_challenge_method:'S256',
  ...(q.get('hint') ? {kc_idp_hint:q.get('hint')} : {})});
location.assign(u);
</script>`;
const callbackPage = `<!doctype html><meta charset=utf-8><title>callback</title><pre id=out></pre><script type=module>
const q = new URLSearchParams(location.search);
const {verifier, state} = JSON.parse(sessionStorage.getItem('pkce'));
let r;
if (q.get('error')) r = {error:q.get('error'), error_description:q.get('error_description')};
else if (q.get('state') !== state) r = {error:'state_mismatch'};
else {
  try {
    const res = await fetch('${REALM}/protocol/openid-connect/token', {method:'POST',
      body:new URLSearchParams({grant_type:'authorization_code', client_id:'spike-host', code:q.get('code'),
        redirect_uri:'${HOST}/callback', code_verifier:verifier})});
    r = await res.json();
  } catch (err) { r = {error:'fetch_failed', error_description:String(err)}; }
}
document.getElementById('out').dataset.done = '1';
document.getElementById('out').textContent = JSON.stringify(r);
</script>`;

function serve() {
  const host = http.createServer((req, res) => {
    const p = new URL(req.url, HOST).pathname;
    if (p === '/start') return res.writeHead(200, { 'content-type': 'text/html' }).end(startPage);
    if (p === '/callback') return res.writeHead(200, { 'content-type': 'text/html' }).end(callbackPage);
    if (p === '/embed') return res.writeHead(200, { 'content-type': 'text/html' }).end(`<iframe src="${XSITE}/read"></iframe>`);
    res.writeHead(404).end();
  });
  // A second site, to prove the engine really blocks third-party cookies.
  let sawCookieInIframe = null;
  const xsite = http.createServer((req, res) => {
    const p = new URL(req.url, XSITE).pathname;
    if (p === '/set') return res.writeHead(200, { 'set-cookie': 'probe=1; Path=/; SameSite=None; Secure', 'content-type': 'text/html' }).end('set');
    if (p === '/read') { sawCookieInIframe = /probe=1/.test(req.headers.cookie || ''); return res.writeHead(200).end('read'); }
    res.writeHead(404).end();
  });
  host.listen(18282, '127.0.0.1');
  xsite.listen(18283, '127.0.0.1');
  return { close: () => { host.close(); xsite.close(); }, sawCookie: () => sawCookieInIframe };
}

const decode = (jwt) => JSON.parse(Buffer.from(jwt.split('.')[1], 'base64url').toString());

// signIn runs one top-level sign-in. Every document a Keycloak origin renders
// is recorded as a prompt with its realm and heading. With answer, the driver
// answers it (upstream password, organization picker, username) as a user
// would; without, it stops at the first prompt and no token is issued.
async function signIn(page, { scope, hint, user, pickOrg, answer = true }) {
  const committed = [];
  const onNav = (f) => { if (f === page.mainFrame()) committed.push(f.url()); };
  page.on('framenavigated', onNav);
  const qs = new URLSearchParams({ scope, ...(hint ? { hint } : {}) });
  await page.goto(`${HOST}/start?${qs}`);
  const prompts = [];
  let result = null;
  for (let i = 0; i < 6 && !result; i++) {
    await page.waitForURL((u) => u.href.startsWith(`${HOST}/callback`) || u.href.startsWith(KC), { timeout: 15000 });
    await page.waitForLoadState('load');
    const url = page.url();
    if (url.startsWith(`${HOST}/callback`)) {
      await page.locator('#out[data-done="1"]').waitFor({ state: 'attached', timeout: 15000 });
      result = JSON.parse(await page.locator('#out').textContent());
      break;
    }
    const kind = await classify(page);
    const heading = ((await page.locator('#kc-page-title, h1').first().textContent().catch(() => '')) || '').trim();
    const alert = ((await page.locator('.pf-v5-c-alert__title, .alert-error, #input-error').first().textContent({ timeout: 500 }).catch(() => '')) || '').trim();
    prompts.push({ kind, realm: url.match(/realms\/([^/]+)/)?.[1], path: new URL(url).pathname.replace(/^\/realms\/[^/]+/, ''), heading, ...(alert ? { alert } : {}) });
    if (!answer) { result = { error: 'stopped_at_prompt' }; break; }
    const nav = page.waitForNavigation({ timeout: 15000 });
    if (kind === 'organization-picker') {
      await page.getByText(pickOrg, { exact: true }).first().click();
    } else if (kind === 'username') {
      await page.locator('#username').fill(user);
      await page.locator('#kc-login, [type=submit]').first().click();
    } else if (kind === 'username-password' || kind === 'password') {
      if (kind === 'username-password') await page.locator('#username').fill(user);
      await page.locator('#password').fill(PASSWORD);
      await page.locator('#kc-login, [type=submit]').first().click();
    } else {
      throw new Error(`unexpected page ${url}: ${(await page.locator('body').innerText()).slice(0, 1500)}`);
    }
    await nav;
  }
  if (!result) throw new Error(`no result at ${page.url()} prompts=${JSON.stringify(prompts)}`);
  page.off('framenavigated', onNav);
  const kcCommitted = committed.filter((u) => u.startsWith(KC)).map((u) => new URL(u).pathname);
  return { result, prompts, kcCommitted };
}

async function classify(page) {
  const hasUser = (await page.locator('#username').count()) > 0;
  const hasPass = (await page.locator('#password').count()) > 0;
  if (hasUser && hasPass) return 'username-password';
  if (hasUser) return 'username';
  if (hasPass) return 'password';
  if (/organization/i.test(await page.locator('body').innerText())) return 'organization-picker';
  return 'unknown';
}

// Drops the spike-s1 realm's own session cookies and keeps the upstream
// realm's: what a browser holds when the brokered IdP session outlives the
// Keycloak one.
async function dropBrokerSession(ctx) {
  const keep = (await ctx.cookies()).filter((c) => !c.path.startsWith('/realms/spike-s1'));
  await ctx.clearCookies();
  await ctx.addCookies(keep);
}

// Chromium 159 ignores --test-third-party-cookie-phaseout and profile
// preferences under Playwright; the DevTools cookie controls do block third-
// party cookies, which the probe below confirms on every run. WebKit blocks
// them by default (ITP).
const engines = [
  { name: 'chromium-3pc-blocked', type: chromium, blockThirdPartyCookies: true },
  { name: 'webkit', type: webkit },
];

async function newPage(e, ctx) {
  const page = await ctx.newPage();
  if (e.blockThirdPartyCookies) {
    const cdp = await ctx.newCDPSession(page);
    await cdp.send('Network.enable');
    await cdp.send('Network.setCookieControls', { enableThirdPartyCookieRestriction: true, disableThirdPartyCookieMetadata: true, disableThirdPartyCookieHeuristics: true });
  }
  return page;
}

const srv = serve();
const tokens = [];
const report = { engines: {} };
const record = (engine, user, step, scope, hint, r, expect) => {
  const entry = { step, scope, hint: hint || null, prompts: r.prompts, keycloakDocumentsCommitted: r.kcCommitted };
  if (r.result.access_token) {
    entry.claims = decode(r.result.access_token);
    tokens.push({ engine, user, step, scope, expect, token: r.result.access_token });
  } else entry.error = r.result;
  report.engines[engine].push({ user, ...entry });
  console.log(engine, user, step, `scope="${scope}"`, hint ? `hint=${hint}` : '', 'prompts:', JSON.stringify(r.prompts.map((p) => p.kind)), r.result.error || 'token');
};

try {
  for (const e of engines) {
    const browser = await e.type.launch();
    report.engines[e.name] = [];
    report[`${e.name}:version`] = browser.version();

    // Third-party cookie probe: set a SameSite=None cookie first-party on
    // XSITE, then load XSITE in an iframe under HOST.
    {
      const ctx = await browser.newContext();
      const p = await newPage(e, ctx);
      await p.goto(`${XSITE}/set`);
      await p.goto(`${HOST}/embed`);
      await p.waitForTimeout(1000);
      report[`${e.name}:thirdPartyCookieSentToIframe`] = srv.sawCookie();
      await ctx.close();
    }

    // Single-org user.
    {
      const ctx = await browser.newContext();
      const page = await newPage(e, ctx);
      const u = 'one@org-a.test';
      let r = await signIn(page, { scope: 'openid organization', hint: 'upstream', user: u });
      record(e.name, u, 'first sign-in (establishes the brokered IdP session)', 'openid organization', 'upstream', r, 'A');
      r = await signIn(page, { scope: 'openid organization', user: u });
      record(e.name, u, 'top-level redirect, Keycloak SSO session', 'openid organization', null, r, 'A');
      r = await signIn(page, { scope: 'openid organization:org-a', user: u });
      record(e.name, u, 'top-level redirect, Keycloak SSO session', 'openid organization:org-a', null, r, 'A');
      r = await signIn(page, { scope: 'openid organization:org-b', user: u, answer: false });
      record(e.name, u, 'top-level redirect, alias of an org the user is not in', 'openid organization:org-b', null, r, '401');
      await dropBrokerSession(ctx);
      r = await signIn(page, { scope: 'openid organization', hint: 'upstream', user: u });
      record(e.name, u, 'top-level redirect, upstream IdP session only (Keycloak session cookies dropped)', 'openid organization', 'upstream', r, 'A');
      await dropBrokerSession(ctx);
      r = await signIn(page, { scope: 'openid organization', user: u, answer: false });
      record(e.name, u, 'top-level redirect, upstream IdP session only, no kc_idp_hint', 'openid organization', null, r, 'A');
      await ctx.close();
    }

    // Two-org user.
    {
      const ctx = await browser.newContext();
      const page = await newPage(e, ctx);
      const u = 'multi@org-a.test';
      let r = await signIn(page, { scope: 'openid organization', hint: 'upstream', user: u, pickOrg: 'org-b' });
      record(e.name, u, 'first sign-in, picker expected, pick org-b', 'openid organization', 'upstream', r, 'B');
      r = await signIn(page, { scope: 'openid organization', user: u, pickOrg: 'org-a' });
      record(e.name, u, 'top-level redirect, Keycloak SSO session, picker expected, pick org-a', 'openid organization', null, r, 'A');
      r = await signIn(page, { scope: 'openid organization:org-b', user: u });
      record(e.name, u, 'top-level redirect, Keycloak SSO session, alias hint', 'openid organization:org-b', null, r, 'B');
      r = await signIn(page, { scope: 'openid organization:org-c', user: u, answer: false });
      record(e.name, u, 'top-level redirect, alias of an org the user is not in', 'openid organization:org-c', null, r, '401');
      await ctx.close();
    }

    // Two-org user, first sign-in naming the organization (D1: a host
    // serving many tenants sends organization:<alias>).
    {
      const ctx = await browser.newContext();
      const page = await newPage(e, ctx);
      const u = 'multi@org-a.test';
      const r = await signIn(page, { scope: 'openid organization:org-b', hint: 'upstream', user: u });
      record(e.name, u, 'first sign-in, alias hint', 'openid organization:org-b', 'upstream', r, 'B');
      await ctx.close();
    }
    await browser.close();
  }
} finally {
  srv.close();
}
fs.mkdirSync('out', { recursive: true });
fs.writeFileSync('out/tokens.json', JSON.stringify(tokens, null, 1));
fs.writeFileSync('out/report.json', JSON.stringify(report, null, 1));
console.log('wrote out/tokens.json and out/report.json');
