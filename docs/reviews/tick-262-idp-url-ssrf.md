# Review: validate URL values in tenant IdP specs (TICK-262, PR #183)

- Reviewer: Quality, Testing & CI Advisor
- Head reviewed: `3b776fb`
- CI: **red** at `3b776fb` ([run 40472018768](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40472018768)). 7 of 8 jobs pass (`changes`, `lint`, `unit`, `integration`, `temporal-server`, `stack-db`, `stack-internal`). `stack` fails in `make stack-test-identity`.
- Specs: TICK-262 ask and acceptance criteria, the TICK-203 inventory, `docs/architecture/identity-port.md`.
- Out of scope: the SSRF threat model as a whole. That review belongs to the Security & Multi-Tenancy Reviewer. Finding 2 is routed to them as well.

## Verdict

Request changes. Finding 1 blocks the merge because CI is red. Finding 2 should be fixed in the same PR, since it is a gap in the check this PR adds.

## Findings

### 1. Bug: stack identity tests fail; their fixtures are now refused

`internal/identitystack/directory_test.go:568` `oidcSpec` uses `http://localhost:18081/...` and `http://localhost:8080/...`, so `CreateTenantOrg` now returns `ErrInvalidInput`:

```
directory_test.go:600: CreateTenantOrg: directory: invalid input: idp config key "authorizationUrl": url refused
--- FAIL: TestCreateTenantOrg
directory_test.go:662: CreateTenantOrg over a linked idp = directory: invalid input: idp config key "tokenUrl": url refused
--- FAIL: TestCreateTenantOrgRefusesExistingLinkedIdP
```

Neither test logs in through the created IdP. They only read back its config, so a public-looking https fixture such as `https://idp-tenant-a.example.com/realms/idp-tenant-a/protocol/openid-connect/auth` is enough.

There is a second failure behind the first `t.Fatalf`. The "bad config on an existing alias" case at line 666 uses `"authorizationUrl": "not a url"`. That value now fails in `checkIdPConfig`, before `readIdP`, so the test gets `ErrInvalidInput` rather than `ErrAlreadyExists`. It also stops proving what it was written for: that a config Keycloak answers with 400 does not delete a live IdP. Keep a config that passes the local check but that Keycloak rejects (for example a valid https `authorizationUrl` with `pkceMethod: "bogus"`). Then confirm in the job log that Keycloak still answers 400 for it. If it does not, the ordering guard has lost its test.

Both are test fixtures, and both are under `internal/identitystack/`, which is a security review path.

### 2. Breaks under a glibc resolver: hex IPv4 forms with a-f digits get through

`allDigitsAndDots` (`internal/identity/keycloak/idpurl.go:79`) accepts only `0-9`, `x` and `X` in the last label, so a hex last label containing `a` to `f` passes. Checked against the PR's own `checkIdPURL`:

| Input | Result | What `inet_aton` makes of it |
|---|---|---|
| `https://0x7f.1/` | refused | 127.0.0.1 |
| `https://0x7f.0xa/` | **accepted** | 127.0.0.10 |
| `https://10.0xa/` | **accepted** | 10.0.0.10 |
| `https://0xa.0.0.0xa/` | **accepted** | 10.0.0.10 |

Java's `InetAddress` does not parse these forms as literals, so it passes them to the system resolver. glibc `getaddrinfo` accepts `inet_aton` forms, and the Keycloak image is UBI, which uses glibc. So these probably reach loopback and RFC 1918 hosts. I have not run this against a live Keycloak; the egress policy remains the backstop.

Fix: treat the host as an IP literal when its last label matches `^(0[xX][0-9a-fA-F]*|[0-9]+)$`. No public TLD has that form. Missing test: add `https://0x7f.0xa/` and `https://10.0xa/` to the `bad` list in `TestIdPConfigRefusesUnsafeURLs`.

### 3. Preference: IPv4-compatible IPv6 is not blocked

`https://[::127.0.0.1]/` is accepted, because `Unmap` only strips `::ffff:`. Current Linux kernels do not route `::/96`, so the risk is low. Adding `::/96` to `blockedPrefixes` costs one line.

### 4. Preference: the function name does not say what it does

`allDigitsAndDots` inspects only the last label, and it also allows `x`. A name like `lastLabelIsNumeric` would be accurate.

## What is right

- `checkIdPConfig` runs before `readIdP` and before any admin write (`directory.go:729`), so a refusal leaves nothing behind.
- `idpURLKeys` covers every URL-valued key in both allow-lists (`oidcConfigKeys` and `samlConfigKeys`).
- The unit test runs the full bad list across every URL key, including the two from the ticket (`http://169.254.169.254/` and `https://10.0.0.5/`).
- The doc states that resolution-time checks are not possible here and that the egress policy is required, which is what the ticket asked for.
- The realm import (`deploy/compose/keycloak/import/realm-nava.json`) still uses `http://localhost` URLs for `idp-tenant-a`. It does not go through the adapter, so this PR leaves it alone, which is correct.

## Acceptance criteria

| Criterion | Evidence | Result |
|---|---|---|
| Adversarial test refuses `jwksUrl: http://169.254.169.254/` and `https://10.0.0.5/` before any admin call | `TestIdPConfigRefusesUnsafeURLs`, CI `unit` | pass |
| `task test` and `task lint` green | CI `unit` and `lint` pass; `stack` fails (finding 1) | **fail** |
| Lands before a tenant-facing onboarding endpoint feeds `IdPSource` | No such endpoint on `main` | n/a until merged |

## How to verify

Push the fix to the PR branch and read `gh pr checks 183`. The `stack` job runs `make stack-test-identity`.
