# Pagination cursor keyring

List endpoints (for example `GET /api/v1/entity-types`) sign their cursors with a keyring
read from `NAVA_CURSOR_KEYS` (comma-separated `id=secret`, secret base64url without padding,
at least 32 random bytes) and `NAVA_CURSOR_ACTIVE_KEY_ID` (the key that signs new cursors).

## Unset keyring

If `NAVA_CURSOR_KEYS` is unset, `nava-core` still starts, but every list endpoint answers
`501 not_implemented`. This is deliberate: failing at start-up would change behaviour for
every existing deploy. Set the keyring in every environment that serves list routes.

## Local compose stack

`compose.yaml` passes a dev-only keyring (`dev1`, not a secret) to `nava-core`. Override
both variables in `.env`. Never use the default outside a local stack.

## Rotation

Add the new key to `NAVA_CURSOR_KEYS`, set it active, and drop the old key later.
